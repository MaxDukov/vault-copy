package vault

import (
	"context"
	"vault-copy/internal/logger"
)

// Reader interface for reading secrets
type Reader interface {
	ReadSecret(path string, logger *logger.Logger) (*Secret, error)
	IsDirectory(path string, logger *logger.Logger) (bool, error)
	ListSecrets(path string, logger *logger.Logger) ([]string, error)
	GetAllSecrets(ctx context.Context, rootPath string, logger *logger.Logger) (<-chan *Secret, <-chan error)
	ExpandWildcardPath(pattern string, logger *logger.Logger) ([]string, error)
}

// Writer interface for writing secrets
type Writer interface {
	WriteSecret(path string, data map[string]interface{}, logger *logger.Logger) error
	SecretExists(path string, logger *logger.Logger) (bool, error)
	BatchWriteSecrets(ctx context.Context, secrets <-chan *Secret, basePath string, logger *logger.Logger) <-chan error
}

// ClientInterface combines Reader and Writer
type ClientInterface interface {
	Reader
	Writer
	GetKVEngine(path string) (string, error)
	GetKVEngineVersion(engine string, logger *logger.Logger) (int, error)
}
