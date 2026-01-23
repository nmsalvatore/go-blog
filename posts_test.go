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

	id, err := createPost(blog.db, "Test Title", "Test Content", true)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}

	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
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

	err := updatePost(blog.db, 1, "Updated", "Updated content", true)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
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

	id, err := createPost(blog.db, "Draft Title", "Draft Content", false)
	if err != nil {
		t.Fatalf("createPost() error: %v", err)
	}

	post, _ := getPostByID(blog.db, int(id))
	if post.Published {
		t.Error("expected post to be a draft")
	}
}

func TestUpdatePost_PublishDraft(t *testing.T) {
	blog := setupTestDB(t)

	createPost(blog.db, "Draft", "Content", false)

	err := updatePost(blog.db, 1, "Draft", "Content", true)
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

	err := updatePost(blog.db, 1, "Published", "Content", false)
	if err != nil {
		t.Fatalf("updatePost() error: %v", err)
	}

	post, _ := getPostByID(blog.db, 1)
	if post.Published {
		t.Error("expected post to be draft after update")
	}
}
