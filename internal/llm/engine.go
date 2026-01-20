package llm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kartoza/kartoza-pg-ai/internal/config"
)

// QueryEngine handles natural language to SQL conversion
// This is a simplified rule-based engine. For production, integrate with
// a local LLM like llama.cpp or similar.
type QueryEngine struct {
	schema *config.SchemaCache
}

// NewQueryEngine creates a new query engine
func NewQueryEngine(schema *config.SchemaCache) *QueryEngine {
	return &QueryEngine{schema: schema}
}

// GenerateSQL converts natural language to SQL
func (e *QueryEngine) GenerateSQL(query string, context string) (string, error) {
	if e.schema == nil {
		return "", fmt.Errorf("no schema loaded")
	}

	query = strings.TrimSpace(strings.ToLower(query))

	// Simple pattern matching for common queries
	// In production, replace with actual LLM integration

	// Count queries
	if countMatch := e.matchCountQuery(query); countMatch != "" {
		return countMatch, nil
	}

	// Show/list queries
	if showMatch := e.matchShowQuery(query); showMatch != "" {
		return showMatch, nil
	}

	// Table info queries
	if tableMatch := e.matchTableQuery(query); tableMatch != "" {
		return tableMatch, nil
	}

	// Spatial queries (if PostGIS available)
	if e.schema.HasPostGIS {
		if spatialMatch := e.matchSpatialQuery(query); spatialMatch != "" {
			return spatialMatch, nil
		}
	}

	// Generic select with limit
	if selectMatch := e.matchSelectQuery(query); selectMatch != "" {
		return selectMatch, nil
	}

	// Fallback: try to extract table name and do basic select
	for _, table := range e.schema.Tables {
		tableName := strings.ToLower(table.Name)
		if strings.Contains(query, tableName) {
			return fmt.Sprintf("SELECT * FROM %s.%s LIMIT 50", table.Schema, table.Name), nil
		}
	}

	return "", fmt.Errorf("could not understand query: %s", query)
}

func (e *QueryEngine) matchCountQuery(query string) string {
	countPatterns := []string{
		`how many (?:rows|records|entries) (?:are )?in (?:the )?(?:table )?(\w+)`,
		`count (?:of |all )?(?:rows |records )?(?:in )?(?:the )?(\w+)`,
		`(\w+) (?:row |record )?count`,
		`how many (\w+)`,
	}

	// Check for "all tables" or "each table" pattern
	if strings.Contains(query, "each table") || strings.Contains(query, "all tables") {
		var queries []string
		for _, table := range e.schema.Tables {
			queries = append(queries, fmt.Sprintf(
				"SELECT '%s.%s' as table_name, COUNT(*) as row_count FROM %s.%s",
				table.Schema, table.Name, table.Schema, table.Name,
			))
		}
		if len(queries) > 0 {
			return strings.Join(queries, " UNION ALL ") + " ORDER BY row_count DESC"
		}
	}

	for _, pattern := range countPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			tableName := matches[1]
			if table := e.findTable(tableName); table != nil {
				return fmt.Sprintf("SELECT COUNT(*) as count FROM %s.%s", table.Schema, table.Name)
			}
		}
	}

	return ""
}

func (e *QueryEngine) matchShowQuery(query string) string {
	showPatterns := []string{
		`show (?:me )?(?:the )?(?:first )?(\d+)? ?(?:rows |records )?(?:from |of )?(?:the )?(\w+)`,
		`list (?:the )?(?:first )?(\d+)? ?(\w+)`,
		`get (?:the )?(?:first )?(\d+)? ?(\w+)`,
		`display (?:the )?(?:first )?(\d+)? ?(\w+)`,
	}

	for _, pattern := range showPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			limit := "50"
			tableName := ""

			for i, m := range matches[1:] {
				if m == "" {
					continue
				}
				// Check if it's a number (limit) or table name
				if regexp.MustCompile(`^\d+$`).MatchString(m) {
					limit = m
				} else if i > 0 || !regexp.MustCompile(`^\d+$`).MatchString(matches[1]) {
					tableName = m
				}
			}

			if tableName != "" {
				if table := e.findTable(tableName); table != nil {
					return fmt.Sprintf("SELECT * FROM %s.%s LIMIT %s", table.Schema, table.Name, limit)
				}
			}
		}
	}

	return ""
}

