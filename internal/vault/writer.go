package vault

import (
	"context"
	"fmt"
	"strings"
	"vault-copy/internal/logger"
)

func (c *Client) WriteSecret(path string, data map[string]interface{}, logger *logger.Logger) error {
	// Для KV v2 нужно обернуть данные
	writeData := map[string]interface{}{
		"data": data,
	}

	logger.Verbose("Запись секрета в Vault: %s", path)
	_, err := c.client.Logical().Write(path, writeData)
	if err != nil {
		logger.Error("Ошибка записи секрета %s: %v", path, err)
		return fmt.Errorf("ошибка записи секрета %s: %v", path, err)
	}

	logger.Verbose("Успешно записан секрет: %s", path)
	return nil
}

func (c *Client) SecretExists(path string, logger *logger.Logger) (bool, error) {
	logger.Verbose("Проверка существования секрета: %s", path)
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		logger.Error("Ошибка проверки существования секрета %s: %v", path, err)
		return false, err
	}

	exists := secret != nil
	logger.Verbose("Секрет %s существует: %t", path, exists)
	return exists, nil
}

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

			// Преобразуем путь из source в destination
			destPath := transformPath(secret.Path, basePath)

			err := c.WriteSecret(destPath, secret.Data, logger)
			if err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
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
