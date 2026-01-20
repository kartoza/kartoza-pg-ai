# Development Setup

## Prerequisites

- Go 1.23 or later
- PostgreSQL (for testing)
- Git

## Clone the Repository

```bash
git clone https://github.com/kartoza/kartoza-pg-ai.git
cd kartoza-pg-ai
```

## Using Nix (Recommended)

The easiest way to get a development environment:

```bash
# Enter development shell
nix develop

# Or use direnv
echo "use flake" > .envrc
direnv allow
```

This provides:

- Go toolchain
- Linting tools (golangci-lint)
- Documentation tools (mkdocs)
- All dependencies

## Manual Setup

### Install Go

```bash
# Download from https://go.dev/dl/
# Or use your package manager
sudo apt install golang-go  # Debian/Ubuntu
brew install go             # macOS
```

### Install Dependencies

```bash
go mod download
go mod tidy
```

### Install Development Tools

```bash
# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Documentation
pip install mkdocs mkdocs-material mkdocs-minify-plugin pymdown-extensions
```

## Build and Run

```bash
# Build
make build

# Run
./bin/kartoza-pg-ai

# Or run directly
go run .
```

## Testing

```bash
# Run all tests
make test

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
```

## Linting

```bash
# Run linter
make lint

# Or directly
golangci-lint run --timeout 5m
```

## Documentation

```bash
# Serve documentation locally
make docs-serve
# Open http://localhost:8000

# Build static docs
make docs
```

## Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

## Project Structure

See [Architecture](architecture.md) for detailed package documentation.

## Making Changes

1. Create a feature branch
2. Make your changes
3. Run tests: `make test`
4. Run linter: `make lint`
5. Update documentation if needed
6. Submit a pull request

## Release Process

Releases are automated via GitHub Actions:

1. Create a tag: `git tag v0.1.0`
2. Push the tag: `git push origin v0.1.0`
3. GitHub Actions builds binaries for all platforms
4. Binaries are uploaded to the GitHub release
