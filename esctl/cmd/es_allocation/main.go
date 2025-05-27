package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Command line flags
var (
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
	status     string
	indexName  string
	shardID    string
	primaryFlag bool

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "es_allocation",
		Short: "Control shard allocation in Elasticsearch",
		Long:  `View and modify shard allocation settings and get detailed allocation explanations.`,
		PersistentPreRunE: initConfig,
		RunE:  getStatus, // Default action is to get status
	}

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
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "", "Output format (rich, plain, json, csv)")

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
	v := viper.New()

	// Use config file from the flag if provided
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Use default config locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.config/esctl")
		v.AddConfigPath("/etc/esctl")
	}

	// Set defaults
	v.SetDefault("elasticsearch.addresses", []string{"http://localhost:9200"})
	v.SetDefault("output.format", "rich")

	// Read config file if it exists
	if err := v.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	}

	// Enable environment variable binding
	v.SetEnvPrefix("ESCTL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind flags to viper
	if cmd.Flags().Changed("es-addresses") {
		v.Set("elasticsearch.addresses", addresses)
	}
	if cmd.Flags().Changed("es-username") {
		v.Set("elasticsearch.username", username)
	}
	if cmd.Flags().Changed("es-password") {
		v.Set("elasticsearch.password", password)
	}
	if cmd.Flags().Changed("es-ca-cert") {
		v.Set("elasticsearch.ca_cert", caCert)
	}
	if cmd.Flags().Changed("es-insecure") {
		v.Set("elasticsearch.insecure", insecure)
	}
	if cmd.Flags().Changed("es-disable-retry") {
		v.Set("elasticsearch.disable_retry", disableRetry)
	}
	if cmd.Flags().Changed("format") {
		v.Set("output.format", outputFormat)
	}

	// Store the viper instance in the context for later use
	cmd.SetContext(config.WithViper(cmd.Context(), v))

	return nil
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
