package main

import (
	"testing"
)

func setupTestDB(t *testing.T) *Blog {
	t.Helper()
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	if err = initDB(db); err != nil {
		t.Fatalf("initializing test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return &Blog{db: db}
}

func TestGetPosts_Empty(t *testing.T) {
	blog := setupTestDB(t)

	posts, err := getPosts(blog.db)
	if err != nil {
		t.Fatalf("getPosts() error: %v", err)
	}

	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}

func TestCreatePost(t *testing.T) {
	blog := setupTestDB(t)

	slug, err := createPost(blog.db, "Test Title", "Test Content", true)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}

	if slug != "test-title" {
		t.Errorf("expected slug 'test-title', got %q", slug)
	}

	post, err := getPostByID(blog.db, 1)
	if err != nil {
		t.Fatalf("getPostByID() error: %v", err)
	}

	if post.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", post.Title)
	}
	if post.Content != "Test Content" {
		t.Errorf("expected content 'Test Content', got '%s'", post.Content)
	}
	if !post.Published {
		t.Error("expected post to be published")
	}
}

func TestGetPosts_Order(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "First", "Content 1", true)
	createPost(blog.db, "Second", "Content 2", true)
	createPost(blog.db, "Third", "Content 3", true)

	posts, err := getPosts(blog.db)
	if err != nil {
		t.Fatalf("getPosts() error: %v", err)
	}

	if len(posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}

	// Should be in reverse order (newest first)
	if posts[0].Title != "Third" {
		t.Errorf("expected first post to be 'Third', got '%s'", posts[0].Title)
	}
	if posts[2].Title != "First" {
		t.Errorf("expected last post to be 'First', got '%s'", posts[2].Title)
	}
}

func TestGetPostByID_NotFound(t *testing.T) {
	blog := setupTestDB(t)

	post, err := getPostByID(blog.db, 999)
	if err != nil {
		t.Fatalf("getPostByID() error: %v", err)
	}

	if post != nil {
		t.Error("expected nil for nonexistent post")
	}
}

func TestUpdatePost(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Original", "Original content", true)

	slug, err := updatePost(blog.db, 1, "Updated", "Updated content", true)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	if slug != "updated" {
		t.Errorf("expected slug 'updated', got %q", slug)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Title != "Updated" {
		t.Errorf("expected title 'Updated', got '%s'", post.Title)
	}
	if post.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got '%s'", post.Content)
	}
}

func TestDeletePost(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "To Delete", "Content", true)

	err := deletePost(blog.db, 1)
	if err != nil {
		t.Fatalf("deletePost() error: %v", err)
	}

	post, _ := getPostByID(blog.db, 1)
	if post != nil {
		t.Error("expected post to be deleted")
	}
}

func TestDeletePost_NonExistent(t *testing.T) {
	blog := setupTestDB(t)

	// Should not error when deleting non-existent post
	err := deletePost(blog.db, 999)
	if err != nil {
		t.Fatalf("deletePost() unexpected error: %v", err)
	}
}

func TestGetPublishedPosts_ExcludesDrafts(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Published Post", "Content", true)
	createPost(blog.db, "Draft Post", "Content", false)

	published, err := getPublishedPosts(blog.db)
	if err != nil {
		t.Fatalf("getPublishedPosts() error: %v", err)
	}

	if len(published) != 1 {
		t.Fatalf("expected 1 published post, got %d", len(published))
	}

	if published[0].Title != "Published Post" {
		t.Errorf("expected 'Published Post', got '%s'", published[0].Title)
	}
}

