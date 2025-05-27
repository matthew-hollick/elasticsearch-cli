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
	addresses    []string
	username     string
	password     string
	caCert       string
	insecure     bool
	disableRetry bool

	// Server fill options
	nodeName string

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "es_fill",
		Short: "Fill servers with data, removing shard allocation exclusion rules",
		Long:  `Use the subcommands to remove shard allocation exclusion rules from one server or all servers.`,
		PersistentPreRunE: initConfig,
	}

	// Server subcommand
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Fill one server with data, removing exclusion rules from it",
		Long:  `This command will remove shard allocation exclusion rules from a particular Elasticsearch node, allowing shards to be allocated to it.`,
		RunE:  runServerFill,
	}

	// All subcommand
	var allCmd = &cobra.Command{
		Use:   "all",
		Short: "Fill all servers with data, removing all exclusion rules",
		Long:  `This command will remove all shard allocation exclusion rules from the cluster, allowing all servers to fill with data.`,
		RunE:  runFillAll,
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

	// Server fill flags
	serverCmd.Flags().StringVarP(&nodeName, "name", "n", "", "Elasticsearch node name to fill (required)")
	serverCmd.MarkFlagRequired("name")

	// Add subcommands
	rootCmd.AddCommand(serverCmd, allCmd)

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

// runServerFill handles the server fill command
func runServerFill(cmd *cobra.Command, args []string) error {
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

	// Fill the server
	remainingExcluded, err := esClient.FillServer(nodeName)
	if err != nil {
		return fmt.Errorf("failed to fill node %s: %w", nodeName, err)
	}

	// Print results
	fmt.Printf("Node %s removed from allocation exclusion rules\n", nodeName)
	if len(remainingExcluded) > 0 {
		fmt.Printf("Nodes still excluded: %s\n", strings.Join(remainingExcluded, ", "))
	} else {
		fmt.Println("No nodes are currently being excluded")
	}

	return nil
}

// runFillAll handles the fill all command
func runFillAll(cmd *cobra.Command, args []string) error {
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

	// Fill all servers
	excludeSettings, err := esClient.FillAll()
	if err != nil {
		return fmt.Errorf("failed to fill all nodes: %w", err)
	}

	// Print results
	fmt.Println("All allocation exclusion rules have been removed")

	// Create formatter
	formatter := format.New(cfg.Output.Format)

	// Check if there are any remaining exclusions (should be none)
	hasExclusions := len(excludeSettings.ExcludeName) > 0 || 
		len(excludeSettings.ExcludeIP) > 0 || 
		len(excludeSettings.ExcludeHost) > 0

	if hasExclusions {
		fmt.Println("\nWarning: Some exclusion settings still exist:")

		// Show any remaining exclusions by name
		if len(excludeSettings.ExcludeName) > 0 {
			fmt.Println("\nNodes still excluded by name:")
			header := []string{"Node Name"}
			rows := [][]string{}
			for _, name := range excludeSettings.ExcludeName {
				rows = append(rows, []string{name})
			}
			if err := formatter.Write(header, rows); err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}
		}

		// Show any remaining exclusions by IP
		if len(excludeSettings.ExcludeIP) > 0 {
			fmt.Println("\nNodes still excluded by IP:")
			header := []string{"IP Address"}
			rows := [][]string{}
			for _, ip := range excludeSettings.ExcludeIP {
				rows = append(rows, []string{ip})
			}
			if err := formatter.Write(header, rows); err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}
		}

		// Show any remaining exclusions by host
		if len(excludeSettings.ExcludeHost) > 0 {
			fmt.Println("\nNodes still excluded by host:")
			header := []string{"Hostname"}
			rows := [][]string{}
			for _, host := range excludeSettings.ExcludeHost {
				rows = append(rows, []string{host})
			}
			if err := formatter.Write(header, rows); err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}
		}
	}

	return nil
}
