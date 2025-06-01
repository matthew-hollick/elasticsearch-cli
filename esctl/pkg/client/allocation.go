package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9/esapi"
)

// AllocationStatus represents the current allocation status
type AllocationStatus struct {
	Status string `json:"status"`
}

// GetAllocationStatus returns the current allocation status
func (c *Client) GetAllocationStatus() (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Cluster.GetSettings(
		c.es.Cluster.GetSettings.WithContext(ctx),
		c.es.Cluster.GetSettings.WithFlatSettings(true),
		c.es.Cluster.GetSettings.WithIncludeDefaults(true),
	)
	if err != nil {
		return "", fmt.Errorf("error getting cluster settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var settings map[string]map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&settings); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	// Check transient settings first
	if transient, ok := settings["transient"]; ok {
		if allocation, ok := transient["cluster.routing.allocation.enable"].(string); ok {
			return allocation, nil
		}
	}

	// Check persistent settings
	if persistent, ok := settings["persistent"]; ok {
		if allocation, ok := persistent["cluster.routing.allocation.enable"].(string); ok {
			return allocation, nil
		}
	}

	// Check default settings
	if defaults, ok := settings["defaults"]; ok {
		if allocation, ok := defaults["cluster.routing.allocation.enable"].(string); ok {
			return allocation, nil
		}
	}

	return "all", nil // Default value if not explicitly set
}

// SetAllocationStatus sets the allocation status
func (c *Client) SetAllocationStatus(status string) error {
	// Validate status
	validStatuses := map[string]bool{
		"all":           true,
		"primaries":     true,
		"new_primaries": true,
		"none":          true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid allocation status: %s. Must be one of: all, primaries, new_primaries, none", status)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request body
	body := map[string]interface{}{
		"persistent": map[string]interface{}{
			"cluster.routing.allocation.enable": status,
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("error encoding request body: %w", err)
	}

	// Execute request
	res, err := c.es.Cluster.PutSettings(
		&buf,
		c.es.Cluster.PutSettings.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error updating cluster settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// GetAllocationExplain returns detailed explanation of shard allocations
func (c *Client) GetAllocationExplain(indexName, shardID string, primary bool) (map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request body if index and shard are specified
	var body map[string]interface{}
	var buf bytes.Buffer

	if indexName != "" && shardID != "" {
		body = map[string]interface{}{
			"index": indexName,
			"shard": shardID,
			"primary": primary,
		}

		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("error encoding request body: %w", err)
		}
	}

	// Execute request
	var res *esapi.Response
	var err error

	if body != nil {
		res, err = c.es.Cluster.AllocationExplain(
			c.es.Cluster.AllocationExplain.WithContext(ctx),
			c.es.Cluster.AllocationExplain.WithBody(&buf),
		)
	} else {
		res, err = c.es.Cluster.AllocationExplain(
			c.es.Cluster.AllocationExplain.WithContext(ctx),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("error getting allocation explanation: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var explanation map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&explanation); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return explanation, nil
}
