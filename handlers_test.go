package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func init() {
	// Initialize auth for tests (uses default admin/password)
	initAuth()
}

func setupTestBlog(t *testing.T) *Blog {
	t.Helper()
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return NewBlog(db)
}

// addCSRFToken adds a CSRF token to the request (cookie + form value)
func addCSRFToken(req *http.Request, form url.Values) {
	token := "test-csrf-token-12345"
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	if form != nil {
		form.Set(csrfFieldName, token)
	}
}

func TestHome(t *testing.T) {
	blog := setupTestBlog(t)

	// Seed a published post
	_, err := createPost(blog.db, "Test Post", "Test content", true)
	if err != nil {
		t.Fatalf("creating test post: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	blog.Home(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Post") {
		t.Error("expected response to contain 'Test Post'")
	}
}

func TestHome_NotFound(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	blog.Home(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDetail(t *testing.T) {
	blog := setupTestBlog(t)

	id, err := createPost(blog.db, "Detail Test", "Detail content", true)
	if err != nil {
		t.Fatalf("creating test post: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/post/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d (post id: %d)", http.StatusOK, w.Code, id)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Detail Test") {
		t.Error("expected response to contain 'Detail Test'")
	}
}

func TestDetail_NotFound(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/post/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDetail_InvalidID(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/post/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreate_GET(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	w := httptest.NewRecorder()

	blog.Create(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestCreate_POST(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("title", "New Post")
	form.Set("content", "New content")
	form.Set("action", "publish")

	req := httptest.NewRequest(http.MethodPost, "/new", nil) // body set after CSRF
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Create(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	// Verify post was created and published
	posts, _ := getPosts(blog.db)
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Title != "New Post" {
		t.Errorf("expected title 'New Post', got '%s'", posts[0].Title)
	}
	if !posts[0].Published {
		t.Error("expected post to be published")
	}
}

func TestCreate_POST_MissingTitle(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("content", "Some content")

	req := httptest.NewRequest(http.MethodPost, "/new", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreate_POST_NoCSRF(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("title", "New Post")
	form.Set("content", "New content")

	req := httptest.NewRequest(http.MethodPost, "/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Create(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestEdit_POST(t *testing.T) {
	blog := setupTestBlog(t)

	_, err := createPost(blog.db, "Original", "Original content", true)
	if err != nil {
		t.Fatalf("creating test post: %v", err)
	}

	form := url.Values{}
	form.Set("title", "Updated")
	form.Set("content", "Updated content")
	form.Set("action", "publish")

	req := httptest.NewRequest(http.MethodPost, "/edit/1", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	blog.Edit(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	// Verify post was updated
	post, _ := getPostByID(blog.db, 1)
	if post.Title != "Updated" {
		t.Errorf("expected title 'Updated', got '%s'", post.Title)
	}
}

func TestDelete_POST(t *testing.T) {
	blog := setupTestBlog(t)

	_, err := createPost(blog.db, "To Delete", "Content", true)
	if err != nil {
		t.Fatalf("creating test post: %v", err)
	}

	form := url.Values{}
	req := httptest.NewRequest(http.MethodPost, "/delete/1", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	blog.Delete(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	// Verify post was deleted
	post, _ := getPostByID(blog.db, 1)
	if post != nil {
		t.Error("expected post to be deleted")
	}
}

func TestCreate_POST_Draft(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("title", "Draft Post")
	form.Set("content", "Draft content")
	form.Set("action", "draft")

	req := httptest.NewRequest(http.MethodPost, "/new", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Create(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	posts, _ := getPosts(blog.db)
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Published {
		t.Error("expected post to be a draft")
	}
}

func TestDetail_Draft_Unauthenticated(t *testing.T) {
	blog := setupTestBlog(t)

	// Create a draft post
	createPost(blog.db, "Draft Post", "Draft content", false)

	req := httptest.NewRequest(http.MethodGet, "/post/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for draft without auth, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDetail_Draft_Authenticated(t *testing.T) {
	blog := setupTestBlog(t)

	// Create a draft post
	createPost(blog.db, "Draft Post", "Draft content", false)

	// Create a session for authentication
	token, _ := createSession(blog.db, 1)

	req := httptest.NewRequest(http.MethodGet, "/post/1", nil)
	req.SetPathValue("id", "1")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for draft with auth, got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Draft Post") {
		t.Error("expected response to contain 'Draft Post'")
	}
}

func TestHome_HidesDraftsFromUnauthenticated(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "Published Post", "Content", true)
	createPost(blog.db, "Draft Post", "Content", false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	blog.Home(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Published Post") {
		t.Error("expected response to contain 'Published Post'")
	}
	if strings.Contains(body, "Draft Post") {
		t.Error("expected response NOT to contain 'Draft Post' for unauthenticated user")
	}
}

func TestHome_ShowsDraftsToAuthenticated(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "Published Post", "Content", true)
	createPost(blog.db, "Draft Post", "Content", false)

	token, _ := createSession(blog.db, 1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w := httptest.NewRecorder()

	blog.Home(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Published Post") {
		t.Error("expected response to contain 'Published Post'")
	}
	if !strings.Contains(body, "Draft Post") {
		t.Error("expected response to contain 'Draft Post' for authenticated user")
	}
}

func TestEdit_ConvertToDraft(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "Published", "Content", true)

	form := url.Values{}
	form.Set("title", "Published")
	form.Set("content", "Content")
	form.Set("action", "draft")

	req := httptest.NewRequest(http.MethodPost, "/edit/1", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	blog.Edit(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Published {
		t.Error("expected post to be converted to draft")
	}
}

func TestEdit_PublishDraft(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "Draft", "Content", false)

	form := url.Values{}
	form.Set("title", "Draft")
	form.Set("content", "Content")
	form.Set("action", "publish")

	req := httptest.NewRequest(http.MethodPost, "/edit/1", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	blog.Edit(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	post, _ := getPostByID(blog.db, 1)
	if !post.Published {
		t.Error("expected draft to be published")
	}
}
