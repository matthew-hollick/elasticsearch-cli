package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
	esAddresses  []string
	esUsername   string
	esPassword   string
	caCert       string
	insecure     bool
	disableRetry bool

	// Query filtering
	thresholdDuration int
	indexPattern      string
	lastPeriod        string
	queryID           string
	maxResults        int
	showUsername      bool
	hideSensitive     bool

	// Output formatting
	outputFormat string
	outputStyle  string
)

// Query represents an Elasticsearch query with metadata
type Query struct {
	ID          string        `json:"id"`
	Statement   string        `json:"statement"`
	Duration    time.Duration `json:"duration"`
	StartedAt   time.Time     `json:"started_at"`
	Index       string        `json:"index"`
	Username    string        `json:"username,omitempty"`
	Application string        `json:"application,omitempty"`
	Status      string        `json:"status,omitempty"`
}

// QueryAnalysis represents the analysis of slow queries
type QueryAnalysis struct {
	CommonPatterns     []string            `json:"common_patterns"`
	ProblemIndices     []string            `json:"problem_indices"`
	UserSpecificIssues map[string][]string `json:"user_specific_issues"`
	Recommendations    []string            `json:"recommendations"`
}

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "es_long_queries",
		Short: "Manage and analyze long-running Elasticsearch queries",
		Long: `Manage and analyze long-running Elasticsearch queries.
This command provides tools to list, analyze, and terminate long-running queries.

Examples:
  # List all queries running for more than 10 seconds
  es_long_queries list --threshold 10

  # Show slow query history for the last 24 hours
  es_long_queries history --last 24h

  # Kill a specific query by ID
  es_long_queries kill --query-id AbCdEf123

  # Analyze slow query patterns
  es_long_queries analyze --last 7d`,
		SilenceUsage: true,
	}

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List currently running long queries",
		Long: `List currently running long queries in Elasticsearch.
Queries can be filtered by duration threshold and index pattern.

Examples:
  # List all queries running for more than 10 seconds
  es_long_queries list --threshold 10

  # List long queries on a specific index pattern
  es_long_queries list --index logs-*

  # List long queries with username information
  es_long_queries list --show-user

  # List long queries with masked usernames
  es_long_queries list --show-user --hide-sensitive`,
		RunE: runListCommand,
	}

	// History subcommand
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Show historical slow query logs",
		Long: `Show historical slow query logs from Elasticsearch.
Queries can be filtered by time period and index pattern.

Examples:
  # Show slow queries from the last hour
  es_long_queries history --last 1h

  # Show slow queries from the last 24 hours on specific indices
  es_long_queries history --last 24h --index logs-*

  # Show slow queries with username information
  es_long_queries history --last 12h --show-user

  # Limit the number of results
  es_long_queries history --last 24h --max-results 50`,
		RunE: runHistoryCommand,
	}

	// Kill subcommand
	killCmd := &cobra.Command{
		Use:   "kill",
		Short: "Terminate a running query",
		Long: `Terminate a running query in Elasticsearch by its ID.
The query ID can be obtained from the 'list' command.

Examples:
  # Kill a specific query
  es_long_queries kill --query-id AbCdEf123`,
		RunE: runKillCommand,
	}

	// Analyze subcommand
	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze slow query patterns",
		Long: `Analyze slow query patterns and provide optimization suggestions.
Analysis is performed on historical slow query data.

Examples:
  # Analyze slow queries from the last 24 hours
  es_long_queries analyze --last 24h

  # Analyze slow queries on specific indices
  es_long_queries analyze --last 7d --index logs-*

  # Include username information in analysis
  es_long_queries analyze --last 3d --show-user`,
		RunE: runAnalyzeCommand,
	}

	// Add subcommands to root command
	rootCmd.AddCommand(listCmd, historyCmd, killCmd, analyzeCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.esctl.yaml)")
	rootCmd.PersistentFlags().StringSliceVar(&esAddresses, "es-address", []string{}, "Elasticsearch address (can be specified multiple times)")
	rootCmd.PersistentFlags().StringVar(&esUsername, "es-username", "", "Elasticsearch username")
	rootCmd.PersistentFlags().StringVar(&esPassword, "es-password", "", "Elasticsearch password")
	rootCmd.PersistentFlags().StringVar(&caCert, "ca-cert", "", "Path to CA certificate file")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "Skip TLS verification")
	rootCmd.PersistentFlags().BoolVar(&disableRetry, "disable-retry", false, "Disable retry on failure")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().StringVar(&outputStyle, "style", "default", "Output style (default, light, dark)")

	// Subcommand flags
	listCmd.Flags().IntVar(&thresholdDuration, "threshold", 30, "Minimum query duration in seconds")
	listCmd.Flags().StringVar(&indexPattern, "index", "*", "Index pattern to filter queries")
	listCmd.Flags().IntVar(&maxResults, "max-results", 100, "Maximum number of results to display")
	listCmd.Flags().BoolVar(&showUsername, "show-user", true, "Show username information")
	listCmd.Flags().BoolVar(&hideSensitive, "hide-sensitive", false, "Mask sensitive information like usernames")

	historyCmd.Flags().StringVar(&lastPeriod, "last", "1h", "Time period to look back (e.g., 1h, 6h, 24h, 7d)")
	historyCmd.Flags().StringVar(&indexPattern, "index", "*", "Index pattern to filter queries")
	historyCmd.Flags().IntVar(&maxResults, "max-results", 100, "Maximum number of results to display")
	historyCmd.Flags().BoolVar(&showUsername, "show-user", true, "Show username information")
	historyCmd.Flags().BoolVar(&hideSensitive, "hide-sensitive", false, "Mask sensitive information like usernames")

	killCmd.Flags().StringVar(&queryID, "query-id", "", "ID of the query to terminate")
	_ = killCmd.MarkFlagRequired("query-id")

	analyzeCmd.Flags().StringVar(&lastPeriod, "last", "24h", "Time period to analyze (e.g., 24h, 7d, 30d)")
	analyzeCmd.Flags().StringVar(&indexPattern, "index", "*", "Index pattern to filter queries")
	analyzeCmd.Flags().BoolVar(&showUsername, "show-user", true, "Include username information in analysis")
	analyzeCmd.Flags().BoolVar(&hideSensitive, "hide-sensitive", false, "Mask sensitive information like usernames")

	// Bind flags to viper
	viper.BindPFlags(rootCmd.PersistentFlags())
	viper.BindPFlags(listCmd.Flags())
	viper.BindPFlags(historyCmd.Flags())
	viper.BindPFlags(killCmd.Flags())
	viper.BindPFlags(analyzeCmd.Flags())

	// Set up cobra
	cobra.OnInitialize(initConfig)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if configFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".esctl" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigName(".esctl")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// runListCommand handles the "list" subcommand
