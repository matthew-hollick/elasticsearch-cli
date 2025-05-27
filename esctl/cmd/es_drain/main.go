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

	// Server drain options
	nodeName string
	stopDrain bool

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "es_drain",
		Short: "Drain a server or see what servers are draining",
		Long:  `Use the subcommands to drain a server or to see what servers are currently draining.`,
		PersistentPreRunE: initConfig,
	}

	// Server subcommand
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Drain a server by excluding shards from it",
		Long:  `This command will set the shard allocation rules to exclude the given server name. This will cause shards to be moved away from this server, draining the data away.`,
		RunE:  runServerDrain,
	}

	// Status subcommand
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "See what servers are set to drain",
		Long:  `This command will display what servers are set in the clusters allocation exclude rules.`,
		RunE:  runDrainStatus,
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

	// Server drain flags
	serverCmd.Flags().StringVarP(&nodeName, "name", "n", "", "Elasticsearch node name to drain (required)")
	serverCmd.Flags().BoolVarP(&stopDrain, "stop", "s", false, "Stop draining the node instead of starting it")
	serverCmd.MarkFlagRequired("name")

	// Add subcommands
	rootCmd.AddCommand(serverCmd, statusCmd)

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

// runServerDrain handles the server drain command
func runServerDrain(cmd *cobra.Command, args []string) error {
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

	var excludedNodes []string
	var action string

	// Either start or stop draining based on the flag
	if stopDrain {
		action = "stop draining"
		excludedNodes, err = esClient.StopDrainServer(nodeName)
	} else {
		action = "draining"
		excludedNodes, err = esClient.DrainServer(nodeName)
	}

	if err != nil {
		return fmt.Errorf("failed to %s node %s: %w", action, nodeName, err)
	}

	// Print results
	fmt.Printf("%s node: %s\n", action, nodeName)
	if len(excludedNodes) > 0 {
		fmt.Printf("Current excluded nodes: %s\n", strings.Join(excludedNodes, ", "))
	} else {
		fmt.Println("No nodes are currently being drained")
	}

	return nil
}

// runDrainStatus handles the drain status command
func runDrainStatus(cmd *cobra.Command, args []string) error {
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

	// Get drain status
	excludeSettings, err := esClient.GetClusterExcludeSettings()
	if err != nil {
		return fmt.Errorf("failed to get drain status: %w", err)
	}

	// Create formatter
	formatter := format.New(cfg.Output.Format)

	// Prepare table data for excluded nodes by name
	if len(excludeSettings.ExcludeName) > 0 {
		fmt.Println("Nodes excluded by name:")
		header := []string{"Node Name"}
		rows := [][]string{}
		for _, name := range excludeSettings.ExcludeName {
			rows = append(rows, []string{name})
		}
		if err := formatter.Write(header, rows); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
	} else {
		fmt.Println("No nodes are excluded by name")
	}

	// Prepare table data for excluded nodes by IP
	if len(excludeSettings.ExcludeIP) > 0 {
		fmt.Println("\nNodes excluded by IP:")
		header := []string{"IP Address"}
		rows := [][]string{}
		for _, ip := range excludeSettings.ExcludeIP {
			rows = append(rows, []string{ip})
		}
		if err := formatter.Write(header, rows); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
	}

	// Prepare table data for excluded nodes by host
	if len(excludeSettings.ExcludeHost) > 0 {
		fmt.Println("\nNodes excluded by host:")
		header := []string{"Hostname"}
		rows := [][]string{}
		for _, host := range excludeSettings.ExcludeHost {
			rows = append(rows, []string{host})
		}
		if err := formatter.Write(header, rows); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
	}

	return nil
}
