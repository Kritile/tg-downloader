package config

import (
	"os"
	"strconv"
)

type Config struct {
	BotToken                  string
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
		BotToken:      os.Getenv("BOT_TOKEN"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		AdminPort:     getEnvOrDefault("ADMIN_PORT", "8080"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		// Shared SOCKS5 proxy used for yt-dlp traffic and all Telegram Bot API requests.
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
