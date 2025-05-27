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

	// Elasticsearch connection
	addresses   []string
	username    string
	password    string
	caCert      string
	insecure    bool
	disableRetry bool

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "es-ping",
		Short: "Check Elasticsearch cluster health",
		Long:  `Check the health and status of an Elasticsearch cluster.`,
		PersistentPreRunE: initConfig,
		RunE:  run,
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

func run(cmd *cobra.Command, args []string) error {
	// Load configuration with context containing viper instance
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Flag overrides are now handled in initConfig

	// Initialize client
	client, err := client.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get cluster health
	rows, err := client.CatHealth()
	if err != nil {
		return fmt.Errorf("failed to get cluster health: %w", err)
	}

	// Output results
	formatter := format.New(cfg.Output.Format)
	return formatter.Write(rows[0], rows[1:])
}
