package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Cache    CacheConfig    `mapstructure:"cache"`
}

type ServerConfig struct {
	Port int
	Host string
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
	// Limbo specific configurations
	MaxConns    int           `mapstructure:"max_conns"`
	Timeout     time.Duration `mapstructure:"timeout"`
	ReadOnly    bool          `mapstructure:"read_only"`
	JournalMode string        `mapstructure:"journal_mode"`
}

type NATSConfig struct {
	URL string
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	TTL time.Duration `mapstructure:"ttl"`
}

// LoadConfig loads the configuration from file and environment variables
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("No config file found, using defaults")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.read_timeout", 5*time.Second)
	viper.SetDefault("server.write_timeout", 10*time.Second)

	// Database defaults
	viper.SetDefault("database.path", "pyrolytic.db")
	viper.SetDefault("database.max_conns", 10)
	viper.SetDefault("database.timeout", 5*time.Second)
	viper.SetDefault("database.read_only", false)
	viper.SetDefault("database.journal_mode", "WAL")

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.timeout", 5*time.Second)
	viper.SetDefault("nats.max_retry", 3)

	// Cache defaults
	viper.SetDefault("cache.ttl", 1*time.Hour)
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Cache.TTL <= 0 {
		return fmt.Errorf("invalid cache TTL: %v", cfg.Cache.TTL)
	}

	return nil
}
