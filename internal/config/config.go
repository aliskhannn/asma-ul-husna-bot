package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

var ErrMissingEnvironmentVariables = errors.New("missing required environment variables")

type Config struct {
	ENV              string `mapstructure:"env"`
	TelegramAPIToken string
	DB               DB `mapstructure:"database"`
}

type DB struct {
	URL               string
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
	if n.URL != "" {
		return fmt.Sprintf("%s&pool_max_conns=%d&pool_max_conn_lifetime=%s",
			n.URL, n.MaxConnections, n.ConnectionTimeout.String(),
		)
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&pool_max_conns=%d&pool_max_conn_lifetime=%s",
		n.User, n.Password, n.Host, n.Port, n.Name, n.SSLMode, n.MaxConnections, n.ConnectionTimeout.String(),
	)
}

func Load() (*Config, error) {
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

	cfg.ENV = os.Getenv("APP_ENV")

	token := os.Getenv("TELEGRAM_API_TOKEN")
	if token == "" {
		return nil, ErrMissingEnvironmentVariables
	}
	cfg.TelegramAPIToken = token

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DB.URL = dbURL
		return &cfg, nil
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	if user == "" || password == "" {
		return nil, ErrMissingEnvironmentVariables
	}

	cfg.DB.User = user
	cfg.DB.Password = password
	return &cfg, nil
}
