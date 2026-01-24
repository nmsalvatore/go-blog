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

func TestDetail(t *testing.T) {
	blog := setupTestBlog(t)

	slug, err := createPost(blog.db, "Detail Test", "Detail content", true)
	if err != nil {
		t.Fatalf("creating test post: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+slug, nil)
	req.SetPathValue("slug", slug)
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d (post slug: %q)", http.StatusOK, w.Code, slug)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Detail Test") {
		t.Error("expected response to contain 'Detail Test'")
	}
}

func TestDetail_NotFound(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.SetPathValue("slug", "nonexistent")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
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
	slug, _ := createPost(blog.db, "Draft Post", "Draft content", false)

	req := httptest.NewRequest(http.MethodGet, "/"+slug, nil)
	req.SetPathValue("slug", slug)
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for draft without auth, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDetail_Draft_Authenticated(t *testing.T) {
	blog := setupTestBlog(t)

	// Create a draft post
	slug, _ := createPost(blog.db, "Draft Post", "Draft content", false)

	// Create a session for authentication
	token, _ := createSession(blog.db, 1)

	req := httptest.NewRequest(http.MethodGet, "/"+slug, nil)
	req.SetPathValue("slug", slug)
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

func TestFeed(t *testing.T) {
	blog := setupTestBlog(t)

	// Create published posts
	createPost(blog.db, "First Post", "First content", true)
	createPost(blog.db, "Second Post", "Second content", true)
	// Create a draft (should not appear)
	createPost(blog.db, "Draft Post", "Draft content", false)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	w := httptest.NewRecorder()

	blog.Feed(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/rss+xml") {
		t.Errorf("expected Content-Type application/rss+xml, got %s", contentType)
	}

	body := w.Body.String()

	// Check RSS structure
	if !strings.Contains(body, `<?xml version="1.0"`) {
		t.Error("expected XML declaration")
	}
	if !strings.Contains(body, `<rss version="2.0">`) {
		t.Error("expected RSS element")
	}
	if !strings.Contains(body, "<channel>") {
		t.Error("expected channel element")
	}

	// Check published posts appear
	if !strings.Contains(body, "First Post") {
		t.Error("expected First Post in feed")
	}
	if !strings.Contains(body, "Second Post") {
		t.Error("expected Second Post in feed")
	}

	// Check draft does not appear
	if strings.Contains(body, "Draft Post") {
		t.Error("draft should not appear in feed")
	}
}

func TestFeed_Empty(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	w := httptest.NewRecorder()

	blog.Feed(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<channel>") {
		t.Error("expected channel element even with no posts")
	}
}

func TestFeed_EscapesXML(t *testing.T) {
	blog := setupTestBlog(t)

	// Create post with special characters
	createPost(blog.db, "Test <script>", "Content with <html> & \"quotes\"", true)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	w := httptest.NewRecorder()

	blog.Feed(w, req)

	body := w.Body.String()

	// Check that special characters are escaped
	if strings.Contains(body, "<script>") {
		t.Error("expected < to be escaped")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("expected &lt;script&gt; in escaped title")
	}
}

// Slug-based URL tests

func TestDetail_BySlug(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "My Test Post", "Test content", true)

	req := httptest.NewRequest(http.MethodGet, "/my-test-post", nil)
	req.SetPathValue("slug", "my-test-post")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "My Test Post") {
		t.Error("expected response to contain 'My Test Post'")
	}
}

func TestDetail_BySlug_NotFound(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.SetPathValue("slug", "nonexistent")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDetail_Draft_BySlug_Unauthenticated(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "Draft Post", "Draft content", false)

	req := httptest.NewRequest(http.MethodGet, "/draft-post", nil)
	req.SetPathValue("slug", "draft-post")
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for draft without auth, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDetail_Draft_BySlug_Authenticated(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "Draft Post", "Draft content", false)
	token, _ := createSession(blog.db, 1)

	req := httptest.NewRequest(http.MethodGet, "/draft-post", nil)
	req.SetPathValue("slug", "draft-post")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w := httptest.NewRecorder()

	blog.Detail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for draft with auth, got %d", http.StatusOK, w.Code)
	}
}

func TestFeed_UsesSlugURLs(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "My Post Title", "Content", true)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	blog.Feed(w, req)

	body := w.Body.String()

	// Should use slug URL, not ID URL
	if !strings.Contains(body, "/my-post-title") {
		t.Error("expected feed to contain slug URL '/my-post-title'")
	}
	if strings.Contains(body, "/1") {
		t.Error("feed should not contain ID-based URL '/1'")
	}
}

func TestEdit_POST_RedirectsToSlug(t *testing.T) {
	blog := setupTestBlog(t)

	createPost(blog.db, "Original Title", "Original content", true)

	form := url.Values{}
	form.Set("title", "Updated Title")
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

	// Verify redirect location is the slug URL, not home
	location := w.Header().Get("Location")
	if location != "/updated-title" {
		t.Errorf("expected redirect to '/updated-title', got %q", location)
	}
}

func TestLegacyPostRedirect(t *testing.T) {
	blog := setupTestBlog(t)

	req := httptest.NewRequest(http.MethodGet, "/post/my-old-slug", nil)
	req.SetPathValue("slug", "my-old-slug")
	w := httptest.NewRecorder()

	blog.LegacyPostRedirect(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, w.Code)
	}

	location := w.Header().Get("Location")
	if location != "/my-old-slug" {
		t.Errorf("expected redirect to '/my-old-slug', got %q", location)
	}
}
