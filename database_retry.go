package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	
	"github.com/jmoiron/sqlx"
)

// RetryConfig holds database retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffMultiplier float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        5,
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// ExecuteWithRetry executes a database operation with retry logic
func ExecuteWithRetry(fn func() error) error {
	config := DefaultRetryConfig()
	return ExecuteWithRetryConfig(fn, config)
}

// ExecuteWithRetryConfig executes a database operation with custom retry configuration
func ExecuteWithRetryConfig(fn func() error, config RetryConfig) error {
	var lastErr error
	delay := config.InitialDelay
	
	for i := 0; i <= config.MaxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
		
		if i < config.MaxRetries {
			log.Printf("[DB RETRY] Attempt %d/%d failed: %v. Retrying in %v...", 
				i+1, config.MaxRetries+1, err, delay)
			time.Sleep(delay)
			
			// Exponential backoff
			delay = time.Duration(float64(delay) * config.BackoffMultiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}
	
	return fmt.Errorf("database operation failed after %d retries: %w", config.MaxRetries+1, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for specific error types
	errStr := err.Error()
	
	// Connection errors
	if containsAny(errStr, 
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"context deadline exceeded",
		"too many connections",
		"cannot acquire connection",
		"driver: bad connection",
		"connection timed out") {
		return true
	}
	
	// Database is starting up or shutting down
	if containsAny(errStr,
		"the database system is starting up",
		"the database system is shutting down",
		"terminating connection") {
		return true
	}
	
	// Check for sql.ErrConnDone
	if err == sql.ErrConnDone {
		return true
	}
	
	return false
}

// containsAny checks if string contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if dbContains(s, substr) {
			return true
		}
	}
	return false
}

// dbContains is a case-insensitive string contains for database errors
func dbContains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > 0 && len(substr) > 0 && 
		 dbContainsIgnoreCase(s, substr))
}

func dbContainsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	// Simple case-insensitive contains
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// QueryWithRetry executes a query with retry logic
func QueryWithRetry(dest interface{}, query string, args ...interface{}) error {
	return ExecuteWithRetry(func() error {
		return db.Get(dest, query, args...)
	})
}

// QueryRowsWithRetry executes a query that returns multiple rows with retry logic
func QueryRowsWithRetry(dest interface{}, query string, args ...interface{}) error {
	return ExecuteWithRetry(func() error {
		return db.Select(dest, query, args...)
	})
}

// ExecWithRetry executes a non-query SQL statement with retry logic
func ExecWithRetry(query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	err := ExecuteWithRetry(func() error {
		var err error
		result, err = db.Exec(query, args...)
		return err
	})
	return result, err
}

// PingWithRetry pings the database with retry logic
func PingWithRetry() error {
	return ExecuteWithRetry(func() error {
		return db.Ping()
	})
}

// TransactionWithRetry executes a transaction with retry logic
func TransactionWithRetry(fn func(*sqlx.Tx) error) error {
	return ExecuteWithRetry(func() error {
		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
				panic(p)
			}
		}()
		
		err = fn(tx)
		if err != nil {
			tx.Rollback()
			return err
		}
		
		return tx.Commit()
	})
}

// MonitorDatabaseHealth continuously monitors database health
func MonitorDatabaseHealth(interval time.Duration) {
	go func() {
		defer recoverFromPanic("database health monitor")
		
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		consecutiveFailures := 0
		const maxConsecutiveFailures = 3
		
		for range ticker.C {
			err := PingWithRetry()
			if err != nil {
				consecutiveFailures++
				log.Printf("[DB HEALTH] Database ping failed (%d/%d): %v", 
					consecutiveFailures, maxConsecutiveFailures, err)
				
				if consecutiveFailures >= maxConsecutiveFailures {
					log.Printf("[DB HEALTH] CRITICAL: Database appears to be down after %d consecutive failures", 
						consecutiveFailures)
					// Could trigger alerts or recovery procedures here
				}
			} else {
				if consecutiveFailures > 0 {
					log.Printf("[DB HEALTH] Database connection restored after %d failures", 
						consecutiveFailures)
				}
				consecutiveFailures = 0
			}
		}
	}()
}