
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type StorageClient struct {
	client     *storage.Client
	bucketName string
	useLocal   bool
}

func NewStorageClient() (*StorageClient, error) {
	// Check if we have Object Storage credentials
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT_ID")
	bucketName := os.Getenv("OBJECT_STORAGE_BUCKET")
	
	if projectID == "" || bucketName == "" {
		log.Println("Object Storage not configured, using local filesystem")
		return &StorageClient{useLocal: true}, nil
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON"))))
	if err != nil {
		log.Printf("Failed to create storage client, falling back to local: %v", err)
		return &StorageClient{useLocal: true}, nil
	}

	return &StorageClient{
		client:     client,
		bucketName: bucketName,
		useLocal:   false,
	}, nil
}

func (s *StorageClient) SaveJSON(filename string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if s.useLocal {
		// Ensure directory exists
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		return os.WriteFile(filename, jsonData, 0644)
	}

	// Use Object Storage
	ctx := context.Background()
	obj := s.client.Bucket(s.bucketName).Object(filename)
	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"

	if _, err := w.Write(jsonData); err != nil {
		w.Close()
		return fmt.Errorf("failed to write to object storage: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close object storage writer: %w", err)
	}

	log.Printf("Saved %s to Object Storage", filename)
	return nil
}

func (s *StorageClient) LoadJSON(filename string, target interface{}) error {
	if s.useLocal {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		return json.NewDecoder(file).Decode(target)
	}

	// Use Object Storage
	ctx := context.Background()
	obj := s.client.Bucket(s.bucketName).Object(filename)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to read from object storage: %w", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read object data: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	log.Printf("Loaded %s from Object Storage", filename)
	return nil
}

func (s *StorageClient) FileExists(filename string) bool {
	if s.useLocal {
		_, err := os.Stat(filename)
		return !os.IsNotExist(err)
	}

	// Check Object Storage
	ctx := context.Background()
	obj := s.client.Bucket(s.bucketName).Object(filename)
	_, err := obj.Attrs(ctx)
	return err == nil
}

func (s *StorageClient) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
