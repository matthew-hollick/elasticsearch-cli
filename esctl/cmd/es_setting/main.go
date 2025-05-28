package main

import (
	"fmt"
	"log"
	"os"

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
	settingToUpdate string
	valueToUpdate   string
	removeValue     bool

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "es-setting",
		Short: "Interact with cluster settings",
		Long:  `Use the subcommands to view or update cluster settings.`,
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

	// Create update command
	var updateCmd = &cobra.Command{
		Use:              "update",
		Short:            "Create or update a cluster setting",
		Long:             `This command will create a new setting or update an existing cluster setting with the provided value.`,
		PersistentPreRunE: initConfig,
		RunE:             runUpdate,
	}

	// Update command flags
	updateCmd.Flags().StringVarP(&settingToUpdate, "setting", "s", "", "Elasticsearch cluster setting to update (required)")
	updateCmd.MarkFlagRequired("setting")
	updateCmd.Flags().StringVarP(&valueToUpdate, "value", "v", "", "Value of the Elasticsearch cluster setting to update (can't be used with \"--remove\")")
	updateCmd.Flags().BoolVar(&removeValue, "remove", false, "Remove provided cluster setting, resetting it to default configuration (can't be used with \"--value|-v\")")

	// Create get command
	var getCmd = &cobra.Command{
		Use:              "get",
		Short:            "Get cluster settings",
		Long:             `This command will display the cluster's settings.`,
		PersistentPreRunE: initConfig,
		RunE:             runGet,
	}

	// Add commands to root
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(getCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	return config.InitializeConfig(cmd, configFile, addresses, username, password, caCert, insecure, disableRetry, outputFormat)
}

// runUpdate executes the update command
func runUpdate(cmd *cobra.Command, args []string) error {
	// Validate flags
	if cmd.Flags().Changed("value") && cmd.Flags().Changed("remove") {
		fmt.Println("Can't set both \"--value|-v\" and \"--remove\" options")
		fmt.Print(cmd.UsageString())
		os.Exit(1)
	}
	if !cmd.Flags().Changed("value") && !cmd.Flags().Changed("remove") {
		fmt.Println("Error: requires one of \"--value|-v\" or \"--remove\"")
		fmt.Print(cmd.UsageString())
		os.Exit(1)
	}

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

	// Prepare value
	var ptrValueToUpdate *string
	if removeValue {
		ptrValueToUpdate = nil
	} else {
		ptrValueToUpdate = &valueToUpdate
	}

	// Update setting
	existingValue, newValue, err := c.SetClusterSetting(settingToUpdate, ptrValueToUpdate)
	if err != nil {
		return fmt.Errorf("error updating setting %s to %s: %w", 
			settingToUpdate, 
			printableValue(ptrValueToUpdate), 
			err)
	}

	// Output results
	if existingValue == nil {
		fmt.Printf("Created new setting %s\n", settingToUpdate)
		fmt.Printf("\tValue: %s\n", printableValue(newValue))
	} else {
		fmt.Printf("Updated setting %s\n", settingToUpdate)
		fmt.Printf("\tOld value: %s\n", printableValue(existingValue))
		fmt.Printf("\tNew value: %s\n", printableValue(newValue))
	}

	return nil
}

// runGet executes the get command
func runGet(cmd *cobra.Command, args []string) error {
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

	// Get settings
	settings, err := c.GetClusterSettings(true)
	if err != nil {
		return fmt.Errorf("error getting cluster settings: %w", err)
	}

	// Output results
	fmt.Println("Cluster Settings:")
	
	if transient, ok := settings["transient"]; ok && len(transient) > 0 {
		fmt.Println("\nTransient Settings:")
		for k, v := range transient {
			fmt.Printf("\t%s: %v\n", k, v)
		}
	}
	
	if persistent, ok := settings["persistent"]; ok && len(persistent) > 0 {
		fmt.Println("\nPersistent Settings:")
		for k, v := range persistent {
			fmt.Printf("\t%s: %v\n", k, v)
		}
	}
	
	if defaults, ok := settings["defaults"]; ok && len(defaults) > 0 {
		fmt.Println("\nDefault Settings:")
		fmt.Printf("\t%d default settings available\n", len(defaults))
		fmt.Println("\tUse '--format json' to see all default settings")
	}

	return nil
}

// printableValue returns a string representation of a string pointer value
func printableValue(value *string) string {
	if value == nil {
		return "null"
	}
	return *value
}
