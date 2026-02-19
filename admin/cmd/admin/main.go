package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/admin/internal/repository"
	"github.com/mediaharvester/tg-downloader/admin/internal/transport"
	"github.com/mediaharvester/tg-downloader/admin/internal/usecase"
	"github.com/mediaharvester/tg-downloader/shared/config"
	_ "github.com/lib/pq"
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
	
	// Try to create default admin
	// In production, this should be done via CLI command
	username := "admin"
	password := "admin123" // Change this!
	
	err := authService.CreateAdmin(ctx, username, password)
	if err == nil {
		log.Printf("⚠️  Default admin created: username=%s, password=%s", username, password)
		log.Println("⚠️  Please change the password immediately after first login!")
	} else {
		log.Printf("Admin user may already exist: %v", err)
	}
}
