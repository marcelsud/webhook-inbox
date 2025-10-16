package config

import (
	"fmt"

	"github.com/spf13/viper"
)

/* Config é um pacote auxiliar. Poderia ser uma lib externa*/

type Config struct {
	Port             string `mapstructure:"PORT"`
	DBName           string `mapstructure:"DBNAME"`
	TursoDatabaseURL string `mapstructure:"TURSO_DATABASE_URL"`
	TursoAuthToken   string `mapstructure:"TURSO_AUTH_TOKEN"`

	// PostgreSQL Configuration
	PostgresHost               string `mapstructure:"POSTGRES_HOST"`
	PostgresPort               string `mapstructure:"POSTGRES_PORT"`
	PostgresDB                 string `mapstructure:"POSTGRES_DB"`
	PostgresUser               string `mapstructure:"POSTGRES_USER"`
	PostgresPassword           string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresSSLMode            string `mapstructure:"POSTGRES_SSLMODE"`
	PostgresMaxOpenConns       int    `mapstructure:"POSTGRES_MAX_OPEN_CONNS"`
	PostgresMaxIdleConns       int    `mapstructure:"POSTGRES_MAX_IDLE_CONNS"`
	PostgresConnMaxLifeMinutes int    `mapstructure:"POSTGRES_CONN_MAX_LIFE_MINUTES"`
}

// PostgresConnectionString retorna a connection string para PostgreSQL
// SEGURANÇA: Para produção, considere usar autenticação IAM ou AWS Secrets Manager
// em vez de armazenar credenciais em variáveis de ambiente em texto plano.
func (c *Config) PostgresConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.PostgresHost, c.PostgresPort, c.PostgresUser,
		c.PostgresPassword, c.PostgresDB, c.PostgresSSLMode,
	)
}

// ValidatePostgres valida se todas as configurações obrigatórias do PostgreSQL estão presentes
func (c *Config) ValidatePostgres() error {
	if c.PostgresHost == "" {
		return fmt.Errorf("POSTGRES_HOST é obrigatório")
	}
	if c.PostgresPort == "" {
		return fmt.Errorf("POSTGRES_PORT é obrigatório")
	}
	if c.PostgresDB == "" {
		return fmt.Errorf("POSTGRES_DB é obrigatório")
	}
	if c.PostgresUser == "" {
		return fmt.Errorf("POSTGRES_USER é obrigatório")
	}
	if c.PostgresPassword == "" {
		return fmt.Errorf("POSTGRES_PASSWORD é obrigatório")
	}
	if c.PostgresSSLMode == "" {
		return fmt.Errorf("POSTGRES_SSLMODE é obrigatório (use 'disable' para desenvolvimento)")
	}
	return nil
}

// GetPostgresMaxOpenConns retorna o máximo de conexões abertas ou o padrão (25)
func (c *Config) GetPostgresMaxOpenConns() int {
	if c.PostgresMaxOpenConns <= 0 {
		return 25 // default
	}
	return c.PostgresMaxOpenConns
}

// GetPostgresMaxIdleConns retorna o máximo de conexões inativas ou o padrão (5)
func (c *Config) GetPostgresMaxIdleConns() int {
	if c.PostgresMaxIdleConns <= 0 {
		return 5 // default
	}
	return c.PostgresMaxIdleConns
}

// GetPostgresConnMaxLifeMinutes retorna a duração máxima em minutos ou o padrão (5)
func (c *Config) GetPostgresConnMaxLifeMinutes() int {
	if c.PostgresConnMaxLifeMinutes <= 0 {
		return 5 // default: 5 minutes
	}
	return c.PostgresConnMaxLifeMinutes
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
