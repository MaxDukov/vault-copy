package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"vault-copy/internal/config"
	"vault-copy/internal/sync"
	"vault-copy/internal/vault"
)

func main() {
	// Парсинг аргументов командной строки
	srcPath := flag.String("src-path", "", "Путь к секрету или папке в источнике (обязательно)")
	dstPath := flag.String("dst-path", "", "Путь назначения в целевом Vault (обязательно)")
	recursive := flag.Bool("recursive", false, "Рекурсивно копировать все секреты из папки(по умолчанию отключено)")
	dryRun := flag.Bool("dry-run", false, "Показать что будет скопировано без выполнения")
	overwrite := flag.Bool("overwrite", false, "Перезаписывать существующие секреты (по умолчанию отключено)")
	parallel := flag.Int("parallel", 5, "Количество параллельных операций")
	verbose := flag.Bool("v", false, "Включить подробный вывод")

	// Флаги для source Vault
	sourceAddr := flag.String("src-addr", "", "URL Vault-источника (по-умолчанию будет использована переменная окружения VAULT_SOURCE_ADDR)")
	sourceToken := flag.String("src-token", "", "Токен для Vault-источника (по-умолчанию будет использована переменная окружения VAULT_SOURCE_TOKEN)")

	// Флаги для destination Vault
	destAddr := flag.String("dst-addr", "", "URL Vault-приемника (по-умолчанию будет использована переменная окружения VAULT_DEST_ADDR)")
	destToken := flag.String("dst-token", "", "Токен для Vault-приемника (по-умолчанию будет использована переменная окруженияи VAULT_DEST_TOKEN)")

	flag.Parse()

	// Валидация аргументов
	if *srcPath == "" || *dstPath == "" {
		message := `
пример использования 

export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"

./vault-sync --src-path="secret/data/apps/production" --dst-path="secret/data/backup/production" --recursive --parallel=10`
		fmt.Println("Требуются как минимум 2 параметра: --src-path и --dst-path, в этом случае секреты будут скопированы внутри VAULT_SOURCE_ADDR")
		fmt.Println(message)
		fmt.Println("введите --help для получения справки")
		os.Exit(1)
	}

	// Создание конфигурации
	cfg, err := config.NewConfig(
		*srcPath,
		*dstPath,
		*recursive,
		*dryRun,
		*overwrite,
		*verbose,
		*parallel,
		*sourceAddr,
		*sourceToken,
		*destAddr,
		*destToken,
	)
	if err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	// Инициализация клиентов Vault
	sourceClient, err := vault.NewClient(cfg.SourceAddr, cfg.SourceToken)
	if err != nil {
		log.Fatalf("Ошибка создания клиента Vault-источника: %v", err)
	}

	destClient, err := vault.NewClient(cfg.DestAddr, cfg.DestToken)
	if err != nil {
		log.Fatalf("Ошибка создания клиента Vault-приемника: %v", err)
	}

	// Создание менеджера синхронизации
	syncManager := sync.NewManager(sourceClient, destClient, cfg)

	// Выполнение синхронизации
	ctx := context.Background()
	stats, err := syncManager.Sync(ctx)
	if err != nil {
		log.Fatalf("Ошибка синхронизации: %v", err)
	}

	// Вывод статистики
	fmt.Printf("\nСинхронизация завершена:\n")
	fmt.Printf("  Прочитано секретов: %d\n", stats.SecretsRead)
	fmt.Printf("  Записано секретов: %d\n", stats.SecretsWritten)
	fmt.Printf("  Пропущено (существуют): %d\n", stats.SecretsSkipped)
	fmt.Printf("  Ошибок: %d\n", stats.Errors)

	if *dryRun {
		fmt.Println("\nРежим dry-run - ничего не было записано")
	}
}