func TestGetPosts_IncludesDrafts(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Published Post", "Content", true)
	createPost(blog.db, "Draft Post", "Content", false)

	all, err := getPosts(blog.db)
	if err != nil {
		t.Fatalf("getPosts() error: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(all))
	}
}

func TestCreatePost_Draft(t *testing.T) {
	blog := setupTestDB(t)

	_, err := createPost(blog.db, "Draft Title", "Draft Content", false)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Published {
		t.Error("expected post to be a draft")
	}
}

func TestUpdatePost_PublishDraft(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Draft", "Content", false)

	_, err := updatePost(blog.db, 1, "Draft", "Content", true)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	post, _ := getPostByID(blog.db, 1)
	if !post.Published {
		t.Error("expected post to be published after update")
	}
}

func TestUpdatePost_UnpublishPost(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Published", "Content", true)

	_, err := updatePost(blog.db, 1, "Published", "Content", false)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Published {
		t.Error("expected post to be draft after update")
	}
}

// Slug tests

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{"simple title", "Hello World", "hello-world"},
		{"lowercase", "hello world", "hello-world"},
		{"uppercase", "HELLO WORLD", "hello-world"},
		{"special chars", "Hello, World!", "hello-world"},
		{"multiple spaces", "Hello   World", "hello-world"},
		{"leading/trailing spaces", "  Hello World  ", "hello-world"},
		{"numbers", "Top 10 Posts", "top-10-posts"},
		{"apostrophe", "It's a Test", "its-a-test"},
		{"ampersand", "Cats & Dogs", "cats-dogs"},
		{"dashes preserved", "Pre-existing Slug", "pre-existing-slug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.title)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestGenerateSlug_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{"empty string", "", ""},
		{"only special chars", "!@#$%", ""},
		{"leading hyphens", "---Hello", "hello"},
		{"trailing hyphens", "Hello---", "hello"},
		{"multiple hyphens", "Hello---World", "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.title)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestEnsureUniqueSlug(t *testing.T) {
	blog := setupTestDB(t)

	// Create a post with slug "hello-world"
	createPost(blog.db, "Hello World", "Content", true)

	tests := []struct {
		name      string
		slug      string
		excludeID int
		expected  string
	}{
		{"unique slug", "different-slug", 0, "different-slug"},
		{"duplicate slug", "hello-world", 0, "hello-world-2"},
		{"same post (excluded)", "hello-world", 1, "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ensureUniqueSlug(blog.db, tt.slug, tt.excludeID)
			if err != nil {
				t.Fatalf("ensureUniqueSlug() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ensureUniqueSlug(%q, %d) = %q, want %q", tt.slug, tt.excludeID, result, tt.expected)
			}
		})
	}
}

func TestIsReservedSlug(t *testing.T) {
	tests := []struct {
		slug     string
		expected bool
	}{
		{"admin", true},
		{"feed", true},
		{"logout", true},
		{"new", true},
		{"edit", true},
		{"delete", true},
		{"settings", true},
		{"static", true},
		{"my-post", false},
		{"hello-world", false},
		{"admin-panel", false}, // suffix doesn't make it reserved
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			result := isReservedSlug(tt.slug)
			if result != tt.expected {
				t.Errorf("isReservedSlug(%q) = %v, want %v", tt.slug, result, tt.expected)
			}
		})
	}
}

func TestEnsureUniqueSlug_ReservedSlugs(t *testing.T) {
	blog := setupTestDB(t)

	tests := []struct {
		name     string
		slug     string
		expected string
	}{
		{"reserved: admin", "admin", "admin-2"},
		{"reserved: feed", "feed", "feed-2"},
		{"reserved: settings", "settings", "settings-2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ensureUniqueSlug(blog.db, tt.slug, 0)
			if err != nil {
				t.Fatalf("ensureUniqueSlug() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ensureUniqueSlug(%q, 0) = %q, want %q", tt.slug, result, tt.expected)
			}
		})
	}
}

func TestCreatePost_ReservedSlug(t *testing.T) {
	blog := setupTestDB(t)

	// Create a post titled "Feed" - should get slug "feed-2" to avoid collision
	slug, err := createPost(blog.db, "Feed", "Content about feeds", true)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}

	if slug != "feed-2" {
		t.Errorf("expected slug 'feed-2' for reserved word, got %q", slug)
	}
}

func TestEnsureUniqueSlug_MultipleDuplicates(t *testing.T) {
	blog := setupTestDB(t)

	// Create posts with slugs hello-world, hello-world-2
	createPost(blog.db, "Hello World", "Content", true)
	createPost(blog.db, "Hello World", "Content", true) // Should get hello-world-2

	// Third duplicate should get hello-world-3
	slug, err := ensureUniqueSlug(blog.db, "hello-world", 0)
	if err != nil {
		t.Fatalf("ensureUniqueSlug() error: %v", err)
	}
	if slug != "hello-world-3" {
		t.Errorf("expected 'hello-world-3', got %q", slug)
	}
}

func TestCreatePost_GeneratesSlug(t *testing.T) {
	blog := setupTestDB(t)

	slug, err := createPost(blog.db, "My First Post", "Content", true)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}

	if slug != "my-first-post" {
		t.Errorf("expected slug 'my-first-post', got %q", slug)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Slug != "my-first-post" {
		t.Errorf("expected post.Slug 'my-first-post', got %q", post.Slug)
	}
}

