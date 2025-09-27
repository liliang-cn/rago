
.PHONY: help build build-web build-dev run test clean install check build-all build-full proto

# Get the latest git tag (fallback to v0.0.0 if no tags)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
# Variable to hold the Go linker flags
LDFLAGS := -ldflags="-X 'github.com/liliang-cn/rago/v2/cmd/rago-cli.version=$(GIT_TAG)'"

# Default target - shows help
all: help

# Help target - shows all available commands
help:
	@echo "RAGO Build System"
	@echo "================="
	@echo ""
	@echo "Production Build:"
	@echo "  make build-full  - Build complete binary with embedded web assets"
	@echo "  make build       - Alias for build-full (backward compatibility)"
	@echo "  make install     - Build and install rago to GOPATH/bin"
	@echo ""
	@echo "Development:"
	@echo "  make build-dev   - Build binary only (uses existing web assets)"
	@echo "  make build-web   - Build web assets only"
	@echo "  make dev         - Run backend in development mode"
	@echo "  make run-web     - Run frontend development server"
	@echo "  make run         - Run rago directly with go run"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test        - Run all tests"
	@echo "  make check       - Run format, vet, and race tests"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean       - Remove all build artifacts"
	@echo "  make clean-web   - Remove web build artifacts only"
	@echo ""
	@echo "Current version: $(GIT_TAG)"

# Build the application with embedded web assets
build-full: build-web
	@echo "Building rago version $(GIT_TAG) with embedded web assets..."
	@go build $(LDFLAGS) -o rago-cli ./cmd/rago-cli

# Build just the Go binary (for development, uses existing web assets)
build-dev:
	@echo "Building rago version $(GIT_TAG) (development mode)..."
	@go build $(LDFLAGS) -o rago-cli ./cmd/rago-cli

# Alias for backward compatibility
build: build-full

# Build the web application
build-web:
	@echo "Building web assets..."
	@cd web && npm install && npm run build

# Run the application (development mode)
run:
	@go run $(LDFLAGS) main.go

# Run web development server
run-web:
	@echo "Starting web development server..."
	@cd web && npm run dev

# Run both backend and frontend in development mode (requires two terminals)
dev:
	@echo "Starting development mode..."
	@echo "Run 'make run' in one terminal and 'make run-web' in another"
	@go run $(LDFLAGS) main.go serve --port 7127

# Run tests
test:
	@go test ./...

# Clean up build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f rago-cli
	@rm -rf web/dist
	@rm -rf web/node_modules

# Clean only web artifacts
clean-web:
	@echo "Cleaning web artifacts..."
	@rm -rf web/dist
	@rm -rf web/node_modules

# Install the application with embedded web assets
install: build-full
	@echo "Installing rago-cli version $(GIT_TAG)..."
	@go install $(LDFLAGS) ./cmd/rago-cli

# Run checks (lint, format check, tests)
check:
	@echo "Running checks..."
	@go fmt ./...
	@go vet ./...
	@go test -race -coverprofile=coverage.out ./...

# Build all platforms with embedded web assets (used by CI)
build-all: build-full

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@mkdir -p proto/rago
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/rago/rago.proto

