package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Secret struct {
	Path     string
	Data     map[string]interface{}
	Metadata map[string]interface{}
}

func (c *Client) ReadSecret(path string) (*Secret, error) {
	secret, err := c.client.Logical().Read(path)
	if err != nil {
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

func (c *Client) IsDirectory(path string) (bool, error) {
	// Пробуем получить листинг
	listPath := strings.Replace(path, "/data/", "/metadata/", 1)
	if !strings.Contains(listPath, "/metadata/") {
		listPath = path + "/"
	}

	secret, err := c.client.Logical().List(listPath)
	if err != nil {
		// Если ошибка 405 или 404, это не папка
		if strings.Contains(err.Error(), "405") ||
			strings.Contains(err.Error(), "404") ||
			strings.Contains(err.Error(), "permission denied") {
			return false, nil
		}
		return false, err
	}

	return secret != nil && secret.Data != nil, nil
}

func (c *Client) ListSecrets(path string) ([]string, error) {
	// Для KV v2 используем metadata endpoint для листинга
	listPath := strings.Replace(path, "/data/", "/metadata/", 1)
	if !strings.Contains(listPath, "/metadata/") {
		listPath = path + "/"
	}

	secret, err := c.client.Logical().List(listPath)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	var result []string
	for _, key := range keys {
		if str, ok := key.(string); ok {
			result = append(result, str)
		}
	}

	return result, nil
}

func (c *Client) GetAllSecrets(ctx context.Context, rootPath string) (<-chan *Secret, <-chan error) {
	secretsChan := make(chan *Secret, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(secretsChan)
		defer close(errChan)

		c.walkSecrets(ctx, rootPath, secretsChan, errChan)
	}()

	return secretsChan, errChan
}

func (c *Client) walkSecrets(ctx context.Context, path string, secretsChan chan<- *Secret, errChan chan<- error) {
	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return
	default:
	}

	isDir, err := c.IsDirectory(path)
	if err != nil {
		errChan <- err
		return
	}

	if isDir {
		items, err := c.ListSecrets(path)
		if err != nil {
			errChan <- err
			return
		}

		var wg sync.WaitGroup
		for _, item := range items {
			wg.Add(1)
			go func(itemPath string) {
				defer wg.Done()
				c.walkSecrets(ctx, itemPath, secretsChan, errChan)
			}(buildPath(path, item))
		}
		wg.Wait()
	} else {
		secret, err := c.ReadSecret(path)
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
