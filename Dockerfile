ARG TAILWIND_VERSION=v3.4.17

# Stage 1: CSS build using Tailwind standalone binary
FROM alpine AS css-builder
ARG TAILWIND_VERSION
ARG TARGETARCH
RUN apk add --no-cache curl && \
    ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "arm64" || echo "x64") && \
    curl -sL "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${ARCH}" \
         -o /usr/local/bin/tailwindcss && \
    chmod +x /usr/local/bin/tailwindcss
WORKDIR /app
COPY tailwind.config.js .
COPY assets/ ./assets/
COPY web/templates/ ./web/templates/
RUN tailwindcss -i assets/css/app.css -o web/static/css/app.css --minify

# Stage 2: Go build
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=css-builder /app/web/static/css/app.css ./web/static/css/app.css
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/rustymanager ./cmd/server

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache wget && adduser -D -u 1001 appuser && mkdir /data && chown appuser /data
COPY --from=go-builder /bin/rustymanager /rustymanager
USER appuser
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/rustymanager"]
