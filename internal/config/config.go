// Package config loads and provides application configuration from YAML files and env vars.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	ShortURL ShortURLConfig  `mapstructure:"short_url"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

// DSN returns the PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
	CacheTTL time.Duration `mapstructure:"cache_ttl"`
}

// RateLimitConfig holds token-bucket rate limiter settings.
type RateLimitConfig struct {
	Enabled    bool  `mapstructure:"enabled"`
	Rate       int   `mapstructure:"rate"`        // tokens per second
	BucketSize int   `mapstructure:"bucket_size"` // max burst
	KeyPrefix  string `mapstructure:"key_prefix"`
}

// ShortURLConfig holds URL shortening parameters.
type ShortURLConfig struct {
	BaseURL    string `mapstructure:"base_url"`
	CodeLength int    `mapstructure:"code_length"`
	MaxRetries int    `mapstructure:"max_retries"`
}

// TelemetryConfig holds observability settings.
type TelemetryConfig struct {
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	MetricsPath    string `mapstructure:"metrics_path"`
	TracingEnabled bool   `mapstructure:"tracing_enabled"`
	OTLPEndpoint   string `mapstructure:"otlp_endpoint"`
	ServiceName    string `mapstructure:"service_name"`
}

// Load reads configuration from the given config file path and environment variables.
// Environment variables take precedence: GATEWAY_SERVER_PORT, GATEWAY_DATABASE_HOST, etc.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Defaults.
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.idle_timeout", "120s")
	v.SetDefault("server.shutdown_timeout", "15s")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "gateway")
	v.SetDefault("database.password", "gateway")
	v.SetDefault("database.dbname", "gateway")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.max_lifetime", "5m")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.cache_ttl", "1h")

	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.rate", 100)
	v.SetDefault("rate_limit.bucket_size", 200)
	v.SetDefault("rate_limit.key_prefix", "rl:")

	v.SetDefault("short_url.base_url", "http://localhost:8080")
	v.SetDefault("short_url.code_length", 7)
	v.SetDefault("short_url.max_retries", 3)

	v.SetDefault("telemetry.metrics_enabled", true)
	v.SetDefault("telemetry.metrics_path", "/metrics")
	v.SetDefault("telemetry.tracing_enabled", false)
	v.SetDefault("telemetry.otlp_endpoint", "localhost:4317")
	v.SetDefault("telemetry.service_name", "gateway")

	// Config file.
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("app.dev")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Environment variables: GATEWAY_SERVER_PORT, etc.
	v.SetEnvPrefix("GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		// Config file not found is acceptable — rely on defaults and env vars.
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}
