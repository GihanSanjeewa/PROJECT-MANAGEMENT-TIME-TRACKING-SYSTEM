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
	"project-management-system/internal/config"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: 'up' or 'down'")
	dsnFlag := flag.String("dsn", "", "PostgreSQL DSN connection string (optional, defaults to DB_DSN)")
	flag.Parse()

	cfg, _ := config.Load()
	dsn := cfg.DatabaseDSN
	if *dsnFlag != "" {
		dsn = *dsnFlag
	}

	fmt.Println("==================================================")
	fmt.Printf("📦 PulseFlow Database Migration Tool (%s)\n", strings.ToUpper(*direction))
	fmt.Println("==================================================")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("✖ Failed to open DB connection: %v\n", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Printf("✖ Cannot reach PostgreSQL at DSN: %s\n", dsn)
		fmt.Printf("  Error: %v\n\n", err)
		fmt.Println("👉 Please ensure PostgreSQL is running. See options below:")
		fmt.Println("   1. Local PostgreSQL service on port 5432")
		fmt.Println("   2. Docker: docker compose up -d")
		fmt.Println("   3. Cloud DB (Neon / Supabase): set DB_DSN='postgres://user:pass@ep-xyz.../pulseflow_pm'")
		os.Exit(1)
	}

	fmt.Println("✔ Successfully connected to PostgreSQL!")

	// Create schema_migrations tracking table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(100) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatalf("✖ Failed to create migrations tracking table: %v", err)
	}

	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = filepath.Join("Backend", "migrations")
	}

	pattern := filepath.Join(migrationsDir, fmt.Sprintf("*.%s.sql", *direction))
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		log.Fatalf("✖ No %s migration files found in %s", *direction, migrationsDir)
	}

	sort.Strings(files)
	if *direction == "down" {
		// Roll back in reverse order
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	for _, file := range files {
		baseName := filepath.Base(file)
		version := strings.Split(baseName, ".")[0]

		if *direction == "up" {
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
			if err == nil && exists {
				fmt.Printf("  [SKIP] %s (already applied)\n", baseName)
				continue
			}

			content, err := os.ReadFile(file)
			if err != nil {
				log.Fatalf("✖ Failed to read %s: %v", file, err)
			}

			fmt.Printf("  [APPLY] %s...", baseName)
			if _, err := db.Exec(string(content)); err != nil {
				log.Fatalf("\n✖ Failed to execute %s: %v", baseName, err)
			}

			_, _ = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
			fmt.Println(" ✔ DONE")
		} else {
			content, err := os.ReadFile(file)
			if err != nil {
				log.Fatalf("✖ Failed to read %s: %v", file, err)
			}

			fmt.Printf("  [ROLLBACK] %s...", baseName)
			if _, err := db.Exec(string(content)); err != nil {
				log.Fatalf("\n✖ Failed to rollback %s: %v", baseName, err)
			}

			_, _ = db.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
			fmt.Println(" ✔ DONE")
		}
	}

	fmt.Println("\n🎉 Database migration process completed successfully!")
}
