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
	// Игнорируем логгер для тестов
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
	// Игнорируем логгер для тестов
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.CheckErrors[path]; ok {
		return false, err
	}

	// Проверяем, есть ли в списке директорий
	if isDir, ok := m.Directories[path]; ok {
		return isDir, nil
	}

	// Проверяем, есть ли в списке результатов List
	if _, ok := m.ListResults[path]; ok {
		return true, nil
	}

	return false, nil
}

func (m *MockClient) ListSecrets(path string, logger *logger.Logger) ([]string, error) {
	// Игнорируем логгер для тестов
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.ListErrors[path]; ok {
		return nil, err
	}

	// Проверяем, является ли путь директорией
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
	// Игнорируем логгер для тестов
	secretsChan := make(chan *vault.Secret, 100) // Увеличиваем буфер
	errChan := make(chan error, 1)

	go func() {
		defer close(secretsChan)
		defer close(errChan)

		m.mu.RLock()
		defer m.mu.RUnlock()

		// Рекурсивно собираем все секреты
		m.collectSecrets(ctx, rootPath, secretsChan, errChan)
	}()

	return secretsChan, errChan
}

func (m *MockClient) collectSecrets(ctx context.Context, rootPath string, secretsChan chan *vault.Secret, errChan chan error) {
	// Проверяем, является ли rootPath секретом
	if secret, ok := m.Secrets[rootPath]; ok {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		case secretsChan <- secret:
		}
	}

	// Проверяем, является ли rootPath директорией
	if isDir, ok := m.Directories[rootPath]; ok && isDir {
		// Получаем список элементов в директории
		if items, ok := m.ListResults[rootPath]; ok {
			// Для каждого элемента проверяем, является ли он секретом или директорией
			for _, item := range items {
				fullPath := rootPath + "/" + item

				// Рекурсивно обрабатываем элемент
				m.collectSecrets(ctx, fullPath, secretsChan, errChan)
			}
		}
	} else {
		// Проверяем, есть ли в списке результатов List
		// Это может быть случай, когда путь не помечен как директория,
		// но у него есть список элементов
		if items, ok := m.ListResults[rootPath]; ok {
			// Для каждого элемента проверяем, является ли он секретом или директорией
			for _, item := range items {
				fullPath := rootPath + "/" + item

				// Рекурсивно обрабатываем элемент
				m.collectSecrets(ctx, fullPath, secretsChan, errChan)
			}
		}
	}
}

func (m *MockClient) WriteSecret(path string, data map[string]interface{}, logger *logger.Logger) error {
	// Игнорируем логгер для тестов
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
	// Игнорируем логгер для тестов
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.CheckErrors[path]; ok {
		return false, err
	}

	// Проверяем, является ли путь директорией
	if isDir, ok := m.Directories[path]; ok && isDir {
		return false, nil // Директории не считаются секретами
	}

	_, exists := m.Secrets[path]
	return exists, nil
}

func (m *MockClient) BatchWriteSecrets(ctx context.Context, secrets <-chan *vault.Secret, basePath string, logger *logger.Logger) <-chan error {
	// Игнорируем логгер для тестов
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
		// Убираем слэш в конце, если есть
		trimmedDest := strings.TrimSuffix(baseDestPath, "/")
		if relativePath != "" {
			return trimmedDest + "/" + relativePath
		}
		return trimmedDest
	}

	// Иначе добавляем движок из source
	if relativePath != "" {
		return parts[0] + "/data/" + baseDestPath + "/" + relativePath
	}
	return parts[0] + "/data/" + baseDestPath
}

func (m *MockClient) GetKVEngine(path string) (string, error) {
	// Простая реализация для тестов
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "secret", nil // По умолчанию
	}
	return parts[0], nil
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
