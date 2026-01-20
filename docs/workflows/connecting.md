# Connecting to Databases

This guide covers how to configure and connect to PostgreSQL databases.

## Prerequisites

- PostgreSQL database running and accessible
- Network access to the database server
- Valid credentials

## Configuration

### pg_service.conf

Kartoza PG AI uses the standard PostgreSQL service file for connections.

Create or edit `~/.pg_service.conf`:

```ini
[mydb]
host=localhost
port=5432
dbname=mydb
user=myuser
password=mypassword
sslmode=prefer

[production]
host=db.example.com
port=5432
dbname=proddb
user=admin
password=secret
sslmode=require
```

### Connection Options

| Option | Description |
|--------|-------------|
| `host` | Database server hostname or IP |
| `port` | Connection port (default: 5432) |
| `dbname` | Database name |
| `user` | Username |
| `password` | Password |
| `sslmode` | SSL mode (disable, allow, prefer, require) |

### Security

- Never commit pg_service.conf to version control
- Set restrictive permissions: `chmod 600 ~/.pg_service.conf`
- Use SSL in production (`sslmode=require`)

## Connecting

1. Launch the application: `kartoza-pg-ai`
2. Select "Database Connections" from the menu
3. Choose a database from the list
4. Wait for connection test and schema harvesting

## Troubleshooting

### Connection Refused

- Check if PostgreSQL is running
- Verify host and port
- Check firewall rules

### Authentication Failed

- Verify username and password
- Check pg_hba.conf on the server

### SSL Errors

- Try `sslmode=prefer` or `sslmode=disable` for local development
- Ensure server has valid SSL certificates for production

### Schema Harvesting Slow

- Large databases may take time
- Schema is cached for subsequent connections
- Use "Refresh Schema" to update cache
