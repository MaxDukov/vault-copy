# vault-copy

Командная утилита для копирования секретов между экземплярами HashiCorp Vault или внутри одного экземпляра.

## Описание

`vault-copy` - это инструмент командной строки, написанный на Go, который позволяет копировать секреты между различными экземплярами Vault или внутри одного экземпляра. Он поддерживает рекурсивное копирование каталогов, параллельную обработку, режим "dry-run" для предварительного просмотра операций и перезапись существующих секретов.

## Установка

Для установки утилиты вам понадобится Go 1.21 или выше.

```bash
# Клонируйте репозиторий
git clone <repository-url>
cd vault-copy

# Соберите проект
go build -o vault-copy cmd/vault-copy/main.go

# (Опционально) Установите в систему
sudo cp vault-copy /usr/local/bin/
```

## Использование

### Базовое использование

```bash
# Экспортируйте переменные окружения
export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"

# Скопируйте секрет
./vault-copy --src-path="secret/data/apps/production" --dst-path="secret/data/backup/production"
```

### Рекурсивное копирование

```bash
# Рекурсивно скопируйте все секреты из каталога
./vault-copy --src-path="secret/data/apps" --dst-path="secret/data/backup/apps" --recursive --parallel=10
```

### Копирование между различными экземплярами Vault

```bash
# Экспортируйте переменные окружения для обоих Vault
export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"
export VAULT_DEST_TOKEN="dest_token"
export VAULT_DEST_ADDR="https://vault2:8200"

# Копирование между экземплярами
./vault-copy --src-path="secret/data/apps/production" --dst-path="secret/data/apps/production" --recursive
```

### Режим Dry-run

```bash
# Предварительный просмотр того, что будет скопировано, без выполнения операций
./vault-copy --src-path="secret/data/apps" --dst-path="secret/data/backup/apps" --recursive --dry-run
```

## Параметры командной строки

| Параметр | Описание | Обязательный | Значение по умолчанию |
|----------|----------|--------------|----------------------|
| `--src-path` | Путь к секрету или папке в источнике (поддерживает подстановочные знаки) | Да | - |
| `--dst-path` | Путь назначения в целевом Vault | Да | - |
| `--recursive` | Рекурсивно копировать все секреты из папки | Нет | false |
| `--dry-run` | Показать, что будет скопировано, без выполнения | Нет | false |
| `--overwrite` | Перезаписать существующие секреты | Нет | false |
| `--parallel` | Количество параллельных операций | Нет | 5 |
| `--src-addr` | URL исходного Vault | Нет | VAULT_SOURCE_ADDR или VAULT_ADDR |
| `--src-token` | Токен для исходного Vault | Нет | VAULT_SOURCE_TOKEN или VAULT_TOKEN |
| `--dst-addr` | URL целевого Vault | Нет | VAULT_DEST_ADDR или VAULT_ADDR |
| `--dst-token` | Токен для целевого Vault | Нет | VAULT_DEST_TOKEN или VAULT_TOKEN |

## Поддержка подстановочных знаков

Параметр `--src-path` теперь поддерживает шаблоны подстановочных знаков для копирования нескольких секретов или каталогов одновременно:

- `secret/apps/app1/postgre*` - соответствует всем секретам или каталогам, начинающимся с "postgre" в указанном пути
- `secret/apps/*/database` - соответствует секретам "database" во всех каталогах приложений

Пример:
```bash
# Скопируйте все секреты, связанные с postgres
./vault-copy --src-path="secret/data/apps/app1/postgre*" --dst-path="secret/data/backup/postgres" --recursive
```

## Файл конфигурации

Вы также можете указать параметры конфигурации в файле `config.yaml`:

```yaml
source:
  address: "https://vault-source:8200"
  token: "source-token"
destination:
  address: "https://vault-dest:8200"
  token: "dest-token"
settings:
  recursive: true
  dry_run: false
  overwrite: false
  parallel: 5
  verbose: false
```

Приоритет конфигурации:
1. Аргументы командной строки (наивысший приоритет)
2. Переменные окружения
3. Файл конфигурации (низший приоритет)

## Лицензия

Этот проект лицензирован по лицензии AGPL - смотрите файл [LICENSE](LICENSE) для получения подробной информации.