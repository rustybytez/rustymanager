# rustymanager

A self-hosted project manager with kanban, team chat, and web push notifications.

## Stack

- **HTTP**: Echo v4
- **DB**: SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **SQL**: `sqlc` for type-safe queries
- **CSS**: Tailwind CSS v4 + DaisyUI v5 (built via Bun)
- **Real-time**: WebSockets (`nhooyr.io/websocket`)
- **Push**: Web Push via VAPID (`webpush-go`)
- **Auth**: Username + password (bcrypt), cookie-based sessions

## Features

- **User accounts** — register with username, password, and an admin token; bcrypt password hashing
- **Projects** — create, edit, archive projects
- **Kanban** — slide-out drawer with To Do / In Progress / Done columns
- **Project chat** — real-time WebSocket chat per project with message history
- **Web push notifications** — offline push alerts for new chat messages (PWA-ready)

## Getting Started

### Requirements

- Go 1.22+
- Bun (for CSS build)
- [sqlc](https://sqlc.dev) (for code generation)

### Environment variables

| Variable | Default | Required |
|---|---|---|
| `AUTH_TOKEN` | — | Yes — used to gate account registration |
| `DATABASE_URL` | `rustymanager.db` | No |
| `PORT` | `8080` | No |
| `VAPID_PUBLIC_KEY` | — | Yes |
| `VAPID_PRIVATE_KEY` | — | Yes |
| `VAPID_SUBSCRIBER` | `admin@example.com` | No |

Generate VAPID keys:

```bash
make vapid
```

### Run locally

```bash
cp .env.example .env  # fill in values
make dev              # builds CSS + runs server with hot reload
```

### Create your first account

Navigate to `/register` and provide a display name, username, password, and the `AUTH_TOKEN` from your `.env`.

## Commands

```bash
make build          # compile
make run            # go run ./cmd/server
make dev            # build CSS + run with air (hot reload)
make test           # run all tests
make lint           # golangci-lint
make fmt            # go fmt ./...
make tidy           # go mod tidy
make css            # build Tailwind CSS → web/static/css/output.css
make css-watch      # watch mode for CSS
make generate       # sqlc generate (re-run after changing sql/ files)
make docker-build   # build Docker image (rustymanager:latest)
make docker-run     # run container on port 8080
make vapid          # generate VAPID key pair
```

## Docker

```bash
make docker-build
make docker-run
```

The Docker image uses a multi-stage build: a Bun stage compiles the CSS, then a Go stage builds the binary. Runs as a non-root user on Alpine.

## Project layout

```
cmd/
  server/         # entry point
  vapid/          # VAPID key generator
internal/
  db/             # sqlc-generated (do not edit)
  handler/        # Echo handlers
  middleware/     # auth, project middleware
  push/           # web push sender + handler
  store/          # DB wrapper + migrations
sql/
  schema/         # source-of-truth DDL
  queries/        # sqlc-annotated SQL
web/
  templates/
    layout.html   # base layout with navbar
    auth/         # login, register
    projects/     # project pages (show, new, edit, select)
    users/        # user management
    settings/     # notification settings
  static/         # CSS, service worker, icons
assets/css/       # Tailwind source
```
