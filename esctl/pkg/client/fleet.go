package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
)

// FleetClient extends KibanaClient with Fleet-specific methods
type FleetClient struct {
	*KibanaClient
}

// AgentPolicy represents a Fleet agent policy
type AgentPolicy struct {
	ID                string   `json:"id,omitempty"`
	Name              string   `json:"name"`
	Namespace         string   `json:"namespace"`
	Description       string   `json:"description,omitempty"`
	MonitoringEnabled []string `json:"monitoring_enabled,omitempty"`
	Status            string   `json:"status,omitempty"`
	Revision          int      `json:"revision,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
	UpdatedBy         string   `json:"updated_by,omitempty"`
	IsDefault         bool     `json:"is_default,omitempty"`
	IsManaged         bool     `json:"is_managed,omitempty"`
	IsDeletable       bool     `json:"is_deletable,omitempty"`
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

// Agent represents a Fleet agent
type Agent struct {
	ID                 string                 `json:"id"`
	PolicyID           string                 `json:"policy_id"`
	Type               string                 `json:"type"`
	Active             bool                   `json:"active"`
	Status             string                 `json:"status"`
	LastCheckin        string                 `json:"last_checkin"`
	EnrolledAt         string                 `json:"enrolled_at"`
	UnenrolledAt       string                 `json:"unenrolled_at,omitempty"`
	UpgradedAt         string                 `json:"upgraded_at,omitempty"`
	UpgradeStatus      string                 `json:"upgrade_status,omitempty"`
	LocalMetadata      map[string]interface{} `json:"local_metadata,omitempty"`
	UserMetadata       map[string]interface{} `json:"user_metadata,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	AccessAPIKeyID     string                 `json:"access_api_key_id,omitempty"`
}

// AgentResponse represents the response from the Fleet API for agents
type AgentResponse struct {
	Item   Agent    `json:"item,omitempty"`
	Items  []Agent  `json:"items"`
	Page   int      `json:"page"`
	PerPage int     `json:"perPage"`
	Total  int      `json:"total"`
}

