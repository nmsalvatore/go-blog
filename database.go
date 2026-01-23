package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func initDB(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		published BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	if err := migrateDB(db); err != nil {
		return err
	}

	return nil
}

func migrateDB(db *sql.DB) error {
	// Check if published column exists
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='published'`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = db.Exec(`ALTER TABLE posts ADD COLUMN published BOOLEAN NOT NULL DEFAULT 1`)
		if err != nil {
			return err
		}
	}

	return nil
}

func seedDB(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	posts := []Post{
		{Title: "Hey now", Content: "Everything is awesome!", Published: true},
		{Title: "What's the deal?", Content: "What is happening?!", Published: true},
		{Title: "Football", Content: "Niners and stuff.", Published: true},
	}

	stmt := "INSERT INTO posts (title, content, published) VALUES (?, ?, ?)"
	for _, post := range posts {
		_, err := db.Exec(stmt, post.Title, post.Content, post.Published)
		if err != nil {
			return err
		}
	}

	fmt.Println("successfully seeded test data")
	return nil
}

func seedSettings(db *sql.DB) error {
	// Seed default intro text if not exists
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings WHERE key = 'intro'").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaultIntro := "Lorem ipsum dolor sit amet consectetur adipisicing elit. Dicta incidunt ipsa numquam impedit nostrum, ut cum a autem soluta animi, error, ea tenetur?"
	_, err := db.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", "intro", defaultIntro)
	return err
}
