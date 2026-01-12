
.PHONY: help dev build test clean deps frontend build-web backend frontend-dev

# Get the latest git tag (fallback to v0.0.0 if no tags)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
# Variable to hold the Go linker flags
LDFLAGS := -ldflags="-X 'main.version=$(GIT_TAG)'"

# Default target - shows help
all: help

# Help target - shows all available commands
help:
	@echo "RAGO - Simple Commands"
	@echo "====================="
	@echo ""
	@echo "  dev         - Start development server (builds web + runs backend)"
	@echo "  frontend-dev - Start frontend dev server with hot reload (port 5555)"
	@echo "  build       - Build complete application with web UI"
	@echo "  frontend    - Build only the web UI (alias: build-web)"
	@echo "  build-web   - Alias for frontend"
	@echo "  backend     - Build only the Go binary (without rebuilding frontend)"
	@echo "  test        - Run all tests"
	@echo "  clean       - Clean build artifacts and databases"
	@echo "  deps        - Download and install all dependencies"
	@echo ""
	@echo "Current version: $(GIT_TAG)"

# Start development server (builds web + runs backend)
dev:
	@echo "Starting development mode..."
	@echo "Building web assets..."
	@cd web && npm install && npm run build
	@echo "Starting RAGO server on port 7127..."
	@go run $(LDFLAGS) ./cmd/rago-cli serve --ui --port 7127 --host 0.0.0.0

# Build complete application with web UI
build: frontend backend
	@echo "✅ Build complete! Run with: ./rago-cli serve --ui --port 7127"

# Build only the frontend
frontend:
	@echo "Building web assets..."
	@cd web && npm install && npm run build
	@echo "✅ Frontend built to internal/web/dist/"

# Alias for frontend
build-web: frontend

# Alias for build
build-all: build

# Run checks (tests for now, can add linting later)
check: test

# Start frontend development server with hot reload
frontend-dev:
	@echo "Starting frontend development server with hot reload..."
	@echo "📦 Installing dependencies..."
	@cd web && npm install
	@echo "🚀 Starting dev server on http://localhost:5555"
	@echo "✨ Hot reload enabled - changes will auto-refresh!"
	@cd web && npm run dev -- --port 5555 --host 0.0.0.0

# Build only the backend (Go binary)
backend:
	@echo "Building rago version $(GIT_TAG) with embedded web assets..."
	@go build $(LDFLAGS) -o rago-cli ./cmd/rago-cli
	@echo "✅ Backend binary built: rago-cli"

# Run all tests
test:
	@echo "Running Go tests..."
	@go test ./... -v
	# Web tests skipped (no test script in package.json)
	# @cd web && npm test

# Clean build artifacts and databases
clean:
	@echo "Cleaning build artifacts..."
	@rm -f rago-cli
	@rm -rf web/dist
	@rm -rf web/node_modules
	@echo "Cleaning databases..."
	@rm -rf .rago/data/*.db

# Download and install all dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "Installing web dependencies..."
	@cd web && npm install
	@echo "✅ All dependencies installed!"

