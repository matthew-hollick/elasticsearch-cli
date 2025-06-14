package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/matthew-hollick/elasticsearch-cli/pkg/client"
	"github.com/matthew-hollick/elasticsearch-cli/pkg/config"
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

	// Settings options
	settingName   string
	settingValue  string
	settingType   string
	includeDefaults bool
	flat          bool

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "es_settings",
		Short: "Manage Elasticsearch cluster settings",
		Long:  `View and modify Elasticsearch cluster settings, including transient and persistent settings.

This command provides comprehensive control over Elasticsearch's cluster-wide configuration settings.
It allows you to view all current settings or filter by specific setting names, and supports both
transient (temporary until restart) and persistent (survives restarts) setting types.

Key capabilities include:
- Listing all cluster settings with optional filtering
- Viewing settings in flat or hierarchical format
- Including default values in the output
- Updating settings with new values
- Specifying setting persistence type

Cluster settings control critical aspects of Elasticsearch behavior including shard allocation,
threading, memory usage, discovery, and more. This command helps you inspect and tune these
settings for optimal performance and stability.

Example usage:
  es_settings
  es_settings --name=cluster.routing.allocation.enable
  es_settings --include-defaults
  es_settings update --name=cluster.routing.allocation.enable --value=none --type=transient`,
		Example: `es_settings
es_settings --name=cluster.routing.allocation.enable
es_settings --include-defaults --flat
es_settings update --name=cluster.routing.allocation.enable --value=none --type=transient`,
		PersistentPreRunE: initConfig,
		RunE:  listSettings, // Default action is to list settings
	}
	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// List settings subcommand (same as root command, but explicit)
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List cluster settings",
		Long:  `List all cluster settings, optionally including default values.`,
		RunE:  listSettings,
	}

	// Get setting subcommand
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get a specific setting",
		Long:  `Get the value of a specific cluster setting.`,
		RunE:  getSetting,
	}

	// Set setting subcommand
	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "Set a cluster setting",
		Long:  `Set a cluster setting to a specific value, either as a transient or persistent setting.`,
		RunE:  setSetting,
	}

	// Reset setting subcommand
	var resetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset a cluster setting",
		Long:  `Reset a cluster setting to its default value.`,
		RunE:  resetSetting,
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

	// List command flags
	rootCmd.Flags().BoolVarP(&includeDefaults, "defaults", "d", false, "Include default settings")
	listCmd.Flags().BoolVarP(&includeDefaults, "defaults", "d", false, "Include default settings")

	// Get command flags
	getCmd.Flags().StringVarP(&settingName, "name", "n", "", "Setting name to get (required)")
	getCmd.Flags().BoolVarP(&includeDefaults, "defaults", "d", true, "Include default settings")
	getCmd.MarkFlagRequired("name")

	// Set command flags
	setCmd.Flags().StringVarP(&settingName, "name", "n", "", "Setting name to set (required)")
	setCmd.Flags().StringVarP(&settingValue, "value", "v", "", "Setting value (required)")
	setCmd.Flags().StringVarP(&settingType, "type", "t", "transient", "Setting type (transient or persistent)")
	setCmd.MarkFlagRequired("name")
	setCmd.MarkFlagRequired("value")

	// Reset command flags
	resetCmd.Flags().StringVarP(&settingName, "name", "n", "", "Setting name to reset (required)")
	resetCmd.Flags().StringVarP(&settingType, "type", "t", "transient", "Setting type (transient or persistent)")
	resetCmd.MarkFlagRequired("name")

	// Add subcommands
	rootCmd.AddCommand(listCmd, getCmd, setCmd, resetCmd)

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

// listSettings handles the list settings command
func listSettings(cmd *cobra.Command, args []string) error {
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

	// Get cluster settings
	settings, err := esClient.GetClusterSettings(includeDefaults)
	if err != nil {
		return fmt.Errorf("failed to get cluster settings: %w", err)
	}

	// Format and print settings
	settingsJSON, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format settings: %w", err)
	}

	fmt.Println(string(settingsJSON))
	return nil
}

// getSetting handles the get setting command
func getSetting(cmd *cobra.Command, args []string) error {
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

	// Get setting value
	value, valueType, err := esClient.GetSettingValue(settingName, includeDefaults)
	if err != nil {
		return fmt.Errorf("failed to get setting value: %w", err)
	}

	// Format and print setting
	fmt.Printf("Setting: %s\n", settingName)
	fmt.Printf("Type: %s\n", valueType)
	
	// Format value as JSON if it's complex
	valueJSON, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format setting value: %w", err)
	}
	fmt.Printf("Value: %s\n", string(valueJSON))

	return nil
}

// setSetting handles the set setting command
func setSetting(cmd *cobra.Command, args []string) error {
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

	// Parse setting value as JSON if it looks like JSON
	var settingValueInterface interface{} = settingValue
	if strings.HasPrefix(settingValue, "{") || strings.HasPrefix(settingValue, "[") {
		if err := json.Unmarshal([]byte(settingValue), &settingValueInterface); err != nil {
			// If it's not valid JSON, use it as a string
			settingValueInterface = settingValue
		}
	} else if settingValue == "true" {
		settingValueInterface = true
	} else if settingValue == "false" {
		settingValueInterface = false
	} else if num, err := json.Number(settingValue).Int64(); err == nil {
		settingValueInterface = num
	} else if num, err := json.Number(settingValue).Float64(); err == nil {
		settingValueInterface = num
	}

	// Prepare settings map
	settings := map[string]interface{}{
		settingName: settingValueInterface,
	}

	// Update setting
	if err := esClient.UpdateClusterSettings(settingType, settings); err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}

	fmt.Printf("Setting %s updated successfully as %s setting\n", settingName, settingType)
	return nil
}

// resetSetting handles the reset setting command
func resetSetting(cmd *cobra.Command, args []string) error {
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

	// Reset setting
	if err := esClient.ResetClusterSetting(settingType, settingName); err != nil {
		return fmt.Errorf("failed to reset setting: %w", err)
	}

	fmt.Printf("Setting %s reset successfully\n", settingName)
	return nil
}
