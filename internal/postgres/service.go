package postgres

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

// ServiceEntry represents a PostgreSQL service configuration
type ServiceEntry struct {
	Name     string
	Host     string
	Port     string
	DBName   string
	User     string
	Password string
	SSLMode  string
	Options  map[string]string
}

// ParsePGServiceFile parses the pg_service.conf file
func ParsePGServiceFile() ([]ServiceEntry, error) {
	paths := getPGServicePaths()

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return parsePGServiceFileAt(path)
		}
	}

	return nil, fmt.Errorf("no pg_service.conf found in standard locations")
}

// getPGServicePaths returns possible pg_service.conf locations
func getPGServicePaths() []string {
	paths := []string{}

	// Check PGSERVICEFILE env var first
	if envPath := os.Getenv("PGSERVICEFILE"); envPath != "" {
		paths = append(paths, envPath)
	}

	// User's home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".pg_service.conf"))
	}

	// System-wide location
	paths = append(paths, "/etc/pg_service.conf")
	paths = append(paths, "/etc/postgresql-common/pg_service.conf")

	return paths
}

// parsePGServiceFileAt parses a pg_service.conf file at the given path
func parsePGServiceFileAt(path string) ([]ServiceEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var services []ServiceEntry
	var current *ServiceEntry

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for service header [servicename]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if current != nil {
				services = append(services, *current)
			}
			serviceName := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			current = &ServiceEntry{
				Name:    serviceName,
				Options: make(map[string]string),
			}
			continue
		}

		// Parse key=value pairs
		if current != nil && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "host":
					current.Host = value
				case "port":
					current.Port = value
				case "dbname":
					current.DBName = value
				case "user":
					current.User = value
				case "password":
					current.Password = value
				case "sslmode":
					current.SSLMode = value
				default:
					current.Options[key] = value
				}
			}
		}
	}

	// Don't forget the last service
	if current != nil {
		services = append(services, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return services, nil
}

// ConnectionString returns a PostgreSQL connection string for the service
func (s *ServiceEntry) ConnectionString() string {
	parts := []string{}

	if s.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", s.Host))
	}
	if s.Port != "" {
		parts = append(parts, fmt.Sprintf("port=%s", s.Port))
	}
	if s.DBName != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", s.DBName))
	}
	if s.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", s.User))
	}
	if s.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", s.Password))
	}
	if s.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", s.SSLMode))
	} else {
		parts = append(parts, "sslmode=prefer")
	}

	for k, v := range s.Options {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(parts, " ")
}

// Connect creates a database connection to this service
func (s *ServiceEntry) Connect() (*sql.DB, error) {
	return sql.Open("postgres", s.ConnectionString())
}

// TestConnection tests if a connection can be established
func (s *ServiceEntry) TestConnection() error {
	db, err := s.Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Ping()
}

// GetServiceByName finds a service by name from the list
func GetServiceByName(services []ServiceEntry, name string) (*ServiceEntry, error) {
	for _, s := range services {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("service '%s' not found", name)
}
