package postgres

import (
	"database/sql"
	"time"

	"github.com/kartoza/kartoza-pg-ai/internal/config"
)

// ProgressCallback is called with progress updates during schema harvesting
type ProgressCallback func(current, total int, message string)

// SchemaHarvester harvests database schema information
type SchemaHarvester struct {
	db       *sql.DB
	progress ProgressCallback
}

// NewSchemaHarvester creates a new schema harvester
func NewSchemaHarvester(db *sql.DB) *SchemaHarvester {
	return &SchemaHarvester{db: db}
}

// SetProgressCallback sets a callback function for progress updates
func (h *SchemaHarvester) SetProgressCallback(cb ProgressCallback) {
	h.progress = cb
}

// reportProgress safely reports progress if callback is set
func (h *SchemaHarvester) reportProgress(current, total int, message string) {
	if h.progress != nil {
		h.progress(current, total, message)
	}
}

// SchemaCounts holds the counts of schema objects
type SchemaCounts struct {
	Tables    int
	Views     int
	Functions int
	Total     int
}

// CountSchemaObjects counts all schema objects without fetching details
func (h *SchemaHarvester) CountSchemaObjects() (*SchemaCounts, error) {
	counts := &SchemaCounts{}

	// Count tables
	err := h.db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
	`).Scan(&counts.Tables)
	if err != nil {
		return nil, err
	}

	// Count views
	err = h.db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.views
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
	`).Scan(&counts.Views)
	if err != nil {
		return nil, err
	}

	// Count functions (limited)
	err = h.db.QueryRow(`
		SELECT COUNT(*)
		FROM (
			SELECT 1 FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
			  AND p.prokind = 'f'
			LIMIT 500
		) AS funcs
	`).Scan(&counts.Functions)
	if err != nil {
		return nil, err
	}

	counts.Total = counts.Tables + counts.Views + counts.Functions
	return counts, nil
}

// Harvest harvests the complete database schema
func (h *SchemaHarvester) Harvest(serviceName string) (*config.SchemaCache, error) {
	cache := &config.SchemaCache{
		ServiceName: serviceName,
		Tables:      []config.TableInfo{},
		Views:       []config.ViewInfo{},
		Functions:   []config.FunctionInfo{},
		CachedAt:    time.Now(),
	}

	// First count all objects for progress reporting
	counts, err := h.CountSchemaObjects()
	if err != nil {
		counts = &SchemaCounts{Total: 1} // Fallback to avoid division by zero
	}

	current := 0
	h.reportProgress(current, counts.Total, "Checking PostGIS availability...")

	// Check PostGIS availability
	hasPostGIS, err := h.checkPostGIS()
	if err == nil {
		cache.HasPostGIS = hasPostGIS
	}

	// Get PostgreSQL version
	version, err := h.getVersion()
	if err == nil {
		cache.Version = version
	}

	h.reportProgress(current, counts.Total, "Harvesting tables...")

	// Harvest tables with progress
	tables, err := h.harvestTablesWithProgress(counts, &current)
	if err != nil {
		return nil, err
	}
	cache.Tables = tables

	h.reportProgress(current, counts.Total, "Harvesting views...")

	// Harvest views with progress
	views, err := h.harvestViewsWithProgress(counts, &current)
	if err != nil {
		return nil, err
	}
	cache.Views = views

	h.reportProgress(current, counts.Total, "Harvesting functions...")

	// Harvest functions with progress
	functions, err := h.harvestFunctionsWithProgress(counts, &current)
	if err != nil {
		return nil, err
	}
	cache.Functions = functions

	h.reportProgress(counts.Total, counts.Total, "Schema harvesting complete!")

	return cache, nil
}

