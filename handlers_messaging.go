package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

// MessagingUser represents a user in the messaging context with ID
type MessagingUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Email    string `json:"email,omitempty"`
}

// Message represents a chat message between users
type Message struct {
	ID           int                    `json:"id"`
	ConversationID string               `json:"conversation_id"`
	SenderID     int                    `json:"sender_id"`
	SenderName   string                 `json:"sender_name"`
	SenderRole   string                 `json:"sender_role"`
	RecipientID  *int                   `json:"recipient_id,omitempty"`
	RecipientName *string              `json:"recipient_name,omitempty"`
	Content      string                 `json:"content"`
	MessageType  string                 `json:"message_type"` // text, location, emergency
	Attachments  []MessageAttachment    `json:"attachments,omitempty"`
	Status       string                 `json:"status"` // sent, delivered, read
	CreatedAt    time.Time              `json:"created_at"`
	ReadAt       *time.Time             `json:"read_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MessageAttachment represents an attachment to a message
type MessageAttachment struct {
	ID       int    `json:"id"`
	Type     string `json:"type"` // image, document, location
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"` // direct, group, broadcast
	Name         string    `json:"name"`
	Participants []MessagingUser    `json:"participants"`
	LastMessage  *Message  `json:"last_message,omitempty"`
	UnreadCount  int       `json:"unread_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// messagingHandler serves the messaging interface
func messagingHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get conversations for the user
	userID := getUserID(session.Username)
	conversations, err := getUserConversations(userID)
	if err != nil {
		log.Printf("Failed to get conversations: %v (user_id=%d)", err, userID)
	}

	// Get contacts (other drivers and managers)
	contacts, err := getAvailableContacts(session)
	if err != nil {
		log.Printf("Failed to get contacts: %v (user_id=%d)", err, userID)
	}

	data := struct {
		Title         string
		Username      string
		UserType      string
		UserID        int
		CSPNonce      string
		Conversations []Conversation
		Contacts      []MessagingUser
	}{
		Title:         "Messaging",
		Username:      session.Username,
		UserType:      session.Role,
		UserID:        userID,
		CSPNonce:      generateNonce(),
		Conversations: conversations,
		Contacts:      contacts,
	}

	tmpl := template.Must(template.ParseFiles("templates/messaging.html"))
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error rendering messaging page: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// API Endpoints

// getConversationHandler returns messages for a conversation
func getConversationHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID := r.URL.Query().Get("id")
	if conversationID == "" {
		http.Error(w, "Conversation ID required", http.StatusBadRequest)
		return
	}

	// Verify user has access to conversation
	userID := getUserID(session.Username)
	hasAccess, err := userHasAccessToConversation(userID, conversationID)
	if err != nil || !hasAccess {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get messages
	messages, err := getConversationMessages(conversationID, 50)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		http.Error(w, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	// Mark messages as read
	markMessagesAsRead(conversationID, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"conversation_id": conversationID,
	})
}

// sendMessageHandler handles sending a new message
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ConversationID string                 `json:"conversation_id"`
		RecipientID    *int                   `json:"recipient_id"`
		Content        string                 `json:"content"`
		MessageType    string                 `json:"message_type"`
		Metadata       map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate content
	if req.Content == "" {
		http.Error(w, "Message content required", http.StatusBadRequest)
		return
	}

	// Create or get conversation
	conversationID := req.ConversationID
	if conversationID == "" && req.RecipientID != nil {
		// Create direct conversation
		var err error
		conversationID, err = createDirectConversation(getUserID(session.Username), *req.RecipientID)
		if err != nil {
			log.Printf("Failed to create conversation: %v", err)
			http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
			return
		}
	}

	// Create message
	message := Message{
		ConversationID: conversationID,
		SenderID:       getUserID(session.Username),
		SenderName:     session.Username,
		SenderRole:     session.Role,
		RecipientID:    req.RecipientID,
		Content:        req.Content,
		MessageType:    req.MessageType,
		Status:         "sent",
		CreatedAt:      time.Now(),
		Metadata:       req.Metadata,
	}

	if message.MessageType == "" {
		message.MessageType = "text"
	}

	// Save message
	messageID, err := saveMessage(&message)
	if err != nil {
		log.Printf("Failed to save message: %v", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}
	message.ID = messageID

	// Broadcast via WebSocket
	broadcastMessage(&message)

	// Send push notification if recipient is offline
	if req.RecipientID != nil {
		sendMessageNotification(*req.RecipientID, &message)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
		"conversation_id": conversationID,
	})
}

// createConversationHandler creates a new conversation
func createConversationHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Type         string `json:"type"`
		Name         string `json:"name"`
		ParticipantIDs []int `json:"participant_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" {
		req.Type = "direct"
	}

	if len(req.ParticipantIDs) == 0 {
		http.Error(w, "Participants required", http.StatusBadRequest)
		return
	}

	// Create conversation
	userID := getUserID(session.Username)
	conversationID := fmt.Sprintf("conv_%d_%d", time.Now().Unix(), userID)
	
	// Add current user to participants
	participants := append(req.ParticipantIDs, userID)
	
	// Save conversation
	err := createConversation(conversationID, req.Type, req.Name, participants)
	if err != nil {
		log.Printf("Failed to create conversation: %v", err)
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"conversation_id": conversationID,
	})
}

