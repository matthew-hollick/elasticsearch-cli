package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfigName = "config"
	defaultConfigType = "yaml"
)

// Config holds all configuration for the application
type Config struct {
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch" mapstructure:"elasticsearch"`
	Kibana        KibanaConfig        `yaml:"kibana" mapstructure:"kibana"`
	Output        OutputConfig        `yaml:"output" mapstructure:"output"`
}

// ElasticsearchConfig holds Elasticsearch specific configuration
type ElasticsearchConfig struct {
	Addresses    []string `yaml:"addresses" mapstructure:"addresses"`
	Username     string   `yaml:"username" mapstructure:"username"`
	Password     string   `yaml:"password" mapstructure:"password"`
	CACert       string   `yaml:"ca_cert" mapstructure:"ca_cert"`
	Insecure     bool     `yaml:"insecure" mapstructure:"insecure"`
	DisableRetry bool     `yaml:"disable_retry" mapstructure:"disable_retry"`
}

// KibanaConfig holds Kibana specific configuration
type KibanaConfig struct {
	Addresses []string `yaml:"addresses" mapstructure:"addresses"`
	Username  string   `yaml:"username" mapstructure:"username"`
	Password  string   `yaml:"password" mapstructure:"password"`
	CACert    string   `yaml:"ca_cert" mapstructure:"ca_cert"`
	Insecure  bool     `yaml:"insecure" mapstructure:"insecure"`
}

// OutputConfig holds output formatting configuration
type OutputConfig struct {
	Format string `yaml:"format" mapstructure:"format"` // plain, json, csv
	Style  string `yaml:"style" mapstructure:"style"`  // Style for fancy output format
}

// Context key for viper instance
type contextKey string
const viperKey contextKey = "viper"

// WithViper adds a viper instance to the context
func WithViper(ctx context.Context, v *viper.Viper) context.Context {
	return context.WithValue(ctx, viperKey, v)
}

// FromContext retrieves the viper instance from the context
func FromContext(ctx context.Context) *viper.Viper {
	v, ok := ctx.Value(viperKey).(*viper.Viper)
	if !ok {
		return nil
	}
	return v
}

// Load loads configuration from context, file, and environment variables
func Load(ctx ...context.Context) (*Config, error) {
	var v *viper.Viper
	
	// Check if viper instance is provided in context
	if len(ctx) > 0 && ctx[0] != nil {
		v = FromContext(ctx[0])
	}
	
	// Create new viper instance if not provided
	if v == nil {
		v = viper.New()
		v.SetConfigName(defaultConfigName)
		v.SetConfigType(defaultConfigType)
		v.AddConfigPath(".")                    // Current directory
		v.AddConfigPath("$HOME/.config/esctl") // User config directory
		v.AddConfigPath("/etc/esctl")          // System config directory

		// Set defaults
		v.SetDefault("elasticsearch.addresses", []string{"http://localhost:9200"})
		v.SetDefault("kibana.addresses", []string{"http://localhost:5601"})
		v.SetDefault("output.format", "fancy")
		v.SetDefault("output.style", "dark") // Default style for fancy output

		// Read config file if it exists
		if err := v.ReadInConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if !errors.As(err, &configFileNotFoundError) {
				return nil, fmt.Errorf("error reading config: %w", err)
			}
		}

		// Enable environment variable binding
		v.SetEnvPrefix("ESCTL")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	v := viper.New()
	v.Set("elasticsearch", c.Elasticsearch)
	v.Set("output", c.Output)

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	return v.WriteConfigAs(path)
}

// InitializeConfig provides a standardized way to initialize configuration for Cobra commands
// It handles config file loading, environment variables, and command-line flags
func InitializeConfig(cmd *cobra.Command, configFile string, addresses []string, username, password, caCert string, insecure, disableRetry bool, outputFormat string) error {
	return initializeConfigInternal(cmd, configFile, addresses, username, password, caCert, insecure, disableRetry, nil, "", "", "", false, outputFormat)
}

// InitializeKibanaConfig provides a standardized way to initialize configuration for Cobra commands that use Kibana
// It handles config file loading, environment variables, and Kibana-specific command-line flags
func InitializeKibanaConfig(cmd *cobra.Command, configFile string, kbAddresses []string, kbUsername, kbPassword, kbCaCert string, kbInsecure bool, outputFormat string) error {
	return initializeConfigInternal(cmd, configFile, nil, "", "", "", false, false, kbAddresses, kbUsername, kbPassword, kbCaCert, kbInsecure, outputFormat)
}

// initializeConfigInternal is the internal implementation of InitializeConfig and InitializeKibanaConfig
// It handles both Elasticsearch and Kibana configuration
func initializeConfigInternal(cmd *cobra.Command, configFile string, 
	esAddresses []string, esUsername, esPassword, esCaCert string, esInsecure, esDisableRetry bool,
	kbAddresses []string, kbUsername, kbPassword, kbCaCert string, kbInsecure bool,
	outputFormat string) error {
	v := viper.New()

	// Use config file from the flag if provided
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Use default config locations
		v.SetConfigName(defaultConfigName)
		v.SetConfigType(defaultConfigType)
		v.AddConfigPath(".")                    // Current directory
		v.AddConfigPath("$HOME/.config/esctl") // User config directory
		v.AddConfigPath("/etc/esctl")          // System config directory
	}

	// Set defaults
	v.SetDefault("elasticsearch.addresses", []string{"http://localhost:9200"})
	v.SetDefault("kibana.addresses", []string{"http://localhost:5601"})
	v.SetDefault("output.format", "plain")

	// Read config file if it exists
	if err := v.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	}

	// Enable environment variable binding
	v.SetEnvPrefix("ESCTL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind flags to viper
	// Elasticsearch flags
	if cmd.Flags().Changed("es-addresses") && esAddresses != nil {
		v.Set("elasticsearch.addresses", esAddresses)
	}
	if cmd.Flags().Changed("es-username") && esUsername != "" {
		v.Set("elasticsearch.username", esUsername)
	}
	if cmd.Flags().Changed("es-password") && esPassword != "" {
		v.Set("elasticsearch.password", esPassword)
	}
	if cmd.Flags().Changed("es-ca-cert") && esCaCert != "" {
		v.Set("elasticsearch.ca_cert", esCaCert)
	}
	if cmd.Flags().Changed("es-insecure") {
		v.Set("elasticsearch.insecure", esInsecure)
	}
	if cmd.Flags().Changed("es-disable-retry") {
		v.Set("elasticsearch.disable_retry", esDisableRetry)
	}
	
	// Kibana flags
	if cmd.Flags().Changed("kb-addresses") && kbAddresses != nil {
		v.Set("kibana.addresses", kbAddresses)
	}
	if cmd.Flags().Changed("kb-username") && kbUsername != "" {
		v.Set("kibana.username", kbUsername)
	}
	if cmd.Flags().Changed("kb-password") && kbPassword != "" {
		v.Set("kibana.password", kbPassword)
	}
	if cmd.Flags().Changed("kb-ca-cert") && kbCaCert != "" {
		v.Set("kibana.ca_cert", kbCaCert)
	}
	if cmd.Flags().Changed("kb-insecure") {
		v.Set("kibana.insecure", kbInsecure)
	}
	if cmd.Flags().Changed("format") {
		v.Set("output.format", outputFormat)
	}

	// Store the viper instance in the context for later use
	cmd.SetContext(WithViper(cmd.Context(), v))

	return nil
}
