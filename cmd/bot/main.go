package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/config"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/delivery/telegram"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramAPIToken)
	if err != nil {
		// TODO: replace standard logger with Zap Logger.
		log.Panic(err)
	}

	// Set commands.
	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "Запустить бота",
		},
		{
			Command:     "all",
			Description: "Посмотреть все имена",
		},
		{
			Command:     "random",
			Description: "Получить случайное имя",
		},
		{
			Command:     "range",
			Description: "Получить имена в диапазоне (использование: /range 25 30)",
		},
		{
			Command:     "progress",
			Description: "Показать прогресс",
		},
		{
			Command:     "settings",
			Description: "Настройки",
		},
		{
			Command:     "help",
			Description: "Помощь",
		},
	}

	_, err = bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		log.Printf("Failed to set bot commands: %v", err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize repositories and use cases.
	nameRepo, err := repository.NewNameRepository("assets/data/names.json")
	if err != nil {
		log.Fatal(err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DB.DSN())
	if err != nil {
		log.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)

	nameService := service.NewNameService(nameRepo)
	userService := service.NewUserService(userRepo)

	progressRepo := repository.NewProgressRepository(pool)
	settingsRepo := repository.NewSettingsRepository(pool)

	progressService := service.NewProgressService(progressRepo)
	settingsService := service.NewSettingsService(settingsRepo)

	handler := telegram.NewHandler(
		bot,
		nameService,
		userService,
		progressService,
		settingsService,
	)
	if err := handler.Run(ctx); err != nil {
		log.Panic(err)
	}

	<-ctx.Done()
	log.Println("shutdown signal received")
}
