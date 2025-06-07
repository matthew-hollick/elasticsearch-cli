package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/format"
	"github.com/spf13/cobra"
)

// Command line flags
var (
	// Config file
	configFile string

	// Kibana connection
	addresses []string
	username  string
	password  string
	caCert    string
	insecure  bool

	// Output
	outputFormat string
	outputStyle  string

	// Agent filtering
	kuery string
	agentID string

	// Agent operations
	agentTags []string
	policyID string
	forceDelete bool
	metadataFile string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:               "kb_fleet_agents",
		Short:             "Manage Kibana Fleet agents",
		Long:              `Manage Elastic Agents in Kibana Fleet.

This command provides agent management capabilities aligned with the Fleet policy dependency graph:
- Package Policy -> Agent Policy -> Agent

Operations include:
- Listing all agents with filtering
- Viewing detailed agent information
- Updating agent metadata and tags
- Reassigning agents between policies
- Unenrolling/deleting agents

Example usage:
  kb_fleet_agents --kb-addresses=https://kibana:5601
  kb_fleet_agents --kuery="policy_id:default-policy"
  kb_fleet_agents get --agent-id=12345678-1234-1234-1234-123456789012`,
		Example:           `kb_fleet_agents
kb_fleet_agents --kuery="policy_id:default-policy"
kb_fleet_agents get --agent-id=12345678-1234-1234-1234-123456789012`,
		PersistentPreRunE: initConfig,
		RunE:              listAgents, // Default action is to list agents
	}

	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Config file flag
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default is ./config.yaml, ~/.config/esctl/config.yaml, or /etc/esctl/config.yaml)")

	// Kibana connection flags
	rootCmd.PersistentFlags().StringSliceVar(&addresses, "kb-addresses", nil, "Kibana addresses (comma-separated list)")
	rootCmd.PersistentFlags().StringVar(&username, "kb-username", "", "Kibana username")
	rootCmd.PersistentFlags().StringVar(&password, "kb-password", "", "Kibana password")
	rootCmd.PersistentFlags().StringVar(&caCert, "kb-ca-cert", "", "Path to CA certificate for Kibana")
	rootCmd.PersistentFlags().BoolVar(&insecure, "kb-insecure", false, "Skip TLS certificate validation (insecure)")

	// Output flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "", "Output format (fancy, plain, json, csv)")
	rootCmd.PersistentFlags().StringVar(&outputStyle, "style", "", "Table style for fancy output (dark, light, bright, blue, double)")

	// Agent filtering flag for root command (list)
	rootCmd.Flags().StringVar(&kuery, "kuery", "", "Filter agents using KQL syntax (e.g. 'policy_id:\"default-policy\"')")

	// Get command
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a specific Fleet agent",
		Long:  "Get detailed information about a specific Fleet agent by ID",
		RunE:  getAgent,
	}
	getCmd.Flags().StringVar(&agentID, "agent-id", "", "ID of the agent to get (required)")
	getCmd.MarkFlagRequired("agent-id")
	rootCmd.AddCommand(getCmd)

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a Fleet agent",
		Long:  "Update a Fleet agent's tags or metadata",
		RunE:  updateAgent,
	}
	updateCmd.Flags().StringVar(&agentID, "agent-id", "", "ID of the agent to update (required)")
	updateCmd.Flags().StringSliceVar(&agentTags, "tags", nil, "Tags to set on the agent (comma-separated)")
	updateCmd.Flags().StringVar(&metadataFile, "metadata-file", "", "Path to JSON file containing user metadata")
	updateCmd.MarkFlagRequired("agent-id")
	rootCmd.AddCommand(updateCmd)

	// Reassign command
	reassignCmd := &cobra.Command{
		Use:   "reassign",
		Short: "Reassign an agent to a different policy",
		Long:  "Move an agent from its current policy to a different agent policy",
		RunE:  reassignAgent,
	}
	reassignCmd.Flags().StringVar(&agentID, "agent-id", "", "ID of the agent to reassign (required)")
	reassignCmd.Flags().StringVar(&policyID, "policy-id", "", "ID of the policy to assign the agent to (required)")
	reassignCmd.MarkFlagRequired("agent-id")
	reassignCmd.MarkFlagRequired("policy-id")
	rootCmd.AddCommand(reassignCmd)

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete/unenroll a Fleet agent",
		Long:  "Unenroll an agent from Fleet, optionally with force flag for offline agents",
		RunE:  deleteAgent,
	}
	deleteCmd.Flags().StringVar(&agentID, "agent-id", "", "ID of the agent to delete (required)")
	deleteCmd.Flags().BoolVar(&forceDelete, "force", false, "Force delete the agent even if it's offline")
	deleteCmd.MarkFlagRequired("agent-id")
	rootCmd.AddCommand(deleteCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	// Use the centralized Kibana config initialization function
	return config.InitializeKibanaConfig(cmd, configFile, addresses, username, password, caCert, insecure, outputFormat)
}

// listAgents lists all agents with optional filtering
func listAgents(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	fleetClient, err := client.NewFleet(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Fleet client: %w", err)
	}

	// Get Fleet agents
	headers, rows, err := fleetClient.GetAgentsFormatted(kuery)
	if err != nil {
		return fmt.Errorf("failed to get Fleet agents: %w", err)
	}

	// Output results
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	return formatter.Write(headers, rows)
}

// getAgent gets a specific agent by ID
func getAgent(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	fleetClient, err := client.NewFleet(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Fleet client: %w", err)
	}

	// Get agent
	agent, err := fleetClient.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Output result as JSON for detailed view
	jsonData, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling agent data: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// updateAgent updates an agent's tags or metadata
func updateAgent(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	fleetClient, err := client.NewFleet(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Fleet client: %w", err)
	}

	// Load metadata from file if provided
	var metadata map[string]interface{}
	if metadataFile != "" {
		data, err := ioutil.ReadFile(metadataFile)
		if err != nil {
			return fmt.Errorf("failed to read metadata file: %w", err)
		}
		if err := json.Unmarshal(data, &metadata); err != nil {
			return fmt.Errorf("failed to parse metadata JSON: %w", err)
		}
	}

	// Update agent
	if err := fleetClient.UpdateAgent(agentID, metadata, agentTags); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	fmt.Printf("Agent %s updated successfully\n", agentID)
	return nil
}

// reassignAgent assigns an agent to a different policy
func reassignAgent(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	fleetClient, err := client.NewFleet(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Fleet client: %w", err)
	}

	// Reassign agent
	if err := fleetClient.ReassignAgent(agentID, policyID); err != nil {
		return fmt.Errorf("failed to reassign agent: %w", err)
	}

	fmt.Printf("Agent %s reassigned to policy %s successfully\n", agentID, policyID)
	return nil
}

// deleteAgent deletes/unenrolls an agent
func deleteAgent(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	fleetClient, err := client.NewFleet(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Fleet client: %w", err)
	}

	// Delete agent
	if err := fleetClient.DeleteAgent(agentID, forceDelete); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	fmt.Printf("Agent %s deleted successfully\n", agentID)
	return nil
}
