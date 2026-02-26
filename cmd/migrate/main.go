package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	migrationsDir := flag.String("dir", "", "Directory containing migration files (optional)")
	flag.Parse()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	resolvedDir, files, err := resolveMigrationFiles(*migrationsDir)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Using migrations directory: %s", resolvedDir)
	log.Printf("Found %d migration files", len(files))
	for _, file := range files {
		log.Printf(" - %s", filepath.Base(file))
	}

	for _, file := range files {
		if strings.HasSuffix(filepath.Base(file), ".down.sql") {
			continue
		}

		log.Printf("Executing migration: %s", filepath.Base(file))

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
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

func resolveMigrationFiles(flagDir string) (string, []string, error) {
	candidates := []string{}
	if flagDir != "" {
		candidates = append(candidates, flagDir)
	} else {
		candidates = append(candidates, "./migrations", "/app/migrations")
	}

	for _, dir := range candidates {
		files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
		if err != nil {
			return "", nil, fmt.Errorf("failed to read migrations directory %s: %w", dir, err)
		}
		if len(files) == 0 {
			continue
		}
		sort.Strings(files)
		return dir, files, nil
	}

	return "", nil, fmt.Errorf("no migration files found in: %s", strings.Join(candidates, ", "))
}
