package mocks

import (
	"context"
	"strings"
	"sync"

	"vault-copy/internal/logger"
	"vault-copy/internal/vault"
)

// MockClient implements vault.Client interface for testing
type MockClient struct {
	Secrets     map[string]*vault.Secret
	Directories map[string]bool
	ListResults map[string][]string
	WriteErrors map[string]error
	ReadErrors  map[string]error
	ListErrors  map[string]error
	CheckErrors map[string]error

	mu sync.RWMutex
}

func NewMockClient() *MockClient {
	return &MockClient{
		Secrets:     make(map[string]*vault.Secret),
		Directories: make(map[string]bool),
		ListResults: make(map[string][]string),
		WriteErrors: make(map[string]error),
		ReadErrors:  make(map[string]error),
		ListErrors:  make(map[string]error),
		CheckErrors: make(map[string]error),
	}
}

func (m *MockClient) ReadSecret(path string, logger *logger.Logger) (*vault.Secret, error) {
	// Ignore logger for tests
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.ReadErrors[path]; ok {
		return nil, err
	}

	secret, ok := m.Secrets[path]
	if !ok {
		return nil, nil
	}

	return secret, nil
}

func (m *MockClient) IsDirectory(path string, logger *logger.Logger) (bool, error) {
	// Ignore logger for tests
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.CheckErrors[path]; ok {
		return false, err
	}

	// Check if in directory list
	if isDir, ok := m.Directories[path]; ok {
		return isDir, nil
	}

	// Check if in List results list
	if _, ok := m.ListResults[path]; ok {
		return true, nil
	}

	return false, nil
}

func (m *MockClient) ListSecrets(path string, logger *logger.Logger) ([]string, error) {
	// Ignore logger for tests
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.ListErrors[path]; ok {
		return nil, err
	}

	// Check if path is a directory
	if isDir, ok := m.Directories[path]; !ok || !isDir {
		return []string{}, nil
	}

	items, ok := m.ListResults[path]
	if !ok {
		return []string{}, nil
	}

	return items, nil
}

func (m *MockClient) GetAllSecrets(ctx context.Context, rootPath string, logger *logger.Logger) (<-chan *vault.Secret, <-chan error) {
	// Ignore logger for tests
	secretsChan := make(chan *vault.Secret, 100) // Increase buffer
	errChan := make(chan error, 1)

	go func() {
		defer close(secretsChan)
		defer close(errChan)

		// Recursively collect all secrets
		m.collectSecrets(ctx, rootPath, secretsChan, errChan)
	}()

	return secretsChan, errChan
}

func (m *MockClient) collectSecrets(ctx context.Context, rootPath string, secretsChan chan *vault.Secret, errChan chan error) {
	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return
	default:
	}

	// Check if rootPath itself is a secret (path can be both secret and directory in Vault)
	m.mu.RLock()
	secret, isSecret := m.Secrets[rootPath]
	m.mu.RUnlock()

	if isSecret {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		case secretsChan <- secret:
		}
	}

	// Check if it's a directory
	isDir, err := m.IsDirectory(rootPath, nil)
	if err != nil {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
		case errChan <- err:
		}
		return
	}

	if isDir {
		// Get list of items in directory
		items, err := m.ListSecrets(rootPath, nil)
		if err != nil {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
			case errChan <- err:
			}
			return
		}

		// For each item, recursively process it
		for _, item := range items {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			fullPath := buildPath(rootPath, item)
			// Recursively process item (whether it's a secret or directory)
			m.collectSecrets(ctx, fullPath, secretsChan, errChan)
		}
	} else if !isSecret {
		// If not a directory and not already sent as secret, try to read it as a secret
		secret, err := m.ReadSecret(rootPath, nil)
		if err != nil {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
			case errChan <- err:
			}
			return
		}

		if secret != nil {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case secretsChan <- secret:
			}
		}
	}
}

func (m *MockClient) WriteSecret(path string, data map[string]interface{}, logger *logger.Logger) error {
	// Ignore logger for tests
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.WriteErrors[path]; ok {
		return err
	}

	m.Secrets[path] = &vault.Secret{
		Path: path,
		Data: data,
	}

	return nil
}

