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
	outputStyle string
	// Config file
	configFile string

	// Kibana connection
	addresses    []string
	username     string
	password     string
	caCert       string
	insecure     bool
	disableRetry bool

	// Command specific
	searchTerm          string
	objectTypes         []string
	includeDependencies bool
	perPage             int
	page                int

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "es_obj_search",
		Short: "Search for Kibana saved objects",
		Long: `Search for Kibana saved objects by name, ID, and type.

Saved objects are used to store content created by Kibana users, such as dashboards, 
visualizations, index patterns, and more. This command allows you to search for saved 
objects across all types or filtered by specific types.

The command returns object details including ID, type, title, last update time, and references
to other saved objects. You can paginate through results and include dependencies.

Example usage:
  es_obj_search --search "dashboard" --type dashboard,visualization
  es_obj_search --search "logs" --include-dependencies
  es_obj_search --per-page 50 --page 2`,
		Example: `es_obj_search --search "dashboard"
es_obj_search --type dashboard,visualization
es_obj_search --search "logs" --include-dependencies --per-page 50`,
		PersistentPreRunE: initConfig,
		RunE:              runSearch,
	}
	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Config file flag
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default is ./config.yaml, ~/.config/esctl/config.yaml, or /etc/esctl/config.yaml)")

	// Kibana connection flags
	rootCmd.PersistentFlags().StringSliceVar(&addresses, "kb-addresses", nil, "Kibana addresses (comma-separated list)")
	rootCmd.PersistentFlags().StringVar(&username, "kb-username", "", "Kibana username")
	rootCmd.PersistentFlags().StringVar(&password, "kb-password", "", "Kibana password")
	rootCmd.PersistentFlags().StringVar(&caCert, "kb-ca-cert", "", "Path to CA certificate for Kibana")
	rootCmd.PersistentFlags().BoolVar(&insecure, "kb-insecure", false, "Skip TLS certificate validation (insecure)")
	rootCmd.PersistentFlags().BoolVar(&disableRetry, "kb-disable-retry", false, "Disable retry on Kibana connection failure")

	// Command specific flags
	rootCmd.Flags().StringVarP(&searchTerm, "search", "s", "", "Search term to filter objects by name or ID")
	rootCmd.Flags().StringSliceVarP(&objectTypes, "type", "t", nil, "Filter by object type (comma-separated list)")
	rootCmd.Flags().BoolVarP(&includeDependencies, "include-dependencies", "d", false, "Include objects that the discovered objects depend on")
	rootCmd.Flags().IntVar(&perPage, "per-page", 20, "Number of results per page")
	rootCmd.Flags().IntVar(&page, "page", 1, "Page number")

	// Output format flag
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	return config.InitializeKibanaConfig(cmd, configFile, addresses, username, password, caCert, insecure, outputFormat)
}

// runSearch executes the search command
func runSearch(cmd *cobra.Command, args []string) error {
	// Get config from context
	cfg, err := config.Load(cmd.Context())
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Create Kibana client
	c, err := client.NewKibana(cfg)
	if err != nil {
		return fmt.Errorf("error creating Kibana client: %w", err)
	}

	// If no types specified, get all available types
	if len(objectTypes) == 0 {
		types, err := c.GetSavedObjectsTypes()
		if err != nil {
			// If we can't get types, just continue without filtering
			fmt.Fprintln(os.Stderr, "Warning: Could not retrieve saved object types, searching across all types")
		} else {
			objectTypes = types
		}
	}

	// Search for saved objects
	response, err := c.SearchSavedObjects(searchTerm, objectTypes, includeDependencies, perPage, page)
	if err != nil {
		return fmt.Errorf("error searching for saved objects: %w", err)
	}

	// Format and output results
	if response.Total == 0 {
		fmt.Println("No saved objects found")
		return nil
	}

	// Print pagination info
	fmt.Printf("Page %d of %d (showing %d of %d results)\n\n",
		response.Page,
		(response.Total+response.PerPage-1)/response.PerPage,
		len(response.SavedObjects),
		response.Total)

	// Create table headers and rows
	headers := []string{"ID", "Type", "Title", "Updated", "References"}
	rows := make([][]string, 0, len(response.SavedObjects))

	// Create table rows
	for _, obj := range response.SavedObjects {
		// Extract title from attributes if available
		title := ""
		if titleVal, ok := obj.Attributes["title"]; ok {
			title = fmt.Sprintf("%v", titleVal)
		} else if nameVal, ok := obj.Attributes["name"]; ok {
			title = fmt.Sprintf("%v", nameVal)
		} else if descVal, ok := obj.Attributes["description"]; ok {
			title = fmt.Sprintf("%v", descVal)
		}

		// Format references
		references := ""
		if len(obj.References) > 0 {
			refStrings := make([]string, 0, len(obj.References))
			for _, ref := range obj.References {
				refStrings = append(refStrings, fmt.Sprintf("%s:%s", ref.Type, ref.ID))
			}
			references = strings.Join(refStrings, ", ")
			if len(references) > 50 {
				references = references[:47] + "..."
			}
		}

		// Add row
		rows = append(rows, []string{
			obj.ID,
			obj.Type,
			title,
			obj.UpdatedAt,
			references,
		})
	}

	// Create formatter and write output
	formatter := format.NewWithStyle(cfg.Output.Format, cfg.Output.Style)
	return formatter.Write(headers, rows)
}
