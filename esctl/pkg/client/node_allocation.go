package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// NodeAllocation represents disk allocation information for a node
type NodeAllocation struct {
	Name         string
	IP           string
	ID           string
	Role         string
	Master       string
	Version      string
	Jdk          string
	DiskTotal    string
	DiskUsed     string
	DiskAvail    string
	DiskPercent  string
	DiskIndices  string
	Shards       string
}

// GetNodeAllocations returns disk allocation information for all nodes in the cluster
func (c *Client) GetNodeAllocations() ([]NodeAllocation, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get node stats for disk information
	statsRes, err := c.es.Nodes.Stats(
		c.es.Nodes.Stats.WithContext(ctx),
		c.es.Nodes.Stats.WithMetric("fs,indices"),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting node stats: %w", err)
	}
	defer statsRes.Body.Close()

	if statsRes.IsError() {
		return nil, fmt.Errorf("error response from stats: %s", statsRes.String())
	}

	// Parse stats response
	var statsResponse map[string]interface{}
	if err := json.NewDecoder(statsRes.Body).Decode(&statsResponse); err != nil {
		return nil, fmt.Errorf("error parsing stats response: %w", err)
	}

	// Get node info for role, version, etc.
	infoRes, err := c.es.Nodes.Info(
		c.es.Nodes.Info.WithContext(ctx),
		c.es.Nodes.Info.WithMetric("jvm"),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting node info: %w", err)
	}
	defer infoRes.Body.Close()

	if infoRes.IsError() {
		return nil, fmt.Errorf("error response from info: %s", infoRes.String())
	}

	// Parse info response
	var infoResponse map[string]interface{}
	if err := json.NewDecoder(infoRes.Body).Decode(&infoResponse); err != nil {
		return nil, fmt.Errorf("error parsing info response: %w", err)
	}

	// Get cluster state for master node information
	stateRes, err := c.es.Cluster.State(
		c.es.Cluster.State.WithContext(ctx),
		c.es.Cluster.State.WithMetric("master_node"),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting cluster state: %w", err)
	}
	defer stateRes.Body.Close()

	if stateRes.IsError() {
		return nil, fmt.Errorf("error response from cluster state: %s", stateRes.String())
	}

	// Parse state response
	var stateResponse map[string]interface{}
	if err := json.NewDecoder(stateRes.Body).Decode(&stateResponse); err != nil {
		return nil, fmt.Errorf("error parsing cluster state response: %w", err)
	}

	// Extract master node ID
	masterNodeID, _ := stateResponse["master_node"].(string)

	// Extract node stats
	statsNodes, ok := statsResponse["nodes"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected stats response format: nodes data not found")
	}

	// Extract node info
	infoNodes, ok := infoResponse["nodes"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected info response format: nodes data not found")
	}

	var nodeAllocations []NodeAllocation
	for nodeID, statsNodeData := range statsNodes {
		statsNode, ok := statsNodeData.(map[string]interface{})
		if !ok {
			continue
		}

		// Get node info data
		infoNodeData, ok := infoNodes[nodeID]
		if !ok {
			continue
		}
		infoNode, ok := infoNodeData.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract basic node information
		name, _ := statsNode["name"].(string)
		ip, _ := infoNode["ip"].(string)

		// Determine node role
		var role string
		roles, ok := infoNode["roles"].([]interface{})
		if ok && len(roles) > 0 {
			role = fmt.Sprintf("%v", roles[0])
		} else {
			role = "-"
		}

		// Determine if node is master
		master := " "
		if nodeID == masterNodeID {
			master = "*"
		}

		// Extract version info
		version := "-"
		jdk := "-"
		if versionInfo, ok := infoNode["version"].(string); ok {
			version = versionInfo
		}
		if jvmInfo, ok := infoNode["jvm"].(map[string]interface{}); ok {
			if jvmVersion, ok := jvmInfo["version"].(string); ok {
				jdk = jvmVersion
			}
		}

		// Extract disk information
		diskTotal := "-"
		diskUsed := "-"
		diskAvail := "-"
		diskPercent := "-"
		diskIndices := "-"
		shards := "-"

		if fs, ok := statsNode["fs"].(map[string]interface{}); ok {
			if total, ok := fs["total"].(map[string]interface{}); ok {
				// Get total bytes first since we need it for calculations
				var totalBytesVal float64
				if val, ok := total["total_in_bytes"].(float64); ok {
					totalBytesVal = val
					diskTotal = ByteCountSI(int64(totalBytesVal))
				}
				
				if freeBytes, ok := total["free_in_bytes"].(float64); ok {
					diskAvail = ByteCountSI(int64(freeBytes))
				}
				
				if availableBytes, ok := total["available_in_bytes"].(float64); ok && totalBytesVal > 0 {
					diskUsed = ByteCountSI(int64(totalBytesVal) - int64(availableBytes))
					usedPercent := (totalBytesVal - availableBytes) / totalBytesVal * 100
					diskPercent = fmt.Sprintf("%.1f%%", usedPercent)
				}
			}
		}

		// Extract indices information
		if indices, ok := statsNode["indices"].(map[string]interface{}); ok {
			if store, ok := indices["store"].(map[string]interface{}); ok {
				if sizeBytes, ok := store["size_in_bytes"].(float64); ok {
					diskIndices = ByteCountSI(int64(sizeBytes))
				}
			}
			if shardStats, ok := indices["shards_stats"].(map[string]interface{}); ok {
				if count, ok := shardStats["count"].(float64); ok {
					shards = fmt.Sprintf("%d", int(count))
				}
			}
		}

		nodeAllocations = append(nodeAllocations, NodeAllocation{
			Name:        name,
			IP:          ip,
			ID:          nodeID,
			Role:        role,
			Master:      master,
			Version:     version,
			Jdk:         jdk,
			DiskTotal:   diskTotal,
			DiskUsed:    diskUsed,
			DiskAvail:   diskAvail,
			DiskPercent: diskPercent,
			DiskIndices: diskIndices,
			Shards:      shards,
		})
	}

	return nodeAllocations, nil
}
