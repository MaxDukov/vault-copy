package sync

import (
	"context"
	"errors"
	"testing"

	"vault-copy/internal/config"
	"vault-copy/mocks"
)

func TestSyncSingleSecret(t *testing.T) {
	sourceMock := mocks.NewMockClient()
	destMock := mocks.NewMockClient()

	// Setup source secret
	sourceData := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
	}
	sourceMock.AddSecret("secret/data/source/app", sourceData)

	cfg := &config.Config{
		SourcePath:      "secret/data/source/app",
		DestinationPath: "secret/data/dest/app",
		Recursive:       false,
		DryRun:          false,
		Overwrite:       false,
		ParallelWorkers: 1,
	}

	manager := NewManager(sourceMock, destMock, cfg)

	stats, err := manager.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if stats.SecretsRead != 1 {
		t.Errorf("SecretsRead = %d, want 1", stats.SecretsRead)
	}

	if stats.SecretsWritten != 1 {
		t.Errorf("SecretsWritten = %d, want 1", stats.SecretsWritten)
	}

	// Verify secret was written to destination
	destSecret, err := destMock.ReadSecret("secret/data/dest/app")
	if err != nil {
		t.Fatalf("Read secret from destination error = %v", err)
	}

	if destSecret == nil {
		t.Fatal("Secret was not written to destination")
	}

	if destSecret.Data["username"] != "admin" {
		t.Errorf("Destination secret username = %v, want admin", destSecret.Data["username"])
	}
}

func TestSyncSingleSecretDryRun(t *testing.T) {
	sourceMock := mocks.NewMockClient()
	destMock := mocks.NewMockClient()

	sourceData := map[string]interface{}{"key": "value"}
	sourceMock.AddSecret("secret/data/source/app", sourceData)

	cfg := &config.Config{
		SourcePath:      "secret/data/source/app",
		DestinationPath: "secret/data/dest/app",
		Recursive:       false,
		DryRun:          true, // Dry run mode
		Overwrite:       false,
		ParallelWorkers: 1,
	}

	manager := NewManager(sourceMock, destMock, cfg)

	stats, err := manager.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if stats.SecretsRead != 1 {
		t.Errorf("SecretsRead = %d, want 1", stats.SecretsRead)
	}

	if stats.SecretsWritten != 1 {
		t.Errorf("SecretsWritten = %d, want 1", stats.SecretsWritten)
	}

	// In dry run mode, secret should NOT be written
	destSecret, _ := destMock.ReadSecret("secret/data/dest/app")
	if destSecret != nil {
		t.Error("Secret was written in dry run mode")
	}
}

func TestSyncSingleSecretAlreadyExists(t *testing.T) {
	sourceMock := mocks.NewMockClient()
	destMock := mocks.NewMockClient()

	// Setup source secret
	sourceMock.AddSecret("secret/data/source/app", map[string]interface{}{"key": "new"})

	// Setup existing destination secret
	destMock.AddSecret("secret/data/dest/app", map[string]interface{}{"key": "old"})

	cfg := &config.Config{
		SourcePath:      "secret/data/source/app",
		DestinationPath: "secret/data/dest/app",
		Recursive:       false,
		DryRun:          false,
		Overwrite:       false, // Do not overwrite
		ParallelWorkers: 1,
	}

	manager := NewManager(sourceMock, destMock, cfg)

	stats, err := manager.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if stats.SecretsRead != 1 {
		t.Errorf("SecretsRead = %d, want 1", stats.SecretsRead)
	}

	if stats.SecretsWritten != 0 {
		t.Errorf("SecretsWritten = %d, want 0", stats.SecretsWritten)
	}

	if stats.SecretsSkipped != 1 {
		t.Errorf("SecretsSkipped = %d, want 1", stats.SecretsSkipped)
	}

	// Verify destination secret was NOT overwritten
	destSecret, _ := destMock.ReadSecret("secret/data/dest/app")
	if destSecret.Data["key"] != "old" {
		t.Errorf("Destination secret was overwritten: key = %v, want old", destSecret.Data["key"])
	}
}

func TestSyncSingleSecretOverwrite(t *testing.T) {
	sourceMock := mocks.NewMockClient()
	destMock := mocks.NewMockClient()

	// Setup source secret
	sourceMock.AddSecret("secret/data/source/app", map[string]interface{}{"key": "new"})

	// Setup existing destination secret
	destMock.AddSecret("secret/data/dest/app", map[string]interface{}{"key": "old"})

	cfg := &config.Config{
		SourcePath:      "secret/data/source/app",
		DestinationPath: "secret/data/dest/app",
		Recursive:       false,
		DryRun:          false,
		Overwrite:       true, // Overwrite enabled
		ParallelWorkers: 1,
	}

	manager := NewManager(sourceMock, destMock, cfg)

	stats, err := manager.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if stats.SecretsWritten != 1 {
		t.Errorf("SecretsWritten = %d, want 1", stats.SecretsWritten)
	}

	// Verify destination secret WAS overwritten
	destSecret, _ := destMock.ReadSecret("secret/data/dest/app")
	if destSecret.Data["key"] != "new" {
		t.Errorf("Destination secret was not overwritten: key = %v, want new", destSecret.Data["key"])
	}
}

