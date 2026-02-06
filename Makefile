.PHONY: help dev build test clean deps backend

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
	@echo "  dev         - Start development mode (runs backend directly)"
	@echo "  build       - Build the rago-cli binary"
	@echo "  test        - Run all Go tests"
	@echo "  clean       - Clean build artifacts and databases"
	@echo "  deps        - Download and install Go dependencies"
	@echo ""
	@echo "Current version: $(GIT_TAG)"

# Start development mode
dev:
	@echo "Starting RAGO server on port 7127..."
	@go run $(LDFLAGS) ./cmd/rago-cli serve --port 7127 --host 0.0.0.0

# Build the application
build: backend
	@echo "✅ Build complete! Run with: ./rago-cli serve --port 7127"

# Build only the backend (Go binary)
backend:
	@echo "Building rago version $(GIT_TAG)..."
	@go build $(LDFLAGS) -o rago-cli ./cmd/rago-cli
	@echo "✅ Backend binary built: rago-cli"

# Run all tests
test:
	@echo "Running Go tests..."
	@go test ./... -v

# Clean build artifacts and databases
clean:
	@echo "Cleaning build artifacts..."
	@rm -f rago-cli
	@echo "Cleaning databases..."
	@rm -rf .rago/data/*.db

# Download and install all dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ All dependencies installed!"