package logger

import (
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/config"
)

// New creates a new zap.Logger instance based on the environment configuration.
// If the environment is "production", it returns a production logger.
// Otherwise, it returns a development logger for easier debugging.
func New(cfg *config.Config) (*zap.Logger, error) {
	if cfg.Env == "production" {
		return zap.NewProduction()
	}

	return zap.NewDevelopment()
}