// Database functions

func getUserConversations(userID int) ([]Conversation, error) {
	query := `
		SELECT DISTINCT c.id, c.type, c.name, c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM messages m 
		        WHERE m.conversation_id = c.id 
		        AND m.sender_id != $1 
		        AND (m.read_at IS NULL OR m.status != 'read'))
		FROM conversations c
		JOIN conversation_participants cp ON c.id = cp.conversation_id
		WHERE cp.user_id = $1
		ORDER BY c.updated_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		err := rows.Scan(&conv.ID, &conv.Type, &conv.Name, 
			&conv.CreatedAt, &conv.UpdatedAt, &conv.UnreadCount)
		if err != nil {
			log.Printf("Failed to scan conversation: %v", err)
			continue
		}

		// Get last message
		conv.LastMessage, _ = getLastMessage(conv.ID)
		
		// Get participants
		conv.Participants, _ = getConversationParticipants(conv.ID)

		conversations = append(conversations, conv)
	}

	return conversations, nil
}

func getConversationMessages(conversationID string, limit int) ([]Message, error) {
	query := `
		SELECT m.id, m.conversation_id, m.sender_id, u.username, u.role,
		       m.recipient_id, m.content, m.message_type, m.status,
		       m.created_at, m.read_at, m.metadata
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2
	`

	rows, err := db.Query(query, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var metadataJSON []byte
		
		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, 
			&msg.SenderName, &msg.SenderRole, &msg.RecipientID,
			&msg.Content, &msg.MessageType, &msg.Status,
			&msg.CreatedAt, &msg.ReadAt, &metadataJSON)
		if err != nil {
			log.Printf("Failed to scan message: %v", err)
			continue
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &msg.Metadata)
		}

		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func saveMessage(msg *Message) (int, error) {
	metadataJSON, _ := json.Marshal(msg.Metadata)
	
	var messageID int
	err := db.QueryRow(`
		INSERT INTO messages (conversation_id, sender_id, recipient_id,
		                     content, message_type, status, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, msg.ConversationID, msg.SenderID, msg.RecipientID,
	   msg.Content, msg.MessageType, msg.Status, metadataJSON, msg.CreatedAt).Scan(&messageID)

	if err != nil {
		return 0, err
	}

	// Update conversation timestamp
	_, err = db.Exec(`
		UPDATE conversations 
		SET updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1
	`, msg.ConversationID)

	return messageID, err
}

func markMessagesAsRead(conversationID string, userID int) error {
	_, err := db.Exec(`
		UPDATE messages 
		SET status = 'read', read_at = CURRENT_TIMESTAMP
		WHERE conversation_id = $1 
		AND sender_id != $2 
		AND (read_at IS NULL OR status != 'read')
	`, conversationID, userID)
	return err
}

