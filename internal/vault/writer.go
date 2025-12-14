package vault

import (
	"context"
	"fmt"
	"strings"
	"vault-copy/internal/logger"
)

// WriteSecret writes a secret to Vault at the specified path.
// For KV v2, it wraps the data in a "data" key as required by the API.
// For KV v1, it writes the data directly.
func (c *Client) WriteSecret(path string, data map[string]interface{}, logger *logger.Logger) error {
	// Determine if this is a KV v2 path (contains /data/)
	var writeData map[string]interface{}

	if strings.Contains(path, "/data/") {
		// For KV v2, we need to wrap the data
		writeData = map[string]interface{}{
			"data": data,
		}
	} else {
		// For KV v1 or other engines, write data directly
		writeData = data
	}

	logger.Verbose("Writing secret to Vault: %s", path)
	_, err := c.client.Logical().Write(path, writeData)
	if err != nil {
		logger.Error("Error writing secret %s: %v", path, err)
		return fmt.Errorf("error writing secret %s: %v", path, err)
	}

	logger.Verbose("Successfully wrote secret: %s", path)
	return nil
}

// SecretExists checks if a secret exists at the specified path in Vault.
// It returns true if the secret exists, false otherwise.
func (c *Client) SecretExists(path string, logger *logger.Logger) (bool, error) {
	logger.Verbose("Checking secret existence: %s", path)
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		logger.Error("Error checking secret existence %s: %v", path, err)
		return false, err
	}

	exists := secret != nil
	logger.Verbose("Secret %s exists: %t", path, exists)
	return exists, nil
}

// BatchWriteSecrets writes multiple secrets to Vault in batch.
// It reads secrets from the secrets channel and writes them to Vault.
// Errors are sent to the returned error channel.
func (c *Client) BatchWriteSecrets(ctx context.Context, secrets <-chan *Secret, basePath string, logger *logger.Logger) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		for secret := range secrets {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			// Transform path from source to destination
			destPath := transformPath(secret.Path, basePath)

			err := c.WriteSecret(destPath, secret.Data, logger)
			if err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
}

// transformPath transforms a source path to a destination path.
// It extracts the relative path from the source path and appends it to the base destination path.
func transformPath(sourcePath, baseDestPath string) string {
	// Extract relative path from the last element
	// For example: secret/data/apps/app1/config -> secret/data/destination/app1/config
	parts := strings.Split(sourcePath, "/")
	if len(parts) < 3 {
		return baseDestPath
	}

	// Take path after engine/data/
	engineAndData := parts[0] + "/" + parts[1] + "/"
	relativePath := strings.TrimPrefix(sourcePath, engineAndData)

	// If baseDestPath already contains engine, use it
	if strings.Contains(baseDestPath, "/data/") {
		return baseDestPath + "/" + relativePath
	}

	// Otherwise add engine from source
	return parts[0] + "/data/" + baseDestPath + "/" + relativePath
}
