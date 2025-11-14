# Makefile for sysc-walls
# Provides convenient targets for building, testing, and syncing

.PHONY: all build clean install sync test help version check-sync

# Default target
all: build

# Build all binaries
build:
	@echo "Building sysc-walls..."
	@go build -o bin/sysc-walls-display cmd/display/main.go
	@go build -o bin/sysc-walls-daemon cmd/daemon/main.go
	@echo "✓ Build complete"
	@echo "  Display: bin/sysc-walls-display"
	@echo "  Daemon:  bin/sysc-walls-daemon"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f bin/sysc-walls-display bin/sysc-walls-daemon
	@rm -f display daemon
	@echo "✓ Clean complete"

# Install to system
install: build
	@echo "Installing sysc-walls..."
	@sudo cp bin/sysc-walls-display /usr/local/bin/
	@sudo cp bin/sysc-walls-daemon /usr/local/bin/
	@echo "✓ Install complete"

# Sync sysc-Go animations from main repository
sync:
	@echo "Syncing sysc-Go animations..."
	@./scripts/sync-sysc-go.sh

# Check if sync is needed (for pre-build hooks)
check-sync:
	@echo "Checking sysc-Go sync status..."
	@MAIN_VERSION=$$(grep 'const version = ' /home/nomadx/Documents/sysc-Go/cmd/syscgo/main.go | cut -d'"' -f2); \
	LOCAL_VERSION=$$(grep 'const version = ' sysc-Go/cmd/syscgo/main.go 2>/dev/null | cut -d'"' -f2 || echo "unknown"); \
	if [ "$$LOCAL_VERSION" != "$$MAIN_VERSION" ]; then \
		echo "⚠ Warning: Local sysc-Go ($$LOCAL_VERSION) differs from main ($$MAIN_VERSION)"; \
		echo "  Run 'make sync' to update"; \
		exit 1; \
	else \
		echo "✓ sysc-Go is up to date ($$LOCAL_VERSION)"; \
	fi

# Test all effects
test:
	@echo "Testing effects..."
	@./bin/sysc-walls-display --effect fire --theme nord -no-clear &
	@PID=$$!; \
	sleep 2; \
	kill $$PID 2>/dev/null || true
	@echo "✓ Basic test passed"
	@echo ""
	@echo "Available effects:"
	@go run cmd/display/main.go -h 2>&1 | grep -A 20 "Available effects:" || true

# Show version information
version:
	@echo "sysc-walls version information:"
	@echo ""
	@SYSCGO_VERSION=$$(grep 'const version = ' sysc-Go/cmd/syscgo/main.go 2>/dev/null | cut -d'"' -f2 || echo "unknown"); \
	MIN_VERSION=$$(grep 'MinimumSyscGoVersion = ' internal/config/config.go | cut -d'"' -f2); \
	echo "  sysc-Go library: $$SYSCGO_VERSION"; \
	echo "  Required minimum: $$MIN_VERSION"; \
	if [ "$$SYSCGO_VERSION" != "unknown" ] && [ "$$SYSCGO_VERSION" \< "$$MIN_VERSION" ]; then \
		echo "  ⚠ WARNING: Version mismatch!"; \
	fi

# Pre-build check (runs sync check before building)
safe-build: check-sync build

# Help target
help:
	@echo "sysc-walls Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       - Build display and daemon binaries"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install binaries to /usr/local/bin"
	@echo "  sync        - Sync animations from main sysc-Go repo"
	@echo "  check-sync  - Check if sync is needed (fails if outdated)"
	@echo "  test        - Run basic effect tests"
	@echo "  version     - Show version information"
	@echo "  safe-build  - Check sync status before building"
	@echo "  help        - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make sync build install"
	@echo "  make safe-build"
	@echo "  make version"
