.PHONY: build run kill test test-pkg lint fmt tidy css css-watch tailwind-install generate docker-build docker-run dev install-dev-tools

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

install-dev-tools: $(TAILWIND_BIN)
	go mod download

tailwind-install: $(TAILWIND_BIN)

kill:
	@PID=$$(lsof -ti:$(PORT)); \
	if [ -n "$$PID" ]; then \
		kill -9 $$PID && echo "killed $$PID on :$(PORT)"; \
	else \
		echo "nothing running on :$(PORT)"; \
	fi

build:
	go build ./...

run:
	go run ./cmd/server

test:
	go test ./...

test-pkg:
	go test ./$(PKG)/...

lint:
	go tool golangci-lint run

fmt:
	go fmt ./...

tidy:
	go mod tidy

css: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i assets/css/app.css -o web/static/css/app.css --minify

css-watch: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i assets/css/app.css -o web/static/css/app.css --watch

generate:
	go tool sqlc generate

docker-build:
	docker build -t rustymanager:latest .

docker-run:
	docker run --rm -p 8080:8080 rustymanager:latest

dev: css
	go tool air
