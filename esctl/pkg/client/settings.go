package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetClusterSettings returns the current cluster settings
func (c *Client) GetClusterSettings(includeDefaults bool) (map[string]map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Cluster.GetSettings(
		c.es.Cluster.GetSettings.WithContext(ctx),
		c.es.Cluster.GetSettings.WithFlatSettings(true),
		c.es.Cluster.GetSettings.WithIncludeDefaults(includeDefaults),
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

	return settings, nil
}

// UpdateClusterSettings updates cluster settings
func (c *Client) UpdateClusterSettings(settingType string, settings map[string]interface{}) error {
	// Validate setting type
	if settingType != "transient" && settingType != "persistent" {
		return fmt.Errorf("invalid setting type: %s. Must be either 'transient' or 'persistent'", settingType)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request body
	body := map[string]interface{}{
		settingType: settings,
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

// ResetClusterSetting resets a cluster setting to its default value
func (c *Client) ResetClusterSetting(settingType, settingName string) error {
	// Validate setting type
	if settingType != "transient" && settingType != "persistent" {
		return fmt.Errorf("invalid setting type: %s. Must be either 'transient' or 'persistent'", settingType)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request body to reset the setting (set to null)
	body := map[string]interface{}{
		settingType: map[string]interface{}{
			settingName: nil,
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
		return fmt.Errorf("error resetting cluster setting: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// GetSettingValue returns the value of a specific setting
func (c *Client) GetSettingValue(settingName string, includeDefaults bool) (interface{}, string, error) {
	// Get all settings
	settings, err := c.GetClusterSettings(includeDefaults)
	if err != nil {
		return nil, "", err
	}

	// Check transient settings first
	if transient, ok := settings["transient"]; ok {
		if value, ok := transient[settingName]; ok {
			return value, "transient", nil
		}
	}

	// Check persistent settings
	if persistent, ok := settings["persistent"]; ok {
		if value, ok := persistent[settingName]; ok {
			return value, "persistent", nil
		}
	}

	// Check default settings
	if includeDefaults {
		if defaults, ok := settings["defaults"]; ok {
			if value, ok := defaults[settingName]; ok {
				return value, "default", nil
			}
		}
	}

	return nil, "", fmt.Errorf("setting not found: %s", settingName)
}
