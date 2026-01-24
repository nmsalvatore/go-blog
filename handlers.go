package main

import (
	"crypto/subtle"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

func (b *Blog) render(w http.ResponseWriter, tmpl string, data map[string]any) {
	if err := b.templates[tmpl].ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("rendering template %s: %v", tmpl, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (b *Blog) getDisplaySettings() (theme, font, blogName string) {
	theme, _ = getSetting(b.db, "theme")
	font, _ = getSetting(b.db, "font")
	blogName = getBlogName(b.db)
	return
}

func (b *Blog) Home(w http.ResponseWriter, r *http.Request) {
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

	theme, font, blogName := b.getDisplaySettings()
	data := map[string]any{
		"Title":           "Home",
		"Posts":           posts,
		"Drafts":          drafts,
		"Intro":           intro,
		"Description":     truncate(intro, 160),
		"IsAuthenticated": isAuth,
		"CSRFToken":       ensureCSRFToken(w, r),
		"Theme":           theme,
		"Font":            font,
		"BlogName":        blogName,
	}

	b.render(w, "home.html", data)
}

// LegacyPostRedirect redirects old /post/{slug} URLs to /{slug}
func (b *Blog) LegacyPostRedirect(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	http.Redirect(w, r, "/"+url.PathEscape(slug), http.StatusMovedPermanently)
}

func (b *Blog) Detail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	post, err := getPostBySlug(b.db, slug)
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

	theme, font, blogName := b.getDisplaySettings()
	data := map[string]any{
		"Title":           post.Title,
		"Post":            post,
		"Description":     truncate(post.Content, 160),
		"IsAuthenticated": isAuth,
		"CSRFToken":       ensureCSRFToken(w, r),
		"Theme":           theme,
		"Font":            font,
		"BlogName":        blogName,
	}

	b.render(w, "detail.html", data)
}

func (b *Blog) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		theme, font, blogName := b.getDisplaySettings()
		data := map[string]any{
			"Title":           "New Post",
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
			"Theme":           theme,
			"Font":            font,
			"BlogName":        blogName,
		}
		b.render(w, "create.html", data)
		return
	}

	if r.Method == http.MethodPost {
		if !parseFormWithCSRF(w, r) {
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

		slug, err := createPost(b.db, title, content, published)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/"+url.PathEscape(slug), http.StatusSeeOther)
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

		theme, font, blogName := b.getDisplaySettings()
		data := map[string]any{
			"Title":           fmt.Sprintf("Editing %q", post.Title),
			"Post":            post,
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
			"Theme":           theme,
			"Font":            font,
			"BlogName":        blogName,
		}
		b.render(w, "edit.html", data)
		return
	}

	if r.Method == http.MethodPost {
		if !parseFormWithCSRF(w, r) {
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

		newSlug, err := updatePost(b.db, id, title, content, published)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/"+url.PathEscape(newSlug), http.StatusSeeOther)
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

		theme, font, blogName := b.getDisplaySettings()
		data := map[string]any{
			"Title":           fmt.Sprintf("Deleting %q", post.Title),
			"Post":            post,
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
			"Theme":           theme,
			"Font":            font,
			"BlogName":        blogName,
		}
		b.render(w, "delete.html", data)
		return
	}

	if r.Method == http.MethodPost {
		if !parseFormWithCSRF(w, r) {
			return
		}

		if err := deletePost(b.db, id); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (b *Blog) Settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		intro, err := getSetting(b.db, "intro")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		theme, font, blogName := b.getDisplaySettings()
		data := map[string]any{
			"Title":           "Settings",
			"Intro":           intro,
			"IsAuthenticated": true,
			"CSRFToken":       ensureCSRFToken(w, r),
			"Theme":           theme,
			"Font":            font,
			"BlogName":        blogName,
		}
		b.render(w, "settings.html", data)
		return
	}

	if r.Method == http.MethodPost {
		if !parseFormWithCSRF(w, r) {
			return
		}

		intro := r.FormValue("intro")
		theme := r.FormValue("theme")
		font := r.FormValue("font")
		blogName := r.FormValue("blog_name")

		if err := setSetting(b.db, "intro", intro); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := setSetting(b.db, "theme", theme); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := setSetting(b.db, "font", font); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := setSetting(b.db, "blog_name", blogName); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (b *Blog) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		theme, font, blogName := b.getDisplaySettings()
		data := map[string]any{
			"Title":     "Login",
			"CSRFToken": ensureCSRFToken(w, r),
			"Theme":     theme,
			"Font":      font,
			"BlogName":  blogName,
		}
		b.render(w, "admin.html", data)
		return
	}

	if r.Method == http.MethodPost {
		if !parseFormWithCSRF(w, r) {
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		if subtle.ConstantTimeCompare([]byte(username), []byte(adminUsername)) != 1 || !checkPassword(adminPassword, password) {
			theme, font, blogName := b.getDisplaySettings()
			data := map[string]any{
				"Title":     "Login",
				"Error":     "Invalid username or password",
				"CSRFToken": getCSRFToken(r),
				"Theme":     theme,
				"Font":      font,
				"BlogName":  blogName,
			}
			w.WriteHeader(http.StatusUnauthorized)
			b.render(w, "admin.html", data)
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

	if !parseFormWithCSRF(w, r) {
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

func (b *Blog) Feed(w http.ResponseWriter, r *http.Request) {
	posts, err := getPublishedPosts(b.db)
	if err != nil {
		log.Printf("fetching posts for feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + r.Host

	items := make([]rssItem, len(posts))
	for i, post := range posts {
		postURL := fmt.Sprintf("%s/%s", baseURL, post.Slug)
		items[i] = rssItem{
			Title:       post.Title,
			Link:        postURL,
			GUID:        postURL,
			PubDate:     post.CreatedAt.UTC().Format(time.RFC1123Z),
			Description: post.Content,
		}
	}

	blogName := getBlogName(b.db)
	feed := rss{
		Version: "2.0",
		Channel: rssChannel{
			Title:       blogName,
			Link:        baseURL,
			Description: "A personal blog",
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("encoding RSS feed: %v", err)
	}
}
