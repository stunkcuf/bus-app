package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"github.com/gorilla/websocket"
)

// responseWriterHijacker wraps ResponseWriter to ensure it implements http.Hijacker
type responseWriterHijacker struct {
	http.ResponseWriter
}

// Ensure our wrapper implements the Hijacker interface
var _ http.Hijacker = (*responseWriterHijacker)(nil)

func (w *responseWriterHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")
	}
	return hijacker.Hijack()
}

// WebSocketFixMiddleware ensures WebSocket upgrades work properly
func WebSocketFixMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a WebSocket upgrade request
		if websocket.IsWebSocketUpgrade(r) {
			// Ensure the ResponseWriter can be hijacked
			if _, ok := w.(http.Hijacker); !ok {
				// Wrap it if it doesn't implement Hijacker
				w = &responseWriterHijacker{w}
			}
		}
		next.ServeHTTP(w, r)
	})
}