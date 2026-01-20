# Installation

## Prerequisites

- PostgreSQL client libraries (for connecting to databases)
- A terminal that supports ANSI colors (most modern terminals)

## Installation Methods

### Using Go

If you have Go installed:

```bash
go install github.com/kartoza/kartoza-pg-ai@latest
```

### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/kartoza/kartoza-pg-ai/releases).

#### Linux (amd64)

```bash
curl -LO https://github.com/kartoza/kartoza-pg-ai/releases/latest/download/kartoza-pg-ai-linux-amd64.tar.gz
tar xzf kartoza-pg-ai-linux-amd64.tar.gz
sudo mv kartoza-pg-ai /usr/local/bin/
```

#### macOS (Apple Silicon)

```bash
curl -LO https://github.com/kartoza/kartoza-pg-ai/releases/latest/download/kartoza-pg-ai-darwin-arm64.tar.gz
tar xzf kartoza-pg-ai-darwin-arm64.tar.gz
sudo mv kartoza-pg-ai /usr/local/bin/
```

#### macOS (Intel)

```bash
curl -LO https://github.com/kartoza/kartoza-pg-ai/releases/latest/download/kartoza-pg-ai-darwin-amd64.tar.gz
tar xzf kartoza-pg-ai-darwin-amd64.tar.gz
sudo mv kartoza-pg-ai /usr/local/bin/
```

### Using Nix

If you use Nix:

```bash
nix run github:kartoza/kartoza-pg-ai
```

Or add to your flake:

```nix
{
  inputs.kartoza-pg-ai.url = "github:kartoza/kartoza-pg-ai";
}
```

### Debian/Ubuntu

```bash
curl -LO https://github.com/kartoza/kartoza-pg-ai/releases/latest/download/kartoza-pg-ai_VERSION_amd64.deb
sudo dpkg -i kartoza-pg-ai_VERSION_amd64.deb
```

### Fedora/RHEL

```bash
curl -LO https://github.com/kartoza/kartoza-pg-ai/releases/latest/download/kartoza-pg-ai-VERSION-1.x86_64.rpm
sudo rpm -i kartoza-pg-ai-VERSION-1.x86_64.rpm
```

## Verify Installation

```bash
kartoza-pg-ai version
```

## Next Steps

- [Quick Start Guide](quickstart.md)
- [Connecting to Databases](../workflows/connecting.md)
