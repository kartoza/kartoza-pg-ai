package llm

import (
	"strings"
	"testing"

	"github.com/kartoza/kartoza-pg-ai/internal/config"
)

func TestNewQueryEngine(t *testing.T) {
	schema := &config.SchemaCache{
		ServiceName: "test",
		Tables: []config.TableInfo{
			{Schema: "public", Name: "users"},
		},
	}

	engine := NewQueryEngine(schema)
	if engine == nil {
		t.Fatal("NewQueryEngine returned nil")
	}

	if engine.schema != schema {
		t.Error("schema not set correctly")
	}
}

func TestGenerateSQLNoSchema(t *testing.T) {
	engine := NewQueryEngine(nil)

	_, err := engine.GenerateSQL("test query", "")
	if err == nil {
		t.Error("expected error with nil schema")
	}
}

func TestGenerateSQLCountQuery(t *testing.T) {
	schema := &config.SchemaCache{
		Tables: []config.TableInfo{
			{Schema: "public", Name: "users"},
			{Schema: "public", Name: "orders"},
		},
	}

	engine := NewQueryEngine(schema)

	tests := []struct {
		query    string
		expected string
	}{
		{
			query:    "how many records in users",
			expected: "SELECT COUNT(*) as count FROM public.users",
		},
		{
			query:    "count of users",
			expected: "SELECT COUNT(*) as count FROM public.users",
		},
		{
			query:    "how many users",
			expected: "SELECT COUNT(*) as count FROM public.users",
		},
	}

	for _, tt := range tests {
		sql, err := engine.GenerateSQL(tt.query, "")
		if err != nil {
			t.Errorf("query '%s' failed: %v", tt.query, err)
			continue
		}

		if sql != tt.expected {
			t.Errorf("query '%s': expected '%s', got '%s'", tt.query, tt.expected, sql)
		}
	}
}

func TestGenerateSQLAllTablesCount(t *testing.T) {
	schema := &config.SchemaCache{
		Tables: []config.TableInfo{
			{Schema: "public", Name: "users"},
			{Schema: "public", Name: "orders"},
		},
	}

	engine := NewQueryEngine(schema)

	sql, err := engine.GenerateSQL("how many records in each table", "")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	// Should have UNION ALL for each table
	if !strings.Contains(sql, "UNION ALL") {
		t.Error("expected UNION ALL in SQL")
	}

	if !strings.Contains(sql, "public.users") {
		t.Error("expected public.users in SQL")
	}

	if !strings.Contains(sql, "public.orders") {
		t.Error("expected public.orders in SQL")
	}
}

func TestGenerateSQLShowQuery(t *testing.T) {
	schema := &config.SchemaCache{
		Tables: []config.TableInfo{
			{Schema: "public", Name: "customers"},
		},
	}

	engine := NewQueryEngine(schema)

	tests := []struct {
		query      string
		expectSQL  string
		expectLike string
	}{
		{
			query:      "show me the first 10 customers",
			expectLike: "SELECT * FROM public.customers LIMIT",
		},
		{
			query:      "list customers",
			expectLike: "SELECT * FROM public.customers",
		},
		{
			query:      "display the first 5 customers",
			expectLike: "LIMIT 5",
		},
	}

	for _, tt := range tests {
		sql, err := engine.GenerateSQL(tt.query, "")
		if err != nil {
			t.Errorf("query '%s' failed: %v", tt.query, err)
			continue
		}

		if !strings.Contains(sql, tt.expectLike) {
			t.Errorf("query '%s': expected to contain '%s', got '%s'", tt.query, tt.expectLike, sql)
		}
	}
}

