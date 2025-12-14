package vault_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vault-copy/internal/vault"
	"vault-copy/mocks"
)

func TestReadSecret(t *testing.T) {
	mockClient := mocks.NewMockClient()

	testData := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
	}

	mockClient.AddSecret("secret/data/test", testData)

	// For testing real client, we need to use interface
	// Instead, we test logic through mock
	secret, err := mockClient.ReadSecret("secret/data/test", nil)
	if err != nil {
		t.Fatalf("ReadSecret() error = %v", err)
	}

	if secret == nil {
		t.Fatal("ReadSecret() returned nil secret")
	}

	if secret.Path != "secret/data/test" {
		t.Errorf("Secret path = %v, want %v", secret.Path, "secret/data/test")
	}

	if secret.Data["username"] != "admin" {
		t.Errorf("Data username = %v, want admin", secret.Data["username"])
	}
}

func TestIsDirectory(t *testing.T) {
	mockClient := mocks.NewMockClient()

	mockClient.AddDirectory("secret/data/apps", []string{"app1", "app2"})

	isDir, err := mockClient.IsDirectory("secret/data/apps", nil)
	if err != nil {
		t.Fatalf("IsDirectory() error = %v", err)
	}

	if !isDir {
		t.Error("IsDirectory() = false, want true")
	}

	// Test non-directory
	isDir, err = mockClient.IsDirectory("secret/data/nonexistent", nil)
	if err != nil {
		t.Fatalf("IsDirectory() error for non-existent = %v", err)
	}

	if isDir {
		t.Error("IsDirectory() for non-existent = true, want false")
	}
}

func TestListSecrets(t *testing.T) {
	mockClient := mocks.NewMockClient()

	items := []string{"database", "api", "cache"}
	mockClient.AddDirectory("secret/data/apps", items)

	listed, err := mockClient.ListSecrets("secret/data/apps", nil)
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(listed) != len(items) {
		t.Errorf("ListSecrets() returned %d items, want %d", len(listed), len(items))
	}

	for i, item := range items {
		if listed[i] != item {
			t.Errorf("ListSecrets()[%d] = %v, want %v", i, listed[i], item)
		}
	}
}

func TestGetAllSecrets(t *testing.T) {
	mockClient := mocks.NewMockClient()

	// Setup test data
	secrets := map[string]map[string]interface{}{
		"secret/data/apps": {
			"config": "apps-config",
		},
		"secret/data/apps/db": {
			"host": "localhost",
			"port": "5432",
		},
		"secret/data/apps/api": {
			"key":    "api-key",
			"secret": "api-secret",
		},
		"secret/data/other/config": {
			"value": "test",
		},
	}

	for path, data := range secrets {
		mockClient.AddSecret(path, data)
	}

	mockClient.AddDirectory("secret/data/apps", []string{"db", "api"})
	mockClient.AddDirectory("secret/data", []string{"apps", "other"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secretsChan, errChan := mockClient.GetAllSecrets(ctx, "secret/data/apps", nil)

	var receivedSecrets []*vault.Secret
	for secret := range secretsChan {
		receivedSecrets = append(receivedSecrets, secret)
	}

	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Fatalf("GetAllSecrets() error = %v", err)
		}
	default:
	}

	// Should receive 3 secrets from apps directory and subdirectories
	// The mock client's GetAllSecrets implementation collects all secrets recursively
	// including the root path itself if it's a secret
	if len(receivedSecrets) != 3 {
		t.Errorf("GetAllSecrets() received %d secrets, want 3", len(receivedSecrets))
	}

	// Verify we got the correct secrets
	foundDB := false
	foundAPI := false
	foundAppsDir := false

	for _, secret := range receivedSecrets {
		switch secret.Path {
		case "secret/data/apps/db":
			foundDB = true
			if secret.Data["host"] != "localhost" {
				t.Errorf("DB secret host = %v, want localhost", secret.Data["host"])
			}
		case "secret/data/apps/api":
			foundAPI = true
			if secret.Data["key"] != "api-key" {
				t.Errorf("API secret key = %v, want api-key", secret.Data["key"])
			}
		case "secret/data/apps":
			foundAppsDir = true
			if secret.Data["config"] != "apps-config" {
				t.Errorf("Apps secret config = %v, want apps-config", secret.Data["config"])
			}
		}
	}

	// Check that we found the expected secrets
	if !foundDB {
		t.Error("GetAllSecrets() did not return db secret")
	}
	if !foundAPI {
		t.Error("GetAllSecrets() did not return api secret")
	}
	if !foundAppsDir {
		t.Error("GetAllSecrets() did not return apps directory secret")
	}
}

func TestBuildPath(t *testing.T) {
	tests := []struct {
		name string
		base string
		item string
		want string
	}{
		{
			name: "base with trailing slash",
			base: "secret/data/apps/",
			item: "database",
			want: "secret/data/apps/database",
		},
		{
			name: "base without trailing slash",
			base: "secret/data/apps",
			item: "database",
			want: "secret/data/apps/database",
		},
		{
			name: "nested item",
			base: "secret/data",
			item: "apps/database",
			want: "secret/data/apps/database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vault.BuildPath(tt.base, tt.item)
			if got != tt.want {
				t.Errorf("BuildPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadSecretError(t *testing.T) {
	mockClient := mocks.NewMockClient()

	expectedErr := errors.New("permission denied")
	mockClient.SetReadError("secret/data/restricted", expectedErr)

	_, err := mockClient.ReadSecret("secret/data/restricted", nil)
	if err == nil {
		t.Fatal("ReadSecret() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("ReadSecret() error = %v, want %v", err, expectedErr)
	}
}
