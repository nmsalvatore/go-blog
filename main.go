package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

type Blog struct {
	db        *sql.DB
	templates map[string]*template.Template
}

func NewBlog(db *sql.DB) *Blog {
	return &Blog{
		db:        db,
		templates: loadTemplates(),
	}
}

func main() {
	godotenv.Load()

	initAuth()

	db, err := openDB("blog.db")
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	if err = initDB(db); err != nil {
		log.Fatalf("initializing database: %v", err)
	}

	if err = seedDB(db); err != nil {
		log.Fatalf("seeding database: %v", err)
	}

	if err = seedSettings(db); err != nil {
		log.Fatalf("seeding settings: %v", err)
	}

	if err = cleanupExpiredSessions(db); err != nil {
		log.Printf("cleaning up expired sessions: %v", err)
	}

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			if err := cleanupExpiredSessions(db); err != nil {
				log.Printf("cleaning up expired sessions: %v", err)
			}
		}
	}()

	blog := NewBlog(db)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	http.HandleFunc("/", blog.Home)
	http.HandleFunc("/post/{slug}", blog.Detail)
	http.HandleFunc("/feed", blog.Feed)
	http.HandleFunc("/admin", blog.Login)
	http.HandleFunc("/logout", blog.Logout)

	// Protected routes
	http.HandleFunc("/new", blog.requireAuth(blog.Create))
	http.HandleFunc("/edit/{id}", blog.requireAuth(blog.Edit))
	http.HandleFunc("/delete/{id}", blog.requireAuth(blog.Delete))
	http.HandleFunc("/settings", blog.requireAuth(blog.Settings))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
