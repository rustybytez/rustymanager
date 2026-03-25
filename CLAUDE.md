# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`rustymanager` is a Go web application — a project manager. Stack:

- **HTTP**: Echo v4
- **CSS**: Tailwind CSS standalone CLI + DaisyUI via CDN (Bootstrap-like components)
- **DB**: SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **SQL**: `sqlc` generates type-safe Go from `.sql` files

## Preferred Technology Stack

When building new Go web applications (or extending this one), always use the following unless there's a compelling reason not to:

### HTTP & Routing
- **Echo v4** (`github.com/labstack/echo/v4`) — router, middleware chain, renderer interface
- Middleware: `middleware.Logger()` and `middleware.Recover()` always enabled

### Database & SQL
- **SQLite** via `modernc.org/sqlite` — pure Go, no CGO, single-file DB, great for self-hosted apps
- **sqlc** (`github.com/sqlc-dev/sqlc`) — generate type-safe Go from annotated SQL; never write raw `db.Query` by hand
- **Migrations**: embed a `migrations.sql` via `//go:embed` and run at startup in `store.Migrate()`; split statements on `;` and ignore "duplicate column name" errors for idempotency

### Templating & Frontend
- **stdlib `html/template`** — no third-party template engine
- **`//go:embed`** via `fs.FS` in a `web/` package for static files and templates
- **Rendering pattern**: clone the base layout template and parse the page file on each request (avoids block conflicts across pages)
- **Tailwind CSS** standalone CLI binary (downloaded by Makefile, not npm) + **DaisyUI** via CDN for components
- **HTMX or vanilla JS** for interactivity — avoid heavy frontend frameworks

### Real-time Features
- **WebSockets**: `nhooyr.io/websocket` (not `gorilla/websocket`)
- **Web Push notifications**: `github.com/SherClockHolmes/webpush-go` with VAPID keys; generate keys with a dedicated `cmd/vapid/` binary

### External APIs
- Use **raw `net/http`** for simple external API calls (e.g. GitHub API) — avoid pulling in large SDK dependencies unless necessary

### Auth
- **Simple cookie-based auth** with a single `AUTH_TOKEN` env var — no sessions library, no JWT for personal/self-hosted tools
- Protect routes with a custom Echo middleware that checks the cookie against the env var

### Config & Environment
- **`github.com/joho/godotenv`** — auto-load `.env` in dev via `godotenv.Load()` at the top of `main()`
- Fail fast with `log.Fatal` if required env vars are missing
- Standard env vars: `DATABASE_URL` (default: `<appname>.db`), `PORT` (default: `8080`)

### Dev Tooling (tracked in `go.mod` `tool` directive)
- **`air`** (`github.com/air-verse/air`) — hot reload during development
- **`golangci-lint`** — linting
- **`sqlc`** — SQL code generation
- Tailwind binary downloaded by Makefile (not npm/node)

### Project Layout
```
cmd/
  server/         # main entry point: wires Echo, DB, renderer, routes
  <toolname>/     # other entry points (e.g. vapid key generator)
internal/
  db/             # sqlc-generated — never edit directly
  handler/        # Echo handlers, one file per feature area
  middleware/     # custom Echo middleware (auth, etc.)
  store/          # wraps sqlc Querier; Migrate() runs DDL on startup
  <feature>/      # self-contained feature packages (e.g. push, github)
sql/
  schema/         # source-of-truth DDL
  queries/        # sqlc-annotated SQL
web/
  web.go          # //go:embed for static + templates
  static/         # CSS, JS, service workers
  templates/
    layout.html   # base layout — defines "layout" template
    <feature>/    # page templates — each defines only {{ define "content" }}
assets/css/       # Tailwind source files
```

### Deployment
- **Docker** — always include `docker-build` and `docker-run` Makefile targets

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
