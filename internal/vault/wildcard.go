package vault

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"vault-copy/internal/logger"
)

// ExpandWildcardPath expands a path with wildcards to a list of matching paths
// For example: "secret/apps/app1/postgre*" will match "secret/apps/app1/postgres" and "secret/apps/app1/postgresql"
func (c *Client) ExpandWildcardPath(pattern string, logger *logger.Logger) ([]string, error) {
	logger.Verbose("Expanding wildcard path: %s", pattern)

	// Check if pattern contains wildcard
	if !strings.Contains(pattern, "*") {
		// No wildcard, return as is
		return []string{pattern}, nil
	}

	// Split path into parts
	parts := strings.Split(pattern, "/")
	if len(parts) == 0 {
		return []string{pattern}, nil
	}

	// Find the part with wildcard
	wildcardIndex := -1
	for i, part := range parts {
		if strings.Contains(part, "*") {
			wildcardIndex = i
			break
		}
	}

	if wildcardIndex == -1 {
		// No wildcard found, return as is
		return []string{pattern}, nil
	}

	// Get the base path up to the wildcard part
	basePath := strings.Join(parts[:wildcardIndex], "/")
	if basePath == "" {
		basePath = "/"
	}

	// Get the wildcard pattern
	wildcardPattern := parts[wildcardIndex]

	// List items in the base path
	items, err := c.ListSecrets(basePath, logger)
	if err != nil {
		logger.Error("Error listing secrets in %s: %v", basePath, err)
		return nil, err
	}

	var matchingPaths []string

	// Check each item against the wildcard pattern
	for _, item := range items {
		// Use filepath.Match for pattern matching
		matched, err := filepath.Match(wildcardPattern, item)
		if err != nil {
			logger.Error("Error matching pattern %s with %s: %v", wildcardPattern, item, err)
			return nil, fmt.Errorf("error matching pattern: %v", err)
		}

		if matched {
			// Build full path
			var fullPath string
			if basePath == "/" || basePath == "" {
				fullPath = item
			} else {
				fullPath = basePath + "/" + item
			}

			// Check if this is a directory
			isDir, err := c.IsDirectory(fullPath, logger)
			if err != nil {
				logger.Error("Error checking if %s is directory: %v", fullPath, err)
				return nil, err
			}

			if isDir {
				// For directories, we need to get all secrets under it
				// But first, we need to handle the rest of the path parts
				if wildcardIndex < len(parts)-1 {
					// There are more parts after the wildcard
					restPattern := strings.Join(parts[wildcardIndex+1:], "/")
					expanded, err := c.expandPathWithRest(fullPath, restPattern, logger)
					if err != nil {
						return nil, err
					}
					matchingPaths = append(matchingPaths, expanded...)
				} else {
					// This is the final part, get all secrets under this directory
					expanded, err := c.getAllPathsUnder(fullPath, logger)
					if err != nil {
						return nil, err
					}
					matchingPaths = append(matchingPaths, expanded...)
				}
			} else {
				// For files, check if there are more parts
				if wildcardIndex < len(parts)-1 {
					// This shouldn't happen for files, but let's handle it
					restPattern := strings.Join(parts[wildcardIndex+1:], "/")
					expanded, err := c.expandPathWithRest(fullPath, restPattern, logger)
					if err != nil {
						return nil, err
					}
					matchingPaths = append(matchingPaths, expanded...)
				} else {
					// Simple file match
					matchingPaths = append(matchingPaths, fullPath)
				}
			}
		}
	}

	logger.Verbose("Expanded wildcard path %s to %d paths", pattern, len(matchingPaths))
	return matchingPaths, nil
}