func runListCommand(cmd *cobra.Command, args []string) error {
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

	// Get running queries
	queries, err := getRunningQueries(esClient, thresholdDuration, indexPattern)
	if err != nil {
		return fmt.Errorf("failed to get running queries: %w", err)
	}

	// Process usernames if needed
	if showUsername && hideSensitive {
		for i := range queries {
			queries[i].Username = maskUsername(queries[i].Username)
		}
	} else if !showUsername {
		for i := range queries {
			queries[i].Username = ""
		}
	}

	// Format and display results
	return formatAndDisplayQueries(cmd, queries, "Running Queries")
}

// runHistoryCommand handles the "history" subcommand
func runHistoryCommand(cmd *cobra.Command, args []string) error {
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

	// Parse time period
	duration, err := parseDuration(lastPeriod)
	if err != nil {
		return fmt.Errorf("invalid time period format: %w", err)
	}

	// Get historical slow queries
	queries, err := getHistoricalSlowQueries(esClient, duration, indexPattern, maxResults)
	if err != nil {
		return fmt.Errorf("failed to get historical slow queries: %w", err)
	}

	// Process usernames if needed
	if showUsername && hideSensitive {
		for i := range queries {
			queries[i].Username = maskUsername(queries[i].Username)
		}
	} else if !showUsername {
		for i := range queries {
			queries[i].Username = ""
		}
	}

	// Format and display results
	return formatAndDisplayQueries(cmd, queries, "Historical Slow Queries")
}

// runKillCommand handles the "kill" subcommand
func runKillCommand(cmd *cobra.Command, args []string) error {
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

	// Validate query ID
	if !isValidQueryID(queryID) {
		return fmt.Errorf("invalid query ID format: %s", queryID)
	}

	// Terminate query
	err = terminateQuery(esClient, queryID)
	if err != nil {
		return fmt.Errorf("failed to terminate query: %w", err)
	}

	fmt.Printf("Query %s successfully terminated\n", queryID)
	return nil
}

// runAnalyzeCommand handles the "analyze" subcommand
func runAnalyzeCommand(cmd *cobra.Command, args []string) error {
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

	// Parse time period
	duration, err := parseDuration(lastPeriod)
	if err != nil {
		return fmt.Errorf("invalid time period format: %w", err)
	}

	// Get historical slow queries for analysis
	queries, err := getHistoricalSlowQueries(esClient, duration, indexPattern, maxResults)
	if err != nil {
		return fmt.Errorf("failed to get historical slow queries: %w", err)
	}

	// Process usernames if needed
	if hideSensitive {
		for i := range queries {
			queries[i].Username = maskUsername(queries[i].Username)
		}
	} else if !showUsername {
		for i := range queries {
			queries[i].Username = ""
		}
	}

	// Analyze queries
	analysis, err := analyzeSlowQueries(queries)
	if err != nil {
		return fmt.Errorf("failed to analyze queries: %w", err)
	}

	// Display analysis results
	return displayAnalysisResults(cmd, analysis)
}

