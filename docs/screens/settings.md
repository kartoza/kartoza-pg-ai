# Settings

The settings screen allows you to configure application behavior.

## Available Settings

### Max History Size

Number of queries to keep in history.

- Default: 100
- Range: 10-1000

### Default Row Limit

Maximum rows to return by default.

- Default: 50
- Range: 10-500

### Schema Cache TTL

How long to cache database schemas (in minutes).

- Default: 1440 (24 hours)
- Range: 1-10080 (1 week)

### Enable Spatial Operations

Whether to enable PostGIS spatial query support.

- Default: true

## Configuration File

Settings are stored in:

```
~/.config/kartoza-pg-ai/config.json
```

## Navigation

| Key | Action |
|-----|--------|
| `Esc` | Return to menu |

## Future Features

- Interactive settings editing
- LLM model selection
- Theme customization
