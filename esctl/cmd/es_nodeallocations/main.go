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

	// Command specific
	shortOutput bool

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:               "es-nodeallocations",
		Short:             "Display node disk allocations",
		Long:              `Show disk allocation information for all nodes in the Elasticsearch cluster.`,
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

	// Command specific flags
	rootCmd.Flags().BoolVarP(&shortOutput, "short", "s", false, "Shorter, more compact table output")

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

	// Get node allocations
	nodes, err := c.GetNodeAllocations()
	if err != nil {
		return fmt.Errorf("error getting node allocations: %w", err)
	}

	// Sort nodes by name
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	// Prepare data for output
	var header []string
	var rows [][]string

	if shortOutput {
		header = []string{"Role", "Name", "Avail", "Used", "Total", "%", "Indices", "Shards", "IP"}
		for _, node := range nodes {
			row := []string{
				fmt.Sprintf("%s%s", node.Master, node.Role),
				node.Name,
				node.DiskAvail,
				node.DiskUsed,
				node.DiskTotal,
				node.DiskPercent,
				node.DiskIndices,
				node.Shards,
				node.IP,
			}
			rows = append(rows, row)
		}
	} else {
		header = []string{"Master", "Role", "Name", "Disk Avail", "Disk Indices", "Disk Percent", "Disk Total", "Disk Used", "Shards", "IP", "ID", "JDK", "Version"}
		for _, node := range nodes {
			row := []string{
				node.Master,
				node.Role,
				node.Name,
				node.DiskAvail,
				node.DiskIndices,
				node.DiskPercent,
				node.DiskTotal,
				node.DiskUsed,
				node.Shards,
				node.IP,
				node.ID,
				node.Jdk,
				node.Version,
			}
			rows = append(rows, row)
		}
	}

	// Create formatter and output
	formatter := format.New(cfg.Output.Format)
	return formatter.Write(header, rows)
}
