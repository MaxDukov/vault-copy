package vault

import (
	"strings"

	"github.com/hashicorp/vault/api"
)

// Client is a wrapper around the Vault API client.
type Client struct {
	// client is the underlying Vault API client
	client *api.Client
	// config holds the client configuration
	config *ClientConfig
}

// Убедимся, что Client реализует ClientInterface
var _ ClientInterface = (*Client)(nil)

// ClientConfig holds the configuration for a Vault client.
type ClientConfig struct {
	// Addr is the address of the Vault server
	Addr string
	// Token is the authentication token for the Vault server
	Token string
}

// NewClient creates a new Vault client with the provided address and token.
// It also verifies the connection to the Vault server by checking its health.
func NewClient(addr, token string) (*Client, error) {
	config := &api.Config{
		Address: addr,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(token)

	// Verify connection
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

// GetKVEngine extracts the KV engine name from a Vault path.
// If the path doesn't contain a slash, it returns "secret" as the default engine.
func (c *Client) GetKVEngine(path string) (string, error) {
	// Determine KV engine from path
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "secret", nil // Default
	}
	return parts[0], nil
}
