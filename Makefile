
.PHONY: build build-web run test clean install check build-all

# Get the latest git tag (fallback to v0.0.0 if no tags)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
# Variable to hold the Go linker flags
LDFLAGS := -ldflags="-X 'main.version=$(GIT_TAG)'"

# Default target
all: build

# Build the application
build:
	@echo "Building rago version $(GIT_TAG)..."
	@go build $(LDFLAGS) -o rago main.go

# Build the web application
build-web:
	@echo "Building web assets..."
	@cd web && npm install && npm run build

# Run the application
run:
	@go run $(LDFLAGS) main.go

# Run tests
test:
	@go test ./...

# Clean up build artifacts
clean:
	@rm -f rago

# Install the application
install:
	@echo "Installing rago version $(GIT_TAG)..."
	@go install $(LDFLAGS) ./...

# Run checks (lint, format check, tests)
check:
	@echo "Running checks..."
	@go fmt ./...
	@go vet ./...
	@go test -race -coverprofile=coverage.out ./...

# Build all platforms (used by CI)
build-all: build

