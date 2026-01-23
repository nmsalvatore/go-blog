package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func (b *Blog) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	isAuth := b.isAuthenticated(r)

	var posts, drafts []Post
	var err error

	if isAuth {
		allPosts, err := getPosts(b.db)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		for _, p := range allPosts {
			if p.Published {
				posts = append(posts, p)
			} else {
				drafts = append(drafts, p)
			}
		}
	} else {
		posts, err = getPublishedPosts(b.db)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	intro, err := getSetting(b.db, "intro")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":           "Home",
		"Posts":           posts,
		"Drafts":          drafts,
		"Intro":           intro,
		"IsAuthenticated": isAuth,
		"CSRFToken":       ensureCSRFToken(w, r),
	}

	err = b.templates["home.html"].ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (b *Blog) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := getPostByID(b.db, id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.NotFound(w, r)
		return
	}

	isAuth := b.isAuthenticated(r)

	if !post.Published && !isAuth {
		http.NotFound(w, r)
		return
	}

	data := map[string]any{
		"Title":           post.Title,
		"Post":            post,
		"IsAuthenticated": isAuth,
		"CSRFToken":       ensureCSRFToken(w, r),
	}

	err = b.templates["detail.html"].ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (b *Blog) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := map[string]any{
			"Title":           "New Post",
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
		}
		err := b.templates["create.html"].ExecuteTemplate(w, "create.html", data)
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

		title := r.FormValue("title")
		content := r.FormValue("content")
		action := r.FormValue("action")

		if title == "" || content == "" {
			http.Error(w, "Title and content are required", http.StatusBadRequest)
			return
		}

		published := action == "publish"

		_, err := createPost(b.db, title, content, published)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (b *Blog) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		post, err := getPostByID(b.db, id)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if post == nil {
			http.NotFound(w, r)
			return
		}

		data := map[string]any{
			"Title":           fmt.Sprintf("Editing %q", post.Title),
			"Post":            post,
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
		}
		err = b.templates["edit.html"].ExecuteTemplate(w, "base", data)
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

		title := r.FormValue("title")
		content := r.FormValue("content")
		action := r.FormValue("action")

		if title == "" || content == "" {
			http.Error(w, "Title and content are required", http.StatusBadRequest)
			return
		}

		published := action == "publish"

		if err := updatePost(b.db, id, title, content, published); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (b *Blog) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		post, err := getPostByID(b.db, id)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if post == nil {
			http.NotFound(w, r)
			return
		}

		data := map[string]any{
			"Title":           fmt.Sprintf("Deleting %q", post.Title),
			"Post":            post,
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
		}
		err = b.templates["delete.html"].ExecuteTemplate(w, "base", data)
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

		if err := deletePost(b.db, id); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
