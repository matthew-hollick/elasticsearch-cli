package main

import (
	"fmt"
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

	// Output format
	outputFormat string
	outputStyle  string

	// Common policy parameters
	policyID          string
	customPolicyID    string
	policyName        string
	policyDescription string
	policyNamespace   string
	monitoringOptions []string
	
	// Delete-specific flags
	forceDelete bool
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "kb_fleet_agent_policy",
		Short: "Manage Kibana Fleet agent policies",
		Long: `Manage agent policies in Kibana Fleet.

Agent policies define configurations for Elastic Agents and determine
which integrations are deployed to the agents. This command provides
full lifecycle management of agent policies.`,
		Example: `kb_fleet_agent_policy list
kb_fleet_agent_policy create --name="Production Servers" --description="Policy for production web servers"
kb_fleet_agent_policy update --policy-id=123abc --name="Updated Name"
kb_fleet_agent_policy delete --policy-id=123abc`,
		PersistentPreRunE: initConfig,
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

	// List command
	var listCmd = &cobra.Command{
		Use:     "list",
		Short:   "List agent policies",
		Long:    "List agent policies from Kibana Fleet with optional filtering",
		Example: "kb_fleet_agent_policy list",
		RunE:    listPolicies,
	}
	rootCmd.AddCommand(listCmd)

	// Create command
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new agent policy",
		Long:  "Create a new agent policy in Kibana Fleet",
		Example: `kb_fleet_agent_policy create --name="Production Servers" --description="Policy for production web servers"
kb_fleet_agent_policy create --name="Database Hosts" --namespace=prod --monitoring=logs,metrics
kb_fleet_agent_policy create --id=my-custom-id-001 --name="Custom ID Policy"`,
		RunE: createPolicy,
	}
	createCmd.Flags().StringVar(&customPolicyID, "id", "", "Custom ID for the agent policy (optional, auto-generated if not provided). Must be lowercase alphanumeric with hyphens/underscores, max 36 chars.")
	createCmd.Flags().StringVar(&policyName, "name", "", "Name of the agent policy (required)")
	createCmd.Flags().StringVar(&policyDescription, "description", "", "Description of the agent policy")
	createCmd.Flags().StringVar(&policyNamespace, "namespace", "default", "Namespace for the agent policy")
	createCmd.Flags().StringSliceVar(&monitoringOptions, "monitoring", nil, "Monitoring options to enable (logs, metrics, synthetics)")
	createCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(createCmd)

	// Update command
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update an agent policy",
		Long:  "Update an existing agent policy in Kibana Fleet",
		Example: `kb_fleet_agent_policy update --policy-id=123abc --name="Updated Name"
kb_fleet_agent_policy update --policy-id=123abc --description="New description" --monitoring=logs,metrics`,
		RunE: updatePolicy,
	}
	updateCmd.Flags().StringVar(&policyID, "policy-id", "", "ID of the agent policy to update (required)")
	updateCmd.Flags().StringVar(&policyName, "name", "", "New name for the agent policy")
	updateCmd.Flags().StringVar(&policyDescription, "description", "", "New description for the agent policy")
	updateCmd.Flags().StringVar(&policyNamespace, "namespace", "", "New namespace for the agent policy")
	updateCmd.Flags().StringSliceVar(&monitoringOptions, "monitoring", nil, "Monitoring options to enable (logs, metrics, synthetics)")
	updateCmd.MarkFlagRequired("policy-id")
	rootCmd.AddCommand(updateCmd)

	// Delete command
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete an agent policy",
		Long:  "Delete an agent policy from Kibana Fleet",
		Example: `kb_fleet_agent_policy delete --policy-id=123abc
kb_fleet_agent_policy delete --policy-id=123abc --force`,
		RunE: deletePolicy,
	}
	deleteCmd.Flags().StringVar(&policyID, "policy-id", "", "ID of the agent policy to delete (required)")
	deleteCmd.Flags().BoolVar(&forceDelete, "force", false, "Force deletion even if agents are assigned to the policy")
	deleteCmd.MarkFlagRequired("policy-id")
	rootCmd.AddCommand(deleteCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	return config.InitializeKibanaConfig(cmd, configFile, addresses, username, password, caCert, insecure, outputFormat)
}

// listPolicies handles listing agent policies
func listPolicies(cmd *cobra.Command, args []string) error {
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

	// Get and format agent policies
	headers, rows, err := fleetClient.GetAgentPoliciesFormatted()
	if err != nil {
		return fmt.Errorf("failed to get Fleet agent policies: %w", err)
	}

	// Output results
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	return formatter.Write(headers, rows)
}

// createPolicy handles agent policy creation
func createPolicy(cmd *cobra.Command, args []string) error {
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

	// Create policy object
	policy := client.AgentPolicy{
		ID:                customPolicyID, // Will be empty if not provided
		Name:              policyName,
		Description:       policyDescription,
		Namespace:         policyNamespace,
		MonitoringEnabled: monitoringOptions,
	}

	// Create the policy
	createdPolicy, err := fleetClient.CreateAgentPolicy(policy)
	if err != nil {
		return fmt.Errorf("failed to create agent policy: %w", err)
	}

	// Output success message with created policy ID
	fmt.Printf("Agent policy created successfully\nID: %s\nName: %s\nNamespace: %s\n", 
		createdPolicy.ID, createdPolicy.Name, createdPolicy.Namespace)
	return nil
}

// updatePolicy handles agent policy updates
func updatePolicy(cmd *cobra.Command, args []string) error {
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

	// First get the existing policy
	policies, err := fleetClient.GetAgentPolicies()
	if err != nil {
		return fmt.Errorf("failed to get agent policies: %w", err)
	}

	var existingPolicy *client.AgentPolicy
	for i, p := range policies {
		if p.ID == policyID {
			existingPolicy = &policies[i]
			break
		}
	}

	if existingPolicy == nil {
		return fmt.Errorf("policy with ID %s not found", policyID)
	}

	// Only update fields that were explicitly provided
	if policyName != "" {
		existingPolicy.Name = policyName
	}
	if policyDescription != "" {
		existingPolicy.Description = policyDescription
	}
	if policyNamespace != "" {
		existingPolicy.Namespace = policyNamespace
	}
	if monitoringOptions != nil {
		existingPolicy.MonitoringEnabled = monitoringOptions
	}

	// Update the policy
	updatedPolicy, err := fleetClient.UpdateAgentPolicy(policyID, *existingPolicy)
	if err != nil {
		return fmt.Errorf("failed to update agent policy: %w", err)
	}

	// Output success message with updated policy info
	fmt.Printf("Agent policy updated successfully\nID: %s\nName: %s\nRevision: %d\n", 
		updatedPolicy.ID, updatedPolicy.Name, updatedPolicy.Revision)
	return nil
}

// deletePolicy handles agent policy deletion
func deletePolicy(cmd *cobra.Command, args []string) error {
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

	// Delete the policy with force flag if specified
	err = fleetClient.DeleteAgentPolicy(policyID, forceDelete)
	if err != nil {
		return fmt.Errorf("failed to delete agent policy: %w", err)
	}

	fmt.Printf("Agent policy %s deleted successfully\n", policyID)
	return nil
}
