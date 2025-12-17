package sync

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"vault-copy/internal/config"
	"vault-copy/internal/logger"
	"vault-copy/internal/vault"
)

// SyncStats holds statistics about the synchronization process.
type SyncStats struct {
	// SecretsRead is the number of secrets read from the source
	SecretsRead int64
	// SecretsWritten is the number of secrets written to the destination
	SecretsWritten int64
	// SecretsSkipped is the number of secrets skipped (already existed)
	SecretsSkipped int64
	// Errors is the number of errors encountered during synchronization
	Errors int64
}

// SyncManager handles the synchronization of secrets between Vault instances.
type SyncManager struct {
	// sourceClient is the client for the source Vault instance
	sourceClient vault.ClientInterface
	// destClient is the client for the destination Vault instance
	destClient vault.ClientInterface
	// config holds the configuration for the synchronization
	config *config.Config
	// logger is the logger instance for the manager
	logger *logger.Logger
}

// NewManager creates a new SyncManager instance with the provided clients and configuration.
func NewManager(sourceClient, destClient vault.ClientInterface, cfg *config.Config) *SyncManager {
	return &SyncManager{
		sourceClient: sourceClient,
		destClient:   destClient,
		config:       cfg,
		logger:       logger.NewLogger(cfg),
	}
}

// Sync synchronizes secrets from the source to the destination according to the configuration.
// It returns statistics about the synchronization process and any errors encountered.
func (m *SyncManager) Sync(ctx context.Context) (*SyncStats, error) {
	stats := &SyncStats{}

	m.logger.Info("Starting synchronization from %s to %s",
		m.config.SourcePath, m.config.DestinationPath)

	if m.config.DryRun {
		m.logger.Info("Dry-run mode - secrets will not be written")
	}

	// Detailed output when verbose mode is enabled
	m.logger.Verbose("Synchronization configuration:")
	m.logger.Verbose("  Source: %s", m.config.SourcePath)
	m.logger.Verbose("  Destination: %s", m.config.DestinationPath)
	m.logger.Verbose("  Recursive: %t", m.config.Recursive)
	m.logger.Verbose("  Dry-run: %t", m.config.DryRun)
	m.logger.Verbose("  Overwrite: %t", m.config.Overwrite)
	m.logger.Verbose("  Parallel workers: %d", m.config.ParallelWorkers)
	m.logger.Verbose("  Source Vault: %s", m.config.SourceAddr)
	m.logger.Verbose("  Destination Vault: %s", m.config.DestAddr)

	// Check if source path contains wildcard
	if strings.Contains(m.config.SourcePath, "*") {
		m.logger.Verbose("Source path contains wildcard: %s", m.config.SourcePath)
		// Expand wildcard paths
		expandedPaths, err := m.sourceClient.ExpandWildcardPath(m.config.SourcePath, m.logger)
		if err != nil {
			return nil, fmt.Errorf("error expanding wildcard path: %v", err)
		}

		m.logger.Verbose("Expanded wildcard path to %d paths", len(expandedPaths))
		if len(expandedPaths) == 0 {
			return nil, fmt.Errorf("no paths matched wildcard pattern: %s", m.config.SourcePath)
		}

		// Sync each expanded path
		return m.syncMultiplePaths(ctx, stats, expandedPaths)
	}

	// Check if source is a directory
	m.logger.Verbose("Checking if source is a directory: %s", m.config.SourcePath)
	isDir, err := m.sourceClient.IsDirectory(m.config.SourcePath, m.logger)
	if err != nil {
		return nil, fmt.Errorf("error checking source path: %v", err)
	}
	m.logger.Verbose("Source %s is directory: %t", m.config.SourcePath, isDir)

	if isDir && !m.config.Recursive {
		return nil, fmt.Errorf("source is a directory, use --recursive to copy")
	}

	if !isDir {
		// Copy single secret
		return m.syncSingleSecret(ctx, stats)
	}

	// Copy directory
	return m.syncDirectory(ctx, stats)
}

