# Spatial Queries with PostGIS

Kartoza PG AI fully supports PostGIS spatial databases.

## Prerequisites

- PostGIS extension installed in your database
- Tables with geometry columns

## Supported Query Types

### Distance Queries

Find features within a distance:

```
Find roads within 1km
Show buildings within 500m
Points within 1 mile of location
```

**Units supported:**

- Meters (m, meters)
- Kilometers (km, kilometers)
- Miles (mi, miles)

### Length Queries

For LineString geometries:

```
What is the length of each road?
Total length of all pipelines
Show roads sorted by length
```

### Area Queries

For Polygon geometries:

```
What is the area of each parcel?
Largest polygons by area
Total area of buildings
```

## Example Session

```
🔮 Ask your database: Do we have PostGIS?

The schema shows PostGIS is installed.

🔮 Ask your database: What geometry tables do we have?

SQL: SELECT table_name FROM geometry_columns...

roads       | LINESTRING  | 4326
buildings   | POLYGON     | 4326
points      | POINT       | 4326

🔮 Ask your database: What is the total length of roads in meters?

SQL: SELECT SUM(ST_Length(geom::geography)) as total_length_meters
     FROM public.roads

total_length_meters
───────────────────
45678.23

🔮 Ask your database: Show me the 5 longest roads

SQL: SELECT *, ST_Length(geom::geography) as length_meters
     FROM public.roads
     ORDER BY ST_Length(geom::geography) DESC
     LIMIT 5

id  │ name         │ length_meters
────┼──────────────┼──────────────
1   │ Main Street  │ 2345.67
2   │ Highway 1    │ 1234.56
...
```

## Tips

### SRID Awareness

The engine uses `::geography` for accurate distance calculations in meters.

### Performance

For large tables, spatial queries may be slower. Consider:

- Creating spatial indexes
- Using LIMIT clauses
- Filtering by bounding box first

### Coordinate Systems

The engine assumes your data is in a geographic coordinate system (like EPSG:4326) for distance calculations. Projected data may give incorrect results.
