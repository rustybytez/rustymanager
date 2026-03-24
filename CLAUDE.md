# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`rustymanager` is a Go web application (Go 1.25.5) — a project manager. Stack:

- **HTTP**: Echo v4
- **CSS**: Tailwind CSS standalone CLI + DaisyUI via CDN (Bootstrap-like components)
- **DB**: SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **SQL**: `sqlc` generates type-safe Go from `.sql` files

## Commands

```bash
make build          # compile
make run            # go run ./cmd/server (port 8080)
make dev            # build CSS then run server
make test           # run all tests
make test-pkg PKG=path/to/pkg  # run tests for a single package
make lint           # golangci-lint (requires golangci-lint installed)
make fmt            # go fmt ./...
make tidy           # go mod tidy
make css            # build Tailwind CSS → web/static/css/app.css (downloads bin/tailwindcss if missing)
make css-watch      # watch mode for CSS
make tailwind-install       # explicitly download bin/tailwindcss (v3.4.17 by default)
make install-dev-tools      # install all dev tools: tailwindcss, air, sqlc, golangci-lint
make generate       # sqlc generate (re-run after changing sql/ files)
make docker-build   # build Docker image (rustymanager:latest)
make docker-run     # run container on port 8080
```

## Code generation

Always run `make fmt` before presenting generated or modified code.

After changing anything in `sql/schema/` or `sql/queries/`, run `make generate` to regenerate `internal/db/`.

## Architecture

```
cmd/server/main.go        Entry point: wires Echo, DB, renderer, routes
internal/
  db/                     sqlc-generated — do not edit directly
  handler/projects.go     Echo handlers for all project routes
  store/
    store.go              Wraps sqlc Querier; Migrate() runs DDL on startup
    migrations.sql        Embedded schema DDL (copy of sql/schema/)
sql/
  schema/                 Source-of-truth DDL
  queries/                sqlc-annotated SQL queries
web/
  web.go                  Owns //go:embed for static + templates
  static/css/app.css      Tailwind build output (gitignored)
  templates/
    layout.html           Defines "layout" template, calls {{ template "content" . }}
    projects/             Page templates — each defines only {{ define "content" }}
assets/css/app.css        Tailwind source (@tailwind directives)
```

### Template rendering pattern

The renderer in `main.go` clones the base layout template and parses the requested page file on each render. This allows every page to define its own `content` block without conflicts. Handlers pass the template path relative to `templates/`, e.g. `"projects/index.html"`.

### Environment variables

| Var | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | `rustymanager.db` | SQLite file path |
| `PORT` | `8080` | HTTP listen port |
