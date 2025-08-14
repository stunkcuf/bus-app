package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// recoveryMiddleware recovers from panics and logs them
func recoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.Printf("[PANIC RECOVERED] %s %s - Error: %v\nStack trace:\n%s", 
					r.Method, r.URL.Path, err, debug.Stack())
				
				// Try to get session info for better debugging
				session, _ := GetSession(r)
				if session != nil {
					log.Printf("[PANIC CONTEXT] User: %s, Role: %s", session.Username, session.Role)
				}
				
				// Send error response to client
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				
				// Render a user-friendly error page
				errorHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Error - HS Bus Fleet Management</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .error-container {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 500px;
            text-align: center;
        }
        .error-icon {
            font-size: 72px;
            color: #dc3545;
            margin-bottom: 20px;
        }
        .error-code {
            font-size: 48px;
            font-weight: bold;
            color: #333;
            margin-bottom: 10px;
        }
        .error-message {
            color: #666;
            margin-bottom: 30px;
        }
        .btn-return {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border: none;
            color: white;
            padding: 12px 30px;
            border-radius: 50px;
            font-weight: 500;
            transition: transform 0.3s;
        }
        .btn-return:hover {
            transform: translateY(-2px);
            color: white;
        }
    </style>
</head>
<body>
    <div class="error-container">
        <div class="error-icon">⚠️</div>
        <div class="error-code">500</div>
        <h4 class="mb-3">Something went wrong</h4>
        <p class="error-message">
            We encountered an unexpected error while processing your request. 
            Our team has been notified and is working to fix the issue.
        </p>
        <div class="d-flex gap-3 justify-content-center">
            <a href="/" class="btn btn-return">Return Home</a>
            <button onclick="window.history.back()" class="btn btn-outline-secondary">Go Back</button>
        </div>
        <div class="mt-4 small text-muted">
            Error ID: ` + fmt.Sprintf("%d", time.Now().Unix()) + `
        </div>
    </div>
</body>
</html>`
				
				fmt.Fprint(w, errorHTML)
			}
		}()
		
		next(w, r)
	}
}

// wrapHandler applies recovery middleware to a handler
func wrapHandler(handler http.HandlerFunc) http.HandlerFunc {
	return recoveryMiddleware(handler)
}

// PanicLogger logs panic information to database for analysis
type PanicLogger struct {
	Timestamp   time.Time
	URL         string
	Method      string
	Error       string
	StackTrace  string
	Username    string
	UserAgent   string
}

// logPanicToDB logs panic information to database
func logPanicToDB(r *http.Request, err interface{}, stack string) {
	// Get session info
	username := "anonymous"
	if session, _ := GetSession(r); session != nil {
		username = session.Username
	}
	
	// Try to log to database (non-blocking)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Failed to log panic to database: %v", r)
			}
		}()
		
		_, dbErr := db.Exec(`
			INSERT INTO error_logs (timestamp, url, method, error, stack_trace, username, user_agent)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, time.Now(), r.URL.Path, r.Method, fmt.Sprint(err), stack, username, r.UserAgent())
		
		if dbErr != nil {
			log.Printf("Error logging panic to database: %v", dbErr)
		}
	}()
}

// recoverFromPanic is a deferred function to recover from panics in goroutines
func recoverFromPanic(context string) {
	if r := recover(); r != nil {
		log.Printf("[GOROUTINE PANIC] Context: %s, Error: %v\nStack: %s", 
			context, r, debug.Stack())
	}
}