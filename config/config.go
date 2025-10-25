package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the webhook inbox system
type Config struct {
	Port string `mapstructure:"PORT"`

	// Redis Configuration
	RedisHost     string `mapstructure:"REDIS_HOST"`
	RedisPort     string `mapstructure:"REDIS_PORT"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	// Webhook Configuration
	RoutesFile               string `mapstructure:"ROUTES_FILE"`
	WebhookDeliveredTTLHours int    `mapstructure:"WEBHOOK_DELIVERED_TTL_HOURS"`
	WebhookFailedTTLHours    int    `mapstructure:"WEBHOOK_FAILED_TTL_HOURS"`

	// Telemetry Configuration
	TelemetryEnabled bool `mapstructure:"TELEMETRY_ENABLED"` // OpenTelemetry metrics export
}

// RedisAddr returns the Redis address in format host:port
func (c *Config) RedisAddr() string {
	if c.RedisPort == "" {
		return c.RedisHost + ":6379" // default Redis port
	}
	return c.RedisHost + ":" + c.RedisPort
}

// ValidateRedis validates if Redis configuration is present
func (c *Config) ValidateRedis() error {
	if c.RedisHost == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	return nil
}

// GetRoutesFile returns the routes file path or default
func (c *Config) GetRoutesFile() string {
	if c.RoutesFile == "" {
		return "routes.yaml" // default
	}
	return c.RoutesFile
}

// GetWebhookDeliveredTTLHours returns the TTL for delivered webhooks in hours (default: 1)
func (c *Config) GetWebhookDeliveredTTLHours() int {
	if c.WebhookDeliveredTTLHours <= 0 {
		return 1 // default: 1 hour
	}
	return c.WebhookDeliveredTTLHours
}

// GetWebhookFailedTTLHours returns the TTL for failed webhooks in hours (default: 24)
func (c *Config) GetWebhookFailedTTLHours() int {
	if c.WebhookFailedTTLHours <= 0 {
		return 24 // default: 24 hours
	}
	return c.WebhookFailedTTLHours
}

func GetConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("parsing config data: %w", err)
	}
	return &config, nil
}
