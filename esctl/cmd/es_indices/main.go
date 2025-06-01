package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/format"
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

	// Index options
	indexPattern string
	indexName    string
	settingsJSON string
	force        bool

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "es_indices",
		Short: "Manage Elasticsearch indices",
		Long:  `View and manage Elasticsearch indices, including listing, deleting, opening, closing, and updating settings.

This command provides comprehensive control over Elasticsearch indices. By default, it lists
all indices with their key metrics such as document count, size, status, and health. You can
filter indices using patterns to focus on specific index groups.

The command supports multiple operations through subcommands:
- list: Display all indices matching a pattern (default action)
- delete: Remove indices from the cluster
- open/close: Control index state to optimize resource usage
- settings: View or update index configuration

Use this command for index maintenance, monitoring storage usage, or applying configuration
changes across your indices.

Example usage:
  es_indices --es-addresses=https://elasticsearch:9200 --es-username=elastic --es-password=changeme
  es_indices --index-pattern="logstash-*" --format=json
  es_indices delete --index-name="old-index" --force`,
		Example: `es_indices
es_indices --index-pattern="logstash-*"
es_indices delete --index-name="old-index" --force`,
		PersistentPreRunE: initConfig,
		RunE:  listIndices, // Default action is to list indices
	}
	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// List subcommand (same as root command, but explicit)
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List indices in the cluster",
		Long:  `List all indices in the Elasticsearch cluster with their status, health, and size information.`,
		RunE:  listIndices,
	}

	// Delete subcommand
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete an index",
		Long:  `Delete an index from the Elasticsearch cluster. This operation is irreversible.`,
		RunE:  deleteIndex,
	}

	// Open subcommand
	var openCmd = &cobra.Command{
		Use:   "open",
		Short: "Open a closed index",
		Long:  `Open a closed index to make it available for search and indexing operations.`,
		RunE:  openIndex,
	}

	// Close subcommand
	var closeCmd = &cobra.Command{
		Use:   "close",
		Short: "Close an open index",
		Long:  `Close an open index to reduce resource usage. Closed indices cannot be searched or indexed.`,
		RunE:  closeIndex,
	}

	// Settings subcommand
	var settingsCmd = &cobra.Command{
		Use:   "settings",
		Short: "Get or update index settings",
		Long:  `Get or update settings for a specific index.`,
		RunE:  getIndexSettings,
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

	// List command flags
	rootCmd.Flags().StringVarP(&indexPattern, "pattern", "p", "", "Index pattern to filter indices (e.g., 'logs-*')")
	listCmd.Flags().StringVarP(&indexPattern, "pattern", "p", "", "Index pattern to filter indices (e.g., 'logs-*')")

	// Delete command flags
	deleteCmd.Flags().StringVarP(&indexName, "name", "n", "", "Name of the index to delete (required)")
	deleteCmd.Flags().BoolVarP(&force, "force", "", false, "Force deletion without confirmation")
	deleteCmd.MarkFlagRequired("name")

	// Open command flags
	openCmd.Flags().StringVarP(&indexName, "name", "n", "", "Name of the index to open (required)")
	openCmd.MarkFlagRequired("name")

	// Close command flags
	closeCmd.Flags().StringVarP(&indexName, "name", "n", "", "Name of the index to close (required)")
	closeCmd.MarkFlagRequired("name")

	// Settings command flags
	settingsCmd.Flags().StringVarP(&indexName, "name", "n", "", "Name of the index to get/update settings for (required)")
	settingsCmd.Flags().StringVarP(&settingsJSON, "settings", "s", "", "JSON string with settings to update (if not provided, current settings will be displayed)")
	settingsCmd.MarkFlagRequired("name")

	// Add subcommands
	rootCmd.AddCommand(listCmd, deleteCmd, openCmd, closeCmd, settingsCmd)

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

// listIndices handles the list indices command
func listIndices(cmd *cobra.Command, args []string) error {
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

	// Get indices
	indices, err := esClient.GetIndices(indexPattern)
	if err != nil {
		return fmt.Errorf("failed to get indices: %w", err)
	}

	if len(indices) == 0 {
		fmt.Println("No indices found")
		return nil
	}

	// Create formatter
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)

	// Prepare table data
	header := []string{"Index", "Status", "Health", "Docs Count", "Docs Deleted", "Store Size", "Primary Store Size"}
	rows := [][]string{}

	for _, idx := range indices {
		row := []string{
			idx.Name,
			idx.Status,
			idx.Health,
			idx.DocsCount,
			idx.DocsDeleted,
			idx.StoreSize,
			idx.PriStoreSize,
		}
		rows = append(rows, row)
	}

	// Print table
	return formatter.Write(header, rows)
}

// deleteIndex handles the delete index command
func deleteIndex(cmd *cobra.Command, args []string) error {
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

	// Confirm deletion if not forced
	if !force {
		fmt.Printf("Are you sure you want to delete index '%s'? This operation cannot be undone. [y/N] ", indexName)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Delete index
	if err := esClient.DeleteIndex(indexName); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	fmt.Printf("Index '%s' deleted successfully\n", indexName)
	return nil
}

// openIndex handles the open index command
func openIndex(cmd *cobra.Command, args []string) error {
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

	// Open index
	if err := esClient.OpenIndex(indexName); err != nil {
		return fmt.Errorf("failed to open index: %w", err)
	}

	fmt.Printf("Index '%s' opened successfully\n", indexName)
	return nil
}

// closeIndex handles the close index command
func closeIndex(cmd *cobra.Command, args []string) error {
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

	// Close index
	if err := esClient.CloseIndex(indexName); err != nil {
		return fmt.Errorf("failed to close index: %w", err)
	}

	fmt.Printf("Index '%s' closed successfully\n", indexName)
	return nil
}

// getIndexSettings handles the get/update index settings command
func getIndexSettings(cmd *cobra.Command, args []string) error {
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

	// If settings are provided, update them
	if settingsJSON != "" {
		// Parse settings JSON
		var settings map[string]interface{}
		if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
			return fmt.Errorf("failed to parse settings JSON: %w", err)
		}

		// Update settings
		if err := esClient.UpdateIndexSettings(indexName, settings); err != nil {
			return fmt.Errorf("failed to update index settings: %w", err)
		}

		fmt.Printf("Settings for index '%s' updated successfully\n", indexName)
		return nil
	}

	// Otherwise, get current settings
	settings, err := esClient.GetIndexSettings(indexName)
	if err != nil {
		return fmt.Errorf("failed to get index settings: %w", err)
	}

	// Format and print settings
	settingsJSON, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format settings: %w", err)
	}

	fmt.Printf("Settings for index '%s':\n%s\n", indexName, string(settingsJSON))
	return nil
}
