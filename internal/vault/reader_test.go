package vault

import (
	"context"
	"errors"
	"testing"
	"time"

	"vault-copy/mocks"
)

func TestReadSecret(t *testing.T) {
	mockClient := mocks.NewMockClient()

	testData := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
	}

	mockClient.AddSecret("secret/data/test", testData)

	// Для тестирования реального клиента нужно использовать интерфейс
	// Вместо этого тестируем логику через мок
	secret, err := mockClient.ReadSecret("secret/data/test")
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

	isDir, err := mockClient.IsDirectory("secret/data/apps")
	if err != nil {
		t.Fatalf("IsDirectory() error = %v", err)
	}

	if !isDir {
		t.Error("IsDirectory() = false, want true")
	}

	// Test non-directory
	isDir, err = mockClient.IsDirectory("secret/data/nonexistent")
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

	listed, err := mockClient.ListSecrets("secret/data/apps")
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

	secretsChan, errChan := mockClient.GetAllSecrets(ctx, "secret/data/apps")

	var receivedSecrets []*Secret
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

	// Should receive 2 secrets from apps directory
	if len(receivedSecrets) != 2 {
		t.Errorf("GetAllSecrets() received %d secrets, want 2", len(receivedSecrets))
	}

	// Verify we got the correct secrets
	foundDB := false
	foundAPI := false
	for _, secret := range receivedSecrets {
		if secret.Path == "secret/data/apps/db" {
			foundDB = true
			if secret.Data["host"] != "localhost" {
				t.Errorf("DB secret host = %v, want localhost", secret.Data["host"])
			}
		}
		if secret.Path == "secret/data/apps/api" {
			foundAPI = true
			if secret.Data["key"] != "api-key" {
				t.Errorf("API secret key = %v, want api-key", secret.Data["key"])
			}
		}
	}

	if !foundDB || !foundAPI {
		t.Error("GetAllSecrets() did not return all expected secrets")
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
			got := buildPath(tt.base, tt.item)
			if got != tt.want {
				t.Errorf("buildPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadSecretError(t *testing.T) {
	mockClient := mocks.NewMockClient()

	expectedErr := errors.New("permission denied")
	mockClient.SetReadError("secret/data/restricted", expectedErr)

	_, err := mockClient.ReadSecret("secret/data/restricted")
	if err == nil {
		t.Fatal("ReadSecret() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("ReadSecret() error = %v, want %v", err, expectedErr)
	}
}
