package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
)

func main() {
	var rootCmd = &cobra.Command{
		Use:              "kb_ping",
		Short:            "Check Kibana status",
		Long:             `Check if Kibana is up and running and display its status.`,
		PersistentPreRunE: initConfig,
		RunE:             run,
	}

	// Config file flag
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default is ./config.yaml, ~/.config/esctl/config.yaml, or /etc/esctl/config.yaml)")

	// Kibana connection flags
	rootCmd.PersistentFlags().StringSliceVar(&addresses, "kb-addresses", nil, "Kibana addresses (comma-separated list)")
	rootCmd.PersistentFlags().StringVar(&username, "kb-username", "", "Kibana username")
	rootCmd.PersistentFlags().StringVar(&password, "kb-password", "", "Kibana password")
	rootCmd.PersistentFlags().StringVar(&caCert, "kb-ca-cert", "", "Path to CA certificate for Kibana")
	rootCmd.PersistentFlags().BoolVar(&insecure, "kb-insecure", false, "Skip TLS certificate validation (insecure)")

	// Output flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "", "Output format (rich, plain, json, csv)")

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
	v.SetDefault("kibana.addresses", []string{"http://localhost:5601"})
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
	if cmd.Flags().Changed("kb-addresses") {
		v.Set("kibana.addresses", addresses)
	}
	if cmd.Flags().Changed("kb-username") {
		v.Set("kibana.username", username)
	}
	if cmd.Flags().Changed("kb-password") {
		v.Set("kibana.password", password)
	}
	if cmd.Flags().Changed("kb-ca-cert") {
		v.Set("kibana.ca_cert", caCert)
	}
	if cmd.Flags().Changed("kb-insecure") {
		v.Set("kibana.insecure", insecure)
	}
	if cmd.Flags().Changed("format") {
		v.Set("output.format", outputFormat)
	}

	// Store the viper instance in the context for later use
	cmd.SetContext(config.WithViper(cmd.Context(), v))

	return nil
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
	formatter := format.New(cfg.Output.Format)
	return formatter.Write(rows[0], rows[1:])
}
