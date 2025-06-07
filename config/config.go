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
	Host     string        `mapstructure:"host"`
	Port     int           `mapstructure:"port"`
	User     string        `mapstructure:"user"`
	Password string        `mapstructure:"password"`
	DBName   string        `mapstructure:"dbname"`
	SSLMode  string        `mapstructure:"sslmode"`
	MaxConns int           `mapstructure:"max_conns"`
	Timeout  time.Duration `mapstructure:"timeout"`
	ReadOnly bool          `mapstructure:"read_only"`
}

type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	Timeout       time.Duration `mapstructure:"timeout"`
	MaxRetry      int           `mapstructure:"max_retry"`
	StreamName    string        `mapstructure:"stream_name"`
	StreamSubject string        `mapstructure:"stream_subject"`
	MaxMessages   int           `mapstructure:"max_messages"`
	MaxAge        string        `mapstructure:"max_age"`
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
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "pyrolytics")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_conns", 10)
	viper.SetDefault("database.timeout", 5*time.Second)
	viper.SetDefault("database.read_only", false)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.timeout", 5*time.Second)
	viper.SetDefault("nats.max_retry", 3)
	viper.SetDefault("nats.stream_name", "pyrolytics")
	viper.SetDefault("nats.stream_subject", "pyrolytics.>")
	viper.SetDefault("nats.max_messages", 1000000)
	viper.SetDefault("nats.max_age", "24h")

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