// getRunningQueries retrieves currently running queries from Elasticsearch
func getRunningQueries(esClient *client.Client, thresholdSecs int, indexPattern string) ([]Query, error) {
	// This is a placeholder implementation
	// In a real implementation, this would call the Elasticsearch API
	// to get running queries filtered by duration and index pattern

	// Mock data for demonstration
	queries := []Query{
		{
			ID:          "task_id_1",
			Statement:   "SELECT * FROM logs WHERE timestamp > now() - 1h",
			Duration:    time.Duration(35) * time.Second,
			StartedAt:   time.Now().Add(-35 * time.Second),
			Index:       "logs-*",
			Username:    "alice",
			Application: "Kibana",
			Status:      "RUNNING",
		},
		{
			ID:          "task_id_2",
			Statement:   "SELECT COUNT(*) FROM events GROUP BY category",
			Duration:    time.Duration(120) * time.Second,
			StartedAt:   time.Now().Add(-120 * time.Second),
			Index:       "events-*",
			Username:    "bob",
			Application: "Grafana",
			Status:      "RUNNING",
		},
		{
			ID:          "task_id_3",
			Statement:   "SELECT * FROM metrics WHERE host = 'server1' ORDER BY timestamp DESC LIMIT 1000",
			Duration:    time.Duration(45) * time.Second,
			StartedAt:   time.Now().Add(-45 * time.Second),
			Index:       "metrics-*",
			Username:    "charlie",
			Application: "Custom App",
			Status:      "RUNNING",
		},
	}

	// Filter by threshold duration
	thresholdDuration := time.Duration(thresholdSecs) * time.Second
	var filtered []Query
	for _, q := range queries {
		if q.Duration >= thresholdDuration && (indexPattern == "*" || strings.Contains(q.Index, strings.TrimSuffix(indexPattern, "*"))) {
			filtered = append(filtered, q)
		}
	}

	return filtered, nil
}

