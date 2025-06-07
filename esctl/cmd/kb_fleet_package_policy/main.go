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

	// Output format
	outputFormat string
	outputStyle  string

	// Common policy parameters
	packagePolicyID      string
	customPackagePolicyID string
	name                 string
	description          string
	namespace            string
	agentPolicyID        string
	packageName          string
	packageVersion       string
	force                bool
	jsonConfigFile       string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "kb_fleet_package_policy",
		Short: "Manage Kibana Fleet package policies (integrations)",
		Long: `Manage package policies (integrations) in Kibana Fleet.

Package policies define the actual integrations that are deployed to agents.
Each package policy is assigned to an agent policy, following the dependency chain:
Package Policy → Agent Policy → Agent`,
		Example: `kb_fleet_package_policy list
kb_fleet_package_policy create --name="system-1" --agent-policy-id=abc123 --package=system --version=1.0.0
kb_fleet_package_policy update --policy-id=xyz789 --name="updated-name"
kb_fleet_package_policy delete --policy-id=xyz789`,
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
		Short:   "List package policies",
		Long:    "List package policies (integrations) from Kibana Fleet",
		Example: "kb_fleet_package_policy list",
		RunE:    listPackagePolicies,
	}
	rootCmd.AddCommand(listCmd)

	// Create command
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new package policy",
		Long: `Create a new package policy (integration) in Kibana Fleet.

You must specify the following:
- name: unique name for the package policy
- agent-policy-id: ID of the agent policy to assign this to
- package: name of the integration package
- version: version of the integration package 
- namespace: namespace for the data (default is "default")

For complex integrations, use --config-json to specify the full configuration.`,
		Example: `kb_fleet_package_policy create --name="system-metrics" --agent-policy-id=abc123 --package=system --version=1.0.0
kb_fleet_package_policy create --id=custom-system-1 --name="custom-system" --agent-policy-id=abc123 --package=system --version=1.0.0
kb_fleet_package_policy create --name="elasticsearch-metrics" --agent-policy-id=abc123 --package=elasticsearch --version=1.0.0 --config-json=config.json`,
		RunE: createPackagePolicy,
	}
	createCmd.Flags().StringVar(&customPackagePolicyID, "id", "", "Custom ID for the package policy (optional, auto-generated if not provided). Must be lowercase alphanumeric with hyphens/underscores, max 36 chars.")
	createCmd.Flags().StringVar(&name, "name", "", "Name of the package policy (required)")
	createCmd.Flags().StringVar(&description, "description", "", "Description of the package policy")
	createCmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace for the package policy")
	createCmd.Flags().StringVar(&agentPolicyID, "agent-policy-id", "", "ID of the agent policy to assign this package policy to (required)")
	createCmd.Flags().StringVar(&packageName, "package", "", "Name of the integration package (required)")
	createCmd.Flags().StringVar(&packageVersion, "version", "", "Version of the integration package (required)")
	createCmd.Flags().StringVar(&jsonConfigFile, "config-json", "", "Path to JSON file containing full integration configuration")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("agent-policy-id")
	createCmd.MarkFlagRequired("package")
	createCmd.MarkFlagRequired("version")
	rootCmd.AddCommand(createCmd)

	// Update command
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update a package policy",
		Long:  "Update an existing package policy in Kibana Fleet",
		Example: `kb_fleet_package_policy update --policy-id=xyz789 --name="updated-name"
kb_fleet_package_policy update --policy-id=xyz789 --description="New description"
kb_fleet_package_policy update --policy-id=xyz789 --config-json=updated-config.json`,
		RunE: updatePackagePolicy,
	}
	updateCmd.Flags().StringVar(&packagePolicyID, "policy-id", "", "ID of the package policy to update (required)")
	updateCmd.Flags().StringVar(&name, "name", "", "New name for the package policy")
	updateCmd.Flags().StringVar(&description, "description", "", "New description for the package policy")
	updateCmd.Flags().StringVar(&namespace, "namespace", "", "New namespace for the package policy")
	updateCmd.Flags().StringVar(&jsonConfigFile, "config-json", "", "Path to JSON file containing updated integration configuration")
	updateCmd.MarkFlagRequired("policy-id")
	rootCmd.AddCommand(updateCmd)

	// Delete command
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a package policy",
		Long:  "Delete a package policy (integration) from Kibana Fleet",
		Example: `kb_fleet_package_policy delete --policy-id=xyz789
kb_fleet_package_policy delete --policy-id=xyz789 --force`,
		RunE: deletePackagePolicy,
	}
	deleteCmd.Flags().StringVar(&packagePolicyID, "policy-id", "", "ID of the package policy to delete (required)")
	deleteCmd.Flags().BoolVar(&force, "force", false, "Force deletion even if the package policy is in use")
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

