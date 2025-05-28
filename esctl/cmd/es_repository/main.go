package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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
	repositoryName string
	repositoryType string
	settings       map[string]string

	// Output
	outputFormat string
)

func main() {
	// Create root command
	var rootCmd = &cobra.Command{
		Use:               "es-repository",
		Short:             "Interact with snapshot repositories",
		Long:              `Use the list, verify, register, and remove subcommands to manage snapshot repositories.`,
		PersistentPreRunE: initConfig,
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

	// Create list command
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List configured snapshot repositories",
		Long:  `This command will list all the snapshot repositories on the cluster.`,
		RunE:  runList,
	}

	// Create verify command
	var verifyCmd = &cobra.Command{
		Use:   "verify",
		Short: "Verify the specified repository",
		Long:  `This command will verify the repository is configured correctly on all nodes.`,
		RunE:  runVerify,
	}
	verifyCmd.Flags().StringVarP(&repositoryName, "repository", "r", "", "Snapshot repository to verify (required)")
	verifyCmd.MarkFlagRequired("repository")

	// Create register command
	var registerCmd = &cobra.Command{
		Use:   "register",
		Short: "Register a snapshot repository",
		Long:  `This command will register a new snapshot repository.`,
		RunE:  runRegister,
	}
	registerCmd.Flags().StringVarP(&repositoryName, "repository", "r", "", "Snapshot repository name to register (required)")
	registerCmd.MarkFlagRequired("repository")
	registerCmd.Flags().StringVarP(&repositoryType, "type", "t", "", "Type of snapshot repository to register (required)")
	registerCmd.MarkFlagRequired("type")
	registerCmd.Flags().StringToStringVarP(&settings, "settings", "s", map[string]string{}, "Settings of the repository to register in key value pairs, i.e. location=/backups,compress=true")

	// Create remove command
	var removeCmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove a snapshot repository",
		Long:  `This command will remove the specified snapshot repository.`,
		RunE:  runRemove,
	}
	removeCmd.Flags().StringVarP(&repositoryName, "repository", "r", "", "Snapshot repository to remove (required)")
	removeCmd.MarkFlagRequired("repository")

	// Add subcommands to root
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(removeCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	return config.InitializeConfig(cmd, configFile, addresses, username, password, caCert, insecure, disableRetry, outputFormat)
}

// runList lists all repositories
func runList(cmd *cobra.Command, args []string) error {
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

	// Get repositories
	repos, err := c.GetRepositories()
	if err != nil {
		return fmt.Errorf("error getting repositories: %w", err)
	}

	// Prepare data for output
	header := []string{"Name", "Type", "Settings"}
	var rows [][]string

	for name, repo := range repos {
		// Format settings as a string
		var settingsStrings []string
		for k, v := range repo.Settings {
			settingsStrings = append(settingsStrings, fmt.Sprintf("%s: %v", k, v))
		}

		row := []string{
			name,
			repo.Type,
			strings.Join(settingsStrings, "\n"),
		}
		rows = append(rows, row)
	}

	// Create formatter and output
	formatter := format.New(cfg.Output.Format)
	return formatter.Write(header, rows)
}

// runVerify verifies a repository
func runVerify(cmd *cobra.Command, args []string) error {
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

	// Verify repository
	verified, err := c.VerifyRepository(repositoryName)
	if err != nil {
		return fmt.Errorf("error verifying repository %s: %w", repositoryName, err)
	}

	// Output result
	if verified {
		fmt.Fprintf(cmd.OutOrStdout(), "Repository %s is verified.\n", repositoryName)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Repository %s is NOT verified.\n", repositoryName)
		os.Exit(1)
	}

	return nil
}

// runRegister registers a repository
func runRegister(cmd *cobra.Command, args []string) error {
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

	// Convert settings to interface map
	settingsInterface := make(map[string]interface{}, len(settings))
	for k, v := range settings {
		settingsInterface[k] = v
	}

	// Create repository
	err = c.CreateRepository(repositoryName, repositoryType, settingsInterface, true)
	if err != nil {
		return fmt.Errorf("error registering repository %s: %w", repositoryName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Repository %s registered successfully.\n", repositoryName)
	return nil
}

// runRemove removes a repository
func runRemove(cmd *cobra.Command, args []string) error {
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

	// Delete repository
	err = c.DeleteRepository(repositoryName)
	if err != nil {
		return fmt.Errorf("error removing repository %s: %w", repositoryName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Repository %s removed successfully.\n", repositoryName)
	return nil
}
