package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/joho/godotenv"
)

type Blog struct {
	db        *sql.DB
	templates map[string]*template.Template
}

func linebreaks(s string) template.HTML {
	// Escape HTML first to prevent XSS, then add our formatting
	s = template.HTMLEscapeString(s)

	paragraphs := strings.Split(s, "\n\n")
	var result []string

	for _, p := range paragraphs {
		if p = strings.TrimSpace(p); p != "" {
			p = strings.ReplaceAll(p, "\n", "<br>")
			result = append(result, "<p>"+p+"</p>")
		}
	}

	return template.HTML(strings.Join(result, "\n"))
}

func loadTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	pages := []string{"home.html", "detail.html", "create.html", "edit.html", "delete.html", "settings.html", "login.html"}

	funcs := template.FuncMap{
		"linebreaks": linebreaks,
	}

	for _, page := range pages {
		templates[page] = template.Must(
			template.New("").Funcs(funcs).ParseFiles(
				"templates/base.html",
				"templates/"+page,
			))
	}

	return templates
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
		log.Fatalf("cleaning up expired sessions: %v", err)
	}

	blog := NewBlog(db)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	http.HandleFunc("/", blog.Home)
	http.HandleFunc("/post/{id}", blog.Detail)
	http.HandleFunc("/login", blog.Login)
	http.HandleFunc("/logout", blog.Logout)

	// Protected routes
	http.HandleFunc("/new", blog.requireAuth(blog.Create))
	http.HandleFunc("/edit/{id}", blog.requireAuth(blog.Edit))
	http.HandleFunc("/delete/{id}", blog.requireAuth(blog.Delete))
	http.HandleFunc("/settings", blog.requireAuth(blog.Settings))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
