package postgres

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePGServiceFile(t *testing.T) {
	// Create temp file with test content
	tmpDir, err := os.MkdirTemp("", "pgservice-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	serviceFile := filepath.Join(tmpDir, "pg_service.conf")
	content := `# Test pg_service.conf

[testdb]
host=localhost
port=5432
dbname=testdb
user=testuser
password=testpass
sslmode=require

[another]
host=192.168.1.100
port=5433
dbname=anotherdb
user=anotheruser
`
	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Set environment variable
	origEnv := os.Getenv("PGSERVICEFILE")
	os.Setenv("PGSERVICEFILE", serviceFile)
	defer os.Setenv("PGSERVICEFILE", origEnv)

	services, err := ParsePGServiceFile()
	if err != nil {
		t.Fatalf("ParsePGServiceFile failed: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}

	// Check first service
	if services[0].Name != "testdb" {
		t.Errorf("expected service name 'testdb', got '%s'", services[0].Name)
	}
	if services[0].Host != "localhost" {
		t.Errorf("expected host 'localhost', got '%s'", services[0].Host)
	}
	if services[0].Port != "5432" {
		t.Errorf("expected port '5432', got '%s'", services[0].Port)
	}
	if services[0].DBName != "testdb" {
		t.Errorf("expected dbname 'testdb', got '%s'", services[0].DBName)
	}
	if services[0].User != "testuser" {
		t.Errorf("expected user 'testuser', got '%s'", services[0].User)
	}
	if services[0].Password != "testpass" {
		t.Errorf("expected password 'testpass', got '%s'", services[0].Password)
	}
	if services[0].SSLMode != "require" {
		t.Errorf("expected sslmode 'require', got '%s'", services[0].SSLMode)
	}

	// Check second service
	if services[1].Name != "another" {
		t.Errorf("expected service name 'another', got '%s'", services[1].Name)
	}
	if services[1].Host != "192.168.1.100" {
		t.Errorf("expected host '192.168.1.100', got '%s'", services[1].Host)
	}
}

func TestServiceConnectionString(t *testing.T) {
	service := ServiceEntry{
		Name:     "test",
		Host:     "localhost",
		Port:     "5432",
		DBName:   "testdb",
		User:     "user",
		Password: "pass",
		SSLMode:  "disable",
	}

	connStr := service.ConnectionString()

	expected := []string{
		"host=localhost",
		"port=5432",
		"dbname=testdb",
		"user=user",
		"password=pass",
		"sslmode=disable",
	}

	for _, e := range expected {
		if !contains(connStr, e) {
			t.Errorf("connection string missing '%s': %s", e, connStr)
		}
	}
}

func TestServiceConnectionStringDefaults(t *testing.T) {
	service := ServiceEntry{
		Name:   "minimal",
		Host:   "localhost",
		DBName: "mydb",
	}

	connStr := service.ConnectionString()

	// Should have default sslmode
	if !contains(connStr, "sslmode=prefer") {
		t.Errorf("expected default sslmode=prefer: %s", connStr)
	}
}

func TestGetServiceByName(t *testing.T) {
	services := []ServiceEntry{
		{Name: "first", Host: "host1"},
		{Name: "second", Host: "host2"},
		{Name: "third", Host: "host3"},
	}

	// Find existing
	found, err := GetServiceByName(services, "second")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Host != "host2" {
		t.Errorf("expected host2, got %s", found.Host)
	}

	// Find non-existing
	_, err = GetServiceByName(services, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent service")
	}
}

func TestParsePGServiceFileEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pgservice-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	serviceFile := filepath.Join(tmpDir, "pg_service.conf")
	content := "# Empty file with only comments\n"
	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	origEnv := os.Getenv("PGSERVICEFILE")
	os.Setenv("PGSERVICEFILE", serviceFile)
	defer os.Setenv("PGSERVICEFILE", origEnv)

	services, err := ParsePGServiceFile()
	if err != nil {
		t.Fatalf("ParsePGServiceFile failed: %v", err)
	}

	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
}

func TestParsePGServiceFileWithOptions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pgservice-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	serviceFile := filepath.Join(tmpDir, "pg_service.conf")
	content := `[myservice]
host=localhost
port=5432
dbname=mydb
user=myuser
connect_timeout=10
application_name=myapp
`
	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	origEnv := os.Getenv("PGSERVICEFILE")
	os.Setenv("PGSERVICEFILE", serviceFile)
	defer os.Setenv("PGSERVICEFILE", origEnv)

	services, err := ParsePGServiceFile()
	if err != nil {
		t.Fatalf("ParsePGServiceFile failed: %v", err)
	}

	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}

	if services[0].Options["connect_timeout"] != "10" {
		t.Errorf("expected connect_timeout=10, got %s", services[0].Options["connect_timeout"])
	}

	if services[0].Options["application_name"] != "myapp" {
		t.Errorf("expected application_name=myapp, got %s", services[0].Options["application_name"])
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
