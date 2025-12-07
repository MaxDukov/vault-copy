package mocks

import (
	"context"
	"strings"
	"sync"

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

func (m *MockClient) ReadSecret(path string) (*vault.Secret, error) {
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

func (m *MockClient) IsDirectory(path string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.CheckErrors[path]; ok {
		return false, err
	}

	isDir, ok := m.Directories[path]
	if !ok {
		return false, nil
	}

	return isDir, nil
}

func (m *MockClient) ListSecrets(path string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.ListErrors[path]; ok {
		return nil, err
	}

	items, ok := m.ListResults[path]
	if !ok {
		return []string{}, nil
	}

	return items, nil
}

func (m *MockClient) GetAllSecrets(ctx context.Context, rootPath string) (<-chan *vault.Secret, <-chan error) {
	secretsChan := make(chan *vault.Secret, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(secretsChan)
		defer close(errChan)

		m.mu.RLock()
		defer m.mu.RUnlock()

		for path, secret := range m.Secrets {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				if isSubPath(path, rootPath) {
					secretsChan <- secret
				}
			}
		}
	}()

	return secretsChan, errChan
}

func (m *MockClient) WriteSecret(path string, data map[string]interface{}) error {
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

func (m *MockClient) SecretExists(path string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.CheckErrors[path]; ok {
		return false, err
	}

	_, exists := m.Secrets[path]
	return exists, nil
}

func (m *MockClient) BatchWriteSecrets(ctx context.Context, secrets <-chan *vault.Secret, basePath string) <-chan error {
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

			// Преобразуем путь из source в destination
			destPath := transformPath(secret.Path, basePath)

			err := m.WriteSecret(destPath, secret.Data)
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
	// Извлекаем относительный путь от последнего элемента
	// Например: secret/data/apps/app1/config -> secret/data/destination/app1/config
	parts := strings.Split(sourcePath, "/")
	if len(parts) < 3 {
		return baseDestPath
	}

	// Берем путь после движка/data/
	engineAndData := parts[0] + "/" + parts[1] + "/"
	relativePath := strings.TrimPrefix(sourcePath, engineAndData)

	// Если baseDestPath уже содержит движок, используем его
	if strings.Contains(baseDestPath, "/data/") {
		return baseDestPath + "/" + relativePath
	}

	// Иначе добавляем движок из source
	return parts[0] + "/data/" + baseDestPath + "/" + relativePath
}

func isSubPath(path, root string) bool {
	if root == "" {
		return true
	}
	if len(path) < len(root) {
		return false
	}
	if path[:len(root)] != root {
		return false
	}
	if len(path) == len(root) {
		return true
	}
	return path[len(root)] == '/'
}
