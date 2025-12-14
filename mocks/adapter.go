package mocks

import (
	"context"
	"vault-copy/internal/logger"
	"vault-copy/internal/vault"
)

// Adapter adapts MockClient to implement vault.ClientInterface
type Adapter struct {
	client *MockClient
}

func NewAdapter(client *MockClient) *Adapter {
	return &Adapter{client: client}
}

func (a *Adapter) ReadSecret(path string, logger *logger.Logger) (*vault.Secret, error) {
	return a.client.ReadSecret(path, logger)
}

func (a *Adapter) IsDirectory(path string, logger *logger.Logger) (bool, error) {
	return a.client.IsDirectory(path, logger)
}

func (a *Adapter) ListSecrets(path string, logger *logger.Logger) ([]string, error) {
	return a.client.ListSecrets(path, logger)
}

func (a *Adapter) GetAllSecrets(ctx context.Context, rootPath string, logger *logger.Logger) (<-chan *vault.Secret, <-chan error) {
	return a.client.GetAllSecrets(ctx, rootPath, logger)
}

func (a *Adapter) WriteSecret(path string, data map[string]interface{}, logger *logger.Logger) error {
	return a.client.WriteSecret(path, data, logger)
}

func (a *Adapter) SecretExists(path string, logger *logger.Logger) (bool, error) {
	return a.client.SecretExists(path, logger)
}

func (a *Adapter) BatchWriteSecrets(ctx context.Context, secrets <-chan *vault.Secret, basePath string, logger *logger.Logger) <-chan error {
	return a.client.BatchWriteSecrets(ctx, secrets, basePath, logger)
}

func (a *Adapter) GetKVEngine(path string) (string, error) {
	return a.client.GetKVEngine(path)
}
