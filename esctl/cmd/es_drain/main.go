package main

import (
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
		Long:  `Safely remove an Elasticsearch node from service by relocating its shards to other nodes.

The drain command is essential for cluster maintenance operations. It allows you to safely
take a node offline by moving all its data to other nodes in the cluster. This is accomplished
by setting allocation rules that prevent new shards from being allocated to the target node
while gradually relocating existing shards elsewhere.

Use cases include:
- Performing maintenance on a specific node
- Decommissioning hardware
- Upgrading node configurations
- Rebalancing cluster workloads

The command offers options to start a drain operation on a specific node or to check the
status of ongoing drain operations. You can also stop a drain operation if needed.

Example usage:
  es_drain start --node=node-1
  es_drain status
  es_drain stop --node=node-1`,
		Example: `es_drain start --node=node-1
es_drain status
es_drain stop --node=node-1`,
		PersistentPreRunE: initConfig,
	}
	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

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
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "", "Output format (fancy, plain, json, csv)")
rootCmd.PersistentFlags().StringVar(&outputStyle, "style", "", "Table style for fancy output (dark, light, bright, blue, double)")

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
	// Use the centralized config initialization function
	return config.InitializeConfig(cmd, configFile, addresses, username, password, caCert, insecure, disableRetry, outputFormat)
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
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)

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
