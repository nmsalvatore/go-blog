# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A minimalist personal blog application built with Go, using SQLite for storage and standard library `html/template` for rendering. Single-user authentication with draft support.

## Commands

```bash
go run main.go       # Run dev server on :8080
go test ./...        # Run all tests
go test -v ./...     # Verbose test output
go build             # Build binary
```

## Architecture

**File Organization:**
- `main.go` - Entry point, routing, Blog struct initialization
- `auth.go` - Sessions, CSRF protection, login/logout handlers
- `handlers.go` - HTTP handlers (Home, Detail, Create, Edit, Delete)
- `posts.go` - Post CRUD database operations
- `database.go` - Database initialization, schema, migrations
- `models.go` - Data structures (Post, Session)
- `settings.go` - Settings management
- `templates/` - HTML templates using base.html layout inheritance
- `static/` - CSS and minimal JavaScript

**Key Patterns:**
- `Blog` struct holds `*sql.DB` and handler methods attach to it
- `requireAuth()` middleware wraps protected routes
- In-memory SQLite (`:memory:`) for isolated testing
- Table-driven subtests throughout test files

**Routes:**
- Public: `/`, `/{slug}`, `/feed`, `/admin`, `/logout`
- Protected: `/new`, `/edit/{id}`, `/delete/{id}`, `/settings`

## Security Patterns

- CSRF: Double-submit cookie with constant-time comparison
- Passwords: bcrypt hashing
- SQL: Parameterized queries only
- XSS: `template.HTMLEscapeString()` for user content
- Sessions: HttpOnly cookies, 24-hour expiration, SameSite policies

## Environment

Copy `.env.example` to `.env` and configure:
- `ADMIN_USER` / `ADMIN_PASS` - Admin credentials
- `SECURE_COOKIES` - Set `true` for HTTPS deployments
