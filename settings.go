package main

import (
	"database/sql"
	"net/http"
)

func getSetting(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

func (b *Blog) Settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		intro, err := getSetting(b.db, "intro")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Title":           "Settings",
			"Intro":           intro,
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
		}
		err = b.templates["settings.html"].ExecuteTemplate(w, "base", data)
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

		intro := r.FormValue("intro")
		err := setSetting(b.db, "intro", intro)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
