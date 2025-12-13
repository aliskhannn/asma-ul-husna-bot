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
	"github.com/aliskhannn/asma-ul-husna-bot/internal/usecase"
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
	dbpool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()

	userRepo := repository.NewUserRepository(dbpool)

	nameUC := usecase.NewNameUseCase(nameRepo)
	userUC := usecase.NewUserUseCase(userRepo)

	handler := telegram.NewHandler(bot, nameUC, userUC)
	if err := handler.Run(ctx); err != nil {
		log.Panic(err)
	}

	<-ctx.Done()
	log.Println("shutdown signal received")
}
