package main

import (
	"fmt"
	"log"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
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
	nodesToGetHotThreads []string

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:               "es-hotthreads",
		Short:             "Display the current hot threads by node in the cluster",
		Long:              `Show the current hot threads across a set of nodes within the Elasticsearch cluster.`,
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
	rootCmd.Flags().StringArrayVarP(&nodesToGetHotThreads, "nodes", "n", []string{}, "Elasticsearch nodes to get hot threads for (optional, omitted will include all nodes)")

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

	var threads string
	if len(nodesToGetHotThreads) == 0 {
		// Get hot threads for all nodes
		threads, err = c.GetHotThreads()
		if err != nil {
			return fmt.Errorf("error getting hot threads: %w", err)
		}
	} else {
		// Get hot threads for specific nodes
		threads, err = c.GetNodesHotThreads(nodesToGetHotThreads)
		if err != nil {
			return fmt.Errorf("error getting hot threads for nodes %v: %w", nodesToGetHotThreads, err)
		}
	}

	// Output the hot threads
	fmt.Fprintln(cmd.OutOrStdout(), threads)
	return nil
}
