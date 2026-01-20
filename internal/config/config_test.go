package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.ActiveService != "" {
		t.Errorf("expected empty ActiveService, got %s", cfg.ActiveService)
	}

	if cfg.CachedSchemas == nil {
		t.Error("CachedSchemas should be initialized")
	}

	if cfg.QueryHistory == nil {
		t.Error("QueryHistory should be initialized")
	}

	if cfg.Settings.MaxHistorySize != 100 {
		t.Errorf("expected MaxHistorySize 100, got %d", cfg.Settings.MaxHistorySize)
	}

	if cfg.Settings.DefaultRowLimit != 50 {
		t.Errorf("expected DefaultRowLimit 50, got %d", cfg.Settings.DefaultRowLimit)
	}

	if !cfg.Settings.EnableSpatialOps {
		t.Error("expected EnableSpatialOps to be true")
	}

	if cfg.Settings.SchemaCacheTTLMin != 1440 {
		t.Errorf("expected SchemaCacheTTLMin 1440, got %d", cfg.Settings.SchemaCacheTTLMin)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override config directory for test
	origConfigDir := configDir
	configDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { configDir = origConfigDir }()

	// Create and save config
	cfg := DefaultConfig()
	cfg.ActiveService = "test-service"
	cfg.Settings.MaxHistorySize = 200

	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load config
	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.ActiveService != "test-service" {
		t.Errorf("expected ActiveService 'test-service', got '%s'", loaded.ActiveService)
	}

	if loaded.Settings.MaxHistorySize != 200 {
		t.Errorf("expected MaxHistorySize 200, got %d", loaded.Settings.MaxHistorySize)
	}
}

// For testing, we need to be able to override the config directory
var configDir = ConfigDir

func TestAddQueryToHistory(t *testing.T) {
	cfg := DefaultConfig()

	entry := QueryHistoryEntry{
		Timestamp:     time.Now(),
		NaturalQuery:  "How many users?",
		GeneratedSQL:  "SELECT COUNT(*) FROM users",
		ServiceName:   "test",
		RowsAffected:  1,
		ExecutionTime: 10.5,
		Success:       true,
	}

	cfg.AddQueryToHistory(entry)

	if len(cfg.QueryHistory) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(cfg.QueryHistory))
	}

	if cfg.QueryHistory[0].NaturalQuery != "How many users?" {
		t.Errorf("unexpected query: %s", cfg.QueryHistory[0].NaturalQuery)
	}
}

func TestHistoryTrimming(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Settings.MaxHistorySize = 5

	// Add more entries than max
	for i := 0; i < 10; i++ {
		cfg.AddQueryToHistory(QueryHistoryEntry{
			Timestamp:    time.Now(),
			NaturalQuery: "Query " + string(rune('A'+i)),
			Success:      true,
		})
	}

	if len(cfg.QueryHistory) != 5 {
		t.Errorf("expected 5 history entries, got %d", len(cfg.QueryHistory))
	}

	// Most recent should be first
	if cfg.QueryHistory[0].NaturalQuery != "Query J" {
		t.Errorf("expected most recent query first, got: %s", cfg.QueryHistory[0].NaturalQuery)
	}
}

func TestIsSchemaCacheValid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Settings.SchemaCacheTTLMin = 60 // 1 hour

	// No cache should be invalid
	if cfg.IsSchemaCacheValid("nonexistent") {
		t.Error("expected false for nonexistent cache")
	}

	// Add fresh cache
	cfg.CachedSchemas["test"] = &SchemaCache{
		ServiceName: "test",
		CachedAt:    time.Now(),
		Tables:      []TableInfo{},
	}

	if !cfg.IsSchemaCacheValid("test") {
		t.Error("expected fresh cache to be valid")
	}

	// Add old cache
	cfg.CachedSchemas["old"] = &SchemaCache{
		ServiceName: "old",
		CachedAt:    time.Now().Add(-2 * time.Hour),
		Tables:      []TableInfo{},
	}

	if cfg.IsSchemaCacheValid("old") {
		t.Error("expected old cache to be invalid")
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath failed: %v", err)
	}

	if !filepath.IsAbs(path) {
		t.Error("expected absolute path")
	}

	if !contains(path, "kartoza-pg-ai") {
		t.Error("path should contain 'kartoza-pg-ai'")
	}

	if !contains(path, "config.json") {
		t.Error("path should contain 'config.json'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
