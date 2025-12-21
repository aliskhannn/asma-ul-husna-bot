package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var ErrMissingEnvironmentVariables = errors.New("missing required environment variables")

// Config holds application configuration loaded from files and environment variables.
type Config struct {
	Env              string `mapstructure:"env"`             // current application environment (local, dev, prod etc)
	TelegramAPIToken string `mapstructure:"-"`               // Telegram API token loaded from environment
	NamesJSONPath    string `mapstructure:"names_json_path"` // path to JSON file with 99 Names metadata
	DB               DB     `mapstructure:"database"`        // database configuration section
}

// DB contains database-related configuration parameters.
type DB struct {
	URL             string        `mapstructure:"-"`                 // database connection string loaded from environment
	MaxConnections  int           `mapstructure:"max_connections"`   // maximum number of open connections in the pool
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"` // maximum lifetime of a single connection
}

// DSN returns the database connection string if it is configured.
func (db DB) DSN() (string, error) {
	if db.URL == "" {
		return "", ErrMissingEnvironmentVariables
	}
	return db.URL, nil
}

// Load reads configuration from config files and environment variables.
func Load() (*Config, error) {
	// Initialize Viper instance and base config options.
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	// Set default values for configuration keys.
	v.SetDefault("env", "local")
	v.SetDefault("names_json_path", "assets/asma-ul-husna-ru.json")
	v.SetDefault("database.max_connections", 20)
	v.SetDefault("database.max_conn_lifetime", "30s")

	// Configure environment variable handling and key mapping.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // map nested keys to ENV style names
	v.AutomaticEnv()

	// Bind explicit environment variables to configuration keys.
	_ = v.BindEnv("telegram_api_token", "TELEGRAM_API_TOKEN")
	_ = v.BindEnv("database_url", "DATABASE_URL")
	_ = v.BindEnv("env", "APP_ENV")

	// Try to read configuration file if present.
	if err := v.ReadInConfig(); err != nil {
		var fileLookupErr viper.ConfigFileNotFoundError
		if !errors.As(err, &fileLookupErr) {
			return nil, fmt.Errorf("error loading config file: %w", err)
		}
	}

	// Unmarshal configuration into strongly typed struct.
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// Load sensitive values from environment variables.
	cfg.TelegramAPIToken = v.GetString("telegram_api_token")
	if cfg.TelegramAPIToken == "" {
		return nil, ErrMissingEnvironmentVariables
	}

	cfg.DB.URL = v.GetString("database_url")
	if cfg.DB.URL == "" {
		return nil, ErrMissingEnvironmentVariables
	}

	return &cfg, nil
}
