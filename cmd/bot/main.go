package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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

	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "Запустить бота",
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
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	repo, err := repository.NewNameRepository("assets/data/names.json")
	if err != nil {
		log.Panic(err)
	}

	uc := usecase.NewNameUseCase(repo)

	handler := telegram.NewHandler(bot, uc)
	if err := handler.Run(ctx); err != nil {
		log.Panic(err)
	}
}
