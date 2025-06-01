package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

	// Kibana connection
	addresses    []string
	username     string
	password     string
	caCert       string
	insecure     bool

	// Command specific
	objectID            string
	objectType          string
	includeDependencies bool
	outputDir           string
	outputFilename      string

	// Output
	outputFormat string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "es_obj_export",
		Short: "Export Kibana saved objects",
		Long: `Export Kibana saved objects to NDJSON files.

This command exports Kibana saved objects (dashboards, visualizations, index patterns, etc.) 
to NDJSON files that can be imported into other Kibana instances. You must specify both the 
object ID and type to export.

The exported file will be saved in the specified output directory with either a custom filename
or a filename derived from the object's title. The file will have a .ndjson extension.

You can optionally include all dependencies of the specified object, which ensures that all
referenced objects are included in the export file.

Example usage:
  es_obj_export --id my-dashboard-id --type dashboard
  es_obj_export --id my-dashboard-id --type dashboard --include-dependencies
  es_obj_export --id my-dashboard-id --type dashboard --output-dir /path/to/exports --filename custom-name`,
		Example: `es_obj_export --id my-dashboard-id --type dashboard
es_obj_export --id my-dashboard-id --type dashboard --include-dependencies
es_obj_export --id my-dashboard-id --type dashboard --output-dir ./exports --filename my-export`,
		PersistentPreRunE: initConfig,
		RunE:              runExport,
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

	// Command specific flags
	rootCmd.Flags().StringVarP(&objectID, "id", "i", "", "ID of the object to export")
	rootCmd.MarkFlagRequired("id")
	
	rootCmd.Flags().StringVarP(&objectType, "type", "t", "", "Type of the object to export")
	rootCmd.MarkFlagRequired("type")
	
	rootCmd.Flags().BoolVarP(&includeDependencies, "include-dependencies", "d", false, "Include objects that the specified object depends on")
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to save the exported file")
	rootCmd.Flags().StringVarP(&outputFilename, "filename", "f", "", "Custom filename for the exported file (without extension)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig(cmd *cobra.Command, args []string) error {
	return config.InitializeKibanaConfig(cmd, configFile, addresses, username, password, caCert, insecure, outputFormat)
}

// runExport executes the export command
func runExport(cmd *cobra.Command, args []string) error {
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

	// First, get the object to determine its name/title if no custom filename provided
	if outputFilename == "" {
		obj, err := c.GetSavedObject(objectID, objectType, false)
		if err != nil {
			return fmt.Errorf("error retrieving object details: %w", err)
		}

		// Extract title from attributes if available
		title := objectID // Default to ID if no title found
		if titleVal, ok := obj.Attributes["title"]; ok {
			title = fmt.Sprintf("%v", titleVal)
		} else if nameVal, ok := obj.Attributes["name"]; ok {
			title = fmt.Sprintf("%v", nameVal)
		} else if descVal, ok := obj.Attributes["description"]; ok {
			title = fmt.Sprintf("%v", descVal)
		}

		// Sanitize the title for use as a filename
		outputFilename = sanitizeFilename(title)
	}

	// Export the object
	data, err := c.ExportSavedObject(objectID, objectType, includeDependencies)
	if err != nil {
		return fmt.Errorf("error exporting object: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Build the full file path
	filePath := filepath.Join(outputDir, outputFilename+".ndjson")

	// Write the data to the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	fmt.Printf("Successfully exported %s to %s\n", objectType, filePath)
	if includeDependencies {
		fmt.Println("Dependencies were included in the export")
	}

	return nil
}

// sanitizeFilename sanitizes a string for use as a filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = reg.ReplaceAllString(name, "_")

	// Trim spaces and limit length
	name = strings.TrimSpace(name)
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}
