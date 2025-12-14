package config

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	// Save original environment variables
	originalEnv := map[string]string{
		"VAULT_SOURCE_ADDR":  os.Getenv("VAULT_SOURCE_ADDR"),
		"VAULT_SOURCE_TOKEN": os.Getenv("VAULT_SOURCE_TOKEN"),
		"VAULT_DEST_ADDR":    os.Getenv("VAULT_DEST_ADDR"),
		"VAULT_DEST_TOKEN":   os.Getenv("VAULT_DEST_TOKEN"),
		"VAULT_ADDR":         os.Getenv("VAULT_ADDR"),
		"VAULT_TOKEN":        os.Getenv("VAULT_TOKEN"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		args        map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with separate tokens",
			envVars: map[string]string{
				"VAULT_SOURCE_TOKEN": "source_token",
				"VAULT_DEST_TOKEN":   "dest_token",
			},
			args: map[string]string{
				"source":      "secret/data/app",
				"destination": "secret/data/backup",
			},
			wantErr: false,
		},
		{
			name: "valid config with fallback token",
			envVars: map[string]string{
				"VAULT_TOKEN": "common_token",
			},
			args: map[string]string{
				"source":      "secret/data/app",
				"destination": "secret/data/backup",
			},
			wantErr: false,
		},
		{
			name: "missing source token",
			envVars: map[string]string{
				"VAULT_DEST_TOKEN": "dest_token",
			},
			args: map[string]string{
				"source":      "secret/data/app",
				"destination": "secret/data/backup",
			},
			wantErr:     true,
			errContains: "source Vault token not found",
		},
		{
			name: "missing destination token",
			envVars: map[string]string{
				"VAULT_SOURCE_TOKEN": "source_token",
			},
			args: map[string]string{
				"source":      "secret/data/app",
				"destination": "secret/data/backup",
			},
			wantErr:     true,
			errContains: "destination Vault token not found",
		},
		{
			name:    "missing source path",
			envVars: map[string]string{},
			args: map[string]string{
				"destination": "secret/data/backup",
			},
			wantErr:     true,
			errContains: "source path cannot be empty",
		},
		{
			name:    "missing destination path",
			envVars: map[string]string{},
			args: map[string]string{
				"source": "secret/data/app",
			},
			wantErr:     true,
			errContains: "destination path cannot be empty",
		},
		{
			name: "custom addresses",
			envVars: map[string]string{
				"VAULT_SOURCE_TOKEN": "source_token",
				"VAULT_DEST_TOKEN":   "dest_token",
				"VAULT_SOURCE_ADDR":  "https://vault1:8200",
				"VAULT_DEST_ADDR":    "https://vault2:8200",
			},
			args: map[string]string{
				"source":      "secret/data/app",
				"destination": "secret/data/backup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			for k := range originalEnv {
				os.Unsetenv(k)
			}

			// Set test variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Prepare arguments
			source := tt.args["source"]
			destination := tt.args["destination"]
			sourceAddr := tt.args["sourceAddr"]
			destAddr := tt.args["destAddr"]
			sourceToken := tt.args["sourceToken"]
			destToken := tt.args["destToken"]

			cfg, err := NewConfig(
				source,
				destination,
				false, // recursive
				false, // dryRun
				false, // overwrite
				false, // verbose
				5,     // parallelWorkers
				sourceAddr,
				sourceToken,
				destAddr,
				destToken,
				"config.yaml", // configFile
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewConfig() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("NewConfig() error = %v, want containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewConfig() unexpected error = %v", err)
				return
			}

			if cfg == nil {
				t.Errorf("NewConfig() returned nil config")
				return
			}

			// Check values - paths should be used as-is without normalization
			if cfg.SourcePath != source {
				t.Errorf("SourcePath = %v, want %v", cfg.SourcePath, source)
			}

			if cfg.DestinationPath != destination {
				t.Errorf("DestinationPath = %v, want %v", cfg.DestinationPath, destination)
			}

			// Check tokens
			expectedSourceToken := tt.envVars["VAULT_SOURCE_TOKEN"]
			if expectedSourceToken == "" {
				expectedSourceToken = tt.envVars["VAULT_TOKEN"]
			}
			if cfg.SourceToken != expectedSourceToken {
				t.Errorf("SourceToken = %v, want %v", cfg.SourceToken, expectedSourceToken)
			}

			expectedDestToken := tt.envVars["VAULT_DEST_TOKEN"]
			if expectedDestToken == "" {
				expectedDestToken = tt.envVars["VAULT_TOKEN"]
			}
			if cfg.DestToken != expectedDestToken {
				t.Errorf("DestToken = %v, want %v", cfg.DestToken, expectedDestToken)
			}
		})
	}
}