// syncSingleSecret synchronizes a single secret from the source to the destination.
func (m *SyncManager) syncSingleSecret(ctx context.Context, stats *SyncStats) (*SyncStats, error) {
	m.logger.Info("Reading secret: %s", m.config.SourcePath)
	m.logger.Verbose("Connecting to source Vault: %s", m.config.SourceAddr)

	secret, err := m.sourceClient.ReadSecret(m.config.SourcePath, m.logger)
	if err != nil {
		m.logger.Error("Error reading secret %s: %v", m.config.SourcePath, err)
		return nil, fmt.Errorf("error reading secret: %v", err)
	}

	atomic.AddInt64(&stats.SecretsRead, 1)
	m.logger.Verbose("Successfully read secret: %s", m.config.SourcePath)

	// Check existence in destination
	destPath := m.transformPath(m.config.SourcePath, m.config.DestinationPath)
	m.logger.Verbose("Checking secret existence in destination: %s", destPath)

	exists, err := m.destClient.SecretExists(destPath, m.logger)
	if err != nil {
		m.logger.Error("Error checking secret existence %s: %v", destPath, err)
		return nil, fmt.Errorf("error checking secret existence: %v", err)
	}

	if exists && !m.config.Overwrite {
		m.logger.Info("Secret already exists in destination: %s (use --overwrite)", destPath)
		atomic.AddInt64(&stats.SecretsSkipped, 1)
		return stats, nil
	}

	if m.config.DryRun {
		m.logger.Info("[DRY-RUN] Will write secret: %s", destPath)
		atomic.AddInt64(&stats.SecretsWritten, 1)
		return stats, nil
	}

	// Write secret
	m.logger.Info("Writing secret: %s", destPath)
	m.logger.Verbose("Connecting to destination Vault: %s", m.config.DestAddr)
	err = m.destClient.WriteSecret(destPath, secret.Data, m.logger)
	if err != nil {
		m.logger.Error("Error writing secret %s: %v", destPath, err)
		atomic.AddInt64(&stats.Errors, 1)
		return nil, fmt.Errorf("error writing secret: %v", err)
	}

	m.logger.Verbose("Successfully wrote secret: %s", destPath)
	atomic.AddInt64(&stats.SecretsWritten, 1)

	return stats, nil
}

// syncDirectory synchronizes all secrets in a directory from the source to the destination.
func (m *SyncManager) syncDirectory(ctx context.Context, stats *SyncStats) (*SyncStats, error) {
	m.logger.Info("Reading directory: %s", m.config.SourcePath)
	m.logger.Verbose("Connecting to source Vault: %s", m.config.SourceAddr)

	// Create channels for parallel processing
	secretsChan := make(chan *vault.Secret, m.config.ParallelWorkers*2)
	errChan := make(chan error, m.config.ParallelWorkers)

	var wg sync.WaitGroup

	// Start readers
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(secretsChan)

		m.logger.Verbose("Getting list of all secrets from: %s", m.config.SourcePath)
		sourceSecrets, sourceErrChan := m.sourceClient.GetAllSecrets(ctx, m.config.SourcePath, m.logger)

		for {
			select {
			case secret, ok := <-sourceSecrets:
				if !ok {
					m.logger.Verbose("Finished reading secrets from: %s", m.config.SourcePath)
					return
				}
				atomic.AddInt64(&stats.SecretsRead, 1)
				m.logger.Verbose("Read secret: %s", secret.Path)
				secretsChan <- secret
			case err := <-sourceErrChan:
				if err != nil {
					m.logger.Error("Error getting list of secrets: %v", err)
					errChan <- err
				}
				return
			case <-ctx.Done():
				m.logger.Verbose("Context cancelled while reading secrets")
				return
			}
		}
	}()

	// Start writers
	writerWg := sync.WaitGroup{}
	for i := 0; i < m.config.ParallelWorkers; i++ {
		writerWg.Add(1)
		go func(workerID int) {
			defer writerWg.Done()
			m.writeWorker(ctx, workerID, secretsChan, errChan, stats)
		}(i)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		m.logger.Verbose("Finished reading all secrets")
		writerWg.Wait()
		m.logger.Verbose("Finished writing all secrets")
		close(errChan)
	}()

	// Process errors
	for err := range errChan {
		atomic.AddInt64(&stats.Errors, 1)
		m.logger.Error("Error: %v", err)
	}

	return stats, nil
}

// writeWorker is a worker function that writes secrets to the destination.
// It processes secrets from the secretsChan and writes them to the destination Vault.
func (m *SyncManager) writeWorker(ctx context.Context, workerID int,
	secretsChan <-chan *vault.Secret, errChan chan<- error, stats *SyncStats) {

	m.logger.Verbose("Worker %d: started", workerID)

	for secret := range secretsChan {
		select {
		case <-ctx.Done():
			m.logger.Verbose("Worker %d: context cancelled", workerID)
			return
		default:
		}

		destPath := m.transformPath(secret.Path, m.config.DestinationPath)
		m.logger.Verbose("Worker %d: processing secret %s -> %s", workerID, secret.Path, destPath)

		// Check existence
		m.logger.Verbose("Worker %d: checking secret existence: %s", workerID, destPath)
		exists, err := m.destClient.SecretExists(destPath, m.logger)
		if err != nil {
			m.logger.Error("Worker %d: error checking %s: %v", workerID, destPath, err)
			errChan <- fmt.Errorf("worker %d: error checking %s: %v", workerID, destPath, err)
			continue
		}

		if exists && !m.config.Overwrite {
			m.logger.Info("Worker %d: skipping existing secret: %s", workerID, destPath)
			atomic.AddInt64(&stats.SecretsSkipped, 1)
			continue
		}

		if m.config.DryRun {
			m.logger.Info("[DRY-RUN] Worker %d: will write %s", workerID, destPath)
			atomic.AddInt64(&stats.SecretsWritten, 1)
			continue
		}

		// Write secret
		m.logger.Verbose("Worker %d: writing secret: %s", workerID, destPath)
		m.logger.Verbose("Worker %d: connecting to destination Vault: %s", workerID, m.config.DestAddr)
		err = m.destClient.WriteSecret(destPath, secret.Data, m.logger)
		if err != nil {
			m.logger.Error("Worker %d: error writing %s: %v", workerID, destPath, err)
			errChan <- fmt.Errorf("worker %d: error writing %s: %v", workerID, destPath, err)
			continue
		}

		m.logger.Info("Worker %d: wrote secret: %s", workerID, destPath)
		m.logger.Verbose("Worker %d: successfully wrote secret: %s", workerID, destPath)
		atomic.AddInt64(&stats.SecretsWritten, 1)
	}

	m.logger.Verbose("Worker %d: finished", workerID)
}