// AgentAction represents an action to be performed on an agent
type AgentAction struct {
	Type        string                 `json:"type"`
	SubType     string                 `json:"subType,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	ExpireAfter string                 `json:"expireAfter,omitempty"`
}

// AgentActionResponse represents the response from creating an agent action
type AgentActionResponse struct {
	Item struct {
		ID string `json:"id"`
	} `json:"item"`
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

// PolicyIDError represents an error related to policy ID validation
type PolicyIDError struct {
	ID     string
	Reason string
}

// Error implements the error interface for PolicyIDError
func (e *PolicyIDError) Error() string {
	return fmt.Sprintf("invalid policy ID '%s': %s", e.ID, e.Reason)
}

// ValidatePolicyID checks if a policy ID matches required format rules
// Policy IDs must be lowercase alphanumeric with hyphens and underscores, 1-36 chars
func ValidatePolicyID(id string) error {
	// Empty ID is valid (system will generate one)
	if id == "" {
		return nil
	}

	// Check length
	if len(id) > 36 {
		return &PolicyIDError{ID: id, Reason: "exceeds maximum length of 36 characters"}
	}

	// Check pattern using regex
	validPattern := regexp.MustCompile("^[a-z0-9][a-z0-9_-]*$")
	if !validPattern.MatchString(id) {
		return &PolicyIDError{
			ID:     id,
			Reason: "must contain only lowercase letters, numbers, hyphens, and underscores, and start with a letter or number",
		}
	}

	return nil
}

// CheckPolicyIDExists checks if a policy ID already exists
func (c *FleetClient) CheckPolicyIDExists(id string) (bool, error) {
	// Get all agent policies
	policies, err := c.GetAgentPolicies()
	if err != nil {
		return false, fmt.Errorf("error fetching existing policies: %w", err)
	}

	// Check if ID exists
	for _, policy := range policies {
		if policy.ID == id {
			return true, nil
		}
	}

	return false, nil
}

// CreateAgentPolicy creates a new agent policy in Fleet
func (c *FleetClient) CreateAgentPolicy(policy AgentPolicy) (*AgentPolicy, error) {
	// Validate the policy ID if provided
	if policy.ID != "" {
		// Format validation
		if err := ValidatePolicyID(policy.ID); err != nil {
			return nil, err
		}

		// Uniqueness validation
		exists, err := c.CheckPolicyIDExists(policy.ID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, &PolicyIDError{
				ID:     policy.ID,
				Reason: "already exists",
			}
		}
	}

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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
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

// CheckPackagePolicyIDExists checks if a package policy ID already exists
func (c *FleetClient) CheckPackagePolicyIDExists(id string) (bool, error) {
	// Get all package policies
	policies, err := c.GetPackagePolicies()
	if err != nil {
		return false, fmt.Errorf("error fetching existing package policies: %w", err)
	}

	// Check if ID exists
	for _, policy := range policies {
		if policy.ID == id {
			return true, nil
		}
	}

	return false, nil
}

// CreatePackagePolicy creates a new package policy in Fleet
func (c *FleetClient) CreatePackagePolicy(policy PackagePolicy) (*PackagePolicy, error) {
	// Validate the policy ID if provided
	if policy.ID != "" {
		// Format validation - uses same rules as agent policies
		if err := ValidatePolicyID(policy.ID); err != nil {
			return nil, err
		}

		// Uniqueness validation
		exists, err := c.CheckPackagePolicyIDExists(policy.ID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, &PolicyIDError{
				ID:     policy.ID,
				Reason: "already exists",
			}
		}
	}

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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
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
	// Get package policies
	policies, err := c.GetPackagePolicies()
	if err != nil {
		return nil, nil, err
	}

	// Define headers
	headers := []string{"ID", "Name", "Description", "Policy ID", "Package", "Version"}

	// Format rows
	rows := make([][]string, 0, len(policies))
	for _, policy := range policies {
		row := []string{
			policy.ID,
			policy.Name,
			policy.Description,
			policy.PolicyID,
			policy.Package.Name,
			policy.Package.Version,
		}
		rows = append(rows, row)
	}

	return headers, rows, nil
}

// GetAgents retrieves agents with optional filtering
func (c *FleetClient) GetAgents(kuery string, page int, perPage int) ([]Agent, int, error) {
	urlPath := fmt.Sprintf("%s/api/fleet/agents", c.baseURL)
	
	// Add query parameters if provided
	params := make([]string, 0)
	if kuery != "" {
		params = append(params, fmt.Sprintf("kuery=%s", url.QueryEscape(kuery)))
	}
	if page > 0 {
		params = append(params, fmt.Sprintf("page=%d", page))
	}
	if perPage > 0 {
		params = append(params, fmt.Sprintf("perPage=%d", perPage))
	}
	
	// Append parameters to URL
	if len(params) > 0 {
		urlPath = fmt.Sprintf("%s?%s", urlPath, strings.Join(params, "&"))
	}
	
	// Create request
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Parse response
	var result AgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}
	
	return result.Items, result.Total, nil
}

// GetAgent retrieves a specific agent by ID
func (c *FleetClient) GetAgent(id string) (*Agent, error) {
	// Create request to Fleet API
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/fleet/agents/%s", c.baseURL, id), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
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
	var result struct {
		Item Agent `json:"item"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	
	return &result.Item, nil
}

