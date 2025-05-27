package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ShardInfo represents information about a single shard
type ShardInfo struct {
	Index              string `json:"index"`
	Shard              string `json:"shard"`
	PrimaryOrReplica   string `json:"prirep"` // p = primary, r = replica
	State              string `json:"state"`
	Docs               string `json:"docs"`
	Store              string `json:"store"`
	IP                 string `json:"ip"`
	Node               string `json:"node"`
	UnassignedReason   string `json:"unassigned.reason,omitempty"`
	UnassignedAt       string `json:"unassigned.at,omitempty"`
	UnassignedDetails  string `json:"unassigned.details,omitempty"`
	UnassignedFor      string `json:"unassigned.for,omitempty"`
	RecoverySource     string `json:"recovery_source,omitempty"`
	RecoveryStage      string `json:"recovery_stage,omitempty"`
	RecoveryType       string `json:"recovery_type,omitempty"`
	RecoveryTimeMillis string `json:"recovery_time_millis,omitempty"`
}

// GetShards returns information about all shards in the cluster
func (c *Client) GetShards(nodes []string) ([]ShardInfo, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the request
	req := map[string]interface{}{
		"format": "json",
		"h": "index,shard,prirep,state,docs,store,ip,node,unassigned.reason,unassigned.at," +
			"unassigned.details,unassigned.for,recovery_source,recovery_stage,recovery_type,recovery_time_millis",
	}

	// Add node filter if specified
	if len(nodes) > 0 {
		req["nodes"] = nodes
	}

	// Execute request
	res, err := c.es.Cat.Shards(
		c.es.Cat.Shards.WithContext(ctx),
		c.es.Cat.Shards.WithFormat("json"),
		c.es.Cat.Shards.WithH(req["h"].(string)),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response: %s", res.String())
	}

	// Parse response
	var shards []ShardInfo
	if err := json.NewDecoder(res.Body).Decode(&shards); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return shards, nil
}

// GetShardsByNode organizes shards by node name
func (c *Client) GetShardsByNode(nodes []string) (map[string][]ShardInfo, []ShardInfo, error) {
	// Get all shards
	shards, err := c.GetShards(nodes)
	if err != nil {
		return nil, nil, err
	}

	// Organize by node
	shardsByNode := make(map[string][]ShardInfo)
	var unassignedShards []ShardInfo

	for _, shard := range shards {
		if shard.State == "UNASSIGNED" {
			unassignedShards = append(unassignedShards, shard)
		} else if shard.Node != "" {
			shardsByNode[shard.Node] = append(shardsByNode[shard.Node], shard)
		}
	}

	return shardsByNode, unassignedShards, nil
}
