package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
)

// FleetClient extends KibanaClient with Fleet-specific methods
type FleetClient struct {
	*KibanaClient
}

// AgentPolicy represents a Fleet agent policy
type AgentPolicy struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Namespace        string   `json:"namespace"`
	MonitoringEnabled []string `json:"monitoring_enabled,omitempty"`
	Status           string   `json:"status,omitempty"`
	IsManaged        bool     `json:"is_managed,omitempty"`
	Revision         int      `json:"revision,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	UpdatedBy        string   `json:"updated_by,omitempty"`
	SchemaVersion    string   `json:"schema_version,omitempty"`
}

// AgentPolicyResponse represents the response from the Fleet API for agent policies
type AgentPolicyResponse struct {
	Item  AgentPolicy   `json:"item,omitempty"`
	Items []AgentPolicy `json:"items,omitempty"`
	Page  int           `json:"page,omitempty"`
	PerPage int         `json:"perPage,omitempty"`
	Total int           `json:"total,omitempty"`
}

// EnrollmentToken represents a Fleet enrollment token
type EnrollmentToken struct {
	ID        string `json:"id"`
	Active    bool   `json:"active"`
	APIKey    string `json:"api_key"`
	APIKeyID  string `json:"api_key_id"`
	CreatedAt string `json:"created_at"`
	Name      string `json:"name"`
	PolicyID  string `json:"policy_id"`
}

// EnrollmentTokenResponse represents the response from the Fleet API for enrollment tokens
type EnrollmentTokenResponse struct {
	Items   []EnrollmentToken `json:"items"`
	Page    int               `json:"page"`
	PerPage int               `json:"perPage"`
	Total   int               `json:"total"`
}

// PackagePolicy represents a Fleet package policy (integration)
type PackagePolicy struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	PolicyID    string                 `json:"policy_id"`
	Package     PackagePolicyPackage   `json:"package"`
	Inputs      map[string]interface{} `json:"inputs"`
}

// PackagePolicyPackage represents the package information in a package policy
type PackagePolicyPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// PackagePolicyResponse represents the response from the Fleet API for package policies
type PackagePolicyResponse struct {
	Item  PackagePolicy   `json:"item,omitempty"`
	Items []PackagePolicy `json:"items,omitempty"`
	Page  int             `json:"page,omitempty"`
	PerPage int           `json:"perPage,omitempty"`
	Total int             `json:"total,omitempty"`
}

// NewFleet creates a new Fleet client
func NewFleet(cfg *config.Config) (*FleetClient, error) {
	kibanaClient, err := NewKibana(cfg)
	if err != nil {
		return nil, err
	}

	return &FleetClient{
		KibanaClient: kibanaClient,
	}, nil
}

// GetAgentPolicies retrieves all agent policies from Fleet
func (c *FleetClient) GetAgentPolicies() ([]AgentPolicy, error) {
	// Create request to Fleet API
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/fleet/agent_policies", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add basic auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Add required headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var result AgentPolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Items, nil
}

// CreateAgentPolicy creates a new agent policy in Fleet
func (c *FleetClient) CreateAgentPolicy(policy AgentPolicy) (*AgentPolicy, error) {
	// Marshal policy to JSON
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("marshaling policy: %w", err)
	}

	// Create request to Fleet API
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/fleet/agent_policies", c.baseURL), bytes.NewBuffer(policyJSON))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add basic auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Add required headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var result AgentPolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result.Item, nil
}

// GetEnrollmentTokens retrieves all enrollment tokens from Fleet
func (c *FleetClient) GetEnrollmentTokens() ([]EnrollmentToken, error) {
	// Create request to Fleet API
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/fleet/enrollment_api_keys", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add basic auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Add required headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var result EnrollmentTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Items, nil
}

// GetPackagePolicies retrieves all package policies from Fleet
func (c *FleetClient) GetPackagePolicies() ([]PackagePolicy, error) {
	// Create request to Fleet API
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/fleet/package_policies", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add basic auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Add required headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var result PackagePolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Items, nil
}

// CreatePackagePolicy creates a new package policy in Fleet
func (c *FleetClient) CreatePackagePolicy(policy PackagePolicy) (*PackagePolicy, error) {
	// Marshal policy to JSON
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("marshaling policy: %w", err)
	}

	// Create request to Fleet API
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/fleet/package_policies", c.baseURL), bytes.NewBuffer(policyJSON))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add basic auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Add required headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var result PackagePolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result.Item, nil
}

// GetAgentPoliciesFormatted returns agent policies formatted for display
func (c *FleetClient) GetAgentPoliciesFormatted() ([]string, [][]string, error) {
	policies, err := c.GetAgentPolicies()
	if err != nil {
		return nil, nil, err
	}

	headers := []string{"ID", "Name", "Namespace", "Status", "Revision", "Updated At"}
	rows := make([][]string, len(policies))

	for i, policy := range policies {
		rows[i] = []string{
			policy.ID,
			policy.Name,
			policy.Namespace,
			policy.Status,
			fmt.Sprintf("%d", policy.Revision),
			policy.UpdatedAt,
		}
	}

	return headers, rows, nil
}

// GetEnrollmentTokensFormatted returns enrollment tokens formatted for display
func (c *FleetClient) GetEnrollmentTokensFormatted() ([]string, [][]string, error) {
	tokens, err := c.GetEnrollmentTokens()
	if err != nil {
		return nil, nil, err
	}

	headers := []string{"ID", "Name", "Policy ID", "Active", "Created At"}
	rows := make([][]string, len(tokens))

	for i, token := range tokens {
		active := "No"
		if token.Active {
			active = "Yes"
		}

		rows[i] = []string{
			token.ID,
			token.Name,
			token.PolicyID,
			active,
			token.CreatedAt,
		}
	}

	return headers, rows, nil
}

// GetPackagePoliciesFormatted returns package policies formatted for display
func (c *FleetClient) GetPackagePoliciesFormatted() ([]string, [][]string, error) {
	policies, err := c.GetPackagePolicies()
	if err != nil {
		return nil, nil, err
	}

	headers := []string{"ID", "Name", "Package", "Version", "Policy ID"}
	rows := make([][]string, len(policies))

	for i, policy := range policies {
		rows[i] = []string{
			policy.ID,
			policy.Name,
			policy.Package.Name,
			policy.Package.Version,
			policy.PolicyID,
		}
	}

	return headers, rows, nil
}
