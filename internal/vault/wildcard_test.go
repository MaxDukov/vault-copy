package vault_test

import (
	"testing"

	"vault-copy/mocks"
)

func TestExpandWildcardPath(t *testing.T) {
	mockClient := mocks.NewMockClient()

	// Setup test data
	secrets := map[string]map[string]interface{}{
		"secret/data/apps/app1/database": {
			"host": "db1.example.com",
			"port": "5432",
		},
		"secret/data/apps/app1/postgres": {
			"host": "pg1.example.com",
			"port": "5432",
		},
		"secret/data/apps/app1/postgresql": {
			"host": "pg2.example.com",
			"port": "5432",
		},
		"secret/data/apps/app1/cache": {
			"host": "redis.example.com",
			"port": "6379",
		},
		"secret/data/apps/app2/database": {
			"host": "db2.example.com",
			"port": "5432",
		},
	}

	for path, data := range secrets {
		mockClient.AddSecret(path, data)
	}

	// Setup directories
	mockClient.AddDirectory("secret/data/apps", []string{"app1", "app2"})
	mockClient.AddDirectory("secret/data/apps/app1", []string{"database", "postgres", "postgresql", "cache"})
	mockClient.AddDirectory("secret/data/apps/app2", []string{"database"})

	// Test expanding wildcard path
	expanded, err := mockClient.ExpandWildcardPath("secret/data/apps/app1/postgre*", nil)
	if err != nil {
		t.Fatalf("ExpandWildcardPath() error = %v", err)
	}

	// Should match postgres and postgresql
	if len(expanded) != 2 {
		t.Errorf("ExpandWildcardPath() returned %d paths, want 2", len(expanded))
	}

	// Check that we got the expected paths
	foundPostgres := false
	foundPostgresql := false

	for _, path := range expanded {
		switch path {
		case "secret/data/apps/app1/postgres":
			foundPostgres = true
		case "secret/data/apps/app1/postgresql":
			foundPostgresql = true
		}
	}

	if !foundPostgres {
		t.Error("ExpandWildcardPath() did not return postgres path")
	}

	if !foundPostgresql {
		t.Error("ExpandWildcardPath() did not return postgresql path")
	}
}

func TestExpandWildcardPathNoWildcard(t *testing.T) {
	mockClient := mocks.NewMockClient()

	// Test expanding path without wildcard
	expanded, err := mockClient.ExpandWildcardPath("secret/data/apps/app1/database", nil)
	if err != nil {
		t.Fatalf("ExpandWildcardPath() error = %v", err)
	}

	// Should return the same path
	if len(expanded) != 1 {
		t.Errorf("ExpandWildcardPath() returned %d paths, want 1", len(expanded))
	}

	if expanded[0] != "secret/data/apps/app1/database" {
		t.Errorf("ExpandWildcardPath() returned %v, want [secret/data/apps/app1/database]", expanded)
	}
}

func TestExpandWildcardPathNoMatch(t *testing.T) {
	mockClient := mocks.NewMockClient()

	// Setup directories
	mockClient.AddDirectory("secret/data/apps", []string{"app1", "app2"})
	mockClient.AddDirectory("secret/data/apps/app1", []string{"database", "postgres", "postgresql", "cache"})

	// Test expanding wildcard path with no matches
	expanded, err := mockClient.ExpandWildcardPath("secret/data/apps/app1/mysql*", nil)
	if err != nil {
		t.Fatalf("ExpandWildcardPath() error = %v", err)
	}

	// Should return empty list
	if len(expanded) != 0 {
		t.Errorf("ExpandWildcardPath() returned %d paths, want 0", len(expanded))
	}
}
