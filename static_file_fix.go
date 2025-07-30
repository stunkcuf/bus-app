package main

import (
	"net/http"
	"path/filepath"
	"strings"
)

// StaticFileHandler serves static files with proper MIME types
func StaticFileHandler(w http.ResponseWriter, r *http.Request) {
	// Remove /static/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	
	// Prevent directory traversal
	if strings.Contains(path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	// Set cache control headers
	w.Header().Set("Cache-Control", "public, max-age=3600")
	
	// Determine content type based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	default:
		// Default to octet-stream for unknown types
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	
	// Add CORS headers if needed
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Serve the file
	http.ServeFile(w, r, filepath.Join("static", path))
}

// UpdateStaticHandler updates the static file handler in main.go
// This function should be called in setupRoutes() to replace the existing handler
func UpdateStaticHandler(mux *http.ServeMux) {
	mux.HandleFunc("/static/", StaticFileHandler)
}