// UpdateAgent updates an agent's metadata or tags
func (c *FleetClient) UpdateAgent(id string, userMeta map[string]interface{}, tags []string) error {
	// Prepare payload
	payload := map[string]interface{}{}
	if userMeta != nil {
		payload["user_metadata"] = userMeta
	}
	if tags != nil {
		payload["tags"] = tags
	}
	
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}
	
	// Create request
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/fleet/agents/%s", c.baseURL, id), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// DeleteAgent unenrolls an agent
func (c *FleetClient) DeleteAgent(id string, force bool) error {
	// Create URL with force parameter if needed
	urlPath := fmt.Sprintf("%s/api/fleet/agents/%s", c.baseURL, id)
	if force {
		urlPath += "?force=true"
	}
	
	// Create request
	req, err := http.NewRequest("DELETE", urlPath, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// ReassignAgent assigns an agent to a different policy
func (c *FleetClient) ReassignAgent(agentID string, policyID string) error {
	// Prepare payload
	payload := map[string]interface{}{
		"policy_id": policyID,
	}
	
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}
	
	// Create request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/fleet/agents/%s/reassign", c.baseURL, agentID), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// GetAgentsFormatted returns agents formatted for display
func (c *FleetClient) GetAgentsFormatted(kuery string) ([]string, [][]string, error) {
	// Get agents with potential filtering
	agents, _, err := c.GetAgents(kuery, 0, 0)
	if err != nil {
		return nil, nil, err
	}
	
	// Define headers
	headers := []string{"ID", "Status", "Policy ID", "Type", "Last Check-in", "Tags", "Enrolled At"}
	
	// Format rows
	rows := make([][]string, 0, len(agents))
	for _, agent := range agents {
		// Format tags as comma-separated list
		tags := strings.Join(agent.Tags, ", ")
		
		row := []string{
			agent.ID,
			agent.Status,
			agent.PolicyID,
			agent.Type,
			agent.LastCheckin,
			tags,
			agent.EnrolledAt,
		}
		rows = append(rows, row)
	}
	
	return headers, rows, nil
}

// UpdateAgentPolicy updates an existing agent policy
func (c *FleetClient) UpdateAgentPolicy(id string, policy AgentPolicy) (*AgentPolicy, error) {
	// Marshal policy to JSON
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("marshaling policy: %w", err)
	}
	
	// Create request
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/fleet/agent_policies/%s", c.baseURL, id), bytes.NewBuffer(policyJSON))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
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

// DeleteAgentPolicy deletes an agent policy
func (c *FleetClient) DeleteAgentPolicy(id string, force bool) error {
	// If force is true, we need to first find and reassign any agents using this policy
	if force {
		// 1. Find default policy ID to reassign to
		defaultPolicyID, err := c.getDefaultPolicyID()
		if err != nil {
			return fmt.Errorf("finding default policy for reassignment: %w", err)
		}
		
		// 2. Find all agents assigned to this policy
		agents, _, err := c.GetAgents(fmt.Sprintf("policy_id:%s", id), 1, 1000) // Get up to 1000 agents on page 1
		if err != nil {
			return fmt.Errorf("finding agents assigned to policy %s: %w", id, err)
		}
		
		// 3. Reassign all agents to the default policy
		for _, agent := range agents {
			if err := c.ReassignAgent(agent.ID, defaultPolicyID); err != nil {
				return fmt.Errorf("reassigning agent %s to default policy: %w", agent.ID, err)
			}
		}
	}

	// Create request to delete policy
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/fleet/agent_policies/%s", c.baseURL, id), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("policy deletion failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// getDefaultPolicyID finds the ID of the default agent policy
func (c *FleetClient) getDefaultPolicyID() (string, error) {
	// Get all agent policies
	policies, err := c.GetAgentPolicies()
	if err != nil {
		return "", err
	}
	
	// Find the default policy
	for _, policy := range policies {
		if policy.IsDefault {
			return policy.ID, nil
		}
	}
	
	// If no default policy found, return error
	return "", fmt.Errorf("no default agent policy found for reassignment")
}

// UpdatePackagePolicy updates an existing package policy
func (c *FleetClient) UpdatePackagePolicy(id string, policy PackagePolicy) (*PackagePolicy, error) {
	// Marshal policy to JSON
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("marshaling policy: %w", err)
	}
	
	// Create request
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/fleet/package_policies/%s", c.baseURL, id), bytes.NewBuffer(policyJSON))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
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

// DeletePackagePolicy deletes a package policy
func (c *FleetClient) DeletePackagePolicy(id string, force bool) error {
	// Create the URL with force parameter if needed
	urlPath := fmt.Sprintf("%s/api/fleet/package_policies/%s", c.baseURL, id)
	if force {
		urlPath = fmt.Sprintf("%s?force=true", urlPath)
	}
	
	// Create request
	req, err := http.NewRequest("DELETE", urlPath, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	// Add auth and headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "true")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("package policy deletion failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}
