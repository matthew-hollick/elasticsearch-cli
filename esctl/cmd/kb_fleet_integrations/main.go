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
	outputStyle string
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
)

func main() {
	var rootCmd = &cobra.Command{
		Use:               "kb_fleet_integrations",
		Short:             "List Kibana Fleet package policies (integrations)",
		Long:              `List all package policies (integrations) from Kibana Fleet.

Package policies (integrations) define the data collection configuration for specific services and systems. Each package policy is associated with an agent policy and determines what data is collected and how it's processed. This command displays integration details including ID, name, package type, version, and the associated agent policy ID.

Example usage:
  kb_fleet_integrations --kb-addresses=https://kibana:5601 --kb-username=elastic --kb-password=changeme
  kb_fleet_integrations --format=json
  kb_fleet_integrations --style=blue`,
		Example:           `kb_fleet_integrations
kb_fleet_integrations --format=json
kb_fleet_integrations --style=blue`,
		PersistentPreRunE: initConfig,
		RunE:              run,
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

func run(cmd *cobra.Command, args []string) error {
	// Load configuration with context containing viper instance
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize client
	fleetClient, err := client.NewFleet(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Fleet client: %w", err)
	}

	// Get Fleet package policies
	headers, rows, err := fleetClient.GetPackagePoliciesFormatted()
	if err != nil {
		return fmt.Errorf("failed to get Fleet package policies: %w", err)
	}

	// Output results
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	return formatter.Write(headers, rows)
}
