package main

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "session"
	csrfCookieName    = "csrf"
	csrfFieldName     = "csrf_token"
	sessionDuration   = 24 * time.Hour
)

var (
	adminUsername string
	adminPassword string
	secureCookies bool
)

func initAuth() {
	adminUsername = os.Getenv("ADMIN_USER")
	if adminUsername == "" {
		adminUsername = "admin"
	}

	pass := os.Getenv("ADMIN_PASS")
	if pass == "" {
		log.Println("WARNING: ADMIN_PASS not set, using default password")
		pass = "password"
	}
	adminPassword = mustHashPassword(pass)

	secureCookies = os.Getenv("SECURE_COOKIES") == "true"
}

func mustHashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

func checkPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func createSession(db *sql.DB, userID int) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(sessionDuration)
	_, err = db.Exec(`
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)`, token, userID, expiresAt)
	if err != nil {
		return "", err
	}

	return token, nil
}

func getSession(db *sql.DB, token string) (*Session, error) {
	row := db.QueryRow(`
		SELECT token, user_id, expires_at
		FROM sessions
		WHERE token = ? AND expires_at > ?`, token, time.Now())

	var session Session
	err := row.Scan(&session.Token, &session.UserID, &session.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func deleteSession(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func cleanupExpiredSessions(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

// CSRF protection using double-submit cookie pattern

func generateCSRFToken() (string, error) {
	return generateToken()
}

func setCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // JS needs to read this if doing AJAX (not needed here, but standard)
		Secure:   secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})
}

func getCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func validateCSRF(r *http.Request) bool {
	cookieToken := getCSRFToken(r)
	formToken := r.FormValue(csrfFieldName)

	if cookieToken == "" || formToken == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(formToken)) == 1
}

// ensureCSRFToken returns existing token or creates a new one
func ensureCSRFToken(w http.ResponseWriter, r *http.Request) string {
	token := getCSRFToken(r)
	if token != "" {
		return token
	}

	token, err := generateCSRFToken()
	if err != nil {
		return ""
	}
	setCSRFCookie(w, token)
	return token
}

// requireAuth is middleware that protects routes requiring authentication
func (b *Blog) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		session, err := getSession(b.db, cookie.Value)
		if err != nil || session == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

// isAuthenticated checks if the current request has a valid session
func (b *Blog) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}

	session, err := getSession(b.db, cookie.Value)
	return err == nil && session != nil
}

func (b *Blog) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := map[string]any{
			"Title":     "Quiet Nothings",
			"CSRFToken": ensureCSRFToken(w, r),
		}
		err := b.templates["login.html"].ExecuteTemplate(w, "base", data)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		if subtle.ConstantTimeCompare([]byte(username), []byte(adminUsername)) != 1 || !checkPassword(adminPassword, password) {
			data := map[string]any{
				"Title":     "Quiet Nothings",
				"Error":     "Invalid username or password",
				"CSRFToken": getCSRFToken(r),
			}
			w.WriteHeader(http.StatusUnauthorized)
			err := b.templates["login.html"].ExecuteTemplate(w, "base", data)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		token, err := createSession(b.db, 1) // userID 1 for admin
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   secureCookies,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(sessionDuration.Seconds()),
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (b *Blog) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		deleteSession(b.db, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