func TestSyncDirectory(t *testing.T) {
	sourceMock := mocks.NewMockClient()
	destMock := mocks.NewMockClient()

	// Setup directory structure
	sourceMock.AddDirectory("secret/data/source/apps", []string{"app1", "app2"})

	// Setup secrets
	sourceMock.AddSecret("secret/data/source/apps/app1", map[string]interface{}{
		"db_host": "localhost",
		"db_port": "5432",
	})

	sourceMock.AddSecret("secret/data/source/apps/app2", map[string]interface{}{
		"api_key": "key123",
	})

	cfg := &config.Config{
		SourcePath:      "secret/data/source/apps",
		DestinationPath: "secret/data/dest/apps",
		Recursive:       true, // Recursive mode
		DryRun:          false,
		Overwrite:       false,
		ParallelWorkers: 2,
	}

	manager := NewManager(sourceMock, destMock, cfg)

	stats, err := manager.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if stats.SecretsRead != 2 {
		t.Errorf("SecretsRead = %d, want 2", stats.SecretsRead)
	}

	if stats.SecretsWritten != 2 {
		t.Errorf("SecretsWritten = %d, want 2", stats.SecretsWritten)
	}

	// Verify all secrets were copied
	app1Secret, _ := destMock.ReadSecret("secret/data/dest/apps/app1")
	if app1Secret == nil {
		t.Error("App1 secret was not copied")
	}

	app2Secret, _ := destMock.ReadSecret("secret/data/dest/apps/app2")
	if app2Secret == nil {
		t.Error("App2 secret was not copied")
	}
}

func TestSyncDirectoryWithSubdirectories(t *testing.T) {
	sourceMock := mocks.NewMockClient()
	destMock := mocks.NewMockClient()

	// Setup complex directory structure
	sourceMock.AddDirectory("secret/data/source", []string{"apps", "infra"})
	sourceMock.AddDirectory("secret/data/source/apps", []string{"web", "api"})
	sourceMock.AddDirectory("secret/data/source/infra", []string{"db", "cache"})

	// Setup secrets at different levels
	sourceMock.AddSecret("secret/data/source/apps/web", map[string]interface{}{
		"port": "8080",
	})

	sourceMock.AddSecret("secret/data/source/apps/api", map[string]interface{}{
		"port": "3000",
	})

	sourceMock.AddSecret("secret/data/source/infra/db", map[string]interface{}{
		"host": "db.local",
	})

	sourceMock.AddSecret("secret/data/source/infra/cache", map[string]interface{}{
		"host": "cache.local",
	})

	cfg := &config.Config{
		SourcePath:      "secret/data/source",
		DestinationPath: "secret/data/backup",
		Recursive:       true,
		DryRun:          false,
		Overwrite:       false,
		ParallelWorkers: 3,
	}

	manager := NewManager(sourceMock, destMock, cfg)

	stats, err := manager.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if stats.SecretsRead != 4 {
		t.Errorf("SecretsRead = %d, want 4", stats.SecretsRead)
	}

	if stats.SecretsWritten != 4 {
		t.Errorf("SecretsWritten = %d, want 4", stats.SecretsWritten)
	}
}

func TestSyncErrorHandling(t *testing.T) {
	sourceMock := mocks.NewMockClient()
	destMock := mocks.NewMockClient()

	// Setup secrets with one that will cause write error
	sourceMock.AddSecret("secret/data/source/app1", map[string]interface{}{"key": "value1"})
	sourceMock.AddSecret("secret/data/source/app2", map[string]interface{}{"key": "value2"})

	// Make app2 fail to write
	destMock.SetWriteError("secret/data/dest/app2", errors.New("permission denied"))

	cfg := &config.Config{
		SourcePath:      "secret/data/source",
		DestinationPath: "secret/data/dest",
		Recursive:       false, // Only sync specific paths
		DryRun:          false,
		Overwrite:       false,
		ParallelWorkers: 1,
	}

	manager := NewManager(sourceMock, destMock, cfg)

	// This should handle the error gracefully
	stats, err := manager.Sync(context.Background())
	if err != nil {
		// Manager should handle individual errors, not fail completely
		t.Logf("Sync() returned error (expected for some secrets): %v", err)
	}

	if stats.Errors == 0 {
		t.Error("Expected errors to be recorded")
	}

	// app1 should still be written successfully
	app1Secret, _ := destMock.ReadSecret("secret/data/dest/app1")
	if app1Secret == nil {
		t.Error("App1 secret should have been written despite app2 error")
	}
}

func TestTransformPathMethod(t *testing.T) {
	manager := &SyncManager{
		config: &config.Config{
			SourcePath: "secret/data/source",
		},
	}

	tests := []struct {
		name       string
		sourcePath string
		destPath   string
		want       string
	}{
		{
			name:       "simple transformation",
			sourcePath: "secret/data/source/app1",
			destPath:   "secret/data/dest",
			want:       "secret/data/dest/app1",
		},
		{
			name:       "nested path",
			sourcePath: "secret/data/source/apps/prod/db",
			destPath:   "secret/data/backup",
			want:       "secret/data/backup/apps/prod/db",
		},
		{
			name:       "destination with engine prefix",
			sourcePath: "secret/data/source/config",
			destPath:   "kv/data/backup",
			want:       "kv/data/backup/config",
		},
		{
			name:       "same level",
			sourcePath: "secret/data/source",
			destPath:   "secret/data/backup",
			want:       "secret/data/backup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.config.SourcePath = "secret/data/source"
			manager.config.DestinationPath = tt.destPath

			got := manager.transformPath(tt.sourcePath, tt.destPath)
			if got != tt.want {
				t.Errorf("transformPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
