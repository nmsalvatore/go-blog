package main

import (
	"testing"
)

func TestOpenDB(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("db.Ping() error: %v", err)
	}
}

func TestInitDB(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("initDB() error: %v", err)
	}

	// Verify posts table exists with correct columns
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name IN ('id', 'title', 'content', 'published', 'created_at')`).Scan(&count)
	if err != nil {
		t.Fatalf("querying posts schema: %v", err)
	}
	if count != 5 {
		t.Errorf("posts table: expected 5 columns, got %d", count)
	}

	// Verify sessions table exists
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions')`).Scan(&count)
	if err != nil {
		t.Fatalf("querying sessions schema: %v", err)
	}
	if count != 3 {
		t.Errorf("sessions table: expected 3 columns, got %d", count)
	}

	// Verify settings table exists
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings')`).Scan(&count)
	if err != nil {
		t.Fatalf("querying settings schema: %v", err)
	}
	if count != 2 {
		t.Errorf("settings table: expected 2 columns, got %d", count)
	}
}

func TestInitDB_Idempotent(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	// Call initDB twice - should not error
	if err := initDB(db); err != nil {
		t.Fatalf("first initDB() error: %v", err)
	}
	if err := initDB(db); err != nil {
		t.Fatalf("second initDB() error: %v", err)
	}
}

func TestMigrateDB_AddsPublishedColumn(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	// Create posts table WITHOUT published column (old schema)
	_, err = db.Exec(`
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("creating old schema: %v", err)
	}

	// Run migration
	if err := migrateDB(db); err != nil {
		t.Fatalf("migrateDB() error: %v", err)
	}

	// Verify published column was added
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='published'`).Scan(&count)
	if err != nil {
		t.Fatalf("checking published column: %v", err)
	}
	if count != 1 {
		t.Error("published column was not added by migration")
	}
}

func TestSeedDB(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("initDB() error: %v", err)
	}

	if err := seedDB(db); err != nil {
		t.Fatalf("seedDB() error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count); err != nil {
		t.Fatalf("counting posts: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 seeded posts, got %d", count)
	}
}

func TestSeedDB_SkipsWhenDataExists(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("initDB() error: %v", err)
	}

	// Insert existing post
	_, err = db.Exec("INSERT INTO posts (title, content, published) VALUES (?, ?, ?)", "Existing", "Content", true)
	if err != nil {
		t.Fatalf("inserting existing post: %v", err)
	}

	// Seed should skip
	if err := seedDB(db); err != nil {
		t.Fatalf("seedDB() error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count); err != nil {
		t.Fatalf("counting posts: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 post (seed skipped), got %d", count)
	}
}

func TestSeedSettings(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("initDB() error: %v", err)
	}

	if err := seedSettings(db); err != nil {
		t.Fatalf("seedSettings() error: %v", err)
	}

	value, err := getSetting(db, "intro")
	if err != nil {
		t.Fatalf("getSetting() error: %v", err)
	}
	if value == "" {
		t.Error("expected intro setting to be seeded")
	}
}

func TestSeedSettings_SkipsWhenExists(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB() error: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("initDB() error: %v", err)
	}

	// Set custom intro
	if err := setSetting(db, "intro", "Custom intro"); err != nil {
		t.Fatalf("setSetting() error: %v", err)
	}

	// Seed should skip
	if err := seedSettings(db); err != nil {
		t.Fatalf("seedSettings() error: %v", err)
	}

	value, err := getSetting(db, "intro")
	if err != nil {
		t.Fatalf("getSetting() error: %v", err)
	}
	if value != "Custom intro" {
		t.Errorf("expected 'Custom intro', got %q", value)
	}
}
