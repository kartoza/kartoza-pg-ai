# Natural Language Queries

This guide covers how to effectively query your database using natural language.

## Query Types

### Count Queries

Get row counts for tables:

```
How many records are in users?
Count of orders
How many customers do we have?
```

For all tables:

```
How many records are in each table?
Row counts for all tables
```

### Select Queries

Retrieve data from tables:

```
Show me the first 10 users
List all products
Display customers
Get the last 50 orders
```

### Schema Queries

Explore database structure:

```
What tables exist?
List all tables
Describe the users table
What are the columns in orders?
```

### Aggregation Queries

Get aggregated data:

```
What are the largest tables?
Total count of all records
```

## Tips for Better Queries

### Be Specific

Instead of:

```
Show me data
```

Try:

```
Show me the first 20 customers
```

### Use Table Names

The engine recognizes table names from your schema:

```
List orders     ← If you have an 'orders' table
Show customers  ← If you have a 'customers' table
```

### Plural Forms

Both singular and plural work:

```
Show me users    ← Works
Show me user     ← Also works
```

## Limitations

The current rule-based engine has some limitations:

1. **Complex JOINs**: Multi-table queries may not be generated correctly
2. **Custom Functions**: User-defined functions won't be recognized
3. **Ambiguous Queries**: Very vague queries may fail

For complex queries, you can still use the generated SQL as a starting point.
