
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"cloud.google.com/go/storage"
)

type ObjectStorage struct {
	client     *storage.Client
	bucketName string
	ctx        context.Context
}

// Initialize Object Storage client
func NewObjectStorage() (*ObjectStorage, error) {
	ctx := context.Background()
	
	// Get bucket name from environment variable
	bucketName := os.Getenv("REPLIT_OBJECT_STORAGE_BUCKET")
	if bucketName == "" {
		return nil, fmt.Errorf("REPLIT_OBJECT_STORAGE_BUCKET environment variable not set")
	}
	
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	
	return &ObjectStorage{
		client:     client,
		bucketName: bucketName,
		ctx:        ctx,
	}, nil
}

// Save JSON data to Object Storage
func (os *ObjectStorage) SaveJSON(filename string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}
	
	obj := os.client.Bucket(os.bucketName).Object(filename)
	w := obj.NewWriter(os.ctx)
	defer w.Close()
	
	if _, err := w.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write to object storage: %v", err)
	}
	
	log.Printf("Saved %s to Object Storage", filename)
	return nil
}

// Load JSON data from Object Storage
func (os *ObjectStorage) LoadJSON(filename string, target interface{}) error {
	obj := os.client.Bucket(os.bucketName).Object(filename)
	r, err := obj.NewReader(os.ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			log.Printf("File %s does not exist in Object Storage, using defaults", filename)
			return err
		}
		return fmt.Errorf("failed to create reader: %v", err)
	}
	defer r.Close()
	
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read data: %v", err)
	}
	
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	
	log.Printf("Loaded %s from Object Storage", filename)
	return nil
}

// Check if file exists in Object Storage
func (os *ObjectStorage) FileExists(filename string) bool {
	obj := os.client.Bucket(os.bucketName).Object(filename)
	_, err := obj.Attrs(os.ctx)
	return err == nil
}

// Migrate local JSON file to Object Storage
func (os *ObjectStorage) MigrateFromLocal(localPath, objectName string) error {
	// Check if local file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		log.Printf("Local file %s does not exist, skipping migration", localPath)
		return nil
	}
	
	// Check if object already exists in storage
	if os.FileExists(objectName) {
		log.Printf("Object %s already exists in storage, skipping migration", objectName)
		return nil
	}
	
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file %s: %v", localPath, err)
	}
	
	// Upload to Object Storage
	obj := os.client.Bucket(os.bucketName).Object(objectName)
	w := obj.NewWriter(os.ctx)
	defer w.Close()
	
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write to object storage: %v", err)
	}
	
	log.Printf("Migrated %s to Object Storage as %s", localPath, objectName)
	return nil
}

// Close the storage client
func (os *ObjectStorage) Close() error {
	return os.client.Close()
}

// Global storage instance
var objStorage *ObjectStorage

// Initialize Object Storage (call this in main)
func initObjectStorage() error {
	var err error
	objStorage, err = NewObjectStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize Object Storage: %v", err)
	}
	
	// Migrate existing local files to Object Storage
	migrations := map[string]string{
		"data/buses.json":            "buses.json",
		"data/users.json":            "users.json",
		"data/routes.json":           "routes.json",
		"data/route_assignments.json": "route_assignments.json",
		"data/students.json":         "students.json",
		"data/driver_logs.json":      "driver_logs.json",
		"data/maintenance.json":      "maintenance.json",
		"data/activities.json":       "activities.json",
		"data/attendance.json":       "attendance.json",
		"data/mileage.json":          "mileage.json",
		"data/vehicle.json":          "vehicle.json",
	}
	
	for localPath, objectName := range migrations {
		if err := objStorage.MigrateFromLocal(localPath, objectName); err != nil {
			log.Printf("Warning: Failed to migrate %s: %v", localPath, err)
		}
	}
	
	return nil
}