func (e *QueryEngine) matchTableQuery(query string) string {
	// Table structure queries
	if strings.Contains(query, "tables") && (strings.Contains(query, "list") || strings.Contains(query, "show") || strings.Contains(query, "what")) {
		return `SELECT table_schema, table_name,
			(SELECT COUNT(*) FROM information_schema.columns c WHERE c.table_schema = t.table_schema AND c.table_name = t.table_name) as column_count
			FROM information_schema.tables t
			WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema')
			ORDER BY table_schema, table_name`
	}

	// Column info
	colPatterns := []string{
		`what (?:are )?(?:the )?columns (?:in |of )?(?:the )?(\w+)`,
		`describe (?:the )?(\w+)`,
		`schema (?:of |for )?(?:the )?(\w+)`,
		`(\w+) structure`,
	}

	for _, pattern := range colPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			tableName := matches[1]
			if table := e.findTable(tableName); table != nil {
				return fmt.Sprintf(`SELECT column_name, data_type, is_nullable, column_default
					FROM information_schema.columns
					WHERE table_schema = '%s' AND table_name = '%s'
					ORDER BY ordinal_position`, table.Schema, table.Name)
			}
		}
	}

	// Largest tables
	if strings.Contains(query, "largest") && strings.Contains(query, "table") {
		var queries []string
		for _, table := range e.schema.Tables {
			queries = append(queries, fmt.Sprintf(
				"SELECT '%s.%s' as table_name, COUNT(*) as row_count FROM %s.%s",
				table.Schema, table.Name, table.Schema, table.Name,
			))
		}
		if len(queries) > 0 {
			return strings.Join(queries, " UNION ALL ") + " ORDER BY row_count DESC LIMIT 10"
		}
	}

	return ""
}

func (e *QueryEngine) matchSpatialQuery(query string) string {
	// Find geometry columns
	var geomTables []config.TableInfo
	for _, table := range e.schema.Tables {
		for _, col := range table.Columns {
			if col.IsGeometry {
				geomTables = append(geomTables, table)
				break
			}
		}
	}

	if len(geomTables) == 0 {
		return ""
	}

	// Distance queries
	distancePatterns := []string{
		`within (\d+(?:\.\d+)?)\s*(?:km|kilometers?|m|meters?|mi|miles?)`,
		`(\d+(?:\.\d+)?)\s*(?:km|kilometers?|m|meters?|mi|miles?) (?:from|of|away)`,
	}

	for _, pattern := range distancePatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			// Simple spatial query template
			table := geomTables[0]
			geomCol := ""
			for _, col := range table.Columns {
				if col.IsGeometry {
					geomCol = col.Name
					break
				}
			}

			if geomCol != "" {
				// Convert distance to meters
				dist := matches[1]
				distMeters := dist
				if strings.Contains(query, "km") || strings.Contains(query, "kilometer") {
					distMeters = dist + " * 1000"
				} else if strings.Contains(query, "mi") || strings.Contains(query, "mile") {
					distMeters = dist + " * 1609.34"
				}

				return fmt.Sprintf(`SELECT * FROM %s.%s
					WHERE ST_DWithin(%s::geography, ST_MakePoint(0, 0)::geography, %s)
					LIMIT 50`, table.Schema, table.Name, geomCol, distMeters)
			}
		}
	}

	// Area queries
	if strings.Contains(query, "area") || strings.Contains(query, "size") {
		for _, table := range geomTables {
			for _, col := range table.Columns {
				if col.IsGeometry && (col.GeomType == "POLYGON" || col.GeomType == "MULTIPOLYGON") {
					return fmt.Sprintf(`SELECT *, ST_Area(%s::geography) as area_sqm
						FROM %s.%s
						ORDER BY ST_Area(%s::geography) DESC
						LIMIT 50`, col.Name, table.Schema, table.Name, col.Name)
				}
			}
		}
	}

	// Length queries
	if strings.Contains(query, "length") || strings.Contains(query, "meters of road") || strings.Contains(query, "distance") {
		for _, table := range geomTables {
			for _, col := range table.Columns {
				if col.IsGeometry && (col.GeomType == "LINESTRING" || col.GeomType == "MULTILINESTRING") {
					if strings.Contains(query, "total") || strings.Contains(query, "sum") {
						return fmt.Sprintf(`SELECT SUM(ST_Length(%s::geography)) as total_length_meters
							FROM %s.%s`, col.Name, table.Schema, table.Name)
					}
					return fmt.Sprintf(`SELECT *, ST_Length(%s::geography) as length_meters
						FROM %s.%s
						ORDER BY ST_Length(%s::geography) DESC
						LIMIT 50`, col.Name, table.Schema, table.Name, col.Name)
				}
			}
		}
	}

	return ""
}

