package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"vault-copy/internal/config"
	"vault-copy/internal/sync"
	"vault-copy/internal/vault"
)

func main() {
	// Parse command line arguments
	configFile := flag.String("config", "config.yaml", "Path to config file")
	srcPath := flag.String("src-path", "", "Source secret or directory path (required)")
	dstPath := flag.String("dst-path", "", "Destination path in target Vault (required)")
	recursive := flag.Bool("recursive", false, "Recursively copy all secrets from folder (disabled by default)")
	dryRun := flag.Bool("dry-run", false, "Show what would be copied without actually copying")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing secrets (disabled by default)")
	parallel := flag.Int("parallel", 5, "Number of parallel operations")
	verbose := flag.Bool("v", false, "Enable verbose output")

	// Source Vault flags
	sourceAddr := flag.String("src-addr", "", "Source Vault URL (environment variable VAULT_SOURCE_ADDR will be used by default)")
	sourceToken := flag.String("src-token", "", "Source Vault token (environment variable VAULT_SOURCE_TOKEN will be used by default)")

	// Destination Vault flags
	destAddr := flag.String("dst-addr", "", "Destination Vault URL (environment variable VAULT_DEST_ADDR will be used by default)")
	destToken := flag.String("dst-token", "", "Destination Vault token (environment variable VAULT_DEST_TOKEN will be used by default)")

	flag.Parse()

	// Validate arguments
	if *srcPath == "" || *dstPath == "" {
		message := `
example usage:

export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"

./vault-sync --src-path="secret/data/apps/production" --dst-path="secret/data/backup/production" --recursive --parallel=10`
		fmt.Println("At least 2 parameters are required: --src-path and --dst-path, in this case secrets will be copied within VAULT_SOURCE_ADDR")
		fmt.Println(message)
		fmt.Println("enter --help for help")
		os.Exit(1)
	}

	// Create configuration
	cfg, err := config.NewConfig(
		*srcPath,
		*dstPath,
		*recursive,
		*dryRun,
		*overwrite,
		*verbose,
		*parallel,
		*sourceAddr,
		*sourceToken,
		*destAddr,
		*destToken,
		*configFile,
	)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize Vault clients
	sourceClient, err := vault.NewClient(cfg.SourceAddr, cfg.SourceToken)
	if err != nil {
		log.Fatalf("Error creating source Vault client: %v", err)
	}

	destClient, err := vault.NewClient(cfg.DestAddr, cfg.DestToken)
	if err != nil {
		log.Fatalf("Error creating destination Vault client: %v", err)
	}

	// Create synchronization manager
	syncManager := sync.NewManager(sourceClient, destClient, cfg)

	// Perform synchronization
	ctx := context.Background()
	stats, err := syncManager.Sync(ctx)
	if err != nil {
		log.Fatalf("Synchronization error: %v", err)
	}

	// Output statistics
	fmt.Printf("\nSynchronization completed:\n")
	fmt.Printf("  Secrets read: %d\n", stats.SecretsRead)
	fmt.Printf("  Secrets written: %d\n", stats.SecretsWritten)
	fmt.Printf("  Skipped (already exist): %d\n", stats.SecretsSkipped)
	fmt.Printf("  Errors: %d\n", stats.Errors)

	if *dryRun {
		fmt.Println("\nDry-run mode - nothing was written")
	}
}
