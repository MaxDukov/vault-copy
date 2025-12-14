package config

import (
	"errors"
	"log"
	"os"
	"strings"
)

// Config holds the configuration for the vault-copy application.
type Config struct {
	// SourcePath is the path to the secret or directory in the source Vault
	SourcePath string
	// DestinationPath is the path where secrets will be copied in the destination Vault
	DestinationPath string
	// Recursive indicates whether to copy directories recursively
	Recursive bool
	// DryRun indicates whether to perform a dry run without actually copying secrets
	DryRun bool
	// Overwrite indicates whether to overwrite existing secrets in the destination
	Overwrite bool
	// ParallelWorkers is the number of parallel workers for copying secrets
	ParallelWorkers int
	// Verbose indicates whether to enable verbose logging
	Verbose bool

	// SourceAddr is the address of the source Vault server
	SourceAddr string
	// SourceToken is the authentication token for the source Vault
	SourceToken string

	// DestAddr is the address of the destination Vault server
	DestAddr string
	// DestToken is the authentication token for the destination Vault
	DestToken string
}

// NewConfig creates a new Config instance with the provided parameters.
// It handles environment variable fallbacks for Vault addresses and tokens.
func NewConfig(
	sourcePath, destinationPath string,
	recursive, dryRun, overwrite, verbose bool,
	parallelWorkers int,
	sourceAddr, sourceToken,
	destAddr, destToken string,
) (*Config, error) {

	cfg := &Config{
		SourcePath:      normalizePath(sourcePath),
		DestinationPath: normalizePath(destinationPath),
		Recursive:       recursive,
		DryRun:          dryRun,
		Overwrite:       overwrite,
		ParallelWorkers: parallelWorkers,
		Verbose:         verbose,
	}
	// Get source Vault configuration
	if sourceAddr == "" {
		sourceAddr = os.Getenv("VAULT_SOURCE_ADDR")
	}
	if sourceAddr == "" {
		sourceAddr = os.Getenv("VAULT_ADDR")
	}
	if sourceAddr == "" {
		sourceAddr = "http://localhost:8200"
	}
	cfg.SourceAddr = sourceAddr

	if sourceToken == "" {
		sourceToken = os.Getenv("VAULT_SOURCE_TOKEN")
	}
	if sourceToken == "" {
		sourceToken = os.Getenv("VAULT_TOKEN")
	}
	if sourceToken == "" {
		return nil, errors.New("source Vault token not found. Set VAULT_SOURCE_TOKEN or VAULT_TOKEN")
	}
	cfg.SourceToken = sourceToken

	// Get destination Vault configuration
	if destAddr == "" {
		destAddr = os.Getenv("VAULT_DEST_ADDR")
	}
	if destAddr == "" {
		destAddr = os.Getenv("VAULT_ADDR")
		log.Println("VAULT_DEST_ADDR not found, using VAULT_ADDR, copying within the same Vault")
	}
	cfg.DestAddr = destAddr

	if destToken == "" {
		destToken = os.Getenv("VAULT_DEST_TOKEN")
	}
	if destToken == "" {
		destToken = os.Getenv("VAULT_TOKEN")
	}
	if destToken == "" {
		return nil, errors.New("destination Vault token not found. Set VAULT_DEST_TOKEN or VAULT_TOKEN")
	}
	cfg.DestToken = destToken

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// normalizePath normalizes Vault secret paths by ensuring they don't get incorrectly modified.
// It preserves paths that already contain /data/ or don't have KV engine prefixes.
func normalizePath(path string) string {
	// Don't modify paths that already contain /data/ or don't have KV engine prefixes
	if strings.Contains(path, "/data/") || (!strings.HasPrefix(path, "secret/") && !strings.HasPrefix(path, "kv/")) {
		return path
	}

	// For paths with secret/ or kv/ prefixes, check if they already contain data
	if strings.HasPrefix(path, "secret/") || strings.HasPrefix(path, "kv/") {
		// If path doesn't contain /data/, don't add it automatically
		// Let Vault API determine the format
		return path
	}

	return path
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.SourcePath == "" {
		return errors.New("source path cannot be empty")
	}

	if c.DestinationPath == "" {
		return errors.New("destination path cannot be empty")
	}

	if c.ParallelWorkers < 1 {
		return errors.New("parallel workers must be >= 1")
	}

	return nil
}
