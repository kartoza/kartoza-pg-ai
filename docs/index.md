# Kartoza PG AI

A beautiful TUI application for querying PostgreSQL databases using natural language.

## Features

- **Natural Language Queries**: Ask questions about your database in plain English
- **PostGIS Support**: Full support for spatial queries and geographic data
- **Schema Caching**: Fast startup times with intelligent schema caching
- **Conversation Context**: Follow-up queries maintain context from previous results
- **Beautiful Interface**: Modern terminal UI with Kartoza branding

## Quick Start

```bash
# Install via go
go install github.com/kartoza/kartoza-pg-ai@latest

# Or download from releases
# https://github.com/kartoza/kartoza-pg-ai/releases

# Run the application
kartoza-pg-ai
```

## Screenshots

The application features:

- Animated splash screen with Kartoza branding
- Main menu with keyboard navigation
- Database connection management via pg_service.conf
- Query interface with natural language input
- Beautiful results tables with syntax highlighting

## Configuration

Kartoza PG AI uses your existing `~/.pg_service.conf` file for database connections.

Example `~/.pg_service.conf`:

```ini
[mydb]
host=localhost
port=5432
dbname=mydb
user=postgres
```

## Example Queries

Once connected to a database, you can ask questions like:

- "How many records are in each table?"
- "Show me the first 10 customers"
- "What are the columns in the orders table?"
- "Find all roads within 1km of this point" (PostGIS)
- "What is the total length of all roads?"

## License

MIT License - See LICENSE file for details.

## Contributing

Contributions are welcome! Please see our [Developer Guide](developer/setup.md) for setup instructions.
