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

	// Filter options
	nodes       []string
	indices     []string
	states      []string
	primaryOnly bool

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:              "es_shards",
		Short:            "Display Elasticsearch shard allocation",
		Long:             `Display Elasticsearch shard allocation by node, including unallocated shards and their reasons.`,
		PersistentPreRunE: initConfig,
		RunE:             run,
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

	// Filter flags
	rootCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nil, "Filter by node names (comma-separated list)")
	rootCmd.PersistentFlags().StringSliceVarP(&indices, "indices", "i", nil, "Filter by index names (comma-separated list)")
	rootCmd.PersistentFlags().StringSliceVarP(&states, "states", "s", nil, "Filter by shard states (comma-separated list)")
	rootCmd.PersistentFlags().BoolVarP(&primaryOnly, "primary", "p", false, "Show only primary shards")

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

	// Initialize client
	esClient, err := client.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Get shards by node
	shardsByNode, unassignedShards, err := esClient.GetShardsByNode(nodes)
	if err != nil {
		return fmt.Errorf("failed to get shards: %w", err)
	}

	// Create formatter
	formatter := format.New(cfg.Output.Format)

	// Print allocated shards by node
	if len(shardsByNode) > 0 {
		for node, shards := range shardsByNode {
			// Filter shards if needed
			filteredShards := filterShards(shards, indices, states, primaryOnly)
			if len(filteredShards) == 0 {
				continue
			}

			fmt.Printf("\nNode: %s\n", node)
			
			// Prepare table data
			header := []string{"Index", "Shard", "Type", "State", "Docs", "Store"}
			rows := [][]string{}
			
			for _, shard := range filteredShards {
				shardType := "replica"
				if shard.PrimaryOrReplica == "p" {
					shardType = "primary"
				}
				
				row := []string{
					shard.Index,
					shard.Shard,
					shardType,
					shard.State,
					shard.Docs,
					shard.Store,
				}
				rows = append(rows, row)
			}
			
			// Print table
			if err := formatter.Write(header, rows); err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}
		}
	}

	// Print unassigned shards if any
	filteredUnassigned := filterShards(unassignedShards, indices, states, primaryOnly)
	if len(filteredUnassigned) > 0 {
		fmt.Printf("\nUnassigned Shards:\n")
		
		// Prepare table data
		header := []string{"Index", "Shard", "Type", "Reason", "Unassigned For", "Details"}
		rows := [][]string{}
		
		for _, shard := range filteredUnassigned {
			shardType := "replica"
			if shard.PrimaryOrReplica == "p" {
				shardType = "primary"
			}
			
			row := []string{
				shard.Index,
				shard.Shard,
				shardType,
				shard.UnassignedReason,
				shard.UnassignedFor,
				shard.UnassignedDetails,
			}
			rows = append(rows, row)
		}
		
		// Print table
		if err := formatter.Write(header, rows); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
	}

	return nil
}

// filterShards applies filters to the shard list
func filterShards(shards []client.ShardInfo, indices, states []string, primaryOnly bool) []client.ShardInfo {
	if len(indices) == 0 && len(states) == 0 && !primaryOnly {
		return shards
	}

	var filtered []client.ShardInfo
	for _, shard := range shards {
		// Filter by primary
		if primaryOnly && shard.PrimaryOrReplica != "p" {
			continue
		}

		// Filter by index
		if len(indices) > 0 {
			matchIndex := false
			for _, idx := range indices {
				if strings.Contains(shard.Index, idx) {
					matchIndex = true
					break
				}
			}
			if !matchIndex {
				continue
			}
		}

		// Filter by state
		if len(states) > 0 {
			matchState := false
			for _, state := range states {
				if strings.EqualFold(shard.State, state) {
					matchState = true
					break
				}
			}
			if !matchState {
				continue
			}
		}

		filtered = append(filtered, shard)
	}

	return filtered
}
