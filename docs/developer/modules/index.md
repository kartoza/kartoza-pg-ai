# Modules Overview

Kartoza PG AI is organized into the following packages:

## Package Hierarchy

```
internal/
├── tui/        # Terminal User Interface
├── postgres/   # PostgreSQL integration
├── llm/        # LLM/query engine
├── config/     # Configuration management
├── models/     # Data models (future)
└── storage/    # Local storage (future)
```

## Package Descriptions

### tui

The terminal user interface package contains all Bubble Tea models and views:

- `app.go` - Main application orchestrator
- `menu.go` - Main menu screen
- `query.go` - Query interface
- `database.go` - Database selection
- `splash.go` - Splash screens
- `widgets.go` - Shared components

### postgres

PostgreSQL integration:

- `service.go` - pg_service.conf parser
- `schema.go` - Schema harvester

### llm

Query engine for natural language to SQL:

- `engine.go` - Rule-based query generator

### config

Application configuration:

- `config.go` - Config loading, saving, and structures

## Module Details

- [TUI Package](tui.md)
- [PostgreSQL Package](postgres.md)
- [LLM Package](llm.md)
- [Config Package](config.md)
