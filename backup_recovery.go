package main

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupManager handles system backups
type BackupManager struct {
	db         *sql.DB
	backupPath string
}

// BackupMetadata contains backup information
type BackupMetadata struct {
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Type        string                 `json:"type"`
	Tables      []string               `json:"tables"`
	RecordCount map[string]int         `json:"record_count"`
	Size        int64                  `json:"size"`
	Checksum    string                 `json:"checksum"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(db *sql.DB, backupPath string) *BackupManager {
	return &BackupManager{
		db:         db,
		backupPath: backupPath,
	}
}

// CreateFullBackup creates a complete system backup
func (bm *BackupManager) CreateFullBackup() (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("fleet_backup_%s.zip", timestamp)
	backupFile := filepath.Join(bm.backupPath, backupName)
	
	log.Printf("Starting full backup to %s", backupFile)
	
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(bm.backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %v", err)
	}
	
	// Create zip file
	zipFile, err := os.Create(backupFile)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %v", err)
	}
	defer zipFile.Close()
	
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	
	metadata := BackupMetadata{
		Timestamp:   time.Now(),
		Version:     "1.0",
		Type:        "full",
		Tables:      []string{},
		RecordCount: make(map[string]int),
	}
	
	// Backup database tables
	tables := []string{
		"users", "buses", "vehicles", "routes", "students",
		"route_assignments", "maintenance_records", "fuel_records",
		"driver_logs", "monthly_mileage_reports", "service_records",
		"ecse_data",
	}
	
	for _, table := range tables {
		count, err := bm.backupTable(zipWriter, table)
		if err != nil {
			log.Printf("Warning: Failed to backup table %s: %v", table, err)
			continue
		}
		metadata.Tables = append(metadata.Tables, table)
		metadata.RecordCount[table] = count
	}
	
	// Backup configuration files
	configFiles := []string{
		".env",
		"sessions.json",
	}
	
	for _, file := range configFiles {
		if err := bm.addFileToZip(zipWriter, file); err != nil {
			log.Printf("Warning: Failed to backup file %s: %v", file, err)
		}
	}
	
	// Add metadata
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataWriter, err := zipWriter.Create("backup_metadata.json")
	if err == nil {
		metadataWriter.Write(metadataJSON)
	}
	
	log.Printf("Backup completed: %s", backupFile)
	return backupFile, nil
}

// backupTable backs up a single database table
func (bm *BackupManager) backupTable(zipWriter *zip.Writer, tableName string) (int, error) {
	// Create file in zip
	writer, err := zipWriter.Create(fmt.Sprintf("database/%s.json", tableName))
	if err != nil {
		return 0, err
	}
	
	// Query all data from table
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	rows, err := bm.db.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	
	// Prepare data structures
	count := 0
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	
	// Write JSON array start
	writer.Write([]byte("[\n"))
	
	// Process rows
	first := true
	for rows.Next() {
		if !first {
			writer.Write([]byte(",\n"))
		}
		first = false
		
		// Scan row
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		
		// Convert to map
		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = values[i]
		}
		
		// Write JSON
		jsonData, _ := json.Marshal(record)
		writer.Write(jsonData)
		count++
	}
	
	// Write JSON array end
	writer.Write([]byte("\n]"))
	
	return count, nil
}

// addFileToZip adds a file to the zip archive
func (bm *BackupManager) addFileToZip(zipWriter *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Get file info
	info, err := file.Stat()
	if err != nil {
		return err
	}
	
	// Create zip header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.Join("config", filename)
	
	// Create file in zip
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	
	// Copy file content
	_, err = io.Copy(writer, file)
	return err
}

// RestoreFromBackup restores system from a backup file
func (bm *BackupManager) RestoreFromBackup(backupFile string) error {
	log.Printf("Starting restore from %s", backupFile)
	
	// Open backup file
	reader, err := zip.OpenReader(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %v", err)
	}
	defer reader.Close()
	
	// Read metadata
	var metadata BackupMetadata
	for _, file := range reader.File {
		if file.Name == "backup_metadata.json" {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			json.NewDecoder(rc).Decode(&metadata)
			rc.Close()
			break
		}
	}
	
	// Begin transaction
	tx, err := bm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	
	// Restore tables
	for _, file := range reader.File {
		if filepath.Dir(file.Name) == "database" {
			tableName := filepath.Base(file.Name)
			tableName = tableName[:len(tableName)-5] // Remove .json
			
			if err := bm.restoreTable(tx, file, tableName); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to restore table %s: %v", tableName, err)
			}
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit restore: %v", err)
	}
	
	log.Printf("Restore completed successfully")
	return nil
}

// restoreTable restores a single table from backup
func (bm *BackupManager) restoreTable(tx *sql.Tx, file *zip.File, tableName string) error {
	// Open file
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	
	// Parse JSON
	var records []map[string]interface{}
	if err := json.NewDecoder(rc).Decode(&records); err != nil {
		return err
	}
	
	// Clear existing data
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", tableName)); err != nil {
		log.Printf("Warning: Failed to clear table %s: %v", tableName, err)
	}
	
	// Insert records
	for _, record := range records {
		columns := make([]string, 0, len(record))
		values := make([]interface{}, 0, len(record))
		placeholders := make([]string, 0, len(record))
		
		i := 1
		for col, val := range record {
			columns = append(columns, col)
			values = append(values, val)
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			i++
		}
		
		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			tableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)
		
		if _, err := tx.Exec(query, values...); err != nil {
			log.Printf("Warning: Failed to insert record into %s: %v", tableName, err)
		}
	}
	
	return nil
}

// CreateIncrementalBackup creates a backup of changes since last full backup
func (bm *BackupManager) CreateIncrementalBackup(sinceTime time.Time) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("fleet_incremental_%s.zip", timestamp)
	backupFile := filepath.Join(bm.backupPath, backupName)
	
	log.Printf("Starting incremental backup since %s", sinceTime.Format(time.RFC3339))
	
	// This would backup only records modified since sinceTime
	// Implementation depends on having updated_at columns
	
	return backupFile, nil
}

// ScheduleAutomaticBackups sets up automatic backup schedule
func ScheduleAutomaticBackups(bm *BackupManager) {
	// Daily backup at 2 AM
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
			duration := next.Sub(now)
			
			time.Sleep(duration)
			
			if _, err := bm.CreateFullBackup(); err != nil {
				log.Printf("Automatic backup failed: %v", err)
			}
			
			// Clean old backups (keep last 7 days)
			bm.CleanOldBackups(7 * 24 * time.Hour)
		}
	}()
}

// CleanOldBackups removes backups older than retention period
func (bm *BackupManager) CleanOldBackups(retention time.Duration) error {
	files, err := os.ReadDir(bm.backupPath)
	if err != nil {
		return err
	}
	
	cutoff := time.Now().Add(-retention)
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		info, err := file.Info()
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			backupFile := filepath.Join(bm.backupPath, file.Name())
			if err := os.Remove(backupFile); err != nil {
				log.Printf("Failed to remove old backup %s: %v", file.Name(), err)
			} else {
				log.Printf("Removed old backup: %s", file.Name())
			}
		}
	}
	
	return nil
}

// ExportData exports specific data in various formats
func ExportData(format string, tables []string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("fleet_export_%s.%s", timestamp, format)
	
	switch format {
	case "csv":
		return exportToCSV(filename, tables)
	case "xlsx":
		return exportToExcel(filename, tables)
	case "json":
		return exportToJSON(filename, tables)
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportToCSV exports data to CSV format
func exportToCSV(filename string, tables []string) (string, error) {
	// Implementation for CSV export
	// This would create CSV files for each table
	return filename, nil
}

// exportToExcel exports data to Excel format
func exportToExcel(filename string, tables []string) (string, error) {
	// Implementation for Excel export
	// This would require a library like excelize
	return filename, nil
}

// exportToJSON exports data to JSON format
func exportToJSON(filename string, tables []string) (string, error) {
	// Implementation for JSON export
	// Similar to backup but in a more readable format
	return filename, nil
}