package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
)

// KibanaClient wraps HTTP client with Kibana-specific methods
type KibanaClient struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
}

// NewKibana creates a new Kibana client
func NewKibana(cfg *config.Config) (*KibanaClient, error) {
	if len(cfg.Kibana.Addresses) == 0 {
		return nil, fmt.Errorf("no Kibana addresses provided")
	}

	// Configure TLS
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	// If insecure mode is enabled, skip certificate verification
	if cfg.Kibana.Insecure {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	// If CA cert is provided, use it for verification (unless insecure mode is enabled)
	if cfg.Kibana.CACert != "" && !cfg.Kibana.Insecure {
		caCert, err := ioutil.ReadFile(cfg.Kibana.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		transport.TLSClientConfig.RootCAs = caCertPool
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Set the transport if we've configured TLS options
	if cfg.Kibana.Insecure || cfg.Kibana.CACert != "" {
		httpClient.Transport = transport
	}

	return &KibanaClient{
		httpClient: httpClient,
		baseURL:    cfg.Kibana.Addresses[0],
		username:   cfg.Kibana.Username,
		password:   cfg.Kibana.Password,
	}, nil
}

// Ping checks if Kibana is up and running
func (c *KibanaClient) Ping() (map[string]interface{}, error) {
	// Create request to Kibana status API
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/status", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add basic auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

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
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result, nil
}

// GetStatus returns formatted status information for display
func (c *KibanaClient) GetStatus() ([][]string, error) {
	status, err := c.Ping()
	if err != nil {
		return nil, err
	}

	// Extract relevant information
	version, _ := status["version"].(map[string]interface{})
	versionNumber := "unknown"
	if version != nil {
		versionNumber, _ = version["number"].(string)
	}

	// Extract status
	statusInfo, _ := status["status"].(map[string]interface{})
	overall := "unknown"
	if statusInfo != nil {
		overall, _ = statusInfo["overall"].(map[string]interface{})["state"].(string)
	}

	// Format as table rows
	return [][]string{
		{
			"Status", "Version",
		},
		{
			overall, versionNumber,
		},
	}, nil
}
