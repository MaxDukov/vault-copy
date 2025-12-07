package vault

import (
	"testing"

	"github.com/hashicorp/vault/api"
)

func TestNewClient(t *testing.T) {
	// Тест создание клиента с минимальной конфигурацией
	// Этот тест требует запущенного Vault или мок-сервера
	// Для изолированных тестов лучше использовать интерфейсы и моки

	t.Run("client creation with defaults", func(t *testing.T) {
		// Пропускаем в CI, так как требует Vault
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		client, err := NewClient("http://localhost:8200", "test-token")
		if err != nil {
			t.Logf("Note: This test requires running Vault: %v", err)
			t.Skip("Vault is not available")
		}

		if client == nil {
			t.Error("Expected client, got nil")
		}
	})
}

func TestGetKVEngine(t *testing.T) {
	client := &Client{
		client: &api.Client{},
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "secret engine",
			path:     "secret/data/path",
			expected: "secret",
		},
		{
			name:     "kv engine",
			path:     "kv/data/path",
			expected: "kv",
		},
		{
			name:     "custom engine",
			path:     "custom-engine/data/path",
			expected: "custom-engine",
		},
		{
			name:     "simple path",
			path:     "path",
			expected: "secret",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.GetKVEngine(tt.path)
			if err != nil {
				t.Errorf("GetKVEngine() error = %v", err)
				return
			}

			if got != tt.expected {
				t.Errorf("GetKVEngine() = %v, want %v", got, tt.expected)
			}
		})
	}
}
