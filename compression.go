package main

import (
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// CompressionLevel defines the compression level
type CompressionLevel int

const (
	CompressionNone    CompressionLevel = 0
	CompressionFastest CompressionLevel = 1
	CompressionDefault CompressionLevel = 6
	CompressionBest    CompressionLevel = 9
)

// CompressionMiddleware provides HTTP response compression
type CompressionMiddleware struct {
	level        CompressionLevel
	minSize      int
	contentTypes []string
	excludePaths []string
	writerPool   sync.Pool
	gzipWriters  sync.Pool
	zlibWriters  sync.Pool
}

// NewCompressionMiddleware creates a new compression middleware
func NewCompressionMiddleware(level CompressionLevel) *CompressionMiddleware {
	cm := &CompressionMiddleware{
		level:   level,
		minSize: 1024, // Only compress responses larger than 1KB
		contentTypes: []string{
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"text/xml",
			"text/plain",
			"image/svg+xml",
		},
		excludePaths: []string{
			"/health",
			"/metrics",
		},
	}

	// Initialize writer pools
	cm.gzipWriters = sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, int(level))
			return w
		},
	}

	cm.zlibWriters = sync.Pool{
		New: func() interface{} {
			w, _ := zlib.NewWriterLevel(nil, int(level))
			return w
		},
	}

	return cm
}

// Handler returns the compression middleware handler
func (cm *CompressionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is excluded
		for _, path := range cm.excludePaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check if client accepts compression
		encoding := cm.selectEncoding(r)
		if encoding == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Create compressed response writer
		cw := &compressedResponseWriter{
			ResponseWriter: w,
			encoding:       encoding,
			middleware:     cm,
			minSize:        cm.minSize,
		}
		defer cw.Close()

		// Set content encoding header
		w.Header().Set("Content-Encoding", encoding)
		w.Header().Add("Vary", "Accept-Encoding")

		// Remove content length as it will change
		w.Header().Del("Content-Length")

		// Serve the request
		next.ServeHTTP(cw, r)
	})
}

// selectEncoding selects the best encoding based on Accept-Encoding header
func (cm *CompressionMiddleware) selectEncoding(r *http.Request) string {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if acceptEncoding == "" {
		return ""
	}

	// Parse Accept-Encoding header
	encodings := parseAcceptEncoding(acceptEncoding)

	// Check for gzip support
	if quality, ok := encodings["gzip"]; ok && quality > 0 {
		return "gzip"
	}

	// Check for deflate support
	if quality, ok := encodings["deflate"]; ok && quality > 0 {
		return "deflate"
	}

	return ""
}

// parseAcceptEncoding parses the Accept-Encoding header
func parseAcceptEncoding(header string) map[string]float64 {
	encodings := make(map[string]float64)

	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split encoding and quality
		encParts := strings.Split(part, ";")
		encoding := strings.TrimSpace(encParts[0])
		quality := 1.0

		// Parse quality value
		if len(encParts) > 1 {
			qPart := strings.TrimSpace(encParts[1])
			if strings.HasPrefix(qPart, "q=") {
				qValue := qPart[2:]
				if q, err := parseFloat(qValue); err == nil {
					quality = q
				}
			}
		}

		encodings[encoding] = quality
	}

	return encodings
}

// parseFloat parses a float from string
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// compressedResponseWriter wraps http.ResponseWriter to provide compression
type compressedResponseWriter struct {
	http.ResponseWriter
	encoding    string
	writer      io.WriteCloser
	middleware  *CompressionMiddleware
	minSize     int
	written     int
	wroteHeader bool
	buffer      []byte
}

// Write implements io.Writer
func (cw *compressedResponseWriter) Write(b []byte) (int, error) {
	if !cw.wroteHeader {
		cw.WriteHeader(http.StatusOK)
	}

	// Buffer small responses
	if cw.writer == nil && cw.written+len(b) < cw.minSize {
		cw.buffer = append(cw.buffer, b...)
		cw.written += len(b)
		return len(b), nil
	}

	// Initialize compression writer if needed
	if cw.writer == nil {
		if err := cw.initWriter(); err != nil {
			return 0, err
		}

		// Write buffered data
		if len(cw.buffer) > 0 {
			if _, err := cw.writer.Write(cw.buffer); err != nil {
				return 0, err
			}
			cw.buffer = nil
		}
	}

	n, err := cw.writer.Write(b)
	cw.written += n
	return n, err
}

