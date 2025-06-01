package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
)

// Client wraps the Elasticsearch client with custom methods
type Client struct {
	es *elasticsearch.Client
}

// New creates a new Elasticsearch client
func New(cfg *config.Config) (*Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Elasticsearch.Addresses,
		Username:  cfg.Elasticsearch.Username,
		Password:  cfg.Elasticsearch.Password,
	}

	// Configure TLS options
	// Create a custom transport for TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	// If insecure mode is enabled, skip certificate verification
	if cfg.Elasticsearch.Insecure {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	// If CA cert is provided, use it for verification (unless insecure mode is enabled)
	if cfg.Elasticsearch.CACert != "" && !cfg.Elasticsearch.Insecure {
		caCert, err := ioutil.ReadFile(cfg.Elasticsearch.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		transport.TLSClientConfig.RootCAs = caCertPool
	}

	// Set the transport if we've configured TLS options
	if cfg.Elasticsearch.Insecure || cfg.Elasticsearch.CACert != "" {
		esCfg.Transport = transport
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	return &Client{es: es}, nil
}

// Ping checks if the cluster is up
func (c *Client) Ping() (map[string]interface{}, error) {
	res, err := c.es.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return r, nil
}

// CatHealth returns cluster health information
func (c *Client) CatHealth() ([][]string, error) {
	req := esapi.CatHealthRequest{
		Format: "json",
		H:      []string{"status", "node.total", "node.data", "shards", "pri", "relo", "init", "unassign"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	var health []struct {
		Status    string `json:"status"`
		NodeTotal string `json:"node.total"`
		NodeData  string `json:"node.data"`
		Shards    string `json:"shards"`
		Pri       string `json:"pri"`
		Relo      string `json:"relo"`
		Init      string `json:"init"`
		Unassign  string `json:"unassign"`
	}

	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(health) == 0 {
		return nil, fmt.Errorf("no health data returned")
	}

	h := health[0]
	return [][]string{
		{
			"Status", "Nodes", "Data Nodes", "Shards", "Primary", "Relocating", "Initializing", "Unassigned",
		},
		{
			h.Status, h.NodeTotal, h.NodeData, h.Shards, h.Pri, h.Relo, h.Init, h.Unassign,
		},
	}, nil
}