func createDirectConversation(user1ID, user2ID int) (string, error) {
	// Check if conversation already exists
	var existingID string
	err := db.QueryRow(`
		SELECT c.id FROM conversations c
		JOIN conversation_participants cp1 ON c.id = cp1.conversation_id
		JOIN conversation_participants cp2 ON c.id = cp2.conversation_id
		WHERE c.type = 'direct' 
		AND cp1.user_id = $1 
		AND cp2.user_id = $2
	`, user1ID, user2ID).Scan(&existingID)

	if err == nil {
		return existingID, nil
	}

	// Create new conversation
	conversationID := fmt.Sprintf("direct_%d_%d_%d", user1ID, user2ID, time.Now().Unix())
	
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Insert conversation
	_, err = tx.Exec(`
		INSERT INTO conversations (id, type, name, created_at, updated_at)
		VALUES ($1, 'direct', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, conversationID)
	if err != nil {
		return "", err
	}

	// Add participants
	_, err = tx.Exec(`
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
	`, conversationID, user1ID, user2ID)
	if err != nil {
		return "", err
	}

	return conversationID, tx.Commit()
}

// WebSocket broadcasting
func broadcastMessage(msg *Message) {
	if wsHub == nil {
		return
	}

	wsMessage := WSMessage{
		Type: "chat_message",
		Data: map[string]interface{}{
			"message": msg,
		},
		Timestamp: time.Now(),
	}

	messageJSON, _ := json.Marshal(wsMessage)

	// Get conversation participants
	participants, _ := getConversationParticipants(msg.ConversationID)
	participantIDs := make(map[int]bool)
	for _, p := range participants {
		participantIDs[p.ID] = true
	}

	// Broadcast to participants
	wsHub.mu.RLock()
	defer wsHub.mu.RUnlock()

	for client := range wsHub.clients {
		if client.user != nil {
			clientUserID := getUserID(client.user.Username)
			if participantIDs[clientUserID] {
				select {
				case client.send <- messageJSON:
				default:
				}
			}
		}
	}
}

// Helper functions
func getAvailableContacts(currentUser *User) ([]MessagingUser, error) {
	var query string
	var args []interface{}

	if currentUser.Role == "manager" {
		// Managers can contact all users
		query = `
			SELECT id, username, email, role, phone 
			FROM users 
			WHERE id != $1 AND approved = true
			ORDER BY role, username
		`
		args = []interface{}{getUserID(currentUser.Username)}
	} else {
		// Drivers can contact managers and drivers on same routes
		query = `
			SELECT DISTINCT u.id, u.username, u.email, u.role, u.phone
			FROM users u
			WHERE u.id != $1 AND u.approved = true
			AND (u.role = 'manager' OR 
			     EXISTS (
			         SELECT 1 FROM route_assignments ra1
			         JOIN route_assignments ra2 ON ra1.route_id = ra2.route_id
			         WHERE ra1.driver = $2 AND ra2.driver = u.username
			     ))
			ORDER BY u.role, u.username
		`
		args = []interface{}{getUserID(currentUser.Username), currentUser.Username}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []MessagingUser
	for rows.Next() {
		var user MessagingUser
		var email, phone sql.NullString
		err := rows.Scan(&user.ID, &user.Username, &email, &user.Role, &phone)
		if err != nil {
			continue
		}
		if email.Valid {
			user.Email = email.String
		}
		contacts = append(contacts, user)
	}

	return contacts, nil
}

func userHasAccessToConversation(userID int, conversationID string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM conversation_participants
			WHERE conversation_id = $1 AND user_id = $2
		)
	`, conversationID, userID).Scan(&exists)
	return exists, err
}

func getLastMessage(conversationID string) (*Message, error) {
	var msg Message
	
	err := db.QueryRow(`
		SELECT m.id, m.sender_id, u.username, m.content, 
		       m.message_type, m.created_at
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id = $1
		ORDER BY m.created_at DESC
		LIMIT 1
	`, conversationID).Scan(&msg.ID, &msg.SenderID, &msg.SenderName,
		&msg.Content, &msg.MessageType, &msg.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &msg, nil
}

func getConversationParticipants(conversationID string) ([]MessagingUser, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.role
		FROM users u
		JOIN conversation_participants cp ON u.id = cp.user_id
		WHERE cp.conversation_id = $1
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []MessagingUser
	for rows.Next() {
		var user MessagingUser
		err := rows.Scan(&user.ID, &user.Username, &user.Role)
		if err != nil {
			continue
		}
		participants = append(participants, user)
	}

	return participants, nil
}

func createConversation(id, convType, name string, participantIDs []int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert conversation
	_, err = tx.Exec(`
		INSERT INTO conversations (id, type, name, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, convType, name)
	if err != nil {
		return err
	}

	// Add participants
	for _, userID := range participantIDs {
		_, err = tx.Exec(`
			INSERT INTO conversation_participants (conversation_id, user_id)
			VALUES ($1, $2)
		`, id, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func sendMessageNotification(recipientID int, msg *Message) {
	// Create notification
	notification := Notification{
		ID:       generateNotificationID(),
		Type:     "message",
		Priority: "medium",
		Recipients: []Recipient{
			{UserID: strconv.Itoa(recipientID)},
		},
		Subject: fmt.Sprintf("New message from %s", msg.SenderName),
		Message: msg.Content,
		Data: map[string]interface{}{
			"conversation_id": msg.ConversationID,
			"sender_id":       msg.SenderID,
			"message_id":      msg.ID,
		},
		Channels:  []string{"in-app", "push"},
		CreatedAt: time.Now(),
	}

	// Send via notification system
	if notificationSystem != nil {
		notificationSystem.Send(notification)
	}
}