// WriteHeader implements http.ResponseWriter
func (cw *compressedResponseWriter) WriteHeader(code int) {
	if cw.wroteHeader {
		return
	}

	// Check if content type is compressible
	contentType := cw.Header().Get("Content-Type")
	if !cw.isCompressible(contentType) {
		cw.Header().Del("Content-Encoding")
		cw.Header().Del("Vary")
		cw.ResponseWriter.WriteHeader(code)
		cw.wroteHeader = true
		return
	}

	cw.ResponseWriter.WriteHeader(code)
	cw.wroteHeader = true
}

// Close closes the compression writer
func (cw *compressedResponseWriter) Close() error {
	// Write any buffered data without compression
	if cw.writer == nil && len(cw.buffer) > 0 {
		cw.Header().Del("Content-Encoding")
		cw.Header().Del("Vary")
		if !cw.wroteHeader {
			cw.WriteHeader(http.StatusOK)
		}
		_, err := cw.ResponseWriter.Write(cw.buffer)
		return err
	}

	if cw.writer == nil {
		return nil
	}

	err := cw.writer.Close()

	// Return writer to pool
	switch cw.encoding {
	case "gzip":
		if gw, ok := cw.writer.(*gzip.Writer); ok {
			gw.Reset(nil)
			cw.middleware.gzipWriters.Put(gw)
		}
	case "deflate":
		if zw, ok := cw.writer.(*zlib.Writer); ok {
			zw.Reset(nil)
			cw.middleware.zlibWriters.Put(zw)
		}
	}

	return err
}

// initWriter initializes the compression writer
func (cw *compressedResponseWriter) initWriter() error {
	switch cw.encoding {
	case "gzip":
		gw := cw.middleware.gzipWriters.Get().(*gzip.Writer)
		gw.Reset(cw.ResponseWriter)
		cw.writer = gw
	case "deflate":
		zw := cw.middleware.zlibWriters.Get().(*zlib.Writer)
		zw.Reset(cw.ResponseWriter)
		cw.writer = zw
	default:
		return fmt.Errorf("unsupported encoding: %s", cw.encoding)
	}

	return nil
}

// isCompressible checks if content type should be compressed
func (cw *compressedResponseWriter) isCompressible(contentType string) bool {
	if contentType == "" {
		return false
	}

	// Extract base content type (before semicolon)
	if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	// Check against allowed content types
	for _, ct := range cw.middleware.contentTypes {
		if strings.HasPrefix(contentType, ct) {
			return true
		}
	}

	return false
}

// Flush implements http.Flusher
func (cw *compressedResponseWriter) Flush() {
	if cw.writer != nil {
		// Flush compression writer
		if fw, ok := cw.writer.(http.Flusher); ok {
			fw.Flush()
		}
	}

	// Flush underlying response writer
	if fw, ok := cw.ResponseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

// CompressionConfig holds compression configuration
type CompressionConfig struct {
	Enabled      bool
	Level        CompressionLevel
	MinSize      int
	ContentTypes []string
	ExcludePaths []string
}

// DefaultCompressionConfig returns default compression configuration
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Enabled: true,
		Level:   CompressionDefault,
		MinSize: 1024,
		ContentTypes: []string{
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"text/xml",
			"text/plain",
			"image/svg+xml",
		},
		ExcludePaths: []string{
			"/health",
			"/metrics",
			"/api/ws", // WebSocket endpoints
		},
	}
}

// EnableCompression creates and configures compression middleware
func EnableCompression(config CompressionConfig) func(http.Handler) http.Handler {
	if !config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	cm := NewCompressionMiddleware(config.Level)
	if config.MinSize > 0 {
		cm.minSize = config.MinSize
	}
	if len(config.ContentTypes) > 0 {
		cm.contentTypes = config.ContentTypes
	}
	if len(config.ExcludePaths) > 0 {
		cm.excludePaths = config.ExcludePaths
	}

	return cm.Handler
}
