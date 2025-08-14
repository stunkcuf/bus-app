# Fleet Management System - Log Issues Fix Plan
## Analysis Date: January 2025

---

## 🔴 Critical Issues to Fix

### 1. WebSocket Connection Failures (620 occurrences)
**Error**: `WebSocket upgrade failed: websocket: response does not implement http.Hijacker`

**Root Cause**: The HTTP ResponseWriter doesn't support hijacking, which is required for WebSocket upgrades.

**Fix Plan**:
```go
// In websocket_realtime.go, check if hijacking is supported before upgrade
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Check if ResponseWriter supports hijacking
    hijacker, ok := w.(http.Hijacker)
    if !ok {
        // Fallback to non-WebSocket communication or skip
        log.Printf("WebSocket not supported in this environment")
        http.Error(w, "WebSocket not supported", http.StatusNotImplemented)
        return
    }
    
    // Proceed with WebSocket upgrade
    upgrader.Upgrade(w, r, nil)
}
```

**Alternative**: Remove WebSocket dependency if real-time features aren't critical.

---

### 2. Application Panics (24 occurrences)
**Error**: Runtime panics in template processing

**Fix Plan**:
```go
// Add panic recovery middleware in main.go
func recoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("PANIC RECOVERED: %v", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next(w, r)
    }
}

// Wrap all handlers with recovery middleware
http.HandleFunc("/", recoverMiddleware(handleLogin))
```

---

### 3. Socket Binding Errors (36 occurrences)
**Error**: `bind: Only one usage of each socket address`

**Fix Plan**:
```go
// In main.go, add graceful shutdown
func main() {
    // Check if port is already in use before binding
    listener, err := net.Listen("tcp", ":"+port)
    if err != nil {
        log.Fatalf("Port %s is already in use: %v", port, err)
    }
    listener.Close()
    
    // Setup graceful shutdown
    srv := &http.Server{Addr: ":" + port}
    
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
        <-sigChan
        
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        srv.Shutdown(ctx)
    }()
    
    log.Fatal(srv.ListenAndServe())
}
```

---

### 4. ECSE Data Cleanup Failures (80 occurrences)
**Error**: `invalid input syntax for type date: ""`

**Fix Plan**:
```sql
-- In fix_database_sync_issues.go, update the cleanup query
UPDATE ecse_students 
SET birthdate = NULL 
WHERE birthdate = '' OR birthdate IS NULL;

-- Or set a default date
UPDATE ecse_students 
SET birthdate = '2000-01-01' 
WHERE birthdate = '' OR birthdate IS NULL OR birthdate = '0000-00-00';
```

---

## 🟡 Major Issues to Fix

### 5. Database Connection Errors (19 occurrences)
**Fix Plan**:
```go
// Add connection retry logic in database.go
func executeQueryWithRetry(query string, args ...interface{}) (*sql.Rows, error) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        rows, err := db.Query(query, args...)
        if err == nil {
            return rows, nil
        }
        
        // Check if it's a connection error
        if isConnectionError(err) {
            log.Printf("Database connection error, retry %d/%d", i+1, maxRetries)
            time.Sleep(time.Second * time.Duration(i+1))
            continue
        }
        
        return nil, err
    }
    return nil, fmt.Errorf("database query failed after %d retries", maxRetries)
}
```

---

## 🟢 Minor Issues & Improvements

### 6. Authentication Failures
**Recommendation**: These are likely legitimate. Add rate limiting:
```go
// Implement rate limiting for login attempts
var loginAttempts = make(map[string][]time.Time)

func rateLimitLogin(username string) bool {
    attempts := loginAttempts[username]
    now := time.Now()
    
    // Remove old attempts (older than 15 minutes)
    var validAttempts []time.Time
    for _, t := range attempts {
        if now.Sub(t) < 15*time.Minute {
            validAttempts = append(validAttempts, t)
        }
    }
    
    if len(validAttempts) >= 5 {
        return false // Too many attempts
    }
    
    loginAttempts[username] = append(validAttempts, now)
    return true
}
```

---

## 📋 Implementation Priority

### Phase 1 - Immediate (This Week)
1. ✅ Fix WebSocket compatibility issue
2. ✅ Add panic recovery middleware
3. ✅ Fix ECSE birthdate data issue
4. ✅ Implement graceful shutdown

### Phase 2 - High Priority (Next Week)
1. ⬜ Add database connection pooling and retry logic
2. ⬜ Implement proper error handling throughout
3. ⬜ Add monitoring and alerting

### Phase 3 - Medium Priority (This Month)
1. ⬜ Implement rate limiting for authentication
2. ⬜ Improve validation error messages
3. ⬜ Add comprehensive logging strategy

---

## 🛠️ Quick Fixes to Apply Now

### 1. Disable WebSocket if not needed
```go
// In main.go, comment out WebSocket routes temporarily
// http.HandleFunc("/ws", handleWebSocket)
```

### 2. Fix ECSE birthdate issue
```sql
-- Run this SQL directly on the database
UPDATE ecse_students 
SET birthdate = NULL 
WHERE birthdate = '' OR birthdate = '0000-00-00';
```

### 3. Add startup port check
```go
// Add to main.go before starting server
if err := checkPortAvailable(port); err != nil {
    log.Fatalf("Cannot start: %v", err)
}
```

---

## 📊 Success Metrics

After implementing fixes, monitor for:
- Zero WebSocket errors (or feature disabled)
- Zero panic occurrences
- Zero port binding errors
- Successful ECSE data cleanup
- Database errors reduced by 90%
- Authentication failures only for invalid credentials

---

## 🔍 Monitoring Recommendations

1. Set up log rotation to prevent large log files
2. Implement structured logging with levels (DEBUG, INFO, WARN, ERROR)
3. Add health check endpoint that validates all critical components
4. Create dashboard for real-time error monitoring
5. Set up alerts for critical errors

---

**Next Steps**:
1. Review and approve this fix plan
2. Create feature branch for fixes
3. Implement fixes in priority order
4. Test each fix thoroughly
5. Deploy to staging first
6. Monitor logs for 24 hours before production deployment