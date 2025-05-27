package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// IndexInfo represents information about a single index
type IndexInfo struct {
	Name       string `json:"index"`
	Status     string `json:"status"`
	Health     string `json:"health"`
	DocsCount  string `json:"docs.count"`
	DocsDeleted string `json:"docs.deleted"`
	StoreSize  string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

// GetIndices returns information about all indices in the cluster
func (c *Client) GetIndices(pattern string) ([]IndexInfo, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request
	indexPattern := "*"
	if pattern != "" {
		indexPattern = pattern
	}

	// Execute request
	res, err := c.es.Cat.Indices(
		c.es.Cat.Indices.WithContext(ctx),
		c.es.Cat.Indices.WithFormat("json"),
		c.es.Cat.Indices.WithH("index,status,health,docs.count,docs.deleted,store.size,pri.store.size"),
		c.es.Cat.Indices.WithIndex(indexPattern),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var indices []IndexInfo
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return indices, nil
}

// DeleteIndex deletes an index from the cluster
func (c *Client) DeleteIndex(indexName string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Indices.Delete(
		[]string{indexName},
		c.es.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error deleting index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// OpenIndex opens a closed index
func (c *Client) OpenIndex(indexName string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Indices.Open(
		[]string{indexName},
		c.es.Indices.Open.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error opening index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// CloseIndex closes an open index
func (c *Client) CloseIndex(indexName string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Indices.Close(
		[]string{indexName},
		c.es.Indices.Close.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error closing index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// GetIndexSettings gets settings for an index
func (c *Client) GetIndexSettings(indexName string) (map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Indices.GetSettings(
		c.es.Indices.GetSettings.WithContext(ctx),
		c.es.Indices.GetSettings.WithIndex(indexName),
		c.es.Indices.GetSettings.WithFlatSettings(true),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting index settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var settings map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return settings, nil
}

// UpdateIndexSettings updates settings for an index
func (c *Client) UpdateIndexSettings(indexName string, settings map[string]interface{}) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert settings to JSON
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("error marshaling settings: %w", err)
	}

	// Execute request
	res, err := c.es.Indices.PutSettings(
		strings.NewReader(string(settingsJSON)),
		c.es.Indices.PutSettings.WithContext(ctx),
		c.es.Indices.PutSettings.WithIndex(indexName),
	)
	if err != nil {
		return fmt.Errorf("error updating index settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}
