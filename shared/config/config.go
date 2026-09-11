package config

import (
	"os"
	"strconv"
)

type Config struct {
	BotToken                  string
	MaxBotToken               string
	TelegramAPIURL            string
	DatabaseURL               string
	RedisURL                  string
	GlobalDefaultDailyLimit   int
	GlobalDefaultMonthlyLimit int
	WorkerCount               int
	AdminPort                 string
	SessionSecret             string
	XraySocks5Proxy           string
}

func Load() *Config {
	cfg := &Config{
		BotToken:       os.Getenv("BOT_TOKEN"),
		MaxBotToken:    os.Getenv("MAX_BOT_TOKEN"),
		TelegramAPIURL: getEnvOrDefault("TELEGRAM_API_URL", "http://telegram-bot-api:8081"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		AdminPort:      getEnvOrDefault("ADMIN_PORT", "8080"),
		SessionSecret:  os.Getenv("SESSION_SECRET"),
		// SOCKS5 proxy is used by yt-dlp only. Telegram Local Bot API uses its AWG
		// network namespace, and MAX uses its own direct client.
		XraySocks5Proxy: os.Getenv("XRAY_SOCKS5_PROXY"),
	}

	cfg.GlobalDefaultDailyLimit, _ = strconv.Atoi(getEnvOrDefault("GLOBAL_DEFAULT_DAILY_LIMIT", "10"))
	cfg.GlobalDefaultMonthlyLimit, _ = strconv.Atoi(getEnvOrDefault("GLOBAL_DEFAULT_MONTHLY_LIMIT", "100"))
	cfg.WorkerCount, _ = strconv.Atoi(getEnvOrDefault("WORKER_COUNT", "3"))

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
