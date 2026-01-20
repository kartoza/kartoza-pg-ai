# Query Interface

The query interface is where you interact with your database using natural language.

## Layout

The screen is divided into:

1. **Header**: Shows connection status and statistics
2. **Results Area**: Displays query results or welcome message
3. **Prompt Area**: Where you type your questions

## Asking Questions

Type your question in the prompt area and press `Ctrl+Enter` to execute.

### Example Questions

**Count queries:**

- "How many records are in each table?"
- "Count of users"
- "How many orders"

**List queries:**

- "Show me the first 10 customers"
- "List all products"
- "Display the last 20 orders"

**Schema queries:**

- "What are the columns in users?"
- "Describe the orders table"
- "List all tables"

**Spatial queries (PostGIS):**

- "Find roads within 1km of this point"
- "What is the total area of all polygons?"
- "Show me the length of each road"

## Results Display

Results are shown in a formatted table with:

- Column headers in orange
- Alternating row colors for readability
- Truncated cells for long values
- Row count and execution time

## Conversation Context

The query engine maintains context from previous queries, allowing follow-up questions:

```
You: How many customers are there?
> 1,234 customers

You: Show me the first 5
> (Shows first 5 customers)
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+Enter` | Execute query |
| `Esc` | Return to menu |
| `Ctrl+C` | Cancel current query |
