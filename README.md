# vault-copy

Утилита командной строки для копирования секретов между HashiCorp Vault инстансами или внутри одного инстанса.

## Описание

`vault-copy` - это инструмент командной строки, написанный на Go, который позволяет копировать секреты между различными Vault инстансами или внутри одного инстанса. Поддерживает рекурсивное копирование директорий, параллельную обработку, режим "dry-run" для предварительного просмотра операций и перезапись существующих секретов.

## Установка

Для установки утилиты вам понадобится Go 1.21 или выше.

```bash
# Клонирование репозитория
git clone <repository-url>
cd vault-copy

# Сборка проекта
go build -o vault-copy cmd/vault-copy/main.go

# (Опционально) Установка в систему
sudo cp vault-copy /usr/local/bin/
```

## Использование

### Базовое использование

```bash
# Экспорт переменных окружения
export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"

# Копирование секрета
./vault-copy --src-path="secret/data/apps/production" --dst-path="secret/data/backup/production"
```

### Рекурсивное копирование

```bash
# Рекурсивное копирование всех секретов из директории
./vault-copy --src-path="secret/data/apps" --dst-path="secret/data/backup/apps" --recursive --parallel=10
```

### Копирование между разными Vault инстансами

```bash
# Экспорт переменных окружения для обоих Vault
export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"
export VAULT_DEST_TOKEN="dest_token"
export VAULT_DEST_ADDR="https://vault2:8200"

# Копирование между инстансами
./vault-copy --src-path="secret/data/apps/production" --dst-path="secret/data/apps/production" --recursive
```

### Режим Dry-run

```bash
# Просмотр того, что будет скопировано без выполнения операций
./vault-copy --src-path="secret/data/apps" --dst-path="secret/data/backup/apps" --recursive --dry-run
```

## Параметры командной строки

| Параметр | Описание | Обязательный | Значение по умолчанию |
|----------|----------|--------------|----------------------|
| `--src-path` | Путь к секрету или папке в источнике | Да | - |
| `--dst-path` | Путь назначения в целевом Vault | Да | - |
| `--recursive` | Рекурсивно копировать все секреты из папки | Нет | false |
| `--dry-run` | Показать что будет скопировано без выполнения | Нет | false |
| `--overwrite` | Перезаписывать существующие секреты | Нет | false |
| `--parallel` | Количество параллельных операций | Нет | 5 |
| `--src-addr` | URL Vault-источника | Нет | VAULT_SOURCE_ADDR или VAULT_ADDR |
| `--src-token` | Токен для Vault-источника | Нет | VAULT_SOURCE_TOKEN или VAULT_TOKEN |
| `--dst-addr` | URL Vault-приемника | Нет | VAULT_DEST_ADDR или VAULT_ADDR |
| `--dst-token` | Токен для Vault-приемника | Нет | VAULT_DEST_TOKEN или VAULT_TOKEN |

## Лицензия

Этот проект лицензирован под AGNU License - смотрите файл [LICENSE](LICENSE) для подробностей.