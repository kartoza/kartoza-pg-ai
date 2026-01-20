# Main Menu

The main menu is the starting point of the application after the splash screen.

## Menu Options

### Query Database

Opens the query interface where you can ask natural language questions about your database.

**Note**: You must connect to a database first. If not connected, this will redirect to Database Connections.

### Database Connections

Opens the database connection screen where you can:

- View available PostgreSQL services from `~/.pg_service.conf`
- Test connections
- Select a database to connect to

### Query History

View your previous queries and their results. History is persisted between sessions.

### Settings

Configure application settings (coming soon).

### Quit

Exit the application with an animated exit splash screen.

## Navigation

| Key | Action |
|-----|--------|
| `↑` or `k` | Move selection up |
| `↓` or `j` | Move selection down |
| `Enter` or `Space` | Select item |
| `q` | Quit application |
