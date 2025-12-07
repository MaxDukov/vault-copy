package config

import (
	"errors"
	"log"
	"os"
	"strings"
)

type Config struct {
	SourcePath      string
	DestinationPath string
	Recursive       bool
	DryRun          bool
	Overwrite       bool
	ParallelWorkers int

	// Source Vault
	SourceAddr  string
	SourceToken string

	// Destination Vault
	DestAddr  string
	DestToken string
}

func NewConfig(
	sourcePath, destinationPath string,
	recursive, dryRun, overwrite bool,
	parallelWorkers int,
	sourceAddr, sourceToken,
	destAddr, destToken string,
) (*Config, error) {

	cfg := &Config{
		SourcePath:      normalizePath(sourcePath),
		DestinationPath: normalizePath(destinationPath),
		Recursive:       recursive,
		DryRun:          dryRun,
		Overwrite:       overwrite,
		ParallelWorkers: parallelWorkers,
	}

	// Получение конфигурации  Vault истоника
	if sourceAddr == "" {
		sourceAddr = os.Getenv("VAULT_SOURCE_ADDR")
	}
	if sourceAddr == "" {
		sourceAddr = os.Getenv("VAULT_ADDR")
	}
	if sourceAddr == "" {
		sourceAddr = "http://localhost:8200"
	}
	cfg.SourceAddr = sourceAddr

	if sourceToken == "" {
		sourceToken = os.Getenv("VAULT_SOURCE_TOKEN")
	}
	if sourceToken == "" {
		sourceToken = os.Getenv("VAULT_TOKEN")
	}
	if sourceToken == "" {
		return nil, errors.New("токен Vault-источника не найден. Установите VAULT_SOURCE_TOKEN или VAULT_TOKEN")
	}
	cfg.SourceToken = sourceToken

	// Получение конфигурации Vault-приемника
	if destAddr == "" {
		destAddr = os.Getenv("VAULT_DEST_ADDR")
	}
	if destAddr == "" {
		destAddr = os.Getenv("VAULT_ADDR")
		log.Println("VAULT_DEST_ADDR не найден, используется VAULT_ADDR, копирование внутри одного Vault")
	}
	cfg.DestAddr = destAddr

	if destToken == "" {
		destToken = os.Getenv("VAULT_DEST_TOKEN")
	}
	if destToken == "" {
		destToken = os.Getenv("VAULT_TOKEN")
	}
	if destToken == "" {
		return nil, errors.New("токен Vault-примника не найден. Установите VAULT_DEST_TOKEN или VAULT_TOKEN")
	}
	cfg.DestToken = destToken

	// Валидация
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func normalizePath(path string) string {
	// Убедимся что путь начинается с правильного префикса для KV v2
	if !strings.HasPrefix(path, "secret/data/") && !strings.HasPrefix(path, "kv/data/") {
		// Пробуем определить движок
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			// Если уже есть движок в пути, добавляем data/
			return parts[0] + "/data/" + parts[1]
		}
	}
	return path
}

func (c *Config) Validate() error {
	if c.SourcePath == "" {
		return errors.New("путь источника не может быть пустым")
	}

	if c.DestinationPath == "" {
		return errors.New("путь назначения не может быть пустым")
	}

	if c.ParallelWorkers < 1 {
		return errors.New("количество параллельных работников должно быть >= 1")
	}

	return nil
}