func (m *MockClient) SecretExists(path string, logger *logger.Logger) (bool, error) {
	// Ignore logger for tests
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.CheckErrors[path]; ok {
		return false, err
	}

	// Check if path is a directory
	if isDir, ok := m.Directories[path]; ok && isDir {
		return false, nil // Directories are not considered secrets
	}

	_, exists := m.Secrets[path]
	return exists, nil
}

func (m *MockClient) BatchWriteSecrets(ctx context.Context, secrets <-chan *vault.Secret, basePath string, logger *logger.Logger) <-chan error {
	// Ignore logger for tests
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

			err := m.WriteSecret(destPath, secret.Data, logger)
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	return errChan
}

func (m *MockClient) SetReadError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReadErrors[path] = err
}

func (m *MockClient) SetWriteError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WriteErrors[path] = err
}

func (m *MockClient) SetListError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ListErrors[path] = err
}

func (m *MockClient) AddSecret(path string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Secrets[path] = &vault.Secret{
		Path: path,
		Data: data,
	}
}

func (m *MockClient) AddDirectory(path string, items []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Directories[path] = true
	m.ListResults[path] = items
}

func transformPath(sourcePath, baseDestPath string) string {
	// This should match the logic in internal/vault/writer.go TransformPath
	parts := strings.Split(sourcePath, "/")
	if len(parts) < 3 {
		return baseDestPath
	}

	// Handle case when source doesn't have /data/ prefix
	if !strings.Contains(sourcePath, "/data/") {
		// For paths like "secret/apps/config", take everything after engine
		relativePath := strings.TrimPrefix(sourcePath, parts[0]+"/")
		relativeParts := strings.Split(relativePath, "/")

		if strings.Contains(baseDestPath, "/data/") {
			// Take everything except the first part
			if len(relativeParts) > 1 {
				restPath := strings.Join(relativeParts[1:], "/")
				return baseDestPath + "/" + restPath
			}
			return baseDestPath + "/" + relativePath
		}
		// If baseDestPath doesn't have /data/, add engine and /data/
		if len(relativeParts) > 1 {
			restPath := strings.Join(relativeParts[1:], "/")
			return parts[0] + "/data/" + baseDestPath + "/" + restPath
		}
		return parts[0] + "/data/" + baseDestPath + "/" + relativePath
	}

	// Take path after engine/data/
	engineAndData := parts[0] + "/" + parts[1] + "/"
	relativePath := strings.TrimPrefix(sourcePath, engineAndData)

	// Split relative path to get segments
	relativeParts := strings.Split(relativePath, "/")
	if len(relativeParts) == 0 {
		return baseDestPath
	}

	// If baseDestPath already contains engine, use it
	if strings.Contains(baseDestPath, "/data/") {
		// Take all parts except the first one (remove the first segment)
		if len(relativeParts) > 1 {
			restPath := strings.Join(relativeParts[1:], "/")
			return baseDestPath + "/" + restPath
		}
		// If only one segment, take it
		return baseDestPath + "/" + relativePath
	}

	// Otherwise add engine from source
	if len(relativeParts) > 1 {
		restPath := strings.Join(relativeParts[1:], "/")
		return parts[0] + "/data/" + baseDestPath + "/" + restPath
	}
	return parts[0] + "/data/" + baseDestPath + "/" + relativePath
}

func (m *MockClient) GetKVEngine(path string) (string, error) {
	// Simple implementation for tests
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "secret", nil // Default
	}
	return parts[0], nil
}

// buildPath constructs a full path from a base path and an item name.
// It ensures proper path separators are used.
func buildPath(base, item string) string {
	if strings.HasSuffix(base, "/") {
		return base + item
	}
	return base + "/" + item
}

// ExpandWildcardPath implements wildcard expansion for tests
func (m *MockClient) ExpandWildcardPath(pattern string, logger *logger.Logger) ([]string, error) {
	// For testing purposes, we'll implement a simple wildcard expansion
	// that matches the actual implementation

	// Check if pattern contains wildcard
	if !strings.Contains(pattern, "*") {
		// No wildcard, return as is
		return []string{pattern}, nil
	}

	// For mock purposes, we'll just return the pattern as is
	// In a real implementation, this would expand the wildcard
	return []string{pattern}, nil
}
