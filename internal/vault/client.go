package vault

import (
	"fmt"
	"strings"
	"vault-copy/internal/logger"

	"github.com/hashicorp/vault/api"
)

// Client is a wrapper around the Vault API client.
type Client struct {
	// client is the underlying Vault API client
	client *api.Client
	// config holds the client configuration
	config *ClientConfig
}

// Ensure that Client implements ClientInterface
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

// GetKVEngineVersion gets the version of a KV engine from Vault
func (c *Client) GetKVEngineVersion(engine string, logger *logger.Logger) (int, error) {
	logger.Verbose("Getting KV engine version for: %s", engine)

	// Get engine configuration
	path := fmt.Sprintf("sys/mounts/%s", engine)
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		logger.Error("Error reading engine config for %s: %v", engine, err)
		return 0, fmt.Errorf("error reading engine config for %s: %v", engine, err)
	}

	if secret == nil || secret.Data == nil {
		logger.Verbose("No configuration found for engine: %s", engine)
		return 0, fmt.Errorf("no configuration found for engine: %s", engine)
	}

	// Extract options
	options, ok := secret.Data["options"].(map[string]interface{})
	if !ok {
		logger.Verbose("No options found for engine: %s, assuming v1", engine)
		return 1, nil // Default to v1 if no options
	}

	// Get version from options
	versionStr, ok := options["version"].(string)
	if !ok {
		logger.Verbose("No version found in options for engine: %s, assuming v1", engine)
		return 1, nil // Default to v1 if no version
	}

	// Convert version string to int
	if versionStr == "1" {
		logger.Verbose("Engine %s is KV version 1", engine)
		return 1, nil
	} else if versionStr == "2" {
		logger.Verbose("Engine %s is KV version 2", engine)
		return 2, nil
	} else {
		logger.Verbose("Unknown version %s for engine: %s, assuming v1", versionStr, engine)
		return 1, nil // Default to v1 for unknown versions
	}
}