// checkPostGIS checks if PostGIS is installed
func (h *SchemaHarvester) checkPostGIS() (bool, error) {
	var exists bool
	err := h.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_extension WHERE extname = 'postgis'
		)
	`).Scan(&exists)
	return exists, err
}

// getVersion gets the PostgreSQL version
func (h *SchemaHarvester) getVersion() (string, error) {
	var version string
	err := h.db.QueryRow("SELECT version()").Scan(&version)
	return version, err
}

// harvestTablesWithProgress harvests all user tables with progress reporting
func (h *SchemaHarvester) harvestTablesWithProgress(counts *SchemaCounts, current *int) ([]config.TableInfo, error) {
	query := `
		SELECT
			table_schema,
			table_name,
			COALESCE(obj_description((quote_ident(table_schema) || '.' || quote_ident(table_name))::regclass), '') as comment
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name
	`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []config.TableInfo
	for rows.Next() {
		var t config.TableInfo
		if err := rows.Scan(&t.Schema, &t.Name, &t.Comment); err != nil {
			return nil, err
		}

		// Get columns for this table (metadata only, no data)
		columns, err := h.harvestColumns(t.Schema, t.Name)
		if err != nil {
			return nil, err
		}
		t.Columns = columns

		tables = append(tables, t)

		*current++
		h.reportProgress(*current, counts.Total, "Table: "+t.Schema+"."+t.Name)
	}

	return tables, rows.Err()
}

// harvestColumns harvests columns for a specific table
func (h *SchemaHarvester) harvestColumns(schema, table string) ([]config.ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			COALESCE(tc.constraint_type = 'PRIMARY KEY', false) as is_pk,
			COALESCE(tc.constraint_type = 'FOREIGN KEY', false) as is_fk,
			COALESCE(ccu.table_name, '') as fk_table,
			COALESCE(ccu.column_name, '') as fk_column,
			COALESCE(col_description((quote_ident(c.table_schema) || '.' || quote_ident(c.table_name))::regclass, c.ordinal_position), '') as comment
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON c.table_schema = kcu.table_schema
			AND c.table_name = kcu.table_name
			AND c.column_name = kcu.column_name
		LEFT JOIN information_schema.table_constraints tc
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = tc.table_schema
		LEFT JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.constraint_type = 'FOREIGN KEY'
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := h.db.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []config.ColumnInfo
	seen := make(map[string]bool)

	for rows.Next() {
		var col config.ColumnInfo
		if err := rows.Scan(
			&col.Name, &col.DataType, &col.IsNullable,
			&col.IsPrimaryKey, &col.IsForeignKey,
			&col.FKTable, &col.FKColumn, &col.Comment,
		); err != nil {
			return nil, err
		}

		// Avoid duplicates from JOIN
		if seen[col.Name] {
			continue
		}
		seen[col.Name] = true

		// Check for geometry columns
		if col.DataType == "USER-DEFINED" || col.DataType == "geometry" || col.DataType == "geography" {
			col.IsGeometry = true
			geomInfo, _ := h.getGeometryInfo(schema, table, col.Name)
			if geomInfo != nil {
				col.GeomType = geomInfo.GeomType
				col.SRID = geomInfo.SRID
			}
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// GeometryInfo holds geometry column metadata
type GeometryInfo struct {
	GeomType string
	SRID     int
}

// getGeometryInfo gets geometry column information from PostGIS
func (h *SchemaHarvester) getGeometryInfo(schema, table, column string) (*GeometryInfo, error) {
	query := `
		SELECT type, srid
		FROM geometry_columns
		WHERE f_table_schema = $1 AND f_table_name = $2 AND f_geometry_column = $3
	`

	var info GeometryInfo
	err := h.db.QueryRow(query, schema, table, column).Scan(&info.GeomType, &info.SRID)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// harvestViewsWithProgress harvests all user views with progress reporting
func (h *SchemaHarvester) harvestViewsWithProgress(counts *SchemaCounts, current *int) ([]config.ViewInfo, error) {
	query := `
		SELECT
			table_schema,
			table_name,
			COALESCE(obj_description((quote_ident(table_schema) || '.' || quote_ident(table_name))::regclass), '') as comment
		FROM information_schema.views
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name
	`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []config.ViewInfo
	for rows.Next() {
		var v config.ViewInfo
		if err := rows.Scan(&v.Schema, &v.Name, &v.Comment); err != nil {
			return nil, err
		}

		// Get columns for this view (metadata only, no data)
		columns, err := h.harvestColumns(v.Schema, v.Name)
		if err != nil {
			// Views might have issues, skip columns
			columns = []config.ColumnInfo{}
		}
		v.Columns = columns

		views = append(views, v)

		*current++
		h.reportProgress(*current, counts.Total, "View: "+v.Schema+"."+v.Name)
	}

	return views, rows.Err()
}

// harvestFunctionsWithProgress harvests commonly used functions with progress reporting
func (h *SchemaHarvester) harvestFunctionsWithProgress(counts *SchemaCounts, current *int) ([]config.FunctionInfo, error) {
	query := `
		SELECT
			n.nspname as schema,
			p.proname as name,
			pg_get_function_result(p.oid) as return_type,
			pg_get_function_arguments(p.oid) as arguments,
			COALESCE(obj_description(p.oid), '') as comment
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND p.prokind = 'f'
		ORDER BY n.nspname, p.proname
		LIMIT 500
	`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []config.FunctionInfo
	for rows.Next() {
		var f config.FunctionInfo
		if err := rows.Scan(&f.Schema, &f.Name, &f.ReturnType, &f.Arguments, &f.Comment); err != nil {
			return nil, err
		}
		functions = append(functions, f)

		*current++
		h.reportProgress(*current, counts.Total, "Function: "+f.Schema+"."+f.Name)
	}

	return functions, rows.Err()
}

// GenerateSchemaDescription generates a text description of the schema for LLM
func GenerateSchemaDescription(cache *config.SchemaCache) string {
	var desc string

	desc += "DATABASE SCHEMA:\n"
	desc += "================\n\n"

	if cache.HasPostGIS {
		desc += "PostGIS is installed - spatial queries are supported.\n\n"
	}

	desc += "TABLES:\n"
	for _, t := range cache.Tables {
		desc += "- " + t.Schema + "." + t.Name
		if t.Comment != "" {
			desc += " (" + t.Comment + ")"
		}
		desc += "\n"
		for _, c := range t.Columns {
			desc += "    - " + c.Name + " (" + c.DataType + ")"
			if c.IsPrimaryKey {
				desc += " [PK]"
			}
			if c.IsForeignKey {
				desc += " [FK -> " + c.FKTable + "." + c.FKColumn + "]"
			}
			if c.IsGeometry {
				desc += " [GEOMETRY: " + c.GeomType + ", SRID: " + string(rune(c.SRID)) + "]"
			}
			if c.Comment != "" {
				desc += " - " + c.Comment
			}
			desc += "\n"
		}
		desc += "\n"
	}

	if len(cache.Views) > 0 {
		desc += "VIEWS:\n"
		for _, v := range cache.Views {
			desc += "- " + v.Schema + "." + v.Name
			if v.Comment != "" {
				desc += " (" + v.Comment + ")"
			}
			desc += "\n"
		}
		desc += "\n"
	}

	return desc
}
