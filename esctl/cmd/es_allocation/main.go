package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
	"github.com/spf13/cobra"
)

// Command line flags
var (
	outputStyle string
	// Config file
	configFile string

	// Elasticsearch connection
	addresses    []string
	username     string
	password     string
	caCert       string
	insecure     bool
	disableRetry bool

	// Allocation options
	status      string
	indexName   string
	shardID     string
	primaryFlag bool

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:               "es_allocation",
		Short:             "Control shard allocation in Elasticsearch",
		Long:              `View and modify shard allocation settings and get detailed allocation explanations.

This command gives you precise control over how Elasticsearch allocates shards across your cluster.
It allows you to view current allocation settings, enable/disable allocation, and get detailed
explanations for why specific shards are allocated to particular nodes or remain unallocated.

Key capabilities include:
- Viewing cluster-wide allocation status
- Enabling or disabling allocation (useful during maintenance)
- Getting allocation explanations for specific shards
- Understanding allocation decisions for troubleshooting

Proper shard allocation is critical for cluster performance, stability, and data availability.
Use this command when performing maintenance, troubleshooting allocation issues, or optimizing
cluster resource usage.

Example usage:
  es_allocation status
  es_allocation enable
  es_allocation disable
  es_allocation explain --index=my-index --shard=0 --primary`,
		Example:          `es_allocation status
es_allocation enable
es_allocation disable
es_allocation explain --index=my-index --shard=0 --primary`,
		PersistentPreRunE: initConfig,
		RunE:              getStatus, // Default action is to get status
	}
	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Get status subcommand (same as root command, but explicit)
	var getStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Get current allocation status",
		Long:  `Get the current shard allocation status for the cluster.`,
		RunE:  getStatus,
	}

	// Set status subcommand
	var setStatusCmd = &cobra.Command{
		Use:   "set",
		Short: "Set allocation status",
		Long:  `Set the shard allocation status for the cluster. Valid values are: all, primaries, new_primaries, none.`,
		RunE:  setStatus,
	}

	// Explain subcommand
	var explainCmd = &cobra.Command{
		Use:   "explain",
		Short: "Get allocation explanation",
		Long:  `Get detailed explanation of shard allocations, optionally for a specific shard.`,
		RunE:  explainAllocation,
	}

	// Config file flag
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default is ./config.yaml, ~/.config/esctl/config.yaml, or /etc/esctl/config.yaml)")

	// Elasticsearch connection flags
	rootCmd.PersistentFlags().StringSliceVar(&addresses, "es-addresses", nil, "Elasticsearch addresses (comma-separated list)")
	rootCmd.PersistentFlags().StringVar(&username, "es-username", "", "Elasticsearch username")
	rootCmd.PersistentFlags().StringVar(&password, "es-password", "", "Elasticsearch password")
	rootCmd.PersistentFlags().StringVar(&caCert, "es-ca-cert", "", "Path to CA certificate for Elasticsearch")
	rootCmd.PersistentFlags().BoolVar(&insecure, "es-insecure", false, "Skip TLS certificate validation (insecure)")
	rootCmd.PersistentFlags().BoolVar(&disableRetry, "es-disable-retry", false, "Disable retry on Elasticsearch connection failure")

	// Output flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "", "Output format (fancy, plain, json, csv)")
rootCmd.PersistentFlags().StringVar(&outputStyle, "style", "", "Table style for fancy output (dark, light, bright, blue, double)")

	// Set status command flags
	setStatusCmd.Flags().StringVarP(&status, "status", "s", "", "Allocation status to set (required, one of: all, primaries, new_primaries, none)")
	setStatusCmd.MarkFlagRequired("status")

	// Explain command flags
	explainCmd.Flags().StringVarP(&indexName, "index", "i", "", "Index name (optional)")
	explainCmd.Flags().StringVarP(&shardID, "shard", "s", "", "Shard ID (optional, requires index)")
	explainCmd.Flags().BoolVarP(&primaryFlag, "primary", "p", false, "Whether the shard is primary (only used with index and shard)")

	// Add subcommands
	rootCmd.AddCommand(getStatusCmd, setStatusCmd, explainCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	// Use the centralized config initialization function
	return config.InitializeConfig(cmd, configFile, addresses, username, password, caCert, insecure, disableRetry, outputFormat)
}

// getStatus handles the get allocation status command
func getStatus(cmd *cobra.Command, args []string) error {
	// Load configuration with context containing viper instance
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	esClient, err := client.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Get allocation status
	status, err := esClient.GetAllocationStatus()
	if err != nil {
		return fmt.Errorf("failed to get allocation status: %w", err)
	}

	// Print status with explanation
	fmt.Printf("Current allocation status: %s\n", status)
	fmt.Println("\nStatus explanations:")
	fmt.Println("- all: Allow shard allocation for all shards")
	fmt.Println("- primaries: Allow shard allocation only for primary shards")
	fmt.Println("- new_primaries: Allow shard allocation only for primary shards for new indices")
	fmt.Println("- none: No shard allocation allowed")

	return nil
}

// setStatus handles the set allocation status command
func setStatus(cmd *cobra.Command, args []string) error {
	// Load configuration with context containing viper instance
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	esClient, err := client.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Set allocation status
	if err := esClient.SetAllocationStatus(status); err != nil {
		return fmt.Errorf("failed to set allocation status: %w", err)
	}

	fmt.Printf("Allocation status set to: %s\n", status)
	return nil
}

// explainAllocation handles the allocation explain command
func explainAllocation(cmd *cobra.Command, args []string) error {
	// Load configuration with context containing viper instance
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	esClient, err := client.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Validate that if shard is specified, index is also specified
	if shardID != "" && indexName == "" {
		return fmt.Errorf("shard ID requires an index name")
	}

	// Get allocation explanation
	explanation, err := esClient.GetAllocationExplain(indexName, shardID, primaryFlag)
	if err != nil {
		return fmt.Errorf("failed to get allocation explanation: %w", err)
	}

	// Format and print explanation
	explanationJSON, err := json.MarshalIndent(explanation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format explanation: %w", err)
	}

	fmt.Println(string(explanationJSON))
	return nil
}
