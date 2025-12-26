package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/config"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/delivery/telegram"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres/repository"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/logger"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/service"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/storage"
)

func main() {
	// Load application configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal()
	}

	// Initialize structured logger.
	lg, err := logger.New(cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = lg.Sync()
	}()

	// Create Telegram Bot API client.
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramAPIToken)
	if err != nil {
		lg.Fatal("failed to create bot",
			zap.Error(err),
		)
	}

	// Set commands.
	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "Начать работу с ботом",
		},
		{
			Command:     "next",
			Description: "Начать изучение следующего имени",
		},
		{
			Command:     "today",
			Description: "Список имен на сегодня",
		},
		{
			Command:     "random",
			Description: "Случайное имя: Guided — из /today, Free — из всех 99",
		},
		{
			Command:     "quiz",
			Description: "Пройти квиз по изученным именам",
		},
		{
			Command:     "all",
			Description: "Показать все 99 имён",
		},
		{
			Command:     "range",
			Description: "Показать имена в диапазоне (например, /range 1 10)",
		},
		{
			Command:     "progress",
			Description: "Показать прогресс изучения",
		},
		{
			Command:     "settings",
			Description: "Настройки бота",
		},
		{
			Command:     "help",
			Description: "Помощь и список команд",
		},
		{
			Command:     "reset",
			Description: "Сбросить прогресс и настройки",
		},
	}

	// Register bot commands with Telegram API.
	_, err = bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		lg.Warn("failed to set bot commands",
			zap.Error(err),
		)
	}

	bot.Debug = true // enable debug mode for development

	lg.Info("authorized on account",
		zap.String("username", bot.Self.UserName),
	)

	// Prepare cancellable context that listens for OS termination signals.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize name repository from static JSON file.
	nameRepo, err := repository.NewNameRepository(cfg.NamesJSONPath)
	if err != nil {
		lg.Fatal("failed to init name repository",
			zap.Error(err),
		)
	}

	// Initialize domain services.
	nameService := service.NewNameService(nameRepo)

	// Build database DSN from configuration.
	connString, err := cfg.DB.DSN()
	if err != nil {
		lg.Fatal("failed to get database DSN", zap.Error(err))
	}

	pool, err := postgres.NewPool(ctx, connString, postgres.PoolConfig{
		MaxConns:        cfg.DB.MaxConnections,
		MaxConnLifetime: cfg.DB.MaxConnLifetime,
	})
	if err != nil {
		lg.Fatal("failed to connect to db",
			zap.Error(err),
		)
	}
	defer pool.Close()

	tr := postgres.NewTransactor(pool)

	// Initialize repositories and services.
	userRepo := repository.NewUserRepository(pool)
	userService := service.NewUserService(userRepo)

	settingsRepo := repository.NewSettingsRepository(pool)
	settingsService := service.NewSettingsService(settingsRepo)

	progressRepo := repository.NewProgressRepository(pool)
	progressService := service.NewProgressService(progressRepo, settingsRepo)

	dailyNameRepo := repository.NewDailyNameRepository(pool)
	dailyNameService := service.NewDailyNameService(dailyNameRepo)

	quizRepo := repository.NewQuizRepository(pool)
	quizService := service.NewQuizService(tr, nameRepo, progressRepo, quizRepo, settingsRepo, dailyNameRepo, lg)

	remindersRepo := repository.NewRemindersRepository(pool)
	remindersService := service.NewReminderService(remindersRepo, progressRepo, settingsRepo, nameRepo, dailyNameRepo, lg)

	resetService := service.NewResetService(tr)

	// Initialize in-memory storage for quiz sessions.
	quizStorage := storage.NewQuizStorage()

	// Construct Telegram updates handler with all dependencies.
	handler := telegram.NewHandler(
		bot,
		lg,
		nameService,
		userService,
		progressService,
		settingsService,
		quizService,
		quizStorage,
		remindersService,
		dailyNameService,
		resetService,
	)

	// Register Telegram notifier in reminders service.
	remindersService.SetNotifier(handler)

	// Start background reminder scheduler.
	go remindersService.Start(ctx)

	// Start main Telegram updates handling loop.
	if err := handler.Run(ctx); err != nil {
		lg.Error("handler run failed",
			zap.Error(err),
		)
	}

	<-ctx.Done() // wait for graceful shutdown signal

	lg.Info("shutdown signal received")
}
