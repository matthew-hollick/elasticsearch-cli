package main

import (
	"encoding/json"
	"fmt"
	"log"

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

	// Repository options
	repoName     string
	repoType     string
	repoSettings string
	verify       bool

	// Snapshot options
	snapshotName        string
	indices             []string
	includeGlobalState  bool
	waitForCompletion   bool
	renamePattern       string
	renameReplacement   string

	// Output
	outputFormat string
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "es_snapshot",
		Short: "Manage Elasticsearch snapshots",
		Long:  `Create, restore, and manage Elasticsearch snapshots and repositories.

This command provides comprehensive control over Elasticsearch's backup and restore functionality.
It allows you to manage snapshot repositories (storage locations) and the snapshots themselves.

Key capabilities include:
- Creating and managing snapshot repositories (S3, shared filesystem, etc.)
- Taking full or partial cluster backups
- Restoring indices from snapshots
- Monitoring snapshot status
- Listing and deleting existing snapshots

Snapshots are critical for disaster recovery, data migration, and archiving. This command
provides a streamlined interface for all snapshot-related operations.

Example usage:
  es_snapshot repo list
  es_snapshot repo create --repo-name=my_backups --repo-type=fs --repo-settings='{"location":"/backups"}'  
  es_snapshot create --repo-name=my_backups --snapshot-name=daily_backup
  es_snapshot restore --repo-name=my_backups --snapshot-name=daily_backup --indices=index1,index2`,
		Example: `es_snapshot repo list
es_snapshot create --repo-name=my_backups --snapshot-name=daily_backup
es_snapshot restore --repo-name=my_backups --snapshot-name=daily_backup`,
		PersistentPreRunE: initConfig,
	}
	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Repository commands
	var repoCmd = &cobra.Command{
		Use:   "repo",
		Short: "Manage snapshot repositories",
		Long:  `Create, list, and delete snapshot repositories.

Snapshot repositories are storage locations where Elasticsearch stores backup data. This
command allows you to manage these repositories, including creating new ones with specific
storage types (fs, s3, azure, gcs, etc.), listing existing repositories, and removing them.

When creating repositories, you'll need to specify the repository type and appropriate settings
for that type. For example, a filesystem repository requires a location path, while an S3
repository requires bucket information.

Example usage:
  es_snapshot repo list
  es_snapshot repo create --repo-name=my_backups --repo-type=fs --repo-settings='{"location":"/backups"}'
  es_snapshot repo delete --repo-name=old_backups`,
		Example: `es_snapshot repo list
es_snapshot repo create --repo-name=my_backups --repo-type=fs --repo-settings='{"location":"/backups"}'
es_snapshot repo delete --repo-name=old_backups`,
	}

	var listRepoCmd = &cobra.Command{
		Use:   "list",
		Short: "List snapshot repositories",
		Long:  `List all snapshot repositories.`,
		RunE:  listRepositories,
	}

	var createRepoCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a snapshot repository",
		Long:  `Create a new snapshot repository.`,
		RunE:  createRepository,
	}

	var deleteRepoCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a snapshot repository",
		Long:  `Delete a snapshot repository.`,
		RunE:  deleteRepository,
	}

	// Snapshot commands
	var snapshotCmd = &cobra.Command{
		Use:   "snapshot",
		Short: "Manage snapshots",
		Long:  `Create, list, restore, and delete snapshots.`,
	}

	var listSnapshotCmd = &cobra.Command{
		Use:   "list",
		Short: "List snapshots",
		Long:  `List all snapshots in a repository.`,
		RunE:  listSnapshots,
	}

	var createSnapshotCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a snapshot",
		Long:  `Create a new snapshot in a repository.`,
		RunE:  createSnapshot,
	}

	var deleteSnapshotCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a snapshot",
		Long:  `Delete a snapshot from a repository.`,
		RunE:  deleteSnapshot,
	}

	var restoreSnapshotCmd = &cobra.Command{
		Use:   "restore",
		Short: "Restore a snapshot",
		Long:  `Restore a snapshot from a repository.`,
		RunE:  restoreSnapshot,
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

	// Repository command flags
	createRepoCmd.Flags().StringVarP(&repoName, "name", "n", "", "Repository name (required)")
	createRepoCmd.Flags().StringVarP(&repoType, "type", "t", "fs", "Repository type (fs, url, s3, etc.)")
	createRepoCmd.Flags().StringVarP(&repoSettings, "settings", "s", "", "Repository settings in JSON format (required)")
	createRepoCmd.Flags().BoolVarP(&verify, "verify", "v", true, "Verify repository after creation")
	createRepoCmd.MarkFlagRequired("name")
	createRepoCmd.MarkFlagRequired("settings")

	deleteRepoCmd.Flags().StringVarP(&repoName, "name", "n", "", "Repository name (required)")
	deleteRepoCmd.MarkFlagRequired("name")

	// Snapshot command flags
	listSnapshotCmd.Flags().StringVarP(&repoName, "repo", "r", "", "Repository name (required)")
	listSnapshotCmd.MarkFlagRequired("repo")

	createSnapshotCmd.Flags().StringVarP(&repoName, "repo", "r", "", "Repository name (required)")
	createSnapshotCmd.Flags().StringVarP(&snapshotName, "name", "n", "", "Snapshot name (required)")
	createSnapshotCmd.Flags().StringSliceVarP(&indices, "indices", "i", []string{"_all"}, "Indices to include in snapshot (comma-separated list)")
	createSnapshotCmd.Flags().BoolVarP(&includeGlobalState, "include-global-state", "g", true, "Include global state in snapshot")
	createSnapshotCmd.Flags().BoolVarP(&waitForCompletion, "wait", "w", false, "Wait for snapshot completion")
	createSnapshotCmd.MarkFlagRequired("repo")
	createSnapshotCmd.MarkFlagRequired("name")

	deleteSnapshotCmd.Flags().StringVarP(&repoName, "repo", "r", "", "Repository name (required)")
	deleteSnapshotCmd.Flags().StringVarP(&snapshotName, "name", "n", "", "Snapshot name (required)")
	deleteSnapshotCmd.MarkFlagRequired("repo")
	deleteSnapshotCmd.MarkFlagRequired("name")

	restoreSnapshotCmd.Flags().StringVarP(&repoName, "repo", "r", "", "Repository name (required)")
	restoreSnapshotCmd.Flags().StringVarP(&snapshotName, "name", "n", "", "Snapshot name (required)")
	restoreSnapshotCmd.Flags().StringSliceVarP(&indices, "indices", "i", []string{}, "Indices to restore (comma-separated list)")
	restoreSnapshotCmd.Flags().StringVar(&renamePattern, "rename-pattern", "", "Pattern for renaming indices during restore")
	restoreSnapshotCmd.Flags().StringVar(&renameReplacement, "rename-replacement", "", "Replacement for renaming indices during restore")
	restoreSnapshotCmd.Flags().BoolVarP(&waitForCompletion, "wait", "w", false, "Wait for restore completion")
	restoreSnapshotCmd.MarkFlagRequired("repo")
	restoreSnapshotCmd.MarkFlagRequired("name")

	// Add subcommands
	repoCmd.AddCommand(listRepoCmd, createRepoCmd, deleteRepoCmd)
	snapshotCmd.AddCommand(listSnapshotCmd, createSnapshotCmd, deleteSnapshotCmd, restoreSnapshotCmd)
	rootCmd.AddCommand(repoCmd, snapshotCmd)

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

// listRepositories handles the list repositories command
func listRepositories(cmd *cobra.Command, args []string) error {
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

	// Get repositories
	repositories, err := esClient.GetRepositories()
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	// Format and print repositories
	repoJSON, err := json.MarshalIndent(repositories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format repositories: %w", err)
	}

	fmt.Println(string(repoJSON))
	return nil
}

// createRepository handles the create repository command
func createRepository(cmd *cobra.Command, args []string) error {
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

	// Parse repository settings
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(repoSettings), &settings); err != nil {
		return fmt.Errorf("failed to parse repository settings: %w", err)
	}

	// Create repository
	if err := esClient.CreateRepository(repoName, repoType, settings, verify); err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	fmt.Printf("Repository %s created successfully\n", repoName)
	return nil
}

