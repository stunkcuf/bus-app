package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// SessionStore interface for different session storage backends
type SessionStore interface {
	Get(token string) (*Session, error)
	Set(token string, session *Session) error
	Delete(token string) error
	DeleteByUsername(username string) error
	Cleanup() error
	GetAll() (map[string]*Session, error)
}

// Session represents a user session
type Session struct {
	Username    string            `json:"username"`
	Role        string            `json:"role"`
	CSRFToken   string            `json:"csrf_token"`
	CreatedAt   time.Time         `json:"created_at"`
	LastAccess  time.Time         `json:"last_access"`
	ExpiresAt   time.Time         `json:"expires_at"`
	ImportFiles map[string]string `json:"import_files,omitempty"` // Temporary storage for import file paths
}

// MemorySessionStore is the in-memory implementation
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

func (m *MemorySessionStore) Get(token string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	session, exists := m.sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}
	
	// Update last access time and extend expiration
	session.LastAccess = time.Now()
	// Extend expiration on activity (sliding window)
	session.ExpiresAt = time.Now().Add(24 * time.Hour)
	
	return session, nil
}

func (m *MemorySessionStore) Set(token string, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sessions[token] = session
	return nil
}

func (m *MemorySessionStore) Delete(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.sessions, token)
	return nil
}

func (m *MemorySessionStore) DeleteByUsername(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for token, session := range m.sessions {
		if session.Username == username {
			delete(m.sessions, token)
		}
	}
	return nil
}

func (m *MemorySessionStore) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	for token, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, token)
		}
	}
	return nil
}

func (m *MemorySessionStore) GetAll() (map[string]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to avoid concurrent access issues
	result := make(map[string]*Session)
	for k, v := range m.sessions {
		result[k] = v
	}
	return result, nil
}

// FileSessionStore implements file-based persistent session storage
type FileSessionStore struct {
	mu       sync.RWMutex
	filePath string
	sessions map[string]*Session
}

// NewFileSessionStore creates a new file-based session store
func NewFileSessionStore(filePath string) (*FileSessionStore, error) {
	store := &FileSessionStore{
		filePath: filePath,
		sessions: make(map[string]*Session),
	}
	
	// Load existing sessions from file
	if err := store.load(); err != nil {
		log.Printf("Warning: Could not load sessions from file: %v", err)
		// Not a fatal error - we can start with empty sessions
	}
	
	// Start cleanup routine
	go store.cleanupRoutine()
	
	return store, nil
}

func (f *FileSessionStore) load() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	data, err := readFile(f.filePath)
	if err != nil {
		return err
	}
	
	if len(data) == 0 {
		return nil // Empty file is OK
	}
	
	return json.Unmarshal(data, &f.sessions)
}

func (f *FileSessionStore) save() error {
	data, err := json.Marshal(f.sessions)
	if err != nil {
		return err
	}
	
	return writeFile(f.filePath, data)
}

func (f *FileSessionStore) Get(token string) (*Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	session, exists := f.sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}
	
	// Update last access time and extend expiration
	session.LastAccess = time.Now()
	// Extend expiration on activity (sliding window)
	session.ExpiresAt = time.Now().Add(24 * time.Hour)
	
	// Save the updated session
	f.save()
	
	return session, nil
}

func (f *FileSessionStore) Set(token string, session *Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.sessions[token] = session
	return f.save()
}

func (f *FileSessionStore) Delete(token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	delete(f.sessions, token)
	return f.save()
}

func (f *FileSessionStore) DeleteByUsername(username string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	modified := false
	for token, session := range f.sessions {
		if session.Username == username {
			delete(f.sessions, token)
			modified = true
		}
	}
	
	if modified {
		return f.save()
	}
	return nil
}

func (f *FileSessionStore) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	now := time.Now()
	modified := false
	
	for token, session := range f.sessions {
		if now.After(session.ExpiresAt) {
			delete(f.sessions, token)
			modified = true
		}
	}
	
	if modified {
		return f.save()
	}
	return nil
}

func (f *FileSessionStore) GetAll() (map[string]*Session, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	// Return a copy to avoid concurrent access issues
	result := make(map[string]*Session)
	for k, v := range f.sessions {
		result[k] = v
	}
	return result, nil
}

func (f *FileSessionStore) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		if err := f.Cleanup(); err != nil {
			log.Printf("Error cleaning up sessions: %v", err)
		}
	}
}

// SessionManager handles session operations
type SessionManager struct {
	store SessionStore
}

// NewSessionManager creates a new session manager
func NewSessionManager(store SessionStore) *SessionManager {
	return &SessionManager{
		store: store,
	}
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(username, role string) (string, error) {
	// Generate secure session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	
	// Generate CSRF token
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		return "", err
	}
	csrfToken := base64.URLEncoding.EncodeToString(csrfBytes)
	
	// Create session
	session := &Session{
		Username:   username,
		Role:       role,
		CSRFToken:  csrfToken,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	
	// Store session
	if err := sm.store.Set(token, session); err != nil {
		return "", err
	}
	
	log.Printf("Session created for user %s with token: %s...", username, token[:10])
	return token, nil
}

// GetSession retrieves a session by token
func (sm *SessionManager) GetSession(token string) (*Session, error) {
	return sm.store.Get(token)
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(token string) error {
	return sm.store.Delete(token)
}

// DeleteUserSessions removes all sessions for a user
func (sm *SessionManager) DeleteUserSessions(username string) error {
	return sm.store.DeleteByUsername(username)
}

// GetActiveSessions returns all active sessions
func (sm *SessionManager) GetActiveSessions() (map[string]*Session, error) {
	return sm.store.GetAll()
}

// Global session manager instance
var sessionManager *SessionManager

// InitializeSessionManager sets up the session management system
func initializeSessionManager() error {
	// Check if we should use file-based storage
	sessionFile := getEnvWithDefault("SESSION_STORE_FILE", "sessions.json")
	
	var store SessionStore
	var err error
	
	if sessionFile != "" {
		log.Printf("Using file-based session storage: %s", sessionFile)
		store, err = NewFileSessionStore(sessionFile)
		if err != nil {
			log.Printf("Failed to create file session store, falling back to memory: %v", err)
			store = NewMemorySessionStore()
		}
	} else {
		log.Println("Using in-memory session storage")
		store = NewMemorySessionStore()
	}
	
	sessionManager = NewSessionManager(store)
	
	// Start cleanup routine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			if err := store.Cleanup(); err != nil {
				log.Printf("Error cleaning up sessions: %v", err)
			}
		}
	}()
	
	return nil
}

// Helper functions for file operations
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeFile(path string, data []byte) error {
	// Write to a temporary file first, then rename for atomicity
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}