// transformPath transforms a source path to a destination path based on the configuration.
// It removes the source path prefix and appends the relative path to the destination path.
func (m *SyncManager) transformPath(sourcePath, baseDestPath string) string {
	m.logger.Verbose("Transforming path: %s -> %s", sourcePath, baseDestPath)

	// Remove the source path prefix from the path
	relativePath := strings.TrimPrefix(sourcePath, m.config.SourcePath)
	if strings.HasPrefix(relativePath, "/") {
		relativePath = relativePath[1:]
	}

	m.logger.Verbose("Relative path after prefix removal: %s", relativePath)

	// Simply concatenate baseDestPath and relativePath
	if relativePath != "" {
		// Remove trailing slash from baseDestPath if present
		baseDestPath = strings.TrimSuffix(baseDestPath, "/")
		result := baseDestPath + "/" + relativePath
		m.logger.Verbose("Result path: %s", result)
		return result
	}

	m.logger.Verbose("Result path (no relative path): %s", baseDestPath)
	return baseDestPath
}

// syncMultiplePaths synchronizes multiple paths (from wildcard expansion) from the source to the destination.
func (m *SyncManager) syncMultiplePaths(ctx context.Context, stats *SyncStats, paths []string) (*SyncStats, error) {
	m.logger.Info("Syncing %d paths from wildcard expansion", len(paths))

	// Create channels for parallel processing
	secretsChan := make(chan *vault.Secret, m.config.ParallelWorkers*2)
	errChan := make(chan error, m.config.ParallelWorkers)

	var wg sync.WaitGroup

	// Start readers for all paths
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(secretsChan)

		for _, path := range paths {
			// Check if path is a directory
			isDir, err := m.sourceClient.IsDirectory(path, m.logger)
			if err != nil {
				m.logger.Error("Error checking if path is directory %s: %v", path, err)
				errChan <- fmt.Errorf("error checking path %s: %v", path, err)
				return
			}

			if isDir {
				// Get all secrets under this directory
				sourceSecrets, sourceErrChan := m.sourceClient.GetAllSecrets(ctx, path, m.logger)

				for {
					select {
					case secret, ok := <-sourceSecrets:
						if !ok {
							goto nextPath
						}
						atomic.AddInt64(&stats.SecretsRead, 1)
						m.logger.Verbose("Read secret: %s", secret.Path)
						secretsChan <- secret
					case err := <-sourceErrChan:
						if err != nil {
							m.logger.Error("Error getting secrets from %s: %v", path, err)
							errChan <- fmt.Errorf("error getting secrets from %s: %v", path, err)
							return
						}
						goto nextPath
					case <-ctx.Done():
						m.logger.Verbose("Context cancelled while reading secrets from %s", path)
						return
					}
				}
			nextPath:
			} else {
				// Single secret
				secret, err := m.sourceClient.ReadSecret(path, m.logger)
				if err != nil {
					m.logger.Error("Error reading secret %s: %v", path, err)
					errChan <- fmt.Errorf("error reading secret %s: %v", path, err)
					return
				}
				atomic.AddInt64(&stats.SecretsRead, 1)
				m.logger.Verbose("Read secret: %s", secret.Path)
				secretsChan <- secret
			}
		}
	}()

	// Start writers
	writerWg := sync.WaitGroup{}
	for i := 0; i < m.config.ParallelWorkers; i++ {
		writerWg.Add(1)
		go func(workerID int) {
			defer writerWg.Done()
			m.writeWorker(ctx, workerID, secretsChan, errChan, stats)
		}(i)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		m.logger.Verbose("Finished reading all secrets from expanded paths")
		writerWg.Wait()
		m.logger.Verbose("Finished writing all secrets")
		close(errChan)
	}()

	// Process errors
	for err := range errChan {
		atomic.AddInt64(&stats.Errors, 1)
		m.logger.Error("Error: %v", err)
	}

	return stats, nil
}
