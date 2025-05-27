package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/format"
	"github.com/spf13/cobra"
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

	// Node options
	nodeID string

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "es_nodes",
		Short: "Get information about Elasticsearch nodes",
		Long:  `View information about Elasticsearch nodes, including resource usage and hot threads.`,
		PersistentPreRunE: initConfig,
		RunE:  listNodes, // Default action is to list nodes
	}

	// List subcommand (same as root command, but explicit)
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List nodes in the cluster",
		Long:  `List all nodes in the Elasticsearch cluster with their resource usage information.`,
		RunE:  listNodes,
	}

	// Stats subcommand
	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Get detailed stats for a node",
		Long:  `Get detailed statistics for a specific node in the Elasticsearch cluster.`,
		RunE:  getNodeStats,
	}

	// Hot threads subcommand
	var hotThreadsCmd = &cobra.Command{
		Use:   "hotthreads",
		Short: "Get hot threads information",
		Long:  `Get information about hot threads in the Elasticsearch cluster or a specific node.`,
		RunE:  getHotThreads,
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

	// Stats command flags
	statsCmd.Flags().StringVarP(&nodeID, "id", "i", "", "Node ID to get stats for (required)")
	statsCmd.MarkFlagRequired("id")

	// Hot threads command flags
	hotThreadsCmd.Flags().StringVarP(&nodeID, "id", "i", "", "Node ID to get hot threads for (optional, if not provided, gets hot threads for all nodes)")

	// Add subcommands
	rootCmd.AddCommand(listCmd, statsCmd, hotThreadsCmd)

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

// listNodes handles the list nodes command
func listNodes(cmd *cobra.Command, args []string) error {
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

	// Get nodes
	nodes, err := esClient.GetNodes()
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	if len(nodes) == 0 {
		fmt.Println("No nodes found")
		return nil
	}

	// Create formatter
	formatter := format.New(cfg.Output.Format)

	// Prepare table data
	header := []string{"ID", "Name", "IP", "Role", "CPU", "Load (1m/5m/15m)", "RAM %", "Heap %", "Disk Used %", "Disk Avail", "Uptime"}
	rows := [][]string{}

	for _, node := range nodes {
		load := fmt.Sprintf("%s/%s/%s", node.Load1m, node.Load5m, node.Load15m)
		row := []string{
			node.ID,
			node.Name,
			node.IP,
			node.Role,
			node.CPU,
			load,
			node.RAMPercent,
			node.HeapPercent,
			node.DiskUsedPercent,
			node.DiskAvailable,
			node.Uptime,
		}
		rows = append(rows, row)
	}

	// Print table
	return formatter.Write(header, rows)
}

// getNodeStats handles the node stats command
func getNodeStats(cmd *cobra.Command, args []string) error {
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

	// Get node stats
	stats, err := esClient.GetNodeStats(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node stats: %w", err)
	}

	// Format and print stats
	statsJSON, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format stats: %w", err)
	}

	fmt.Printf("Stats for node '%s':\n%s\n", nodeID, string(statsJSON))
	return nil
}

// getHotThreads handles the hot threads command
func getHotThreads(cmd *cobra.Command, args []string) error {
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

	// Get hot threads
	hotThreads, err := esClient.GetHotThreads(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get hot threads: %w", err)
	}

	// Print hot threads
	fmt.Println(hotThreads)
	return nil
}
