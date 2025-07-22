package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotatingFileWriter implements a log writer that rotates files
type RotatingFileWriter struct {
	mu          sync.Mutex
	file        *os.File
	filename    string
	maxSize     int64 // Maximum size in bytes before rotation
	maxBackups  int   // Maximum number of old log files to keep
	maxAge      int   // Maximum number of days to keep old logs
	currentSize int64
}

// NewRotatingFileWriter creates a new rotating file writer
func NewRotatingFileWriter(filename string, maxSize int64, maxBackups, maxAge int) (*RotatingFileWriter, error) {
	writer := &RotatingFileWriter{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
	}
	
	if err := writer.openFile(); err != nil {
		return nil, err
	}
	
	// Start cleanup goroutine
	go writer.cleanupOldLogs()
	
	return writer, nil
}

// Write implements io.Writer interface
func (w *RotatingFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// Check if rotation is needed
	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	
	n, err = w.file.Write(p)
	w.currentSize += int64(n)
	return n, err
}

// Close closes the log file
func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// openFile opens the log file
func (w *RotatingFileWriter) openFile() error {
	file, err := os.OpenFile(w.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	
	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	
	w.file = file
	w.currentSize = info.Size()
	return nil
}

// rotate performs log rotation
func (w *RotatingFileWriter) rotate() error {
	// Close current file
	if w.file != nil {
		w.file.Close()
	}
	
	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s.gz", w.filename, timestamp)
	
	// Compress and move the current log file
	if err := w.compressFile(w.filename, backupName); err != nil {
		return err
	}
	
	// Remove the original file after compression
	os.Remove(w.filename)
	
	// Open new file
	if err := w.openFile(); err != nil {
		return err
	}
	
	// Clean up old backups
	w.cleanupBackups()
	
	return nil
}

// compressFile compresses a file using gzip
func (w *RotatingFileWriter) compressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	gzipWriter := gzip.NewWriter(dstFile)
	defer gzipWriter.Close()
	
	_, err = io.Copy(gzipWriter, srcFile)
	return err
}

// cleanupBackups removes old backup files
func (w *RotatingFileWriter) cleanupBackups() {
	dir := filepath.Dir(w.filename)
	base := filepath.Base(w.filename)
	pattern := fmt.Sprintf("%s.*.gz", base)
	
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		log.Printf("Error finding backup files: %v", err)
		return
	}
	
	// Remove oldest files if we exceed maxBackups
	if len(matches) > w.maxBackups {
		// Files are sorted by name (timestamp), so oldest are first
		for i := 0; i < len(matches)-w.maxBackups; i++ {
			if err := os.Remove(matches[i]); err != nil {
				log.Printf("Error removing old backup %s: %v", matches[i], err)
			}
		}
	}
}

// cleanupOldLogs runs periodically to remove logs older than maxAge
func (w *RotatingFileWriter) cleanupOldLogs() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		w.mu.Lock()
		
		dir := filepath.Dir(w.filename)
		base := filepath.Base(w.filename)
		pattern := fmt.Sprintf("%s.*.gz", base)
		
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			log.Printf("Error finding backup files: %v", err)
			w.mu.Unlock()
			continue
		}
		
		cutoff := time.Now().AddDate(0, 0, -w.maxAge)
		
		for _, file := range matches {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			
			if info.ModTime().Before(cutoff) {
				if err := os.Remove(file); err != nil {
					log.Printf("Error removing old log %s: %v", file, err)
				}
			}
		}
		
		w.mu.Unlock()
	}
}

// SetupLogRotation configures log rotation for the application
func SetupLogRotation() error {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	
	// Setup rotating file writer
	// 10MB max size, keep 5 backups, keep for 30 days
	writer, err := NewRotatingFileWriter(
		"logs/fleet-management.log",
		10*1024*1024, // 10MB
		5,            // Keep 5 backups
		30,           // Keep for 30 days
	)
	if err != nil {
		return fmt.Errorf("failed to create rotating file writer: %w", err)
	}
	
	// Create a multi-writer to write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, writer)
	
	// Set the log output
	log.SetOutput(multiWriter)
	
	// Add timestamp to log format
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	log.Println("Log rotation initialized successfully")
	return nil
}