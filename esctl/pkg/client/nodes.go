package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// NodeInfo represents information about a single node
type NodeInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	IP              string `json:"ip"`
	Role            string `json:"role"`
	HeapPercent     string `json:"heap.percent"`
	RAMPercent      string `json:"ram.percent"`
	CPU             string `json:"cpu"`
	Load1m          string `json:"load_1m"`
	Load5m          string `json:"load_5m"`
	Load15m         string `json:"load_15m"`
	DiskUsedPercent string `json:"disk.used_percent"`
	DiskTotal       string `json:"disk.total"`
	DiskAvailable   string `json:"disk.avail"`
	Uptime          string `json:"uptime"`
}

// GetNodes returns information about all nodes in the cluster
func (c *Client) GetNodes() ([]NodeInfo, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Cat.Nodes(
		c.es.Cat.Nodes.WithContext(ctx),
		c.es.Cat.Nodes.WithFormat("json"),
		c.es.Cat.Nodes.WithH("id,name,ip,role,heap.percent,ram.percent,cpu,load_1m,load_5m,load_15m,disk.used_percent,disk.total,disk.avail,uptime"),
		c.es.Cat.Nodes.WithFullID(true),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var nodes []NodeInfo
	if err := json.NewDecoder(res.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return nodes, nil
}

// GetNodeStats returns detailed stats for a specific node
func (c *Client) GetNodeStats(nodeID string) (map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Nodes.Stats(
		c.es.Nodes.Stats.WithContext(ctx),
		c.es.Nodes.Stats.WithNodeID(nodeID),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return stats, nil
}

// GetHotThreads returns hot threads information for nodes
func (c *Client) GetHotThreads(nodeID string) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare options
	opts := []func(*esapi.NodesHotThreadsRequest){}
	
	// Add node ID if specified
	if nodeID != "" {
		opts = append(opts, c.es.Nodes.HotThreads.WithNodeID(nodeID))
	}
	
	// Add context
	opts = append(opts, c.es.Nodes.HotThreads.WithContext(ctx))

	// Execute request
	res, err := c.es.Nodes.HotThreads(opts...)
	if err != nil {
		return "", fmt.Errorf("error getting response: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("error response: %s", res.String())
	}

	// Read response body as string
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}
	
	return buf.String(), nil
}
