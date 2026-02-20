package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	migrationsDir := flag.String("dir", "./migrations", "Directory containing migration files")
	flag.Parse()

	// Retry logic for database connection
	var db *sql.DB
	var err error
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			log.Printf("Failed to open database (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}

		if err := db.Ping(); err != nil {
			log.Printf("Failed to ping database (attempt %d/%d): %v", i+1, maxRetries, err)
			db.Close()
			time.Sleep(retryDelay)
			continue
		}

		log.Println("Connected to database")
		break
	}

	if err != nil {
		log.Fatalf("Failed to connect to database after %d attempts: %v", maxRetries, err)
	}
	defer db.Close()

	// Read and execute migration files
	files, err := filepath.Glob(filepath.Join(*migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	if len(files) == 0 {
		log.Println("No migration files found")
		return
	}

	// Sort files by name
	sort.Strings(files)

	// Execute each migration file
	for _, file := range files {
		// Skip down migration files
		if filepath.Base(file) == "00001_initial_schema.down.sql" {
			continue
		}

		log.Printf("Executing migration: %s", filepath.Base(file))

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			// Check if it's a "already exists" error - if so, skip it
			if strings.Contains(err.Error(), "already exists") {
				log.Printf("Skipping migration %s: already applied", filepath.Base(file))
				continue
			}
			log.Fatalf("Failed to execute migration %s: %v", file, err)
		}

		log.Printf("Migration %s completed", filepath.Base(file))
	}

	log.Println("All migrations completed successfully")
}
