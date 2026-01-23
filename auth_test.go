package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// addCSRFTokenAuth adds a CSRF token to the request (cookie + form value)
func addCSRFTokenAuth(req *http.Request, form url.Values) {
	token := "test-csrf-token-12345"
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	if form != nil {
		form.Set(csrfFieldName, token)
	}
}

func TestCheckPassword(t *testing.T) {
	hash := mustHashPassword("secret")

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"correct password", "secret", true},
		{"wrong password", "wrong", false},
		{"empty password", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkPassword(hash, tt.password)
			if got != tt.want {
				t.Errorf("checkPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() error: %v", err)
	}

	if len(token1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected token length 64, got %d", len(token1))
	}

	token2, _ := generateToken()
	if token1 == token2 {
		t.Error("expected unique tokens")
	}
}

func TestCreateAndGetSession(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}

	token, err := createSession(db, 1)
	if err != nil {
		t.Fatalf("createSession() error: %v", err)
	}

	session, err := getSession(db, token)
	if err != nil {
		t.Fatalf("getSession() error: %v", err)
	}

	if session == nil {
		t.Fatal("expected session, got nil")
	}

	if session.UserID != 1 {
		t.Errorf("expected UserID 1, got %d", session.UserID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}

	session, err := getSession(db, "nonexistent")
	if err != nil {
		t.Fatalf("getSession() error: %v", err)
	}

	if session != nil {
		t.Error("expected nil session for nonexistent token")
	}
}

func TestDeleteSession(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}

	token, _ := createSession(db, 1)
	err = deleteSession(db, token)
	if err != nil {
		t.Fatalf("deleteSession() error: %v", err)
	}

	session, _ := getSession(db, token)
	if session != nil {
		t.Error("expected session to be deleted")
	}
}

func TestLogin_GET(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	blog.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Login") {
		t.Error("expected login form in response")
	}
}

func TestLogin_POST_Success(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "password")

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	addCSRFTokenAuth(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Login(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	// Check for session cookie
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			found = true
			if c.Value == "" {
				t.Error("expected non-empty session cookie")
			}
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func TestLogin_POST_InvalidCredentials(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	addCSRFTokenAuth(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid") {
		t.Error("expected error message in response")
	}
}

func TestLogin_POST_NoCSRF(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "password")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Login(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRequireAuth_NoSession(t *testing.T) {
	blog := setupTestBlog(t)

	handlerCalled := false
	handler := blog.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if handlerCalled {
		t.Error("expected handler not to be called without auth")
	}

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect status %d, got %d", http.StatusSeeOther, w.Code)
	}

	if w.Header().Get("Location") != "/login" {
		t.Errorf("expected redirect to /login, got %s", w.Header().Get("Location"))
	}
}

func TestRequireAuth_ValidSession(t *testing.T) {
	blog := setupTestBlog(t)

	// Create a session
	token, _ := createSession(blog.db, 1)

	handlerCalled := false
	handler := blog.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w := httptest.NewRecorder()

	handler(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called with valid session")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLogout(t *testing.T) {
	blog := setupTestBlog(t)

	// Create a session first
	sessionToken, _ := createSession(blog.db, 1)

	form := url.Values{}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	addCSRFTokenAuth(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionToken})
	w := httptest.NewRecorder()

	blog.Logout(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	// Verify session was deleted
	session, _ := getSession(blog.db, sessionToken)
	if session != nil {
		t.Error("expected session to be deleted after logout")
	}

	// Check cookie was cleared
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == sessionCookieName && c.MaxAge != -1 {
			t.Error("expected session cookie to be cleared")
		}
	}
}
