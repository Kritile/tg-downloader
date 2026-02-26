package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/admin/internal/repository"
	"github.com/mediaharvester/tg-downloader/admin/internal/transport"
	"github.com/mediaharvester/tg-downloader/admin/internal/usecase"
	"github.com/mediaharvester/tg-downloader/shared/config"
)

func main() {
	log.Println("Starting MediaHarvester Admin Panel...")

	// Load configuration
	cfg := config.Load()

	// Validate required config
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET is required")
	}

	// Connect to database
	db, err := initDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	adminRepo := repository.NewAdminUserRepository(db)
	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	downloadRepo := repository.NewDownloadRepository(db)

	// Initialize usecases
	authService := usecase.NewAdminUserService(adminRepo)
	userMgmtSvc := usecase.NewAdminUserManagementService(userRepo, settingsRepo)
	statsSvc := usecase.NewAdminStatsService(downloadRepo, userRepo)

	// Create default admin if not exists
	createDefaultAdmin(authService)

	// Initialize admin server
	server := transport.NewAdminServer(
		authService,
		userMgmtSvc,
		statsSvc,
		settingsRepo,
		cfg.SessionSecret,
	)

	log.Printf("Admin panel starting on port %s", cfg.AdminPort)
	if err := server.Run(cfg.AdminPort); err != nil {
		log.Fatalf("Failed to start admin server: %v", err)
	}
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

func createDefaultAdmin(authService domain.AdminUserService) {
	ctx := context.Background()
	username := os.Getenv("ADMIN_INITIAL_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("ADMIN_INITIAL_PASSWORD")
	if password == "" {
		log.Println("ADMIN_INITIAL_PASSWORD is not set; skip default admin creation for security")
		return
	}
	if len(password) < 12 {
		log.Println("ADMIN_INITIAL_PASSWORD must be at least 12 chars; skip admin creation")
		return
	}

	err := authService.CreateAdmin(ctx, username, password)
	if err == nil {
		log.Printf("Default admin created: username=%s", username)
		log.Println("Please rotate ADMIN_INITIAL_PASSWORD after first successful login.")
	} else {
		log.Printf("Admin user may already exist: %v", err)
	}
}
