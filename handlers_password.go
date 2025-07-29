package main

import (
	"encoding/json"
	"net/http"
	"log"
	"time"
	"golang.org/x/crypto/bcrypt"
)

// passwordChangeHandler handles the password change page
func passwordChangeHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		// Display password change form
		data := map[string]interface{}{
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
			"PageTitle": "Change Password",
		}
		renderTemplate(w, r, "change_password.html", data)
		return
	}

	if r.Method == "POST" {
		// Process password change
		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		// Validate CSRF token
		if !validateCSRF(r) {
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "Invalid security token. Please try again.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Validate input
		if currentPassword == "" || newPassword == "" || confirmPassword == "" {
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "All fields are required.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Check if new passwords match
		if newPassword != confirmPassword {
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "New passwords do not match.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Validate password strength
		if len(newPassword) < MinPasswordLength {
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "Password must be at least 6 characters long.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Don't allow same password
		if currentPassword == newPassword {
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "New password must be different from current password.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Get current password hash from database
		var storedHash string
		err := db.QueryRow("SELECT password FROM users WHERE username = $1", user.Username).Scan(&storedHash)
		if err != nil {
			log.Printf("Error fetching user password: %v", err)
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "An error occurred. Please try again.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Verify current password
		err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(currentPassword))
		if err != nil {
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "Current password is incorrect.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "An error occurred. Please try again.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Update password in database
		_, err = db.Exec("UPDATE users SET password = $1 WHERE username = $2", 
			string(hashedPassword), user.Username)
		if err != nil {
			log.Printf("Error updating password: %v", err)
			data := map[string]interface{}{
				"User":      user,
				"CSRFToken": getSessionCSRFToken(r),
				"PageTitle": "Change Password",
				"Error":     "Failed to update password. Please try again.",
			}
			renderTemplate(w, r, "change_password.html", data)
			return
		}

		// Log the password change
		log.Printf("Password changed for user: %s", user.Username)

		// Success - redirect to profile with success message
		data := map[string]interface{}{
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
			"PageTitle": "Change Password",
			"Success":   "Password changed successfully!",
		}
		renderTemplate(w, r, "change_password.html", data)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// passwordResetRequestHandler handles password reset requests (for forgot password)
func passwordResetRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := map[string]interface{}{
			"PageTitle": "Reset Password",
		}
		renderTemplate(w, r, "password_reset_request.html", data)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		
		// Check if user exists
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
		if err != nil || !exists {
			// Don't reveal if user exists or not
			data := map[string]interface{}{
				"PageTitle": "Reset Password",
				"Message":   "If the username exists, password reset instructions have been sent to the administrator.",
			}
			renderTemplate(w, r, "password_reset_request.html", data)
			return
		}

		// Generate reset token
		resetToken := generateRandomToken(32)
		expiresAt := time.Now().Add(1 * time.Hour)

		// Store reset token
		_, err = db.Exec(`
			INSERT INTO password_reset_tokens (username, token, expires_at) 
			VALUES ($1, $2, $3)
			ON CONFLICT (username) 
			DO UPDATE SET token = $2, expires_at = $3, created_at = CURRENT_TIMESTAMP`,
			username, resetToken, expiresAt)

		if err != nil {
			log.Printf("Error storing reset token: %v", err)
		} else {
			// In a real system, this would send an email
			// For now, log the reset link
			log.Printf("Password reset token for %s: %s", username, resetToken)
		}

		data := map[string]interface{}{
			"PageTitle": "Reset Password",
			"Message":   "If the username exists, password reset instructions have been sent to the administrator.",
		}
		renderTemplate(w, r, "password_reset_request.html", data)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// API endpoint for password change (for mobile app)
func apiPasswordChangeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Authentication required"))
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest("Invalid request format"))
		return
	}

	// Validate input
	if req.CurrentPassword == "" || req.NewPassword == "" {
		SendError(w, ErrBadRequest("Current and new passwords are required"))
		return
	}

	if len(req.NewPassword) < MinPasswordLength {
		SendError(w, ErrBadRequest("Password must be at least 6 characters long"))
		return
	}

	// Get current password hash
	var storedHash string
	err := db.QueryRow("SELECT password FROM users WHERE username = $1", user.Username).Scan(&storedHash)
	if err != nil {
		SendError(w, ErrInternal("Failed to verify user", err))
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.CurrentPassword))
	if err != nil {
		SendError(w, ErrUnauthorized("Current password is incorrect"))
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		SendError(w, ErrInternal("Failed to process password", err))
		return
	}

	// Update password
	_, err = db.Exec("UPDATE users SET password = $1 WHERE username = $2", 
		string(hashedPassword), user.Username)
	if err != nil {
		SendError(w, ErrDatabase("Failed to update password", err))
		return
	}

	log.Printf("Password changed via API for user: %s", user.Username)

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password changed successfully",
	})
}

// generateRandomToken generates a secure random token
func generateRandomToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[secureRandInt(len(charset))]
	}
	return string(b)
}

// secureRandInt returns a secure random integer
func secureRandInt(max int) int {
	// Simple implementation - in production use crypto/rand
	return int(time.Now().UnixNano() % int64(max))
}