package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetIndexMappings returns the mappings for a specific index
func (c *Client) GetIndexMappings(indexName string) (map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Indices.GetMapping(
		c.es.Indices.GetMapping.WithContext(ctx),
		c.es.Indices.GetMapping.WithIndex(indexName),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting index mappings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var mappings map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&mappings); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return mappings, nil
}

// GetPrettyIndexMappings returns the mappings for a specific index in a pretty-printed JSON format
func (c *Client) GetPrettyIndexMappings(indexName string) (string, error) {
	mappings, err := c.GetIndexMappings(indexName)
	if err != nil {
		return "", err
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error formatting mappings: %w", err)
	}

	return string(prettyJSON), nil
}

// PutIndexMapping adds or updates a mapping for a specific index
func (c *Client) PutIndexMapping(indexName string, mapping map[string]interface{}) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert mapping to JSON
	body, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("error encoding mapping: %w", err)
	}

	// Execute request - the first two parameters are required (index array and body)
	res, err := c.es.Indices.PutMapping(
		[]string{indexName},
		bytes.NewReader(body),
		c.es.Indices.PutMapping.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error putting index mapping: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}
