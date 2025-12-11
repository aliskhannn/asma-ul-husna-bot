package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var ErrMissingEnvironmentVariables = errors.New("missing required environment variables")

type Config struct {
	TelegramAPIToken string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	token := os.Getenv("TELEGRAM_API_TOKEN")
	if token == "" {
		return nil, ErrMissingEnvironmentVariables
	}
	return &Config{
		TelegramAPIToken: token,
	}, nil
}
