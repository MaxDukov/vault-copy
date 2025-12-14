package config

import (
	"errors"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
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

// FileConfig represents the structure of the YAML config file
type FileConfig struct {
	Source struct {
		Address string `yaml:"address"`
		Token   string `yaml:"token"`
	} `yaml:"source"`
	Destination struct {
		Address string `yaml:"address"`
		Token   string `yaml:"token"`
	} `yaml:"destination"`
	Settings struct {
		Recursive bool `yaml:"recursive"`
		DryRun    bool `yaml:"dry_run"`
		Overwrite bool `yaml:"overwrite"`
		Parallel  int  `yaml:"parallel"`
		Verbose   bool `yaml:"verbose"`
	} `yaml:"settings"`
}

// LoadConfigFromFile loads configuration from a YAML file
func LoadConfigFromFile(filename string) (*FileConfig, error) {
	// If config file doesn't exist, return empty config
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return &FileConfig{}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var fileConfig FileConfig
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return nil, err
	}

	return &fileConfig, nil
}

// NewConfig creates a new Config instance with the provided parameters.
// It handles environment variable fallbacks for Vault addresses and tokens.
// Priority order: function parameters > environment variables > config file > defaults
func NewConfig(
	sourcePath, destinationPath string,
	recursive, dryRun, overwrite, verbose bool,
	parallelWorkers int,
	sourceAddr, sourceToken,
	destAddr, destToken string,
	configFile string,
) (*Config, error) {
	// Load config file
	fileConfig, err := LoadConfigFromFile(configFile)
	if err != nil {
		log.Printf("Warning: could not load config file: %v", err)
		fileConfig = &FileConfig{}
	}

	cfg := &Config{
		SourcePath:      normalizePath(sourcePath),
		DestinationPath: normalizePath(destinationPath),
		Recursive:       recursive,
		DryRun:          dryRun,
		Overwrite:       overwrite,
		ParallelWorkers: parallelWorkers,
		Verbose:         verbose,
	}

	// Validate configuration early to catch path errors before token validation
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Get source Vault configuration
	// Priority: function parameter > environment variable > config file > default
	if sourceAddr == "" {
		sourceAddr = os.Getenv("VAULT_SOURCE_ADDR")
	}
	if sourceAddr == "" {
		sourceAddr = fileConfig.Source.Address
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
		sourceToken = fileConfig.Source.Token
	}
	if sourceToken == "" {
		sourceToken = os.Getenv("VAULT_TOKEN")
	}
	if sourceToken == "" {
		return nil, errors.New("source Vault token not found. Set VAULT_SOURCE_TOKEN or VAULT_TOKEN")
	}
	cfg.SourceToken = sourceToken

	// Get destination Vault configuration
	// Priority: function parameter > environment variable > config file > default
	if destAddr == "" {
		destAddr = os.Getenv("VAULT_DEST_ADDR")
	}
	if destAddr == "" {
		destAddr = fileConfig.Destination.Address
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
		destToken = fileConfig.Destination.Token
	}
	if destToken == "" {
		destToken = os.Getenv("VAULT_TOKEN")
	}
	if destToken == "" {
		return nil, errors.New("destination Vault token not found. Set VAULT_DEST_TOKEN or VAULT_TOKEN")
	}
	cfg.DestToken = destToken

	// Apply default settings from config file if not set by command line
	// Command line has explicit values when flags are provided
	// We need to check if the values are at their default state
	if !recursive {
		cfg.Recursive = fileConfig.Settings.Recursive
	} else {
		cfg.Recursive = recursive
	}

	if !dryRun {
		cfg.DryRun = fileConfig.Settings.DryRun
	} else {
		cfg.DryRun = dryRun
	}

	if !overwrite {
		cfg.Overwrite = fileConfig.Settings.Overwrite
	} else {
		cfg.Overwrite = overwrite
	}

	if parallelWorkers == 5 && fileConfig.Settings.Parallel != 0 {
		// Only use config file value if command line wasn't explicitly set to default
		cfg.ParallelWorkers = fileConfig.Settings.Parallel
	} else {
		cfg.ParallelWorkers = parallelWorkers
	}

	if !verbose {
		cfg.Verbose = fileConfig.Settings.Verbose
	} else {
		cfg.Verbose = verbose
	}

	return cfg, nil
}

// normalizePath normalizes Vault secret paths by ensuring they don't get incorrectly modified.
// It preserves paths that already contain /data/ or don't have KV engine prefixes.
func normalizePath(path string) string {
	// Don't modify paths that already contain /data/ or don't have KV engine prefixes
	if strings.Contains(path, "/data/") || (!strings.HasPrefix(path, "secret") && !strings.HasPrefix(path, "kv")) {
		return path
	}

	// Handle exact matches for root paths
	if path == "secret" {
		return "secret/data"
	}

	if path == "kv" {
		return "kv/data"
	}

	// For paths with secret/ or kv/ prefixes, add /data/ if it's not already there
	if strings.HasPrefix(path, "secret/") && !strings.HasPrefix(path, "secret/data/") {
		// Add /data/ after secret/
		return "secret/data" + strings.TrimPrefix(path, "secret")
	}

	if strings.HasPrefix(path, "kv/") && !strings.HasPrefix(path, "kv/data/") {
		// Add /data/ after kv/
		return "kv/data" + strings.TrimPrefix(path, "kv")
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

	// Check that destination path doesn't contain invalid characters
	// Vault paths should not contain characters like .. or //
	if strings.Contains(c.DestinationPath, "..") {
		return errors.New("destination path cannot contain ..")
	}

	if strings.Contains(c.DestinationPath, "//") {
		return errors.New("destination path cannot contain //")
	}

	// Check that destination path only contains valid characters
	// Valid characters: English letters, digits, hyphens, underscores, and slashes
	for _, r := range c.DestinationPath {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/') {
			return errors.New("destination path can only contain English letters, digits, hyphens, underscores, and slashes")
		}
	}

	if c.ParallelWorkers < 1 {
		return errors.New("parallel workers must be >= 1")
	}

	return nil
}
