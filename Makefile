.PHONY: help build run kill test test-pkg lint fmt tidy css css-watch tailwind-install generate docker-build docker-run dev install-dev-tools vapid vapid-update-env
.DEFAULT_GOAL := help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

TAILWIND_VERSION ?= v3.4.17
PORT             ?= 8080
TAILWIND_BIN     := bin/tailwindcss

# Download the Tailwind standalone binary for the current OS/arch if not present.
$(TAILWIND_BIN):
	mkdir -p bin
	$(eval _OS   := $(shell uname -s | tr '[:upper:]' '[:lower:]' | sed 's/darwin/macos/'))
	$(eval _ARCH := $(shell uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/'))
	curl -sL "https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(_OS)-$(_ARCH)" \
		-o $(TAILWIND_BIN)
	chmod +x $(TAILWIND_BIN)

install-dev-tools: $(TAILWIND_BIN) ## Install dev tools (tailwind, go deps)
	go mod download

tailwind-install: $(TAILWIND_BIN) ## Download Tailwind standalone binary

kill: ## Kill process on PORT (default 8080)
	@PID=$$(lsof -ti:$(PORT)); \
	if [ -n "$$PID" ]; then \
		kill -9 $$PID && echo "killed $$PID on :$(PORT)"; \
	else \
		echo "nothing running on :$(PORT)"; \
	fi

build: ## Compile all packages
	go build ./...

run: ## Run the server
	go run ./cmd/server

test: ## Run all tests
	go test ./...

test-pkg: ## Run tests for a single package (PKG=path/to/pkg)
	go test ./$(PKG)/...

lint: ## Run golangci-lint
	go tool golangci-lint run

fmt: ## Format Go code
	go fmt ./...

tidy: ## Tidy go.mod
	go mod tidy

css: $(TAILWIND_BIN) ## Build Tailwind CSS (minified)
	$(TAILWIND_BIN) -i assets/css/app.css -o web/static/css/app.css --minify

css-watch: $(TAILWIND_BIN) ## Watch and rebuild CSS on changes
	$(TAILWIND_BIN) -i assets/css/app.css -o web/static/css/app.css --watch

generate: ## Regenerate sqlc code
	go tool sqlc generate

vapid: ## Generate VAPID keys
	go run ./cmd/vapid

vapid-update-env: ## Generate VAPID keys and update .env
	go run ./cmd/vapid -update-env

docker-build: ## Build Docker image
	docker build -t rustymanager:latest .

docker-run: ## Run Docker container on port 8080
	docker run --rm -p 8080:8080 rustymanager:latest

dev: css ## Build CSS then run with hot reload
	go tool air
