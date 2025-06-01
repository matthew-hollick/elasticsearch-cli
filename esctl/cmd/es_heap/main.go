package main

import (
	"fmt"
	"log"
	"sort"

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

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:               "es_heap",
		Short:             "Display node heap statistics",
		Long:              `Display detailed Java heap memory usage statistics for all Elasticsearch nodes.

This command provides critical memory usage metrics for each node in your Elasticsearch cluster.
It shows current heap usage, maximum heap size, garbage collection statistics, and memory pressure
indicators. Monitoring heap usage is essential for preventing out-of-memory errors and optimizing
cluster performance.

The output includes:
- Current heap used (absolute and percentage)
- Maximum heap size configured
- Garbage collection frequency and duration
- Memory pressure indicators

Use this command to identify memory-related performance issues, nodes approaching memory limits,
or to verify heap settings across your cluster.

Example usage:
  es_heap --es-addresses=https://elasticsearch:9200 --es-username=elastic --es-password=changeme
  es_heap --format=json
  es_heap --style=blue`,
		Example:          `es_heap
es_heap --format=json
es_heap --style=blue`,
		PersistentPreRunE: initConfig,
		RunE:              run,
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

	// Get node JVM stats
	nodeStats, err := c.GetNodeJVMStats()
	if err != nil {
		return fmt.Errorf("error getting node JVM stats: %w", err)
	}

	// Sort nodes by name
	sort.Slice(nodeStats, func(i, j int) bool {
		return nodeStats[i].Name < nodeStats[j].Name
	})

	// Prepare data for output
	header := []string{"Name", "Role", "Heap Max", "Heap Used", "Heap %", "Non-Heap Committed", "Non-Heap Used"}
	var rows [][]string

	for _, node := range nodeStats {
		row := []string{
			node.Name,
			node.Role,
			client.ByteCountSI(node.JVMStats.HeapMaxBytes),
			client.ByteCountSI(node.JVMStats.HeapUsedBytes),
			fmt.Sprintf("%d%%", node.JVMStats.HeapUsedPercentage),
			client.ByteCountSI(node.JVMStats.NonHeapCommittedBytes),
			client.ByteCountSI(node.JVMStats.NonHeapUsedBytes),
		}
		rows = append(rows, row)
	}

	// Create formatter and output
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	return formatter.Write(header, rows)
}
