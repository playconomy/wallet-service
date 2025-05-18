// Package integration provides integration test utilities and tests for the wallet service
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/playconomy/wallet-service/internal/observability"
	"github.com/playconomy/wallet-service/internal/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	db       *sql.DB
	dbName   = "testdb"
	user     = "postgres"
	password = "postgres"
	port     = "5433" // Different from default to avoid conflicts
)

// TestMain prepares the test environment for integration tests, creating a temporary Postgres
// database in a Docker container.
func TestMain(m *testing.M) {
	// Use a sensible default on Windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Pull and create postgres container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
			fmt.Sprintf("POSTGRES_USER=%s", user),
			fmt.Sprintf("POSTGRES_DB=%s", dbName),
		},
		ExposedPorts: []string{"5432/tcp"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432/tcp": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// Set cleanup function
	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}()

	// Exponential backoff-retry for the database to be ready
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
			user, password, port, dbName))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		log.Fatalf("Could not run migrations: %s", err)
	}

	// Run tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}

// runMigrations applies database migrations needed for tests
func runMigrations() error {
	// Create wallets table
	_, err := db.Exec(`
		CREATE TABLE wallets (
			id SERIAL PRIMARY KEY,
			user_id INT UNIQUE NOT NULL,
			balance NUMERIC(20, 2) NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// Create exchange_rates table
	_, err = db.Exec(`
		CREATE TABLE exchange_rates (
			id SERIAL PRIMARY KEY,
			game_id VARCHAR(50) NOT NULL,
			token_type VARCHAR(20) NOT NULL,
			to_platform_ratio NUMERIC(10, 4) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(game_id, token_type)
		);
	`)
	if err != nil {
		return err
	}

	// Create wallet_logs table
	_, err = db.Exec(`
		CREATE TABLE wallet_logs (
			id SERIAL PRIMARY KEY,
			wallet_id INT NOT NULL,
			user_id INT NOT NULL,
			game_id VARCHAR(50),
			token_type VARCHAR(20),
			amount NUMERIC(20, 2) NOT NULL,
			platform_amount NUMERIC(20, 2) NOT NULL,
			source VARCHAR(20) NOT NULL,
			reference_id VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (wallet_id) REFERENCES wallets(id)
		);
	`)
	if err != nil {
		return err
	}

	// Insert test data - sample exchange rates
	_, err = db.Exec(`
		INSERT INTO exchange_rates (game_id, token_type, to_platform_ratio)
		VALUES 
			('game1', 'gold', 2.5),
			('game2', 'gems', 5.0),
			('game3', 'coins', 1.0);
	`)

	return err
}

// GetTestDB returns the test database instance for integration tests
func GetTestDB() *sql.DB {
	return db
}

// GetTestRepository returns a real postgres repository for integration tests
func GetTestRepository(t *testing.T) repository.WalletRepository {
	// Create observability for test
	obs := observability.NewTestObservability()
	
	// Create repository with test DB
	return repository.NewPostgresRepository(db, obs)
}

// ClearTestData deletes all test data between tests while preserving table structure
func ClearTestData(t *testing.T) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE wallet_logs, wallets RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("Failed to clear test data: %v", err)
	}
}

// CreateTestWallet creates a wallet for testing
func CreateTestWallet(t *testing.T, userID int, balance float64) int {
	t.Helper()

	var walletID int
	err := db.QueryRow(`
		INSERT INTO wallets (user_id, balance, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, balance, time.Now()).Scan(&walletID)

	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}

	return walletID
}
