package main

import (
	"database/sql"
	"fmt"
)

func getPosts(db *sql.DB) ([]Post, error) {
	query := "SELECT id, title, content, published, created_at FROM posts ORDER BY created_at DESC, id DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Published, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning post: %w", err)
		}
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating posts: %w", err)
	}

	return posts, nil
}

func getPublishedPosts(db *sql.DB) ([]Post, error) {
	query := "SELECT id, title, content, published, created_at FROM posts WHERE published = 1 ORDER BY created_at DESC, id DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying published posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Published, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning post: %w", err)
		}
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating posts: %w", err)
	}

	return posts, nil
}

func getPostByID(db *sql.DB, id int) (*Post, error) {
	row := db.QueryRow(`
		SELECT id, title, content, published, created_at
		FROM posts
		WHERE id = ?`, id)

	var post Post
	err := row.Scan(&post.ID, &post.Title, &post.Content, &post.Published, &post.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning post %d: %w", id, err)
	}

	return &post, nil
}

func createPost(db *sql.DB, title, content string, published bool) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO posts (title, content, published)
		VALUES (?, ?, ?)`, title, content, published)
	if err != nil {
		return 0, fmt.Errorf("inserting post: %w", err)
	}
	return result.LastInsertId()
}

func updatePost(db *sql.DB, id int, title, content string, published bool) error {
	_, err := db.Exec(`
		UPDATE posts
		SET title = ?, content = ?, published = ?
		WHERE id = ?`, title, content, published, id)
	if err != nil {
		return fmt.Errorf("updating post %d: %w", id, err)
	}
	return nil
}

func deletePost(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting post %d: %w", id, err)
	}
	return nil
}
