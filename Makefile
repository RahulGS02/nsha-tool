.PHONY: build clean install test run help

APP_NAME := nsha
VERSION := 1.0.0
BUILD_DIR := build
LDFLAGS := -s -w -X main.Version=$(VERSION)

help: ## Show this help message
	@echo "╔═══════════════════════════════════════════════════════════╗"
	@echo "║           NSHA - Null SHA Fixer                           ║"
	@echo "║                   Makefile Commands                       ║"
	@echo "╚═══════════════════════════════════════════════════════════╝"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

build: ## Build for current platform
	@echo "[BUILD] Building $(APP_NAME) for current platform..."
	@go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)
	@echo "[SUCCESS] Build complete: $(BUILD_DIR)/$(APP_NAME)"

build-all: ## Build for all platforms
	@echo "[BUILD] Building $(APP_NAME) for all platforms..."
	@chmod +x build.sh
	@./build.sh

build-windows: ## Build for Windows (amd64)
	@echo "[BUILD] Building for Windows..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe
	@echo "[SUCCESS] Built: $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe"

build-linux: ## Build for Linux (amd64)
	@echo "[BUILD] Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64
	@echo "[SUCCESS] Built: $(BUILD_DIR)/$(APP_NAME)-linux-amd64"

build-mac: ## Build for macOS (amd64)
	@echo "[BUILD] Building for macOS..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64
	@echo "[SUCCESS] Built: $(BUILD_DIR)/$(APP_NAME)-darwin-amd64"

clean: ## Clean build artifacts
	@echo "[CLEAN] Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@echo "[SUCCESS] Clean complete"

install: build ## Install to /usr/local/bin (requires sudo on Linux/Mac)
	@echo "[INSTALL] Installing $(APP_NAME)..."
	@sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)
	@echo "[SUCCESS] Installed to /usr/local/bin/$(APP_NAME)"

test: ## Run tests
	@echo "[TEST] Running tests..."
	@go test -v ./...

run: ## Run the application
	@go run main.go

deps: ## Download dependencies
	@echo "[DEPS] Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "[SUCCESS] Dependencies updated"

fmt: ## Format code
	@echo "[FORMAT] Formatting code..."
	@go fmt ./...
	@echo "[SUCCESS] Code formatted"

lint: ## Run linter
	@echo "[LINT] Running linter..."
	@golangci-lint run || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"

dev: ## Run in development mode with hot reload
	@echo "[DEV] Running in development mode..."
	@go run main.go diagnose

.DEFAULT_GOAL := help

