package main

import "time"

type Post struct {
	ID        int
	Title     string
	Slug      string
	Content   string
	Published bool
	CreatedAt time.Time
}

type Session struct {
	Token     string
	UserID    int
	ExpiresAt time.Time
}