// expandPathWithRest expands a path when there are more parts after a wildcard match
func (c *Client) expandPathWithRest(basePath, restPattern string, logger *logger.Logger) ([]string, error) {
	// Handle recursive patterns like "**"
	if strings.Contains(restPattern, "**") {
		return c.getAllPathsUnder(basePath, logger)
	}

	// Split rest pattern
	restParts := strings.Split(restPattern, "/")
	if len(restParts) == 0 {
		return []string{basePath}, nil
	}

	// Check if first part has wildcard
	if strings.Contains(restParts[0], "*") {
		// Handle nested wildcard
		return c.expandNestedWildcard(basePath, restPattern, logger)
	}

	// Simple path extension
	fullPath := basePath + "/" + restPattern

	// Check if this path exists
	isDir, err := c.IsDirectory(fullPath, logger)
	if err != nil {
		// Path doesn't exist, return empty
		return []string{}, nil
	}

	if isDir {
		// Return all paths under this directory
		return c.getAllPathsUnder(fullPath, logger)
	}

	// Single file
	return []string{fullPath}, nil
}

// expandNestedWildcard handles nested wildcards in path
func (c *Client) expandNestedWildcard(basePath, pattern string, logger *logger.Logger) ([]string, error) {
	logger.Verbose("Expanding nested wildcard: %s under %s", pattern, basePath)

	parts := strings.Split(pattern, "/")
	if len(parts) == 0 {
		return []string{basePath}, nil
	}

	// Get the first part with wildcard
	wildcardPart := parts[0]

	// List items in basePath
	items, err := c.ListSecrets(basePath, logger)
	if err != nil {
		logger.Error("Error listing secrets in %s: %v", basePath, err)
		return nil, err
	}

	var matchingPaths []string

	// Check each item against the wildcard pattern
	for _, item := range items {
		matched, err := filepath.Match(wildcardPart, item)
		if err != nil {
			logger.Error("Error matching pattern %s with %s: %v", wildcardPart, item, err)
			return nil, fmt.Errorf("error matching pattern: %v", err)
		}

		if matched {
			fullPath := basePath + "/" + item

			// If there are more parts, continue expanding
			if len(parts) > 1 {
				restPattern := strings.Join(parts[1:], "/")
				expanded, err := c.expandPathWithRest(fullPath, restPattern, logger)
				if err != nil {
					return nil, err
				}
				matchingPaths = append(matchingPaths, expanded...)
			} else {
				// Check if this is a directory
				isDir, err := c.IsDirectory(fullPath, logger)
				if err != nil {
					logger.Error("Error checking if %s is directory: %v", fullPath, err)
					return nil, err
				}

				if isDir {
					// Get all paths under this directory
					expanded, err := c.getAllPathsUnder(fullPath, logger)
					if err != nil {
						return nil, err
					}
					matchingPaths = append(matchingPaths, expanded...)
				} else {
					// Single file
					matchingPaths = append(matchingPaths, fullPath)
				}
			}
		}
	}

	return matchingPaths, nil
}

// getAllPathsUnder gets all paths (files and directories) under a given path
func (c *Client) getAllPathsUnder(rootPath string, logger *logger.Logger) ([]string, error) {
	logger.Verbose("Getting all paths under: %s", rootPath)

	var allPaths []string

	// Use GetAllSecrets to get all secrets under this path
	ctx := context.Background()
	secretsChan, errChan := c.GetAllSecrets(ctx, rootPath, logger)

	// Collect all secrets
	for {
		select {
		case secret, ok := <-secretsChan:
			if !ok {
				// Channel closed
				goto checkErrors
			}
			allPaths = append(allPaths, secret.Path)
		case err, ok := <-errChan:
			if !ok {
				// Channel closed
				goto done
			}
			if err != nil {
				logger.Error("Error getting secrets under %s: %v", rootPath, err)
				return nil, err
			}
		}
	}

checkErrors:
	// Check for any remaining errors
	for err := range errChan {
		if err != nil {
			logger.Error("Error getting secrets under %s: %v", rootPath, err)
			return nil, err
		}
	}

done:
	logger.Verbose("Found %d paths under %s", len(allPaths), rootPath)
	return allPaths, nil
}