// getHistoricalSlowQueries retrieves historical slow queries from Elasticsearch
func getHistoricalSlowQueries(esClient *client.Client, lookbackPeriod time.Duration, indexPattern string, limit int) ([]Query, error) {
	// This is a placeholder implementation
	// In a real implementation, this would call the Elasticsearch API
	// to get historical slow queries from the slow query log

	// Mock data for demonstration
	queries := []Query{
		{
			ID:          "query_id_1",
			Statement:   "SELECT * FROM logs WHERE timestamp > now() - 24h",
			Duration:    time.Duration(5) * time.Second,
			StartedAt:   time.Now().Add(-6 * time.Hour),
			Index:       "logs-*",
			Username:    "alice",
			Application: "Kibana",
			Status:      "COMPLETED",
		},
		{
			ID:          "query_id_2",
			Statement:   "SELECT COUNT(*) FROM events GROUP BY category, subcategory, host",
			Duration:    time.Duration(12) * time.Second,
			StartedAt:   time.Now().Add(-12 * time.Hour),
			Index:       "events-*",
			Username:    "bob",
			Application: "Grafana",
			Status:      "COMPLETED",
		},
		{
			ID:          "query_id_3",
			Statement:   "SELECT * FROM metrics WHERE host IN (SELECT id FROM hosts WHERE environment = 'production')",
			Duration:    time.Duration(8) * time.Second,
			StartedAt:   time.Now().Add(-18 * time.Hour),
			Index:       "metrics-*",
			Username:    "charlie",
			Application: "Custom App",
			Status:      "COMPLETED",
		},
	}

	// Filter by lookback period and index pattern
	cutoffTime := time.Now().Add(-lookbackPeriod)
	var filtered []Query
	for _, q := range queries {
		if q.StartedAt.After(cutoffTime) && (indexPattern == "*" || strings.Contains(q.Index, strings.TrimSuffix(indexPattern, "*"))) {
			filtered = append(filtered, q)
		}
	}

	// Limit results
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

// terminateQuery terminates a running query in Elasticsearch
func terminateQuery(esClient *client.Client, id string) error {
	// This is a placeholder implementation
	// In a real implementation, this would call the Elasticsearch API
	// to terminate a running query by its ID
	return nil
}

// analyzeSlowQueries analyzes patterns in slow queries
func analyzeSlowQueries(queries []Query) (*QueryAnalysis, error) {
	// This is a placeholder implementation
	// In a real implementation, this would analyze the query patterns
	// and provide meaningful recommendations

	analysis := &QueryAnalysis{
		CommonPatterns: []string{
			"Full table scans without appropriate filters",
			"Complex aggregations on high cardinality fields",
			"Queries missing relevant indices",
		},
		ProblemIndices: []string{
			"logs-*",
			"events-*",
		},
		UserSpecificIssues: map[string][]string{
			"alice": {"Consistently querying over large time ranges"},
			"bob":   {"Complex grouping operations without optimized mappings"},
		},
		Recommendations: []string{
			"Add explicit time range filters to all queries",
			"Create field-specific indices for commonly queried fields",
			"Use keyword fields instead of text fields for filtering and aggregations",
			"Consider using rollup indices for historical data",
			"Implement query result caching for frequent queries",
		},
	}

	return analysis, nil
}

// maskUsername masks a username for privacy
func maskUsername(username string) string {
	if username == "" {
		return ""
	}
	
	if len(username) <= 2 {
		return username[0:1] + "***"
	}
	
	return username[0:2] + "***"
}

// formatAndDisplayQueries formats and displays query results
func formatAndDisplayQueries(cmd *cobra.Command, queries []Query, title string) error {
	// Load configuration with context containing viper instance
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Get formatter
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	
	// Format based on output format
	switch cfg.Output.Format {
	case "table":
		return outputTable(formatter, queries, title)
	case "json":
		return outputJSON(queries)
	case "yaml":
		return outputYAML(queries)
	default:
		return fmt.Errorf("unsupported output format: %s", cfg.Output.Format)
	}
}

// outputTable formats and displays queries in table format
func outputTable(formatter *format.Formatter, queries []Query, title string) error {
	// Set up table headers
	headers := []string{"ID", "Duration", "Started At", "Index", "Status"}
	if showUsername {
		headers = append(headers, "Username")
	}
	headers = append(headers, "Query")
	
	// Create rows
	var rows [][]string
	for _, q := range queries {
		// Format duration
		durationStr := q.Duration.String()
		
		// Format time
		startedStr := q.StartedAt.Format(time.RFC3339)
		
		// Create row
		row := []string{
			q.ID,
			durationStr,
			startedStr,
			q.Index,
			q.Status,
		}
		
		// Add username if requested
		if showUsername {
			row = append(row, q.Username)
		}
		
		// Add query statement (truncate if too long)
		statement := q.Statement
		if len(statement) > 80 {
			statement = statement[:77] + "..."
		}
		row = append(row, statement)
		
		rows = append(rows, row)
	}
	
	// Write table to output
	return formatter.Write(headers, rows)
}

// outputJSON formats and displays queries in JSON format
func outputJSON(queries []Query) error {
	data, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// outputYAML formats and displays queries in YAML format
func outputYAML(queries []Query) error {
	// For simplicity, we'll use JSON output with a note that this would be YAML in a real implementation
	fmt.Println("# YAML output would be implemented here")
	return outputJSON(queries)
}

// displayAnalysisResults formats and displays query analysis results
func displayAnalysisResults(cmd *cobra.Command, analysis *QueryAnalysis) error {
	// For now, we'll directly print the analysis results regardless of the output format
	// In a real implementation, this could be formatted according to the output format
	
	// Display common patterns
	fmt.Println("Common Query Patterns:")
	for i, pattern := range analysis.CommonPatterns {
		fmt.Printf("  %d. %s\n", i+1, pattern)
	}
	fmt.Println()
	
	// Display problem indices
	fmt.Println("Problem Indices:")
	for i, index := range analysis.ProblemIndices {
		fmt.Printf("  %d. %s\n", i+1, index)
	}
	fmt.Println()
	
	// Display user-specific issues if username display is enabled
	if showUsername {
		fmt.Println("User-Specific Issues:")
		for user, issues := range analysis.UserSpecificIssues {
			displayUser := user
			if hideSensitive {
				displayUser = maskUsername(user)
			}
			fmt.Printf("  %s:\n", displayUser)
			for i, issue := range issues {
				fmt.Printf("    %d. %s\n", i+1, issue)
			}
		}
		fmt.Println()
	}
	
	// Display recommendations
	fmt.Println("Recommendations:")
	for i, rec := range analysis.Recommendations {
		fmt.Printf("  %d. %s\n", i+1, rec)
	}
	
	return nil
}

// parseDuration parses a duration string like "24h" or "7d"
func parseDuration(s string) (time.Duration, error) {
	// Handle day format by converting to hours
	if strings.HasSuffix(s, "d") {
		days, err := time.ParseDuration(strings.TrimSuffix(s, "d") + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}
	
	// Parse standard duration format
	return time.ParseDuration(s)
}

// isValidQueryID validates the format of a query ID
func isValidQueryID(id string) bool {
	// This is a placeholder implementation
	// In a real implementation, this would validate the query ID format
	return id != ""
}
