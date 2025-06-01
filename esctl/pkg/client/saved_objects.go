package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"bytes"
)

// SavedObject represents a Kibana saved object
type SavedObject struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	Attributes       map[string]interface{} `json:"attributes"`
	References       []ObjectReference      `json:"references"`
	UpdatedAt        string                 `json:"updated_at,omitempty"`
	Version          string                 `json:"version,omitempty"`
	NamespaceType    string                 `json:"namespaceType,omitempty"`
	Score            float64                `json:"score,omitempty"`
	Meta             map[string]interface{} `json:"meta,omitempty"`
	OriginID         string                 `json:"originId,omitempty"`
	MigrationVersion map[string]string      `json:"migrationVersion,omitempty"`
}

// ObjectReference represents a reference to another saved object
type ObjectReference struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// SavedObjectSearchResponse represents the response from the saved objects API
type SavedObjectSearchResponse struct {
	Page         int           `json:"page"`
	PerPage      int           `json:"per_page"`
	Total        int           `json:"total"`
	SavedObjects []SavedObject `json:"saved_objects"`
}

// SearchSavedObjects searches for saved objects in Kibana
func (c *KibanaClient) SearchSavedObjects(searchTerm string, types []string, includeDependencies bool, perPage, page int) (*SavedObjectSearchResponse, error) {
	// Build the query parameters
	params := url.Values{}
	if searchTerm != "" {
		params.Add("search", searchTerm)
	}
	if len(types) > 0 {
		params.Add("type", strings.Join(types, ","))
	}
	if includeDependencies {
		params.Add("includeDependencies", "true")
	}
	if perPage > 0 {
		params.Add("per_page", fmt.Sprintf("%d", perPage))
	}
	if page > 0 {
		params.Add("page", fmt.Sprintf("%d", page))
	}

	// Build the request URL
	requestURL := fmt.Sprintf("%s/api/saved_objects/_find?%s", c.baseURL, params.Encode())

	// Create the request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add authentication if configured
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if errMsg, ok := errorResp["message"].(string); ok {
				return nil, fmt.Errorf("error from Kibana API: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("error from Kibana API: %s", resp.Status)
	}

	// Parse the response
	var response SavedObjectSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &response, nil
}

// GetSavedObject retrieves a specific saved object by ID and type
func (c *KibanaClient) GetSavedObject(id, objectType string, includeDependencies bool) (*SavedObject, error) {
	// Build the query parameters
	params := url.Values{}
	if includeDependencies {
		params.Add("includeDependencies", "true")
	}

	// Build the request URL
	requestURL := fmt.Sprintf("%s/api/saved_objects/%s/%s", c.baseURL, objectType, id)
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	// Create the request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add authentication if configured
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if errMsg, ok := errorResp["message"].(string); ok {
				return nil, fmt.Errorf("error from Kibana API: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("error from Kibana API: %s", resp.Status)
	}

	// Parse the response
	var response SavedObject
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &response, nil
}

// GetSavedObjectsTypes returns a list of all available saved object types
func (c *KibanaClient) GetSavedObjectsTypes() ([]string, error) {
	// Build the request URL
	requestURL := fmt.Sprintf("%s/api/saved_objects/_types", c.baseURL)

	// Create the request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add authentication if configured
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if errMsg, ok := errorResp["message"].(string); ok {
				return nil, fmt.Errorf("error from Kibana API: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("error from Kibana API: %s", resp.Status)
	}

	// Parse the response
	var types []string
	if err := json.NewDecoder(resp.Body).Decode(&types); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return types, nil
}

// ExportSavedObject exports a saved object by ID and type
// If includeDependencies is true, it will also export objects that the specified object depends on
// Returns the exported objects in NDJSON format
func (c *KibanaClient) ExportSavedObject(id, objectType string, includeDependencies bool) ([]byte, error) {
	// Build the request URL
	requestURL := fmt.Sprintf("%s/api/saved_objects/_export", c.baseURL)

	// Build the request body
	objects := []map[string]string{
		{
			"type": objectType,
			"id":   id,
		},
	}

	requestBody := map[string]interface{}{
		"objects":             objects,
		"includeReferencesDeep": includeDependencies,
	}

	// Convert request body to JSON
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", "application/json")
	
	// Add authentication if configured
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if errMsg, ok := errorResp["message"].(string); ok {
				return nil, fmt.Errorf("error from Kibana API: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("error from Kibana API: %s", resp.Status)
	}

	// Read the response body into a buffer
	respBody := bytes.NewBuffer(nil)
	_, err = respBody.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return respBody.Bytes(), nil
}
