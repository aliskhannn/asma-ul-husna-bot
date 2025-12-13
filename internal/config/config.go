package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var ErrMissingEnvironmentVariables = errors.New("missing required environment variables")

type Config struct {
	TelegramAPIToken string
	DB               DB `mapstructure:"database"`
}

type DB struct {
	User              string
	Password          string
	Host              string        `mapstructure:"host"`
	Port              string        `mapstructure:"port"`
	Name              string        `mapstructure:"name"`
	SSLMode           string        `mapstructure:"ssl_mode"`
	MaxConnections    int           `mapstructure:"max_connections"`
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
}

func (n DB) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&pool_max_conns=%d&pool_max_conn_lifetime=%s",
		n.User, n.Password, n.Host, n.Port, n.Name, n.SSLMode, n.MaxConnections, n.ConnectionTimeout.String(),
	)
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("loading .env file: %w", err)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	token := os.Getenv("TELEGRAM_API_TOKEN")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	if token == "" || user == "" || password == "" {
		return nil, ErrMissingEnvironmentVariables
	}

	cfg.TelegramAPIToken = token
	cfg.DB.User = os.Getenv("DB_USER")
	cfg.DB.Password = os.Getenv("DB_PASSWORD")

	return &cfg, nil
}
