ARG TAILWIND_VERSION=v3.4.17
ARG GO_VERSION=1.24
ARG ALPINE_VERSION=3.21

# Stage 1: CSS build using Tailwind standalone binary
FROM alpine AS css-builder
ARG TAILWIND_VERSION
RUN apk add --no-cache curl && \
    ARCH=$(uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/') && \
    curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${ARCH}" \
         -o /usr/local/bin/tailwindcss && \
    chmod +x /usr/local/bin/tailwindcss
WORKDIR /app
COPY tailwind.config.js .
COPY assets/ ./assets/
COPY web/templates/ ./web/templates/
RUN tailwindcss -i assets/css/app.css -o web/static/css/app.css --minify

# Stage 2: Go build
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=css-builder /app/web/static/css/app.css ./web/static/css/app.css
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Stage 3: Runtime
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