// deleteRepository handles the delete repository command
func deleteRepository(cmd *cobra.Command, args []string) error {
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

	// Delete repository
	if err := esClient.DeleteRepository(repoName); err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	fmt.Printf("Repository %s deleted successfully\n", repoName)
	return nil
}

// listSnapshots handles the list snapshots command
func listSnapshots(cmd *cobra.Command, args []string) error {
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

	// Get snapshots
	snapshots, err := esClient.GetSnapshots(repoName)
	if err != nil {
		return fmt.Errorf("failed to get snapshots: %w", err)
	}

	// Format and print snapshots
	snapshotJSON, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format snapshots: %w", err)
	}

	fmt.Println(string(snapshotJSON))
	return nil
}

// createSnapshot handles the create snapshot command
func createSnapshot(cmd *cobra.Command, args []string) error {
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

	// Create snapshot
	snapshot, err := esClient.CreateSnapshot(repoName, snapshotName, indices, includeGlobalState, waitForCompletion)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	if waitForCompletion {
		// Format and print snapshot info
		snapshotJSON, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format snapshot info: %w", err)
		}
		fmt.Println(string(snapshotJSON))
	} else {
		fmt.Printf("Snapshot %s creation started in repository %s\n", snapshotName, repoName)
	}

	return nil
}

// deleteSnapshot handles the delete snapshot command
func deleteSnapshot(cmd *cobra.Command, args []string) error {
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

	// Delete snapshot
	if err := esClient.DeleteSnapshot(repoName, snapshotName); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	fmt.Printf("Snapshot %s deleted successfully from repository %s\n", snapshotName, repoName)
	return nil
}

// restoreSnapshot handles the restore snapshot command
func restoreSnapshot(cmd *cobra.Command, args []string) error {
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

	// Restore snapshot
	if err := esClient.RestoreSnapshot(repoName, snapshotName, indices, renamePattern, renameReplacement, waitForCompletion); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	if waitForCompletion {
		fmt.Printf("Snapshot %s from repository %s restored successfully\n", snapshotName, repoName)
	} else {
		fmt.Printf("Snapshot %s restore started from repository %s\n", snapshotName, repoName)
	}

	return nil
}
