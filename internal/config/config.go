package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config represents the application configuration
type Config struct {
	ActiveService string                  `json:"active_service"`
	CachedSchemas map[string]*SchemaCache `json:"cached_schemas"`
	QueryHistory  []QueryHistoryEntry     `json:"query_history"`
	Settings      Settings                `json:"settings"`
}

// Settings contains user preferences
type Settings struct {
	MaxHistorySize    int    `json:"max_history_size"`
	DefaultRowLimit   int    `json:"default_row_limit"`
	EnableSpatialOps  bool   `json:"enable_spatial_ops"`
	LLMModelPath      string `json:"llm_model_path"`
	SchemaCacheTTLMin int    `json:"schema_cache_ttl_min"`
	VimModeEnabled    bool   `json:"vim_mode_enabled"`
}

// SchemaCache represents cached database schema
type SchemaCache struct {
	ServiceName string           `json:"service_name"`
	Tables      []TableInfo      `json:"tables"`
	Views       []ViewInfo       `json:"views"`
	Functions   []FunctionInfo   `json:"functions"`
	CachedAt    time.Time        `json:"cached_at"`
	HasPostGIS  bool             `json:"has_postgis"`
	Version     string           `json:"version"`
}

// TableInfo represents a database table
type TableInfo struct {
	Schema  string       `json:"schema"`
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
	Comment string       `json:"comment,omitempty"`
}

// ViewInfo represents a database view
type ViewInfo struct {
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Columns    []ColumnInfo `json:"columns"`
	Comment    string       `json:"comment,omitempty"`
	Definition string       `json:"definition,omitempty"`
}

// ColumnInfo represents a table column
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	IsForeignKey bool   `json:"is_foreign_key"`
	FKTable      string `json:"fk_table,omitempty"`
	FKColumn     string `json:"fk_column,omitempty"`
	Comment      string `json:"comment,omitempty"`
	IsGeometry   bool   `json:"is_geometry"`
	GeomType     string `json:"geom_type,omitempty"`
	SRID         int    `json:"srid,omitempty"`
}

// FunctionInfo represents a database function
type FunctionInfo struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	ReturnType string `json:"return_type"`
	Arguments  string `json:"arguments"`
	Comment    string `json:"comment,omitempty"`
}

// QueryHistoryEntry represents a query in history
type QueryHistoryEntry struct {
	Timestamp      time.Time `json:"timestamp"`
	NaturalQuery   string    `json:"natural_query"`
	GeneratedSQL   string    `json:"generated_sql"`
	ServiceName    string    `json:"service_name"`
	RowsAffected   int       `json:"rows_affected"`
	ExecutionTime  float64   `json:"execution_time_ms"`
	Success        bool      `json:"success"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

// DefaultConfig returns a new config with default values
func DefaultConfig() *Config {
	return &Config{
		ActiveService: "",
		CachedSchemas: make(map[string]*SchemaCache),
		QueryHistory:  []QueryHistoryEntry{},
		Settings: Settings{
			MaxHistorySize:    100,
			DefaultRowLimit:   50,
			EnableSpatialOps:  true,
			LLMModelPath:      "",
			SchemaCacheTTLMin: 1440, // 24 hours
			VimModeEnabled:    true,
		},
	}
}

// ConfigDir returns the configuration directory path
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kartoza-pg-ai"), nil
}

// ConfigPath returns the configuration file path
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Ensure maps are initialized
	if cfg.CachedSchemas == nil {
		cfg.CachedSchemas = make(map[string]*SchemaCache)
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddQueryToHistory adds a query to the history
func (c *Config) AddQueryToHistory(entry QueryHistoryEntry) {
	c.QueryHistory = append([]QueryHistoryEntry{entry}, c.QueryHistory...)

	// Trim to max size
	if len(c.QueryHistory) > c.Settings.MaxHistorySize {
		c.QueryHistory = c.QueryHistory[:c.Settings.MaxHistorySize]
	}
}

// IsSchemaCacheValid checks if the cached schema is still valid
func (c *Config) IsSchemaCacheValid(serviceName string) bool {
	cache, exists := c.CachedSchemas[serviceName]
	if !exists {
		return false
	}

	ttl := time.Duration(c.Settings.SchemaCacheTTLMin) * time.Minute
	return time.Since(cache.CachedAt) < ttl
}
