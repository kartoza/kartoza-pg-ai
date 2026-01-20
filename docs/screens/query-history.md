# Query History

The query history screen shows your previous natural language queries and their results.

## Features

### Persistent History

Query history is saved between sessions in:

```
~/.config/kartoza-pg-ai/config.json
```

### History Entries

Each entry includes:

- Timestamp
- Natural language query
- Generated SQL
- Execution time
- Row count
- Success/failure status

### History Limits

By default, the last 100 queries are stored. This can be configured in Settings.

## Navigation

| Key | Action |
|-----|--------|
| `↑` or `k` | Scroll up |
| `↓` or `j` | Scroll down |
| `Esc` | Return to menu |

## Future Features

- Re-run queries
- Export history
- Search history
- Clear history
