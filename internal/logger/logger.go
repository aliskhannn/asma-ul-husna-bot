package logger

import (
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/config"
)

func New(cfg *config.Config) (*zap.Logger, error) {
	if cfg.Env == "production" {
		return zap.NewProduction()
	}

	return zap.NewDevelopment()
}
