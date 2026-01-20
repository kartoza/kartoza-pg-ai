# Database Selection

The database selection screen allows you to connect to PostgreSQL databases configured in your `~/.pg_service.conf` file.

## Overview

When you select "Database Connections" from the main menu, you'll see a list of all available database services.

## Features

### Service Discovery

The application automatically scans for pg_service.conf in these locations:

1. `$PGSERVICEFILE` environment variable
2. `~/.pg_service.conf`
3. `/etc/pg_service.conf`
4. `/etc/postgresql-common/pg_service.conf`

### Connection Details

When a service is selected, you'll see:

- **Host**: Database server address
- **Port**: Connection port
- **Database**: Database name
- **User**: Username for connection

### Connection Testing

Before connecting, the application tests the connection to ensure it's valid.

## Navigation

| Key | Action |
|-----|--------|
| `↑` or `k` | Move selection up |
| `↓` or `j` | Move selection down |
| `Enter` | Test and connect |
| `r` | Refresh service list |
| `Esc` | Return to menu |

## After Connection

Once connected, the application will:

1. Harvest the database schema
2. Cache the schema locally
3. Navigate to the query interface

Future connections to the same database will use the cached schema for faster startup.
