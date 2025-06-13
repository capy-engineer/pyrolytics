package config

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Solana   SolanaConfig   `mapstructure:"solana"`
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
	URL             string        `mapstructure:"url"`
	ReconnectWait   time.Duration `mapstructure:"reconnect_wait"`
	MaxReconnects   int           `mapstructure:"max_reconnects"`
	ConnectionName  string        `mapstructure:"connection_name"`
	StreamName      string        `mapstructure:"stream_name"`
	StreamSubjects  []string      `mapstructure:"stream_subjects"`
	RetentionPolicy string        `mapstructure:"retention_policy"`
	StorageType     string        `mapstructure:"storage_type"`
	MaxAge          string        `mapstructure:"max_age"`
	Replicas        int           `mapstructure:"replicas"`
}

// GetRetentionPolicy converts the string retention policy to nats.RetentionPolicy
func (c *NATSConfig) GetRetentionPolicy() nats.RetentionPolicy {
	switch c.RetentionPolicy {
	case "workqueue":
		return nats.WorkQueuePolicy
	case "limits":
		return nats.LimitsPolicy
	case "interest":
		return nats.InterestPolicy
	default:
		return nats.WorkQueuePolicy
	}
}

// GetStorageType converts the string storage type to nats.StorageType
func (c *NATSConfig) GetStorageType() nats.StorageType {
	switch c.StorageType {
	case "memory":
		return nats.MemoryStorage
	case "file":
		return nats.FileStorage
	default:
		return nats.FileStorage
	}
}

// GetMaxAge converts the string max age to time.Duration
func (c *NATSConfig) GetMaxAge() (time.Duration, error) {
	return time.ParseDuration(c.MaxAge)
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	TTL time.Duration `mapstructure:"ttl"`
}

type SolanaConfig struct {
	Network    string        `mapstructure:"network"`     // devnet, testnet, mainnet
	RPCURL     string        `mapstructure:"rpc_url"`     // HTTP RPC endpoint
	WSURL      string        `mapstructure:"ws_url"`      // WebSocket endpoint
	Timeout    time.Duration `mapstructure:"timeout"`     // Connection timeout
	MaxRetries int           `mapstructure:"max_retries"` // Max retry attempts
	Commitment string        `mapstructure:"commitment"`  // Commitment level
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
	viper.SetDefault("nats.reconnect_wait", 5*time.Second)
	viper.SetDefault("nats.max_reconnects", 3)
	viper.SetDefault("nats.connection_name", "pyrolytics")
	viper.SetDefault("nats.stream_name", "pyrolytics")
	viper.SetDefault("nats.stream_subjects", []string{"pyrolytics.>"})
	viper.SetDefault("nats.retention_policy", "workqueue")
	viper.SetDefault("nats.storage_type", "file")
	viper.SetDefault("nats.max_age", "24h")
	viper.SetDefault("nats.replicas", 1)

	// Cache defaults
	viper.SetDefault("cache.ttl", 1*time.Hour)

	// Solana defaults
	viper.SetDefault("solana.network", "devnet")
	viper.SetDefault("solana.rpc_url", "https://api.devnet.solana.com")
	viper.SetDefault("solana.ws_url", "wss://api.devnet.solana.com")
	viper.SetDefault("solana.timeout", 30*time.Second)
	viper.SetDefault("solana.max_retries", 3)
	viper.SetDefault("solana.commitment", "confirmed")
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Cache.TTL <= 0 {
		return fmt.Errorf("invalid cache TTL: %v", cfg.Cache.TTL)
	}

	if _, err := cfg.NATS.GetMaxAge(); err != nil {
		return fmt.Errorf("invalid NATS max age: %v", err)
	}

	return nil
}
