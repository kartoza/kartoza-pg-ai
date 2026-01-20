# Kartoza PG AI

A beautiful TUI application for querying PostgreSQL databases using natural language.

```
  ██████╗  ██████╗      █████╗ ██╗
  ██╔══██╗██╔════╝     ██╔══██╗██║
  ██████╔╝██║  ███╗    ███████║██║
  ██╔═══╝ ██║   ██║    ██╔══██║██║
  ██║     ╚██████╔╝    ██║  ██║██║
  ╚═╝      ╚═════╝     ╚═╝  ╚═╝╚═╝
```

## Features

- **Natural Language Queries**: Ask questions about your database in plain English
- **PostGIS Support**: Full support for spatial queries and geographic data
- **Schema Caching**: Fast startup times with intelligent schema caching
- **Conversation Context**: Follow-up queries maintain context from previous results
- **Beautiful Interface**: Modern terminal UI with Kartoza branding
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Quick Start

```bash
# Install
go install github.com/kartoza/kartoza-pg-ai@latest

# Configure database connection
cat > ~/.pg_service.conf << EOF
[mydb]
host=localhost
port=5432
dbname=mydb
user=postgres
EOF

# Run
kartoza-pg-ai
```

## Example Queries

Once connected to a database:

- "How many records are in each table?"
- "Show me the first 10 customers"
- "What are the columns in the orders table?"
- "Find all roads within 1km of this point" (PostGIS)
- "What is the total length of all roads?"

## Installation

### Using Go

```bash
go install github.com/kartoza/kartoza-pg-ai@latest
```

### Download Binary

Download from [releases](https://github.com/kartoza/kartoza-pg-ai/releases).

### Using Nix

```bash
nix run github:kartoza/kartoza-pg-ai
```

## Configuration

Kartoza PG AI uses `~/.pg_service.conf` for database connections:

```ini
[production]
host=db.example.com
port=5432
dbname=production
user=admin

[development]
host=localhost
port=5432
dbname=devdb
user=dev
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+Enter` | Execute query |
| `Esc` | Go back / Cancel |
| `↑/k` | Navigate up |
| `↓/j` | Navigate down |
| `Enter` | Select item |
| `q` | Quit |

## Development

```bash
# Clone
git clone https://github.com/kartoza/kartoza-pg-ai.git
cd kartoza-pg-ai

# Using Nix (recommended)
nix develop

# Or manually
go mod download
make build

# Run tests
make test

# Run linter
make lint

# Build for all platforms
make release
```

## Documentation

Full documentation available at: https://kartoza.github.io/kartoza-pg-ai/

Or serve locally:

```bash
mkdocs serve
```

## License

MIT License - See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please read our contributing guidelines first.

## Credits

Built with:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [lib/pq](https://github.com/lib/pq) - PostgreSQL driver

Made with love by [Kartoza](https://kartoza.com).
