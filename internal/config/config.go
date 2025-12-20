package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var ErrMissingEnvironmentVariables = errors.New("missing required environment variables")

type Config struct {
	Env string `mapstructure:"env"`

	TelegramAPIToken string `mapstructure:"-"`

	NamesJSONPath string `mapstructure:"names_json_path"`
	DB            DB     `mapstructure:"database"`
}

type DB struct {
	URL string `mapstructure:"-"`

	MaxConnections  int           `mapstructure:"max_connections"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
}

func (db DB) DSN() (string, error) {
	if db.URL == "" {
		return "", ErrMissingEnvironmentVariables
	}
	return db.URL, nil
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	v.SetDefault("env", "local")
	v.SetDefault("names_json_path", "assets/asma-ul-husna-ru.json")
	v.SetDefault("database.max_connections", 20)
	v.SetDefault("database.max_conn_lifetime", "30s")

	// ENV overrides
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.BindEnv("telegram_api_token", "TELEGRAM_API_TOKEN")
	_ = v.BindEnv("database_url", "DATABASE_URL")
	_ = v.BindEnv("env", "APP_ENV")

	if err := v.ReadInConfig(); err != nil {
		var fileLookupErr viper.ConfigFileNotFoundError
		if !errors.As(err, &fileLookupErr) {
			return nil, fmt.Errorf("error loading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

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
