package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Format string `yaml:"format" mapstructure:"format"` // rich, plain, json, csv
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
		v.SetDefault("output.format", "rich")

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
