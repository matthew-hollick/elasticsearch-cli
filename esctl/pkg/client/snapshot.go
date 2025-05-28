package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SnapshotInfo represents information about a snapshot
type SnapshotInfo struct {
	Snapshot          string `json:"snapshot"`
	UUID              string `json:"uuid"`
	VersionID         int    `json:"version_id"`
	Version           string `json:"version"`
	Indices           []string `json:"indices"`
	IncludeGlobalState bool   `json:"include_global_state"`
	State             string `json:"state"`
	StartTime         string `json:"start_time"`
	StartTimeInMillis int64  `json:"start_time_in_millis"`
	EndTime           string `json:"end_time"`
	EndTimeInMillis   int64  `json:"end_time_in_millis"`
	DurationInMillis  int64  `json:"duration_in_millis"`
	Failures          []interface{} `json:"failures"`
	Shards            map[string]int `json:"shards"`
}

// RepositoryInfo represents information about a snapshot repository
type RepositoryInfo struct {
	Type     string                 `json:"type"`
	Settings map[string]interface{} `json:"settings"`
}

// GetRepositories returns all snapshot repositories
func (c *Client) GetRepositories() (map[string]RepositoryInfo, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Snapshot.GetRepository(
		c.es.Snapshot.GetRepository.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting repositories: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var repositories map[string]RepositoryInfo
	if err := json.NewDecoder(res.Body).Decode(&repositories); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return repositories, nil
}

// CreateRepository creates a new snapshot repository
func (c *Client) CreateRepository(name string, repoType string, settings map[string]interface{}, verify bool) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare the request body
	body := map[string]interface{}{
		"type": repoType,
		"settings": settings,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("error encoding request body: %w", err)
	}

	// Execute request
	res, err := c.es.Snapshot.CreateRepository(
		name,
		&buf,
		c.es.Snapshot.CreateRepository.WithContext(ctx),
		c.es.Snapshot.CreateRepository.WithVerify(verify),
	)
	if err != nil {
		return fmt.Errorf("error creating repository: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// DeleteRepository deletes a snapshot repository
func (c *Client) DeleteRepository(name string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Snapshot.DeleteRepository(
		[]string{name},
		c.es.Snapshot.DeleteRepository.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// GetSnapshots returns all snapshots in a repository
func (c *Client) GetSnapshots(repository string) ([]SnapshotInfo, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Snapshot.Get(
		repository,
		[]string{"_all"},
		c.es.Snapshot.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting snapshots: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var response struct {
		Snapshots []SnapshotInfo `json:"snapshots"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return response.Snapshots, nil
}

// CreateSnapshot creates a new snapshot
func (c *Client) CreateSnapshot(repository, name string, indices []string, includeGlobalState bool, waitForCompletion bool) (*SnapshotInfo, error) {
	// Create context with timeout (longer for snapshot creation)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Prepare the request body
	body := map[string]interface{}{
		"indices": strings.Join(indices, ","),
		"include_global_state": includeGlobalState,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("error encoding request body: %w", err)
	}

	// Execute request
	res, err := c.es.Snapshot.Create(
		repository,
		name,
		c.es.Snapshot.Create.WithBody(&buf),
		c.es.Snapshot.Create.WithContext(ctx),
		c.es.Snapshot.Create.WithWaitForCompletion(waitForCompletion),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating snapshot: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// If wait for completion is false, just return nil
	if !waitForCompletion {
		return nil, nil
	}

	// Parse response
	var snapshot SnapshotInfo
	if err := json.NewDecoder(res.Body).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &snapshot, nil
}

// VerifyRepository verifies that a repository is properly configured on all nodes
func (c *Client) VerifyRepository(name string) (bool, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute request - first parameter is the repository name
	res, err := c.es.Snapshot.VerifyRepository(
		name,
		c.es.Snapshot.VerifyRepository.WithContext(ctx),
	)
	if err != nil {
		return false, fmt.Errorf("error verifying repository: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("error parsing response: %w", err)
	}

	// Check if nodes responded successfully
	nodesInfo, ok := response["nodes"].(map[string]interface{})
	if !ok || len(nodesInfo) == 0 {
		return false, fmt.Errorf("unexpected response format or no nodes responded")
	}

	// If we got here without errors, the repository is verified
	return true, nil
}

// DeleteSnapshot deletes a snapshot
func (c *Client) DeleteSnapshot(repository, name string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Snapshot.Delete(
		repository,
		[]string{name},
		c.es.Snapshot.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error deleting snapshot: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}

// RestoreSnapshot restores a snapshot
func (c *Client) RestoreSnapshot(repository, name string, indices []string, renamePattern, renameReplacement string, waitForCompletion bool) error {
	// Create context with timeout (longer for restore)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Prepare the request body
	body := map[string]interface{}{}
	
	if len(indices) > 0 {
		body["indices"] = strings.Join(indices, ",")
	}
	
	if renamePattern != "" && renameReplacement != "" {
		body["rename_pattern"] = renamePattern
		body["rename_replacement"] = renameReplacement
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("error encoding request body: %w", err)
	}

	// Execute request
	res, err := c.es.Snapshot.Restore(
		repository,
		name,
		c.es.Snapshot.Restore.WithBody(&buf),
		c.es.Snapshot.Restore.WithContext(ctx),
		c.es.Snapshot.Restore.WithWaitForCompletion(waitForCompletion),
	)
	if err != nil {
		return fmt.Errorf("error restoring snapshot: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response: %s", res.String())
	}

	return nil
}
