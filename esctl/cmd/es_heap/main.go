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
		Use:               "es-heap",
		Short:             "Display node heap statistics",
		Long:              `Show node heap stats and settings for the Elasticsearch cluster.`,
		PersistentPreRunE: initConfig,
		RunE:              run,
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
	formatter := format.New(cfg.Output.Format)
	return formatter.Write(header, rows)
}
