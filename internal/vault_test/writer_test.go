package vault_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vault-copy/internal/config"
	"vault-copy/internal/logger"
	"vault-copy/internal/vault"
	"vault-copy/mocks"
)

func TestWriteSecret(t *testing.T) {
	mockClient := mocks.NewMockClient()
	logger := logger.NewLogger(&config.Config{})

	testData := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
	}

	err := mockClient.WriteSecret("secret/data/test", testData, logger)
	if err != nil {
		t.Fatalf("WriteSecret() error = %v", err)
	}

	// Verify the secret was written
	secret, err := mockClient.ReadSecret("secret/data/test", logger)
	if err != nil {
		t.Fatalf("ReadSecret() after write error = %v", err)
	}

	if secret == nil {
		t.Fatal("Secret was not written")
	}

	if secret.Data["username"] != "testuser" {
		t.Errorf("Written secret username = %v, want testuser", secret.Data["username"])
	}
}

func TestSecretExists(t *testing.T) {
	mockClient := mocks.NewMockClient()
	logger := logger.NewLogger(&config.Config{})

	// Test non-existent secret
	exists, err := mockClient.SecretExists("secret/data/nonexistent", logger)
	if err != nil {
		t.Fatalf("SecretExists() error = %v", err)
	}

	if exists {
		t.Error("SecretExists() for non-existent = true, want false")
	}

	// Test existing secret
	mockClient.AddSecret("secret/data/existing", map[string]interface{}{"key": "value"})

	exists, err = mockClient.SecretExists("secret/data/existing", logger)
	if err != nil {
		t.Fatalf("SecretExists() error for existing = %v", err)
	}

	if !exists {
		t.Error("SecretExists() for existing = false, want true")
	}
}

func TestBatchWriteSecrets(t *testing.T) {
	mockClient := mocks.NewMockClient()
	logger := logger.NewLogger(&config.Config{})

	secretsChan := make(chan *vault.Secret, 3)

	secrets := []*vault.Secret{
		{
			Path: "secret/data/source/app1",
			Data: map[string]interface{}{"key1": "value1"},
		},
		{
			Path: "secret/data/source/app2",
			Data: map[string]interface{}{"key2": "value2"},
		},
		{
			Path: "secret/data/source/app3",
			Data: map[string]interface{}{"key3": "value3"},
		},
	}

	for _, secret := range secrets {
		secretsChan <- secret
	}
	close(secretsChan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errChan := mockClient.BatchWriteSecrets(ctx, secretsChan, "secret/data/destination", logger)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Fatalf("BatchWriteSecrets() errors: %v", errors)
	}

	// Verify secrets were written with transformed paths
	for _, secret := range secrets {
		destPath := vault.TransformPath(secret.Path, "secret/data/destination")
		storedSecret, err := mockClient.ReadSecret(destPath, logger)
		if err != nil {
			t.Errorf("ReadSecret() for %s error = %v", destPath, err)
			continue
		}

		if storedSecret == nil {
			t.Errorf("Secret was not written to %s", destPath)
			continue
		}

		// Compare data
		for key, expectedValue := range secret.Data {
			if storedSecret.Data[key] != expectedValue {
				t.Errorf("Secret data mismatch for %s[%s]: got %v, want %v",
					destPath, key, storedSecret.Data[key], expectedValue)
			}
		}
	}
}

func TestTransformPath(t *testing.T) {
	tests := []struct {
		name       string
		sourcePath string
		baseDest   string
		want       string
	}{
		{
			name:       "simple transformation",
			sourcePath: "secret/data/apps/database",
			baseDest:   "secret/data/backup",
			want:       "secret/data/backup/database",
		},
		{
			name:       "nested source path",
			sourcePath: "secret/data/apps/prod/database/config",
			baseDest:   "secret/data/backup",
			want:       "secret/data/backup/prod/database/config",
		},
		{
			name:       "destination with engine",
			baseDest:   "kv/data/backup",
			sourcePath: "secret/data/apps/config",
			want:       "kv/data/backup/config",
		},
		{
			name:       "source without data prefix",
			sourcePath: "secret/apps/config",
			baseDest:   "secret/data/backup",
			want:       "secret/data/backup/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vault.TransformPath(tt.sourcePath, tt.baseDest)
			if got != tt.want {
				t.Errorf("TransformPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteSecretError(t *testing.T) {
	mockClient := mocks.NewMockClient()
	logger := logger.NewLogger(&config.Config{})

	expectedErr := errors.New("write failed")
	mockClient.SetWriteError("secret/data/error", expectedErr)

	err := mockClient.WriteSecret("secret/data/error", map[string]interface{}{"key": "value"}, logger)
	if err == nil {
		t.Fatal("WriteSecret() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("WriteSecret() error = %v, want %v", err, expectedErr)
	}
}
