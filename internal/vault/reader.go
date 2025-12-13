package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"vault-copy/internal/logger"
)

type Secret struct {
	Path     string
	Data     map[string]interface{}
	Metadata map[string]interface{}
}

func (c *Client) ReadSecret(path string, logger *logger.Logger) (*Secret, error) {
	logger.Verbose("Чтение секрета из Vault: %s", path)
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		logger.Error("Ошибка чтения секрета %s: %v", path, err)
		return nil, err
	}

	if secret == nil {
		return nil, fmt.Errorf("секрет не найден: %s", path)
	}

	// Для KV v2 данные находятся в secret.Data["data"]
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		data = secret.Data // Для KV v1 или других движков
	}

	metadata, _ := secret.Data["metadata"].(map[string]interface{})

	return &Secret{
		Path:     path,
		Data:     data,
		Metadata: metadata,
	}, nil
}

func (c *Client) IsDirectory(path string, logger *logger.Logger) (bool, error) {
	logger.Verbose("Проверка, является ли путь директорией: %s", path)
	// Пробуем получить листинг
	listPath := strings.Replace(path, "/data/", "/metadata/", 1)
	if !strings.Contains(listPath, "/metadata/") {
		listPath = path + "/"
	}

	logger.Verbose("Получение списка из: %s", listPath)
	secret, err := c.client.Logical().List(listPath)
	if err != nil {
		// Если ошибка 405 или 404, это не папка
		if strings.Contains(err.Error(), "405") ||
			strings.Contains(err.Error(), "404") ||
			strings.Contains(err.Error(), "permission denied") {
			logger.Verbose("Путь %s не является директорией (ошибка: %v)", path, err)
			return false, nil
		}
		logger.Error("Ошибка проверки пути %s: %v", path, err)
		return false, err
	}

	isDir := secret != nil && secret.Data != nil
	logger.Verbose("Путь %s является директорией: %t", path, isDir)
	return isDir, nil
}

func (c *Client) ListSecrets(path string, logger *logger.Logger) ([]string, error) {
	logger.Verbose("Получение списка секретов из: %s", path)
	// Для KV v2 используем metadata endpoint для листинга
	listPath := strings.Replace(path, "/data/", "/metadata/", 1)
	if !strings.Contains(listPath, "/metadata/") {
		listPath = path + "/"
	}

	logger.Verbose("Запрос списка из: %s", listPath)
	secret, err := c.client.Logical().List(listPath)
	if err != nil {
		logger.Error("Ошибка получения списка секретов из %s: %v", path, err)
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		logger.Verbose("Нет секретов в: %s", path)
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		logger.Verbose("Нет ключей в ответе для: %s", path)
		return []string{}, nil
	}

	var result []string
	for _, key := range keys {
		if str, ok := key.(string); ok {
			result = append(result, str)
		}
	}

	logger.Verbose("Найдено %d секретов в: %s", len(result), path)
	return result, nil
}

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

func buildPath(base, item string) string {
	if strings.HasSuffix(base, "/") {
		return base + item
	}
	return base + "/" + item
}
