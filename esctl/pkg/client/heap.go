package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// NodeJVMStats represents the JVM stats for a node
type NodeJVMStats struct {
	Name                   string
	Role                   string
	ID                     string
	JVMStats               JVMStats
}

// JVMStats contains the JVM heap and non-heap statistics
type JVMStats struct {
	HeapUsedBytes          int64
	HeapUsedPercentage     int
	HeapMaxBytes           int64
	NonHeapCommittedBytes  int64
	NonHeapUsedBytes       int64
}

// GetNodeJVMStats returns the JVM stats for all nodes in the cluster
func (c *Client) GetNodeJVMStats() ([]NodeJVMStats, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute request
	res, err := c.es.Nodes.Stats(
		c.es.Nodes.Stats.WithContext(ctx),
		c.es.Nodes.Stats.WithMetric("jvm"),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting node JVM stats: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Extract node stats
	nodesData, ok := response["nodes"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: nodes data not found")
	}

	var nodeStats []NodeJVMStats
	for nodeID, nodeData := range nodesData {
		nodeInfo, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract node name and role
		name, _ := nodeInfo["name"].(string)
		
		// Determine node role
		var role string
		roles, ok := nodeInfo["roles"].([]interface{})
		if ok && len(roles) > 0 {
			role = fmt.Sprintf("%v", roles[0])
		} else {
			role = "unknown"
		}

		// Extract JVM stats
		jvmData, ok := nodeInfo["jvm"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract heap stats
		memData, ok := jvmData["mem"].(map[string]interface{})
		if !ok {
			continue
		}

		heapUsedBytes, _ := memData["heap_used_in_bytes"].(float64)
		heapMaxBytes, _ := memData["heap_max_in_bytes"].(float64)
		heapUsedPercent, _ := memData["heap_used_percent"].(float64)
		nonHeapCommittedBytes, _ := memData["non_heap_committed_in_bytes"].(float64)
		nonHeapUsedBytes, _ := memData["non_heap_used_in_bytes"].(float64)

		nodeStats = append(nodeStats, NodeJVMStats{
			Name: name,
			Role: role,
			ID:   nodeID,
			JVMStats: JVMStats{
				HeapUsedBytes:         int64(heapUsedBytes),
				HeapMaxBytes:          int64(heapMaxBytes),
				HeapUsedPercentage:    int(heapUsedPercent),
				NonHeapCommittedBytes: int64(nonHeapCommittedBytes),
				NonHeapUsedBytes:      int64(nonHeapUsedBytes),
			},
		})
	}

	return nodeStats, nil
}

// ByteCountSI converts bytes to a human-readable string in SI format
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
