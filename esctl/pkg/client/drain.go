package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ClusterExcludeSettings represents the cluster allocation exclude settings
type ClusterExcludeSettings struct {
	ExcludeIP   []string `json:"ip"`
	ExcludeName []string `json:"name"`
	ExcludeHost []string `json:"host"`
}

// GetClusterExcludeSettings retrieves the current cluster allocation exclude settings
func (c *Client) GetClusterExcludeSettings() (*ClusterExcludeSettings, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get cluster settings
	res, err := c.es.Cluster.GetSettings(
		c.es.Cluster.GetSettings.WithContext(ctx),
		c.es.Cluster.GetSettings.WithFlatSettings(true),
		c.es.Cluster.GetSettings.WithIncludeDefaults(false),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting cluster settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var settings map[string]map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Extract exclude settings
	excludeSettings := &ClusterExcludeSettings{
		ExcludeIP:   []string{},
		ExcludeName: []string{},
		ExcludeHost: []string{},
	}

	// Check transient settings
	if transient, ok := settings["transient"]; ok {
		extractExcludeSettings(transient, excludeSettings)
	}

	// Check persistent settings
	if persistent, ok := settings["persistent"]; ok {
		extractExcludeSettings(persistent, excludeSettings)
	}

	return excludeSettings, nil
}

// DrainServer adds a node to the cluster allocation exclude list
func (c *Client) DrainServer(nodeName string) ([]string, error) {
	// Get current exclude settings
	settings, err := c.GetClusterExcludeSettings()
	if err != nil {
		return nil, err
	}

	// Check if node is already being drained
	for _, name := range settings.ExcludeName {
		if name == nodeName {
			return settings.ExcludeName, nil // Already draining
		}
	}

	// Add node to exclude list
	newExcludeList := append(settings.ExcludeName, nodeName)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request body
	body := map[string]interface{}{
		"persistent": map[string]interface{}{
			"cluster.routing.allocation.exclude.name": strings.Join(newExcludeList, ","),
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("error encoding request body: %w", err)
	}

	// Update cluster settings
	res, err := c.es.Cluster.PutSettings(
		&buf,
		c.es.Cluster.PutSettings.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("error updating cluster settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	return newExcludeList, nil
}

// StopDrainServer removes a node from the cluster allocation exclude list
func (c *Client) StopDrainServer(nodeName string) ([]string, error) {
	// Get current exclude settings
	settings, err := c.GetClusterExcludeSettings()
	if err != nil {
		return nil, err
	}

	// Check if node is being drained
	found := false
	var newExcludeList []string
	for _, name := range settings.ExcludeName {
		if name == nodeName {
			found = true
		} else {
			newExcludeList = append(newExcludeList, name)
		}
	}

	if !found {
		return settings.ExcludeName, nil // Not being drained
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request body
	body := map[string]interface{}{
		"persistent": map[string]interface{}{
			"cluster.routing.allocation.exclude.name": strings.Join(newExcludeList, ","),
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("error encoding request body: %w", err)
	}

	// Update cluster settings
	res, err := c.es.Cluster.PutSettings(
		&buf,
		c.es.Cluster.PutSettings.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("error updating cluster settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	return newExcludeList, nil
}

// FillServer removes a node from the cluster allocation exclude list (alias for StopDrainServer)
func (c *Client) FillServer(nodeName string) ([]string, error) {
	return c.StopDrainServer(nodeName)
}

// FillAll removes all nodes from the cluster allocation exclude list
func (c *Client) FillAll() (*ClusterExcludeSettings, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request body to clear all exclusion settings
	body := map[string]interface{}{
		"persistent": map[string]interface{}{
			"cluster.routing.allocation.exclude.name": nil,
			"cluster.routing.allocation.exclude.ip":   nil,
			"cluster.routing.allocation.exclude.host": nil,
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("error encoding request body: %w", err)
	}

	// Update cluster settings
	res, err := c.es.Cluster.PutSettings(
		&buf,
		c.es.Cluster.PutSettings.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("error updating cluster settings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Get the updated settings
	return c.GetClusterExcludeSettings()
}

// Helper function to extract exclude settings from a settings map
func extractExcludeSettings(settings map[string]interface{}, excludeSettings *ClusterExcludeSettings) {
	// Extract IP exclude settings
	if ipSetting, ok := settings["cluster.routing.allocation.exclude.ip"]; ok && ipSetting != "" {
		excludeIPs := strings.Split(ipSetting.(string), ",")
		for _, ip := range excludeIPs {
			if ip != "" {
				excludeSettings.ExcludeIP = append(excludeSettings.ExcludeIP, ip)
			}
		}
	}

	// Extract name exclude settings
	if nameSetting, ok := settings["cluster.routing.allocation.exclude.name"]; ok && nameSetting != "" {
		excludeNames := strings.Split(nameSetting.(string), ",")
		for _, name := range excludeNames {
			if name != "" {
				excludeSettings.ExcludeName = append(excludeSettings.ExcludeName, name)
			}
		}
	}

	// Extract host exclude settings
	if hostSetting, ok := settings["cluster.routing.allocation.exclude.host"]; ok && hostSetting != "" {
		excludeHosts := strings.Split(hostSetting.(string), ",")
		for _, host := range excludeHosts {
			if host != "" {
				excludeSettings.ExcludeHost = append(excludeSettings.ExcludeHost, host)
			}
		}
	}
}
