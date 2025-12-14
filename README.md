# vault-copy

Command line utility for copying secrets between HashiCorp Vault instances or within a single instance.

## Description

`vault-copy` is a command-line tool written in Go that allows you to copy secrets between different Vault instances or within a single instance. It supports recursive directory copying, parallel processing, "dry-run" mode for previewing operations, and overwriting existing secrets.

## Installation

To install the utility, you will need Go 1.21 or higher.

```bash
# Clone the repository
git clone <repository-url>
cd vault-copy

# Build the project
go build -o vault-copy cmd/vault-copy/main.go

# (Optional) Install to system
sudo cp vault-copy /usr/local/bin/
```

## Usage

### Basic usage

```bash
# Export environment variables
export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"

# Copy secret
./vault-copy --src-path="secret/data/apps/production" --dst-path="secret/data/backup/production"
```

### Recursive copying

```bash
# Recursively copy all secrets from directory
./vault-copy --src-path="secret/data/apps" --dst-path="secret/data/backup/apps" --recursive --parallel=10
```

### Copying between different Vault instances

```bash
# Export environment variables for both Vaults
export VAULT_SOURCE_TOKEN="source_token"
export VAULT_SOURCE_ADDR="https://vault1:8200"
export VAULT_DEST_TOKEN="dest_token"
export VAULT_DEST_ADDR="https://vault2:8200"

# Copy between instances
./vault-copy --src-path="secret/data/apps/production" --dst-path="secret/data/apps/production" --recursive
```

### Dry-run mode

```bash
# Preview what will be copied without performing operations
./vault-copy --src-path="secret/data/apps" --dst-path="secret/data/backup/apps" --recursive --dry-run
```

## Command line parameters

| Parameter | Description | Required | Default value |
|----------|----------|--------------|----------------------|
| `--src-path` | Path to secret or folder in source | Yes | - |
| `--dst-path` | Destination path in target Vault | Yes | - |
| `--recursive` | Recursively copy all secrets from folder | No | false |
| `--dry-run` | Show what will be copied without performing | No | false |
| `--overwrite` | Overwrite existing secrets | No | false |
| `--parallel` | Number of parallel operations | No | 5 |
| `--src-addr` | Source Vault URL | No | VAULT_SOURCE_ADDR or VAULT_ADDR |
| `--src-token` | Token for source Vault | No | VAULT_SOURCE_TOKEN or VAULT_TOKEN |
| `--dst-addr` | Destination Vault URL | No | VAULT_DEST_ADDR or VAULT_ADDR |
| `--dst-token` | Token for destination Vault | No | VAULT_DEST_TOKEN or VAULT_TOKEN |

## Configuration File

You can also specify configuration options in a `config.yaml` file:

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

The configuration priority is:
1. Command line arguments (highest priority)
2. Environment variables
3. Configuration file (lowest priority)

## License

This project is licensed under the AGNU License - see the [LICENSE](LICENSE) file for details.