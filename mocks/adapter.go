package mocks

import (
	"context"

	"vault-copy/internal/logger"
	"vault-copy/internal/vault"
)

// Adapter implements the vault.ClientInterface using a MockClient
type Adapter struct {
	client *MockClient
}

// NewAdapter creates a new Adapter with the provided MockClient
func NewAdapter(client *MockClient) *Adapter {
	return &Adapter{client: client}
}

// ReadSecret implements the vault.Reader interface
func (a *Adapter) ReadSecret(path string, logger *logger.Logger) (*vault.Secret, error) {
	return a.client.ReadSecret(path, logger)
}

// IsDirectory implements the vault.Reader interface
func (a *Adapter) IsDirectory(path string, logger *logger.Logger) (bool, error) {
	return a.client.IsDirectory(path, logger)
}

// ListSecrets implements the vault.Reader interface
func (a *Adapter) ListSecrets(path string, logger *logger.Logger) ([]string, error) {
	return a.client.ListSecrets(path, logger)
}

// GetAllSecrets implements the vault.Reader interface
func (a *Adapter) GetAllSecrets(ctx context.Context, rootPath string, logger *logger.Logger) (<-chan *vault.Secret, <-chan error) {
	return a.client.GetAllSecrets(ctx, rootPath, logger)
}

// WriteSecret implements the vault.Writer interface
func (a *Adapter) WriteSecret(path string, data map[string]interface{}, logger *logger.Logger) error {
	return a.client.WriteSecret(path, data, logger)
}

// SecretExists implements the vault.Writer interface
func (a *Adapter) SecretExists(path string, logger *logger.Logger) (bool, error) {
	return a.client.SecretExists(path, logger)
}

// BatchWriteSecrets implements the vault.Writer interface
func (a *Adapter) BatchWriteSecrets(ctx context.Context, secrets <-chan *vault.Secret, basePath string, logger *logger.Logger) <-chan error {
	return a.client.BatchWriteSecrets(ctx, secrets, basePath, logger)
}

// GetKVEngine implements the vault.ClientInterface
func (a *Adapter) GetKVEngine(path string) (string, error) {
	return a.client.GetKVEngine(path)
}
