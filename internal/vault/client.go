package vault

import (
	"strings"

	"github.com/hashicorp/vault/api"
)

type Client struct {
	client *api.Client
	config *ClientConfig
}

// Убедимся, что Client реализует ClientInterface
var _ ClientInterface = (*Client)(nil)

type ClientConfig struct {
	Addr  string
	Token string
}

func NewClient(addr, token string) (*Client, error) {
	config := &api.Config{
		Address: addr,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(token)

	// Проверяем соединение
	_, err = client.Sys().Health()
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
		config: &ClientConfig{
			Addr:  addr,
			Token: token,
		},
	}, nil
}

func (c *Client) GetKVEngine(path string) (string, error) {
	// Определяем движок KV из пути
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "secret", nil // По умолчанию
	}
	return parts[0], nil
}
