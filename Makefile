.PHONY: help build run kill test test-pkg lint fmt tidy css css-watch generate docker-build docker-run dev install-dev-tools vapid vapid-update-env
.DEFAULT_GOAL := help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

PORT ?= 8080

node_modules:
	bun install

install-dev-tools: node_modules ## Install dev tools (bun deps + go deps)
	go mod download

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

css: node_modules ## Build Tailwind CSS (minified)
	bun run css

css-watch: node_modules ## Watch and rebuild CSS on changes
	bun run css:watch

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

dev: node_modules ## Run CSS watch + air in parallel
	trap 'kill 0' SIGINT; \
	bun run css:watch & \
	go tool air; \
	wait
