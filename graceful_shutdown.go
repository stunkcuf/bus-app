package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// GracefulShutdown handles graceful server shutdown
type GracefulShutdown struct {
	server          *http.Server
	shutdownTimeout time.Duration
	cleanupFuncs    []func()
}

// NewGracefulShutdown creates a new graceful shutdown handler
func NewGracefulShutdown(server *http.Server, timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		server:          server,
		shutdownTimeout: timeout,
		cleanupFuncs:    make([]func(), 0),
	}
}

// AddCleanup adds a cleanup function to be called during shutdown
func (gs *GracefulShutdown) AddCleanup(fn func()) {
	gs.cleanupFuncs = append(gs.cleanupFuncs, fn)
}

// Start begins listening for shutdown signals
func (gs *GracefulShutdown) Start() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	
	go func() {
		sig := <-sigChan
		log.Printf("[SHUTDOWN] Received signal: %v", sig)
		
		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), gs.shutdownTimeout)
		defer cancel()
		
		// Attempt graceful shutdown
		log.Println("[SHUTDOWN] Attempting graceful shutdown...")
		if err := gs.server.Shutdown(ctx); err != nil {
			log.Printf("[SHUTDOWN] Error during server shutdown: %v", err)
		}
		
		// Run cleanup functions
		log.Println("[SHUTDOWN] Running cleanup functions...")
		for _, cleanup := range gs.cleanupFuncs {
			cleanup()
		}
		
		log.Println("[SHUTDOWN] Graceful shutdown complete")
	}()
}

// CheckPortAvailable checks if a port is available for binding
func CheckPortAvailable(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("port %s is already in use: %w", port, err)
	}
	ln.Close()
	return nil
}

// FindAvailablePort finds an available port starting from the given port
func FindAvailablePort(startPort int, maxAttempts int) (int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		if err := CheckPortAvailable(fmt.Sprintf("%d", port)); err == nil {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found in range %d-%d", startPort, startPort+maxAttempts-1)
}

// StartServerWithRetry attempts to start the server with retry logic for port binding
func StartServerWithRetry(handler http.Handler, preferredPort string, maxRetries int) (*http.Server, error) {
	port := preferredPort
	
	// First try the preferred port
	if err := CheckPortAvailable(port); err != nil {
		log.Printf("[PORT] Preferred port %s is not available: %v", port, err)
		
		// Try to find an alternative port
		portNum := 0
		fmt.Sscanf(port, "%d", &portNum)
		if portNum > 0 {
			if availablePort, err := FindAvailablePort(portNum+1, maxRetries); err == nil {
				port = fmt.Sprintf("%d", availablePort)
				log.Printf("[PORT] Using alternative port: %s", port)
			} else {
				return nil, fmt.Errorf("failed to find available port: %w", err)
			}
		}
	}
	
	// Configure server
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        handler,
		ReadTimeout:    ReadTimeout,
		WriteTimeout:   WriteTimeout,
		IdleTimeout:    IdleTimeout,
		MaxHeaderBytes: MaxHeaderBytes,
	}
	
	// Set up connection state tracking
	server.ConnState = func(conn net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			log.Printf("[CONNECTION] New connection from %s", conn.RemoteAddr())
		case http.StateClosed:
			log.Printf("[CONNECTION] Connection closed from %s", conn.RemoteAddr())
		}
	}
	
	return server, nil
}

// HandlePortConflict attempts to resolve port binding conflicts
func HandlePortConflict(port string) error {
	log.Printf("[PORT CONFLICT] Attempting to resolve conflict on port %s", port)
	
	// Check if another instance is running
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err == nil {
		conn.Close()
		log.Printf("[PORT CONFLICT] Another service is running on port %s", port)
		
		// Try to send a health check
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return fmt.Errorf("another instance of the application is already running on port %s", port)
			}
		}
		
		return fmt.Errorf("port %s is in use by another application", port)
	}
	
	// Port might be in TIME_WAIT state
	log.Printf("[PORT CONFLICT] Port %s may be in TIME_WAIT state, waiting...", port)
	time.Sleep(5 * time.Second)
	
	// Check again
	if err := CheckPortAvailable(port); err != nil {
		return fmt.Errorf("port %s is still not available after waiting: %w", port, err)
	}
	
	return nil
}

// ReleasePort attempts to forcefully release a port (Windows-specific)
func ReleasePort(port string) error {
	// On Windows, we can try to find and kill the process using the port
	// This is a last resort and should be used carefully
	
	log.Printf("[PORT RELEASE] Attempting to release port %s", port)
	
	// Note: This would require elevated permissions and is generally not recommended
	// in production. It's better to handle port conflicts gracefully.
	
	return fmt.Errorf("automatic port release not implemented for safety reasons")
}