package client

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// GetHotThreads returns the hot threads for all nodes in the cluster
func (c *Client) GetHotThreads() (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Nodes.HotThreads(
		c.es.Nodes.HotThreads.WithContext(ctx),
	)
	if err != nil {
		return "", fmt.Errorf("error getting hot threads: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("error response: %s", res.String())
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return string(body), nil
}

// GetNodesHotThreads returns the hot threads for specific nodes in the cluster
func (c *Client) GetNodesHotThreads(nodeIDs []string) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Nodes.HotThreads(
		c.es.Nodes.HotThreads.WithContext(ctx),
		c.es.Nodes.HotThreads.WithNodeID(strings.Join(nodeIDs, ",")),
	)
	if err != nil {
		return "", fmt.Errorf("error getting hot threads for nodes %v: %w", nodeIDs, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("error response: %s", res.String())
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return string(body), nil
}
