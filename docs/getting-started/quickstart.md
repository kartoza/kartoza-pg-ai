# Quick Start

This guide will get you querying your PostgreSQL database in minutes.

## Step 1: Configure Database Connection

Kartoza PG AI uses the standard PostgreSQL service file for connections.

Create or edit `~/.pg_service.conf`:

```ini
[mydb]
host=localhost
port=5432
dbname=mydatabase
user=myuser
password=mypassword
```

## Step 2: Launch the Application

```bash
kartoza-pg-ai
```

You'll see the animated splash screen followed by the main menu.

## Step 3: Connect to a Database

1. Select "Database Connections" from the main menu
2. Your pg_service.conf entries will be listed
3. Use arrow keys to select a database
4. Press Enter to connect

The application will:

- Test the connection
- Harvest the database schema
- Cache the schema for faster future startups

## Step 4: Start Querying

Once connected, select "Query Database" and start asking questions:

```
🔮 Ask your database: How many tables are there?
```

The application will:

1. Convert your question to SQL
2. Execute the query
3. Display results in a formatted table

## Example Session

```
🔮 Ask your database: How many records are in each table?

SQL: SELECT 'public.customers' as table_name, COUNT(*) as row_count FROM public.customers
     UNION ALL
     SELECT 'public.orders' as table_name, COUNT(*) as row_count FROM public.orders
     ORDER BY row_count DESC

12 rows • 45.23ms

table_name        │ row_count
──────────────────┼──────────
public.orders     │ 15420
public.customers  │ 2341
public.products   │ 156
...

🔮 Ask your database: Show me the first 5 customers

SQL: SELECT * FROM public.customers LIMIT 5

5 rows • 12.45ms

id │ name          │ email                  │ created_at
───┼───────────────┼────────────────────────┼───────────────────
1  │ John Smith    │ john@example.com       │ 2024-01-15 10:30:00
2  │ Jane Doe      │ jane@example.com       │ 2024-01-16 14:22:00
...
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Ctrl+Enter | Execute query |
| Esc | Go back / Cancel |
| ↑/k | Navigate up |
| ↓/j | Navigate down |
| q | Quit |

## Next Steps

- [Natural Language Queries](../workflows/queries.md)
- [Spatial Queries with PostGIS](../workflows/spatial.md)
- [Query History](../screens/query-history.md)
