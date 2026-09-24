package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/mediaharvester/tg-downloader/bot/internal/repository"
	"github.com/mediaharvester/tg-downloader/bot/internal/transport"
	"github.com/mediaharvester/tg-downloader/bot/internal/usecase"
	"github.com/mediaharvester/tg-downloader/bot/internal/worker"
	"github.com/mediaharvester/tg-downloader/shared/config"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("Starting MediaHarvester Bot...")

	// Load configuration
	cfg := config.Load()

	// Validate required config
	if cfg.BotToken == "" && cfg.MaxBotToken == "" {
		log.Fatal("BOT_TOKEN or MAX_BOT_TOKEN is required")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL is required")
	}

	// Connect to database
	db, err := initDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	redisClient := initRedis(cfg.RedisURL)
	defer redisClient.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	downloadRepo := repository.NewDownloadRepository(db)
	jobRepo := repository.NewJobRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// Initialize usecases
	urlValidator := usecase.NewURLValidator()
	permissionSvc := usecase.NewPermissionService(settingsRepo)
	limitSvc := usecase.NewLimitService(downloadRepo, settingsRepo)
	userService := usecase.NewUserService(userRepo)
	formatSvc := usecase.NewFormatService(cfg.XraySocks5Proxy, redisClient)

	// Initialize queue service
	queueService := worker.NewQueueService(redisClient, jobRepo)

	// Initialize bot
	var telegramTransport *transport.TelegramTransport
	if cfg.BotToken != "" {
		botAPI, err := initTelegramBotAPILocal(cfg.BotToken, cfg.TelegramAPIURL)
		if err != nil {
			log.Fatalf("Failed to initialize Telegram Local Bot API: %v", err)
		}
		log.Printf("Telegram Local Bot API configured at %s", cfg.TelegramAPIURL)
		telegramTransport = transport.NewTelegramTransport(botAPI)
	}
	var maxTransport *transport.MaxTransport
	if cfg.MaxBotToken != "" {
		maxAPI, err := maxbot.NewApi(cfg.MaxBotToken)
		if err != nil {
			log.Fatalf("Failed to initialize MAX bot: %v", err)
		}
		maxTransport = transport.NewMaxTransport(maxAPI)
		log.Println("MAX bot configured with direct API access (no Xray)")
	}

	// Shared business workflow; platform transports only translate updates and API calls.
	telegramBot := transport.NewBot(
		"telegram", telegramTransport,
		urlValidator,
		queueService,
		userService,
		permissionSvc,
		limitSvc,
		formatSvc,
		"/downloads",
	)
	maxBot := transport.NewBot(
		"max", maxTransport,
		urlValidator,
		queueService,
		userService,
		permissionSvc,
		limitSvc,
		formatSvc,
		"/downloads",
	)
	notifier := transport.NewNotifierRouter(telegramBot, maxBot)

	// Initialize worker pool
	workerPool := worker.NewWorkerPool(queueService, notifier, cfg.WorkerCount, downloadRepo, cfg.XraySocks5Proxy, jobRepo)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start bot in a goroutine
	go func() {
		if telegramTransport == nil {
			return
		}
		if err := telegramTransport.Start(ctx, telegramBot); err != nil {
			log.Printf("Bot error: %v", err)
		}
	}()
	go func() {
		if maxTransport == nil {
			return
		}
		if err := maxTransport.Start(ctx, maxBot); err != nil {
			log.Printf("MAX bot error: %v", err)
		}
	}()

	// Start worker pool in a goroutine
	go func() {
		if err := workerPool.Start(ctx); err != nil {
			log.Printf("Worker pool error: %v", err)
		}
	}()

	log.Println("Bot and workers are running")

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	cancel()

	// Graceful shutdown timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()
	log.Println("Shutdown complete")
}

func initDatabase(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Connected to database")
	return db, nil
}

func initRedis(redisURL string) *redis.Client {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis")
	return client
}