// listPackagePolicies lists all package policies
func listPackagePolicies(cmd *cobra.Command, args []string) error {
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

	// Get package policies
	policies, err := fleetClient.GetPackagePolicies()
	if err != nil {
		return fmt.Errorf("failed to get package policies: %w", err)
	}

	// Output results based on format
	if outputFormat == "json" {
		jsonOutput, err := json.MarshalIndent(policies, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling to JSON: %w", err)
		}
		fmt.Println(string(jsonOutput))
		return nil
	}

	// Format as table for standard display
	headers, rows, err := formatPackagePolicies(policies)
	if err != nil {
		return fmt.Errorf("error formatting package policies: %w", err)
	}

	// Output results
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	return formatter.Write(headers, rows)
}

// formatPackagePolicies converts package policies to tabular format
func formatPackagePolicies(policies []client.PackagePolicy) ([]string, [][]string, error) {
	headers := []string{"ID", "Name", "Package", "Version", "Agent Policy ID"}
	var rows [][]string

	for _, policy := range policies {
		row := []string{
			policy.ID,
			policy.Name,
			policy.Package.Name,
			policy.Package.Version,
			policy.PolicyID,
		}
		rows = append(rows, row)
	}

	return headers, rows, nil
}

// createPackagePolicy creates a new package policy
func createPackagePolicy(cmd *cobra.Command, args []string) error {
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

	// Create basic package policy
	policy := client.PackagePolicy{
		ID:          customPackagePolicyID,
		Name:        name,
		Description: description,
		PolicyID:    agentPolicyID,
		Package: client.PackagePolicyPackage{
			Name:    packageName,
			Version: packageVersion,
		},
		Inputs:      make(map[string]interface{}),
	}

	// Load and merge optional JSON config
	if jsonConfigFile != "" {
		data, err := ioutil.ReadFile(jsonConfigFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		var jsonConfig client.PackagePolicy
		if err := json.Unmarshal(data, &jsonConfig); err != nil {
			return fmt.Errorf("failed to parse config JSON: %w", err)
		}

		// Merge the JSON config with our command-line values
		// Command-line values take precedence for basic fields
		if jsonConfig.Inputs != nil {
			policy.Inputs = jsonConfig.Inputs
		}
		// Keep command line values for package name/version
		// but merge other configuration
	}

	// Create the package policy
	createdPolicy, err := fleetClient.CreatePackagePolicy(policy)
	if err != nil {
		return fmt.Errorf("failed to create package policy: %w", err)
	}

	// Output success message
	fmt.Printf("Package policy created successfully\nID: %s\nName: %s\n",
		createdPolicy.ID, createdPolicy.Name)
	return nil
}

// updatePackagePolicy updates an existing package policy
func updatePackagePolicy(cmd *cobra.Command, args []string) error {
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

	// Get the existing policy
	policies, err := fleetClient.GetPackagePolicies()
	if err != nil {
		return fmt.Errorf("failed to get package policies: %w", err)
	}

	// Find the specific policy
	var existingPolicy *client.PackagePolicy
	for i, p := range policies {
		if p.ID == packagePolicyID {
			existingPolicy = &policies[i]
			break
		}
	}

	if existingPolicy == nil {
		return fmt.Errorf("package policy with ID %s not found", packagePolicyID)
	}

	// Update fields that were explicitly specified
	if name != "" {
		existingPolicy.Name = name
	}
	if description != "" {
		existingPolicy.Description = description
	}

	// Load and merge JSON config if provided
	if jsonConfigFile != "" {
		data, err := ioutil.ReadFile(jsonConfigFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		var jsonConfig client.PackagePolicy
		if err := json.Unmarshal(data, &jsonConfig); err != nil {
			return fmt.Errorf("failed to parse config JSON: %w", err)
		}

		// Update configuration from JSON
		if jsonConfig.Inputs != nil {
			existingPolicy.Inputs = jsonConfig.Inputs
		}
	}

	// Update the policy
	updatedPolicy, err := fleetClient.UpdatePackagePolicy(packagePolicyID, *existingPolicy)
	if err != nil {
		return fmt.Errorf("failed to update package policy: %w", err)
	}

	// Output success message
	fmt.Printf("Package policy updated successfully\nID: %s\nName: %s\n",
		updatedPolicy.ID, updatedPolicy.Name)
	return nil
}

// deletePackagePolicy deletes a package policy
func deletePackagePolicy(cmd *cobra.Command, args []string) error {
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

	// Delete the package policy with force flag if specified
	err = fleetClient.DeletePackagePolicy(packagePolicyID, force)
	if err != nil {
		return fmt.Errorf("failed to delete package policy: %w", err)
	}

	// Output success message
	fmt.Printf("Package policy %s deleted successfully\n", packagePolicyID)
	return nil
}
