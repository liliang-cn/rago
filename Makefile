.PHONY: help dev build test clean deps backend ui ui-dev ui-build ui-deps

# Get the latest git tag (fallback to v0.0.0 if no tags)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
# Variable to hold the Go linker flags
LDFLAGS := -ldflags="-X 'main.version=$(GIT_TAG)'"

# Default target - shows help
all: help

# Help target - shows all available commands
help:
	@echo "RAGO - AI Agent SDK designed for Go developers"
	@echo "====================="
	@echo ""
	@echo "  dev         - Start development mode (runs backend directly)"
	@echo "  build       - Build the rago binary"
	@echo "  test        - Run all Go tests"
	@echo "  clean       - Clean build artifacts and databases"
	@echo "  deps        - Download and install Go dependencies"
	@echo ""
	@echo "UI Commands:"
	@echo "  ui-dev      - Start UI development server (with hot reload)"
	@echo "  ui-build    - Build UI for production (embeds into binary)"
	@echo "  ui-deps     - Install UI dependencies"
	@echo ""
	@echo "Current version: $(GIT_TAG)"

# Start development mode
dev:
	@echo "Starting RAGO server on port 7127..."
	@go run $(LDFLAGS) ./cmd/rago-cli serve --port 7127 --host 0.0.0.0

# Build the application
build: backend

# Build only the backend (Go binary)
backend:
	@echo "Building rago version $(GIT_TAG)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/rago ./cmd/rago-cli
	@echo "✅ Backend binary built: bin/rago"

# UI development server
ui-dev:
	@echo "Starting UI development server..."
	@cd ui && npm run dev

# Install UI dependencies
ui-deps:
	@echo "Installing UI dependencies..."
	@cd ui && npm install
	@echo "✅ UI dependencies installed"

# Build UI static files
ui-static:
	@echo "Building UI static files..."
	@cd ui && npm run build
	@echo "✅ UI static files built"

# Build UI binary (embeds static files)
ui-build: ui-static
	@echo "Building rago-ui binary..."
	@mkdir -p bin
	@cp -r ui/dist cmd/rago-ui/dist
	@go build $(LDFLAGS) -o bin/rago-ui ./cmd/rago-ui
	@rm -rf cmd/rago-ui/dist
	@echo "✅ UI binary built: bin/rago-ui"

# Run all tests
test:
	@echo "Running Go tests..."
	@go test ./... -v

# Alias for CI
check: test

# Clean build artifacts and databases
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf cmd/rago-ui/dist
	@echo "Cleaning databases..."
	@rm -rf .rago/data/*.db

# Download and install all dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ All dependencies installed!"
