package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"vault-copy/internal/logger"
)

// Secret represents a Vault secret with its path, data, and metadata.
type Secret struct {
	// Path is the full path to the secret in Vault
	Path string
	// Data contains the secret's key-value pairs
	Data map[string]interface{}
	// Metadata contains the secret's metadata
	Metadata map[string]interface{}
}

// ReadSecret reads a secret from Vault at the specified path.
// It handles both KV v1 and KV v2 secrets and returns the secret data and metadata.
func (c *Client) ReadSecret(path string, logger *logger.Logger) (*Secret, error) {
	logger.Verbose("Reading secret from Vault: %s", path)
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		logger.Error("Error reading secret %s: %v", path, err)
		return nil, err
	}

	if secret == nil {
		return nil, fmt.Errorf("secret not found: %s", path)
	}

	// For KV v2, data is in secret.Data["data"]
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		data = secret.Data // For KV v1 or other engines
	}

	metadata, _ := secret.Data["metadata"].(map[string]interface{})

	return &Secret{
		Path:     path,
		Data:     data,
		Metadata: metadata,
	}, nil
}

// IsDirectory checks if the given path is a directory in Vault.
// It attempts to list the path and returns true if listing is successful.
func (c *Client) IsDirectory(path string, logger *logger.Logger) (bool, error) {
	logger.Verbose("Checking if path is a directory: %s", path)
	// Try to get listing
	listPath := strings.Replace(path, "/data/", "/metadata/", 1)
	if !strings.Contains(listPath, "/metadata/") {
		listPath = path + "/"
	}

	logger.Verbose("Getting list from: %s", listPath)
	secret, err := c.client.Logical().List(listPath)
	if err != nil {
		// If error is 405 or 404, it's not a directory
		if strings.Contains(err.Error(), "405") ||
			strings.Contains(err.Error(), "404") ||
			strings.Contains(err.Error(), "permission denied") {
			logger.Verbose("Path %s is not a directory (error: %v)", path, err)
			return false, nil
		}
		logger.Error("Error checking path %s: %v", path, err)
		return false, err
	}

	isDir := secret != nil && secret.Data != nil
	logger.Verbose("Path %s is a directory: %t", path, isDir)
	return isDir, nil
}

// ListSecrets lists all secrets at the given path in Vault.
// It returns a slice of secret names/paths relative to the given path.
func (c *Client) ListSecrets(path string, logger *logger.Logger) ([]string, error) {
	logger.Verbose("Getting list of secrets from: %s", path)
	// For KV v2, use metadata endpoint for listing
	listPath := strings.Replace(path, "/data/", "/metadata/", 1)
	if !strings.Contains(listPath, "/metadata/") {
		listPath = path + "/"
	}

	logger.Verbose("Requesting list from: %s", listPath)
	secret, err := c.client.Logical().List(listPath)
	if err != nil {
		logger.Error("Error getting list of secrets from %s: %v", path, err)
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		logger.Verbose("No secrets in: %s", path)
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		logger.Verbose("No keys in response for: %s", path)
		return []string{}, nil
	}

	var result []string
	for _, key := range keys {
		if str, ok := key.(string); ok {
			result = append(result, str)
		}
	}

	logger.Verbose("Found %d secrets in: %s", len(result), path)
	return result, nil
}

// GetAllSecrets recursively retrieves all secrets under the given root path.
// It returns two channels: one for secrets and one for errors.
// The caller must read from both channels until they are closed.
func (c *Client) GetAllSecrets(ctx context.Context, rootPath string, logger *logger.Logger) (<-chan *Secret, <-chan error) {
	secretsChan := make(chan *Secret, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(secretsChan)
		defer close(errChan)

		c.walkSecrets(ctx, rootPath, secretsChan, errChan, logger)
	}()

	return secretsChan, errChan
}

// walkSecrets recursively walks the Vault hierarchy and sends secrets to the secrets channel.
// It sends errors to the error channel. Both channels are closed when the walk is complete.
func (c *Client) walkSecrets(ctx context.Context, path string, secretsChan chan<- *Secret, errChan chan<- error, logger *logger.Logger) {
	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return
	default:
	}

	isDir, err := c.IsDirectory(path, logger)
	if err != nil {
		errChan <- err
		return
	}

	if isDir {
		items, err := c.ListSecrets(path, logger)
		if err != nil {
			errChan <- err
			return
		}

		var wg sync.WaitGroup
		for _, item := range items {
			wg.Add(1)
			go func(itemPath string) {
				defer wg.Done()
				c.walkSecrets(ctx, itemPath, secretsChan, errChan, logger)
			}(buildPath(path, item))
		}
		wg.Wait()
	} else {
		secret, err := c.ReadSecret(path, logger)
		if err != nil {
			errChan <- err
			return
		}

		select {
		case secretsChan <- secret:
		case <-ctx.Done():
			errChan <- ctx.Err()
		}
	}
}

// buildPath constructs a full path from a base path and an item name.
// It ensures proper path separators are used.
func buildPath(base, item string) string {
	if strings.HasSuffix(base, "/") {
		return base + item
	}
	return base + "/" + item
}
