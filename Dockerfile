ARG GO_VERSION=1.26.1
ARG ALPINE_VERSION=3.21

# ── CSS builder ───────────────────────────────────────────────────────────────
FROM oven/bun:alpine AS css-builder
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY assets/ assets/
COPY web/templates/ web/templates/

RUN bunx tailwindcss -i ./assets/css/input.css -o ./web/static/css/output.css --minify

# ── Go builder ────────────────────────────────────────────────────────────────
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=css-builder /app/web/static/css/output.css ./web/static/css/output.css

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# ── Runtime ───────────────────────────────────────────────────────────────────
FROM alpine:${ALPINE_VERSION}
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

RUN mkdir /data && chown appuser:appgroup /data

COPY --from=builder /app/server .
RUN chown appuser:appgroup /app/server

USER appuser

ENV DATABASE_URL=/data/rustymanager.db
EXPOSE 8080
ENTRYPOINT ["/app/server"]