func TestNewConfigWithConfigFile(t *testing.T) {
	// Save original environment variables
	originalEnv := map[string]string{
		"VAULT_SOURCE_ADDR":  os.Getenv("VAULT_SOURCE_ADDR"),
		"VAULT_SOURCE_TOKEN": os.Getenv("VAULT_SOURCE_TOKEN"),
		"VAULT_DEST_ADDR":    os.Getenv("VAULT_DEST_ADDR"),
		"VAULT_DEST_TOKEN":   os.Getenv("VAULT_DEST_TOKEN"),
		"VAULT_ADDR":         os.Getenv("VAULT_ADDR"),
		"VAULT_TOKEN":        os.Getenv("VAULT_TOKEN"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	// Clear environment variables
	for k := range originalEnv {
		os.Unsetenv(k)
	}

	// Create a temporary config file for testing
	configContent := `
source:
  address: "https://vault-source:8200"
  token: "source-file-token"
destination:
  address: "https://vault-dest:8200"
  token: "dest-file-token"
settings:
  recursive: true
  dry_run: true
  overwrite: false
  parallel: 10
  verbose: true
`

	err := os.WriteFile("test-config.yaml", []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove("test-config.yaml")

	cfg, err := NewConfig(
		"secret/data/app",
		"secret/data/backup",
		false,              // recursive (will be overridden by config file)
		false,              // dryRun (will be overridden by config file)
		false,              // overwrite (will be overridden by config file)
		false,              // verbose (will be overridden by config file)
		5,                  // parallelWorkers (will be overridden by config file)
		"",                 // sourceAddr (will be overridden by config file)
		"",                 // sourceToken (will be overridden by config file)
		"",                 // destAddr (will be overridden by config file)
		"",                 // destToken (will be overridden by config file)
		"test-config.yaml", // configFile
	)

	if err != nil {
		t.Errorf("NewConfig() unexpected error = %v", err)
		return
	}

	if cfg == nil {
		t.Errorf("NewConfig() returned nil config")
		return
	}

	// Check that values from config file are used
	if cfg.SourceAddr != "https://vault-source:8200" {
		t.Errorf("SourceAddr = %v, want %v", cfg.SourceAddr, "https://vault-source:8200")
	}

	if cfg.SourceToken != "source-file-token" {
		t.Errorf("SourceToken = %v, want %v", cfg.SourceToken, "source-file-token")
	}

	if cfg.DestAddr != "https://vault-dest:8200" {
		t.Errorf("DestAddr = %v, want %v", cfg.DestAddr, "https://vault-dest:8200")
	}

	if cfg.DestToken != "dest-file-token" {
		t.Errorf("DestToken = %v, want %v", cfg.DestToken, "dest-file-token")
	}

	// Check that settings from config file are used
	if !cfg.Recursive {
		t.Errorf("Recursive = %v, want %v", cfg.Recursive, true)
	}

	if !cfg.DryRun {
		t.Errorf("DryRun = %v, want %v", cfg.DryRun, true)
	}

	if cfg.Overwrite {
		t.Errorf("Overwrite = %v, want %v", cfg.Overwrite, false)
	}

	if cfg.ParallelWorkers != 10 {
		t.Errorf("ParallelWorkers = %v, want %v", cfg.ParallelWorkers, 10)
	}

	if !cfg.Verbose {
		t.Errorf("Verbose = %v, want %v", cfg.Verbose, true)
	}
}

func TestNewConfigPriority(t *testing.T) {
	// Save original environment variables
	originalEnv := map[string]string{
		"VAULT_SOURCE_ADDR":  os.Getenv("VAULT_SOURCE_ADDR"),
		"VAULT_SOURCE_TOKEN": os.Getenv("VAULT_SOURCE_TOKEN"),
		"VAULT_DEST_ADDR":    os.Getenv("VAULT_DEST_ADDR"),
		"VAULT_DEST_TOKEN":   os.Getenv("VAULT_DEST_TOKEN"),
		"VAULT_ADDR":         os.Getenv("VAULT_ADDR"),
		"VAULT_TOKEN":        os.Getenv("VAULT_TOKEN"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	// Clear environment variables
	for k := range originalEnv {
		os.Unsetenv(k)
	}

	// Set environment variables
	os.Setenv("VAULT_SOURCE_ADDR", "https://vault-env:8200")
	os.Setenv("VAULT_SOURCE_TOKEN", "source-env-token")
	os.Setenv("VAULT_DEST_ADDR", "https://vault-env:8200")
	os.Setenv("VAULT_DEST_TOKEN", "dest-env-token")

	// Create a temporary config file for testing
	configContent := `
source:
  address: "https://vault-file:8200"
  token: "source-file-token"
destination:
  address: "https://vault-file:8200"
  token: "dest-file-token"
`

	err := os.WriteFile("priority-test-config.yaml", []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove("priority-test-config.yaml")

	// Test that environment variables have higher priority than config file
	cfg, err := NewConfig(
		"secret/data/app",
		"secret/data/backup",
		false,                       // recursive
		false,                       // dryRun
		false,                       // overwrite
		false,                       // verbose
		5,                           // parallelWorkers
		"",                          // sourceAddr (will be taken from environment)
		"",                          // sourceToken (will be taken from environment)
		"",                          // destAddr (will be taken from environment)
		"",                          // destToken (will be taken from environment)
		"priority-test-config.yaml", // configFile
	)

	if err != nil {
		t.Errorf("NewConfig() unexpected error = %v", err)
		return
	}

	if cfg == nil {
		t.Errorf("NewConfig() returned nil config")
		return
	}

	// Check that environment variables have higher priority
	if cfg.SourceAddr != "https://vault-env:8200" {
		t.Errorf("SourceAddr = %v, want %v", cfg.SourceAddr, "https://vault-env:8200")
	}

	if cfg.SourceToken != "source-env-token" {
		t.Errorf("SourceToken = %v, want %v", cfg.SourceToken, "source-env-token")
	}

	if cfg.DestAddr != "https://vault-env:8200" {
		t.Errorf("DestAddr = %v, want %v", cfg.DestAddr, "https://vault-env:8200")
	}

	if cfg.DestToken != "dest-env-token" {
		t.Errorf("DestToken = %v, want %v", cfg.DestToken, "dest-env-token")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "already normalized kv v2",
			path: "secret/data/app/config",
			want: "secret/data/app/config",
		},
		{
			name: "kv v2 without data prefix",
			path: "secret/app/config",
			want: "secret/data/app/config",
		},
		{
			name: "custom engine without data",
			path: "kv/app/config",
			want: "kv/data/app/config",
		},
		{
			name: "nested path",
			path: "secret/apps/production/database",
			want: "secret/data/apps/production/database",
		},
		{
			name: "root path",
			path: "secret",
			want: "secret/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePath(tt.path)
			if got != tt.want {
				t.Errorf("normalizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data/backup",
				ParallelWorkers: 5,
			},
			wantErr: false,
		},
		{
			name: "empty source path",
			config: &Config{
				SourcePath:      "",
				DestinationPath: "secret/data/backup",
				ParallelWorkers: 5,
			},
			wantErr: true,
		},
		{
			name: "empty destination path",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "",
				ParallelWorkers: 5,
			},
			wantErr: true,
		},
		{
			name: "invalid parallel workers",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data/backup",
				ParallelWorkers: 0,
			},
			wantErr: true,
		},
		{
			name: "negative parallel workers",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data/backup",
				ParallelWorkers: -1,
			},
			wantErr: true,
		},
		{
			name: "destination path with ..",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data/../backup",
				ParallelWorkers: 5,
			},
			wantErr: true,
		},
		{
			name: "destination path with //",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data//backup",
				ParallelWorkers: 5,
			},
			wantErr: true,
		},
		{
			name: "destination path with valid characters",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data/backup_1-test",
				ParallelWorkers: 5,
			},
			wantErr: false,
		},
		{
			name: "destination path with invalid characters - russian letters",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data/бэкап",
				ParallelWorkers: 5,
			},
			wantErr: true,
		},
		{
			name: "destination path with invalid characters - special symbols",
			config: &Config{
				SourcePath:      "secret/data/app",
				DestinationPath: "secret/data/backup@#$",
				ParallelWorkers: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