func TestCreatePost_UniqueSlug(t *testing.T) {
	blog := setupTestDB(t)

	slug1, _ := createPost(blog.db, "Hello World", "Content 1", true)
	slug2, _ := createPost(blog.db, "Hello World", "Content 2", true)

	if slug1 != "hello-world" {
		t.Errorf("expected first slug 'hello-world', got %q", slug1)
	}
	if slug2 != "hello-world-2" {
		t.Errorf("expected second slug 'hello-world-2', got %q", slug2)
	}
}

func TestGetPostBySlug(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Test Post", "Content", true)

	post, err := getPostBySlug(blog.db, "test-post")
	if err != nil {
		t.Fatalf("getPostBySlug() error: %v", err)
	}
	if post == nil {
		t.Fatal("expected post, got nil")
	}
	if post.Title != "Test Post" {
		t.Errorf("expected title 'Test Post', got %q", post.Title)
	}
}

func TestGetPostBySlug_NotFound(t *testing.T) {
	blog := setupTestDB(t)

	post, err := getPostBySlug(blog.db, "nonexistent")
	if err != nil {
		t.Fatalf("getPostBySlug() error: %v", err)
	}
	if post != nil {
		t.Error("expected nil for nonexistent slug")
	}
}

func TestUpdatePost_UpdatesSlug(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Original Title", "Content", true)

	newSlug, err := updatePost(blog.db, 1, "New Title", "Content", true)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	if newSlug != "new-title" {
		t.Errorf("expected new slug 'new-title', got %q", newSlug)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Slug != "new-title" {
		t.Errorf("expected post.Slug 'new-title', got %q", post.Slug)
	}
}

func TestUpdatePost_SameTitleKeepsSlug(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "My Title", "Content", true)

	// Update with same title - slug should remain unchanged
	newSlug, err := updatePost(blog.db, 1, "My Title", "Updated content", true)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	if newSlug != "my-title" {
		t.Errorf("expected slug to remain 'my-title', got %q", newSlug)
	}
}

func TestGetPosts_IncludesSlug(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Test Post", "Content", true)

	posts, err := getPosts(blog.db)
	if err != nil {
		t.Fatalf("getPosts() error: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	if posts[0].Slug != "test-post" {
		t.Errorf("expected slug 'test-post', got %q", posts[0].Slug)
	}
}

func TestCreatePost_EmptySlugFallback(t *testing.T) {
	blog := setupTestDB(t)

	// Title with only special chars produces empty slug - should fallback to "untitled"
	slug, err := createPost(blog.db, "!@#$%", "Content", true)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}

	if slug != "untitled" {
		t.Errorf("expected slug 'untitled', got %q", slug)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Slug != "untitled" {
		t.Errorf("expected post.Slug 'untitled', got %q", post.Slug)
	}
}

func TestCreatePost_MultipleUntitled(t *testing.T) {
	blog := setupTestDB(t)

	// First post with special chars only
	slug1, err := createPost(blog.db, "!@#$%", "Content 1", true)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}
	if slug1 != "untitled" {
		t.Errorf("expected first slug 'untitled', got %q", slug1)
	}

	// Second post with special chars only - should get "untitled-2"
	slug2, err := createPost(blog.db, "^&*()", "Content 2", true)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}
	if slug2 != "untitled-2" {
		t.Errorf("expected second slug 'untitled-2', got %q", slug2)
	}
}

func TestUpdatePost_EmptySlugFallback(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Normal Title", "Content", true)

	// Update to a title that produces empty slug
	newSlug, err := updatePost(blog.db, 1, "!@#$%", "Updated content", true)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	if newSlug != "untitled" {
		t.Errorf("expected slug 'untitled', got %q", newSlug)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Slug != "untitled" {
		t.Errorf("expected post.Slug 'untitled', got %q", post.Slug)
	}
}

func TestUpdatePost_EmptySlugFallback_WhenUntitledExists(t *testing.T) {
	blog := setupTestDB(t)

	// Create a post that will have slug "untitled"
	createPost(blog.db, "!@#$%", "Content 1", true)

	// Create a second post with normal title
	createPost(blog.db, "Normal Title", "Content 2", true)

	// Update second post to a title that produces empty slug
	// Should get "untitled-2" since "untitled" already exists
	newSlug, err := updatePost(blog.db, 2, "^&*()", "Updated content", true)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	if newSlug != "untitled-2" {
		t.Errorf("expected slug 'untitled-2', got %q", newSlug)
	}

	post, _ := getPostByID(blog.db, 2)
	if post.Slug != "untitled-2" {
		t.Errorf("expected post.Slug 'untitled-2', got %q", post.Slug)
	}
}
