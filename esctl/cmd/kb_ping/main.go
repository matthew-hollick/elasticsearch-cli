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
		Use:              "kb_ping",
		Short:            "Check Kibana status",
		Long:             `Check if Kibana is up and running and display its status.

This command connects to Kibana and verifies that the service is operational. It returns
key information about the Kibana instance including version, status, and build details.
Use this command to troubleshoot connectivity issues or verify Kibana deployments.

Example usage:
  kb_ping --kb-addresses=https://kibana:5601 --kb-username=elastic --kb-password=changeme
  kb_ping --format=json
  kb_ping --style=blue`,
		Example:          `kb_ping
kb_ping --format=json
kb_ping --style=blue`,
		PersistentPreRunE: initConfig,
		RunE:             run,
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
	kibanaClient, err := client.NewKibana(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Kibana client: %w", err)
	}

	// Get Kibana status
	rows, err := kibanaClient.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get Kibana status: %w", err)
	}

	// Output results
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	return formatter.Write(rows[0], rows[1:])
}
