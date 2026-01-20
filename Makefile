# Kartoza PG AI Makefile
# ======================

BINARY_NAME := kartoza-pg-ai
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Directories
BIN_DIR := bin
RELEASE_DIR := release
COVERAGE_FILE := coverage.out

# Go commands
GO := go
GOFMT := gofmt
GOLINT := golangci-lint

# Build targets
.PHONY: all build static build-all clean test deps fmt lint check install uninstall
.PHONY: release release-upload release-clean
.PHONY: deb rpm snap flatpak packages packages-clean
.PHONY: dev help info docs docs-serve

# Default target
all: test build

# Build dynamic binary
build:
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) .

# Build static binary (no CGO)
static:
	CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-static .

# Build both
build-all: test build static

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)
	rm -f $(COVERAGE_FILE)
	$(GO) clean

# Run tests with race detection and coverage
test:
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) ./...

# Download and tidy dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Format code
fmt:
	$(GOFMT) -s -w .
	$(GO) fmt ./...

# Run linter
lint:
	$(GOLINT) run --timeout 5m

# Quality gate: format, lint, test
check: fmt lint test

# Install to /usr/local/bin
install: static
	cp $(BIN_DIR)/$(BINARY_NAME)-static /usr/local/bin/$(BINARY_NAME)

# Remove from /usr/local/bin
uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)

# ============================
# Release targets
# ============================

# Build multi-platform release binaries
release:
	@mkdir -p $(RELEASE_DIR)
	@echo "Building linux-amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Building linux-arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-linux-arm64 .
	@echo "Building darwin-amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@echo "Building darwin-arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Building windows-amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Creating tarballs..."
	cd $(RELEASE_DIR) && tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd $(RELEASE_DIR) && tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	cd $(RELEASE_DIR) && tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd $(RELEASE_DIR) && tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	cd $(RELEASE_DIR) && tar -czf $(BINARY_NAME)-windows-amd64.tar.gz $(BINARY_NAME)-windows-amd64.exe
	@echo "Generating checksums..."
	cd $(RELEASE_DIR) && sha256sum *.tar.gz > checksums.txt
	@echo "Release binaries created in $(RELEASE_DIR)/"

# Upload release to GitHub
release-upload: release
ifndef TAG
	$(error TAG is required. Usage: make release-upload TAG=v0.1.0)
endif
	gh release create $(TAG) $(RELEASE_DIR)/*.tar.gz $(RELEASE_DIR)/checksums.txt --generate-notes

# Clean release directory
release-clean:
	rm -rf $(RELEASE_DIR)

# ============================
# Package targets
# ============================

# Build Debian package
deb:
	@echo "Building Debian package..."
	cd packaging/debian && dpkg-buildpackage -us -uc -b

# Build RPM package
rpm:
	@echo "Building RPM package..."
	rpmbuild -bb packaging/rpm/$(BINARY_NAME).spec

# Build Snap package
snap:
	@echo "Building Snap package..."
	cd packaging/snap && snapcraft

# Build Flatpak package
flatpak:
	@echo "Building Flatpak package..."
	flatpak-builder --force-clean build-dir packaging/flatpak/com.kartoza.PgAI.yml

# Build all packages
packages: deb rpm snap flatpak

# Clean packages
packages-clean:
	rm -rf packaging/debian/*.deb
	rm -rf packaging/rpm/RPMS
	rm -rf packaging/snap/*.snap
	rm -rf build-dir

# ============================
# Development targets
# ============================

# Run the application
dev:
	$(GO) run .

# Serve documentation locally
docs-serve:
	mkdocs serve

# Build documentation
docs:
	mkdocs build

# Display build info
info:
	@echo "Binary:  $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Go:      $(shell $(GO) version)"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build   - Build dynamic binary"
	@echo "  make static  - Build static binary"
	@echo "  make release - Build all platforms"

# Show help
help:
	@echo "Kartoza PG AI - Makefile"
	@echo ""
	@echo "Build targets:"
	@echo "  all          Build and test (default)"
	@echo "  build        Build dynamic binary"
	@echo "  static       Build static binary"
	@echo "  build-all    Test and build both binaries"
	@echo "  clean        Remove build artifacts"
	@echo ""
	@echo "Test targets:"
	@echo "  test         Run tests with coverage"
	@echo "  lint         Run linter"
	@echo "  check        Run fmt, lint, and test"
	@echo ""
	@echo "Dependency targets:"
	@echo "  deps         Download and tidy dependencies"
	@echo "  fmt          Format code"
	@echo ""
	@echo "Install targets:"
	@echo "  install      Install to /usr/local/bin"
	@echo "  uninstall    Remove from /usr/local/bin"
	@echo ""
	@echo "Release targets:"
	@echo "  release          Build multi-platform binaries"
	@echo "  release-upload   Upload to GitHub (requires TAG=vX.Y.Z)"
	@echo "  release-clean    Clean release directory"
	@echo ""
	@echo "Package targets:"
	@echo "  deb          Build Debian package"
	@echo "  rpm          Build RPM package"
	@echo "  snap         Build Snap package"
	@echo "  flatpak      Build Flatpak package"
	@echo "  packages     Build all packages"
	@echo ""
	@echo "Documentation targets:"
	@echo "  docs         Build documentation"
	@echo "  docs-serve   Serve documentation locally"
	@echo ""
	@echo "Development targets:"
	@echo "  dev          Run the application"
	@echo "  info         Show build info"
	@echo "  help         Show this help"
