package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// reservedSlugs contains paths that cannot be used as post slugs
// to prevent collision with application routes
var reservedSlugs = map[string]bool{
	"admin":    true,
	"logout":   true,
	"feed":     true,
	"new":      true,
	"edit":     true,
	"delete":   true,
	"settings": true,
	"static":   true,
}

// isReservedSlug checks if a slug conflicts with application routes
func isReservedSlug(slug string) bool {
	return reservedSlugs[slug]
}

// generateSlug creates a URL-friendly slug from a title
func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove all characters except alphanumeric and hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Replace multiple consecutive hyphens with a single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// ensureUniqueSlug checks if a slug exists or is reserved, and appends a number suffix if needed
func ensureUniqueSlug(db *sql.DB, slug string, excludeID int) (string, error) {
	if slug == "" {
		return "", nil
	}

	candidate := slug
	suffix := 2

	for {
		// Check if slug is reserved (conflicts with app routes)
		if isReservedSlug(candidate) {
			candidate = fmt.Sprintf("%s-%d", slug, suffix)
			suffix++
			continue
		}

		// Check database for existing posts with this slug
		var count int
		var err error
		if excludeID > 0 {
			err = db.QueryRow(`SELECT COUNT(*) FROM posts WHERE slug = ? AND id != ?`, candidate, excludeID).Scan(&count)
		} else {
			err = db.QueryRow(`SELECT COUNT(*) FROM posts WHERE slug = ?`, candidate).Scan(&count)
		}
		if err != nil {
			return "", fmt.Errorf("checking slug uniqueness: %w", err)
		}

		if count == 0 {
			return candidate, nil
		}

		candidate = fmt.Sprintf("%s-%d", slug, suffix)
		suffix++
	}
}

func getPosts(db *sql.DB) ([]Post, error) {
	query := "SELECT id, title, slug, content, published, created_at FROM posts ORDER BY created_at DESC, id DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var slug sql.NullString
		err := rows.Scan(&post.ID, &post.Title, &slug, &post.Content, &post.Published, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning post: %w", err)
		}
		post.Slug = slug.String
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating posts: %w", err)
	}

	return posts, nil
}

func getPublishedPosts(db *sql.DB) ([]Post, error) {
	query := "SELECT id, title, slug, content, published, created_at FROM posts WHERE published = 1 ORDER BY created_at DESC, id DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying published posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var slug sql.NullString
		err := rows.Scan(&post.ID, &post.Title, &slug, &post.Content, &post.Published, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning post: %w", err)
		}
		post.Slug = slug.String
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating posts: %w", err)
	}

	return posts, nil
}

func getPostByID(db *sql.DB, id int) (*Post, error) {
	row := db.QueryRow(`
		SELECT id, title, slug, content, published, created_at
		FROM posts
		WHERE id = ?`, id)

	var post Post
	var slug sql.NullString
	err := row.Scan(&post.ID, &post.Title, &slug, &post.Content, &post.Published, &post.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning post %d: %w", id, err)
	}
	post.Slug = slug.String

	return &post, nil
}

func getPostBySlug(db *sql.DB, slug string) (*Post, error) {
	row := db.QueryRow(`
		SELECT id, title, slug, content, published, created_at
		FROM posts
		WHERE slug = ?`, slug)

	var post Post
	var slugVal sql.NullString
	err := row.Scan(&post.ID, &post.Title, &slugVal, &post.Content, &post.Published, &post.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning post by slug %q: %w", slug, err)
	}
	post.Slug = slugVal.String

	return &post, nil
}

func createPost(db *sql.DB, title, content string, published bool) (string, error) {
	slug := generateSlug(title)
	if slug == "" {
		slug = "untitled"
	}
	uniqueSlug, err := ensureUniqueSlug(db, slug, 0)
	if err != nil {
		return "", fmt.Errorf("generating unique slug: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO posts (title, slug, content, published)
		VALUES (?, ?, ?, ?)`, title, uniqueSlug, content, published)
	if err != nil {
		return "", fmt.Errorf("inserting post: %w", err)
	}
	return uniqueSlug, nil
}

func updatePost(db *sql.DB, id int, title, content string, published bool) (string, error) {
	// Generate new slug from title
	slug := generateSlug(title)
	if slug == "" {
		slug = "untitled"
	}
	uniqueSlug, err := ensureUniqueSlug(db, slug, id)
	if err != nil {
		return "", fmt.Errorf("generating unique slug: %w", err)
	}

	_, err = db.Exec(`
		UPDATE posts
		SET title = ?, slug = ?, content = ?, published = ?
		WHERE id = ?`, title, uniqueSlug, content, published, id)
	if err != nil {
		return "", fmt.Errorf("updating post %d: %w", id, err)
	}
	return uniqueSlug, nil
}

func deletePost(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting post %d: %w", id, err)
	}
	return nil
}