func TestGenerateSQLTableInfoQuery(t *testing.T) {
	schema := &config.SchemaCache{
		Tables: []config.TableInfo{
			{Schema: "public", Name: "products"},
		},
	}

	engine := NewQueryEngine(schema)

	tests := []struct {
		query      string
		expectLike string
	}{
		{
			query:      "what are the columns in products",
			expectLike: "information_schema.columns",
		},
		{
			query:      "describe products",
			expectLike: "information_schema.columns",
		},
		{
			query:      "what tables exist",
			expectLike: "information_schema.tables",
		},
	}

	for _, tt := range tests {
		sql, err := engine.GenerateSQL(tt.query, "")
		if err != nil {
			t.Errorf("query '%s' failed: %v", tt.query, err)
			continue
		}

		if !strings.Contains(sql, tt.expectLike) {
			t.Errorf("query '%s': expected to contain '%s', got '%s'", tt.query, tt.expectLike, sql)
		}
	}
}

func TestGenerateSQLSpatialQuery(t *testing.T) {
	schema := &config.SchemaCache{
		HasPostGIS: true,
		Tables: []config.TableInfo{
			{
				Schema: "public",
				Name:   "roads",
				Columns: []config.ColumnInfo{
					{Name: "id", DataType: "integer"},
					{Name: "geom", DataType: "geometry", IsGeometry: true, GeomType: "LINESTRING"},
				},
			},
		},
	}

	engine := NewQueryEngine(schema)

	// Test length query
	sql, err := engine.GenerateSQL("what is the total length of roads", "")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if !strings.Contains(sql, "ST_Length") {
		t.Error("expected ST_Length in SQL")
	}

	// Test distance query
	sql, err = engine.GenerateSQL("find roads within 1km", "")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if !strings.Contains(sql, "ST_DWithin") {
		t.Error("expected ST_DWithin in SQL")
	}
}

func TestFindTable(t *testing.T) {
	schema := &config.SchemaCache{
		Tables: []config.TableInfo{
			{Schema: "public", Name: "users"},
			{Schema: "public", Name: "user_roles"},
			{Schema: "data", Name: "customer"},
		},
	}

	engine := NewQueryEngine(schema)

	// Exact match
	table := engine.findTable("users")
	if table == nil || table.Name != "users" {
		t.Error("failed to find exact match 'users'")
	}

	// Partial match
	table = engine.findTable("customer")
	if table == nil || table.Name != "customer" {
		t.Error("failed to find 'customer'")
	}

	// Plural handling
	table = engine.findTable("customers")
	if table == nil || table.Name != "customer" {
		t.Error("failed to find 'customers' -> 'customer'")
	}

	// Not found
	table = engine.findTable("nonexistent")
	if table != nil {
		t.Error("should not find nonexistent table")
	}
}

func TestSetSchema(t *testing.T) {
	engine := NewQueryEngine(nil)

	schema := &config.SchemaCache{
		ServiceName: "test",
	}

	engine.SetSchema(schema)

	if engine.schema != schema {
		t.Error("SetSchema failed")
	}
}

func TestGetSchemaContext(t *testing.T) {
	engine := NewQueryEngine(nil)

	// Nil schema
	context := engine.GetSchemaContext()
	if context != "" {
		t.Error("expected empty context for nil schema")
	}

	// With schema
	schema := &config.SchemaCache{
		HasPostGIS: true,
		Tables: []config.TableInfo{
			{
				Schema: "public",
				Name:   "users",
				Columns: []config.ColumnInfo{
					{Name: "id", DataType: "integer", IsPrimaryKey: true},
					{Name: "name", DataType: "text"},
				},
			},
		},
	}

	engine.SetSchema(schema)
	context = engine.GetSchemaContext()

	if !strings.Contains(context, "DATABASE SCHEMA") {
		t.Error("expected 'DATABASE SCHEMA' in context")
	}

	if !strings.Contains(context, "PostGIS") {
		t.Error("expected PostGIS mention in context")
	}

	if !strings.Contains(context, "users") {
		t.Error("expected 'users' table in context")
	}

	if !strings.Contains(context, "[PK]") {
		t.Error("expected [PK] marker in context")
	}
}