func (e *QueryEngine) matchSelectQuery(query string) string {
	selectPatterns := []string{
		`select (?:all )?(?:from )?(\w+)`,
		`fetch (?:all )?(?:from )?(\w+)`,
		`retrieve (?:all )?(?:from )?(\w+)`,
	}

	for _, pattern := range selectPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			tableName := matches[1]
			if table := e.findTable(tableName); table != nil {
				return fmt.Sprintf("SELECT * FROM %s.%s LIMIT 50", table.Schema, table.Name)
			}
		}
	}

	return ""
}

func (e *QueryEngine) findTable(name string) *config.TableInfo {
	name = strings.ToLower(name)

	// Exact match
	for _, table := range e.schema.Tables {
		if strings.ToLower(table.Name) == name {
			return &table
		}
	}

	// Partial match
	for _, table := range e.schema.Tables {
		if strings.Contains(strings.ToLower(table.Name), name) {
			return &table
		}
	}

	// Singular/plural handling
	singular := strings.TrimSuffix(name, "s")
	for _, table := range e.schema.Tables {
		tableLower := strings.ToLower(table.Name)
		if tableLower == singular || strings.TrimSuffix(tableLower, "s") == singular {
			return &table
		}
	}

	return nil
}

// SetSchema updates the schema for the query engine
func (e *QueryEngine) SetSchema(schema *config.SchemaCache) {
	e.schema = schema
}

// GetSchemaContext returns the schema description for LLM context
func (e *QueryEngine) GetSchemaContext() string {
	if e.schema == nil {
		return ""
	}
	return generateSchemaDescription(e.schema)
}

func generateSchemaDescription(cache *config.SchemaCache) string {
	var desc strings.Builder

	desc.WriteString("DATABASE SCHEMA:\n")
	desc.WriteString("================\n\n")

	if cache.HasPostGIS {
		desc.WriteString("PostGIS is installed - spatial queries are supported.\n\n")
	}

	desc.WriteString("TABLES:\n")
	for _, t := range cache.Tables {
		desc.WriteString(fmt.Sprintf("- %s.%s", t.Schema, t.Name))
		if t.Comment != "" {
			desc.WriteString(fmt.Sprintf(" (%s)", t.Comment))
		}
		desc.WriteString("\n")
		for _, c := range t.Columns {
			desc.WriteString(fmt.Sprintf("    - %s (%s)", c.Name, c.DataType))
			if c.IsPrimaryKey {
				desc.WriteString(" [PK]")
			}
			if c.IsForeignKey {
				desc.WriteString(fmt.Sprintf(" [FK -> %s.%s]", c.FKTable, c.FKColumn))
			}
			if c.IsGeometry {
				desc.WriteString(fmt.Sprintf(" [GEOMETRY: %s]", c.GeomType))
			}
			desc.WriteString("\n")
		}
		desc.WriteString("\n")
	}

	return desc.String()
}
