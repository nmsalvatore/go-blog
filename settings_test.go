package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGetSetting(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}

	// Insert a setting
	_, err = db.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", "test_key", "test_value")
	if err != nil {
		t.Fatalf("inserting test setting: %v", err)
	}

	value, err := getSetting(db, "test_key")
	if err != nil {
		t.Fatalf("getSetting() error: %v", err)
	}

	if value != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", value)
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}

	value, err := getSetting(db, "nonexistent")
	if err != nil {
		t.Fatalf("getSetting() error: %v", err)
	}

	if value != "" {
		t.Errorf("expected empty string for nonexistent key, got '%s'", value)
	}
}

func TestSetSetting_Insert(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}

	err = setSetting(db, "new_key", "new_value")
	if err != nil {
		t.Fatalf("setSetting() error: %v", err)
	}

	value, err := getSetting(db, "new_key")
	if err != nil {
		t.Fatalf("getSetting() error: %v", err)
	}
	if value != "new_value" {
		t.Errorf("expected 'new_value', got '%s'", value)
	}
}

func TestSetSetting_Update(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}

	// Insert initial value
	err = setSetting(db, "key", "initial")
	if err != nil {
		t.Fatalf("setSetting() initial error: %v", err)
	}

	// Update the value
	err = setSetting(db, "key", "updated")
	if err != nil {
		t.Fatalf("setSetting() update error: %v", err)
	}

	value, err := getSetting(db, "key")
	if err != nil {
		t.Fatalf("getSetting() error: %v", err)
	}
	if value != "updated" {
		t.Errorf("expected 'updated', got '%s'", value)
	}
}

func TestSettings_GET(t *testing.T) {
	blog := setupTestBlog(t)

	// Seed the intro setting
	setSetting(blog.db, "intro", "Test intro text")

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	blog.Settings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test intro text") {
		t.Error("expected response to contain 'Test intro text'")
	}
	if !strings.Contains(body, "Settings") {
		t.Error("expected response to contain 'Settings'")
	}
}

func TestSettings_POST(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("intro", "Updated intro text")

	req := httptest.NewRequest(http.MethodPost, "/settings", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Settings(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	// Verify setting was updated
	value, _ := getSetting(blog.db, "intro")
	if value != "Updated intro text" {
		t.Errorf("expected 'Updated intro text', got '%s'", value)
	}
}

func TestSettings_POST_NoCSRF(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("intro", "Updated intro text")

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Settings(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestHome_ShowsIntroFromSettings(t *testing.T) {
	blog := setupTestBlog(t)

	// Set a custom intro
	setSetting(blog.db, "intro", "Custom intro from settings")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	blog.Home(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Custom intro from settings") {
		t.Error("expected response to contain 'Custom intro from settings'")
	}
}

func TestSettings_ThemeAndFont_POST(t *testing.T) {
	blog := setupTestBlog(t)

	form := url.Values{}
	form.Set("intro", "")
	form.Set("theme", "dark")
	form.Set("font", "inter")

	req := httptest.NewRequest(http.MethodPost, "/settings", nil)
	addCSRFToken(req, form)
	req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	blog.Settings(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	// Verify settings were saved
	theme, _ := getSetting(blog.db, "theme")
	if theme != "dark" {
		t.Errorf("expected theme 'dark', got '%s'", theme)
	}

	font, _ := getSetting(blog.db, "font")
	if font != "inter" {
		t.Errorf("expected font 'inter', got '%s'", font)
	}
}

func TestSettings_GET_ShowsThemeAndFont(t *testing.T) {
	blog := setupTestBlog(t)

	// Set theme and font
	setSetting(blog.db, "theme", "dark")
	setSetting(blog.db, "font", "serif")

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	blog.Settings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	// Check that dark theme radio is checked
	if !strings.Contains(body, `name="theme" value="dark" checked`) {
		t.Error("expected dark theme radio to be checked")
	}
	// Check that serif font radio is checked
	if !strings.Contains(body, `name="font" value="serif" checked`) {
		t.Error("expected serif font radio to be checked")
	}
}

func TestHome_IncludesThemeAndFontAttributes(t *testing.T) {
	blog := setupTestBlog(t)

	setSetting(blog.db, "theme", "dark")
	setSetting(blog.db, "font", "inter")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	blog.Home(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `data-theme="dark"`) {
		t.Error("expected body to contain data-theme=\"dark\"")
	}
	if !strings.Contains(body, `data-font="inter"`) {
		t.Error("expected body to contain data-font=\"inter\"")
	}
}
