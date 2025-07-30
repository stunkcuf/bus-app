package main

import (
	"database/sql"
	"log"
)

// createMessagingTables creates the tables needed for the messaging system
func createMessagingTables(db *sql.DB) error {
	// Create conversations table
	conversationsTable := `
	CREATE TABLE IF NOT EXISTS conversations (
		id VARCHAR(100) PRIMARY KEY,
		type VARCHAR(20) NOT NULL DEFAULT 'direct', -- direct, group, broadcast
		name VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at DESC);
	`

	// Create conversation participants table
	participantsTable := `
	CREATE TABLE IF NOT EXISTS conversation_participants (
		conversation_id VARCHAR(100) REFERENCES conversations(id) ON DELETE CASCADE,
		user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_read_at TIMESTAMP,
		PRIMARY KEY (conversation_id, user_id)
	);
	CREATE INDEX IF NOT EXISTS idx_participants_user ON conversation_participants(user_id);
	`

	// Create messages table
	messagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		conversation_id VARCHAR(100) REFERENCES conversations(id) ON DELETE CASCADE,
		sender_id INTEGER REFERENCES users(id),
		recipient_id INTEGER REFERENCES users(id), -- For direct messages
		content TEXT NOT NULL,
		message_type VARCHAR(20) DEFAULT 'text', -- text, location, emergency, system
		status VARCHAR(20) DEFAULT 'sent', -- sent, delivered, read
		metadata JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		read_at TIMESTAMP,
		CONSTRAINT check_message_type CHECK (message_type IN ('text', 'location', 'emergency', 'system')),
		CONSTRAINT check_status CHECK (status IN ('sent', 'delivered', 'read'))
	);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
	CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(conversation_id, sender_id, read_at) WHERE read_at IS NULL;
	`

	// Create message attachments table
	attachmentsTable := `
	CREATE TABLE IF NOT EXISTS message_attachments (
		id SERIAL PRIMARY KEY,
		message_id INTEGER REFERENCES messages(id) ON DELETE CASCADE,
		type VARCHAR(20) NOT NULL, -- image, document, location
		url TEXT NOT NULL,
		filename VARCHAR(255),
		size BIGINT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_attachments_message ON message_attachments(message_id);
	`

	// Execute table creation
	tables := []struct {
		name  string
		query string
	}{
		{"conversations", conversationsTable},
		{"conversation_participants", participantsTable},
		{"messages", messagesTable},
		{"message_attachments", attachmentsTable},
	}

	for _, table := range tables {
		log.Printf("Creating %s table...", table.name)
		if _, err := db.Exec(table.query); err != nil {
			return err
		}
	}

	// Add trigger to update conversation timestamp when new message is added
	trigger := `
	CREATE OR REPLACE FUNCTION update_conversation_timestamp()
	RETURNS TRIGGER AS $$
	BEGIN
		UPDATE conversations 
		SET updated_at = CURRENT_TIMESTAMP 
		WHERE id = NEW.conversation_id;
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS update_conversation_on_message ON messages;
	
	CREATE TRIGGER update_conversation_on_message
	AFTER INSERT ON messages
	FOR EACH ROW
	EXECUTE FUNCTION update_conversation_timestamp();
	`

	if _, err := db.Exec(trigger); err != nil {
		log.Printf("Warning: Failed to create trigger: %v", err)
		// Non-critical, continue
	}

	log.Println("Messaging tables created successfully")
	return nil
}