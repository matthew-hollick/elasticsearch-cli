package main

import (
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

	// Command specific
	indexName string

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:              "es_mappings",
		Short:            "Display the mappings of the specified index",
		Long:             `View the field mappings and data types for Elasticsearch indices.

This command displays the complete mapping configuration for a specified index, showing how
Elasticsearch interprets and stores each field in your documents. Mappings define field types,
analyzer settings, and other metadata that control how fields are indexed and searched.

The output includes:
- Field names and their data types (text, keyword, date, numeric, etc.)
- Field properties (analyzer settings, doc_values, store settings)
- Multi-field configurations
- Dynamic mapping rules

Understanding mappings is crucial for optimizing search performance, controlling indexing behavior,
and ensuring your data is correctly interpreted by Elasticsearch.

Example usage:
  es_mappings --index=my-index
  es_mappings --index=my-index --format=json
  es_mappings --index=my-index --style=blue`,
		Example:          `es_mappings --index=my-index
es_mappings --index=my-index --format=json`,
		PersistentPreRunE: initConfig,
		RunE:             run,
	}
	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Config file flag
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default is ./config.yaml, ~/.config/esctl/config.yaml, or /etc/esctl/config.yaml)")

	// Elasticsearch connection flags
	rootCmd.PersistentFlags().StringSliceVar(&addresses, "es-addresses", nil, "Elasticsearch addresses (comma-separated list)")
	rootCmd.PersistentFlags().StringVar(&username, "es-username", "", "Elasticsearch username")
	rootCmd.PersistentFlags().StringVar(&password, "es-password", "", "Elasticsearch password")
	rootCmd.PersistentFlags().StringVar(&caCert, "es-ca-cert", "", "Path to CA certificate for Elasticsearch")
	rootCmd.PersistentFlags().BoolVar(&insecure, "es-insecure", false, "Skip TLS certificate validation (insecure)")
	rootCmd.PersistentFlags().BoolVar(&disableRetry, "es-disable-retry", false, "Disable retry on Elasticsearch connection failure")

	// Command specific flags
	rootCmd.Flags().StringVarP(&indexName, "index", "i", "", "Elasticsearch index to retrieve mappings from (required)")
	rootCmd.MarkFlagRequired("index")

	// Output flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "", "Output format (fancy, plain, json, csv)")
rootCmd.PersistentFlags().StringVar(&outputStyle, "style", "", "Table style for fancy output (dark, light, bright, blue, double)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	return config.InitializeConfig(cmd, configFile, addresses, username, password, caCert, insecure, disableRetry, outputFormat)
}

// run executes the command
func run(cmd *cobra.Command, args []string) error {
	// Get config from context
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Create client
	c, err := client.New(cfg)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Get mappings
	mappings, err := c.GetPrettyIndexMappings(indexName)
	if err != nil {
		return fmt.Errorf("error getting mappings: %w", err)
	}

	// Output as JSON (since mappings are already formatted as pretty JSON)
	fmt.Fprintln(cmd.OutOrStdout(), mappings)

	return nil
}
