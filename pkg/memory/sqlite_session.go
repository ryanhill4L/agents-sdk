package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteSession implements Session using SQLite
type SQLiteSession struct {
	sessionID string
	db        *sql.DB
}

// NewSQLiteSession creates a new SQLite-backed session
func NewSQLiteSession(sessionID, dbPath string) (*SQLiteSession, error) {
	if dbPath == "" {
		dbPath = "sessions.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	session := &SQLiteSession{
		sessionID: sessionID,
		db:        db,
	}

	if err := session.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	return session, nil
}

// initialize creates the necessary tables
func (s *SQLiteSession) initialize() error {
	query := `
    CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id TEXT NOT NULL,
        role TEXT NOT NULL,
        content TEXT NOT NULL,
        metadata TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        INDEX idx_session_id (session_id),
        INDEX idx_created_at (created_at)
    )`

	_, err := s.db.Exec(query)
	return err
}

// GetItems retrieves messages from the session
func (s *SQLiteSession) GetItems(ctx context.Context, limit int) ([]Message, error) {
	query := `
    SELECT role, content, metadata, created_at 
    FROM messages 
    WHERE session_id = ? 
    ORDER BY created_at ASC 
    LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, s.sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var metadataJSON sql.NullString

		err := rows.Scan(&msg.Role, &msg.Content, &metadataJSON, &msg.Timestamp)
		if err != nil {
			return nil, err
		}

		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &msg.Metadata)
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// AddItems adds messages to the session
func (s *SQLiteSession) AddItems(ctx context.Context, items []Message) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO messages (session_id, role, content, metadata, created_at)
        VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, msg := range items {
		var metadataJSON []byte
		if msg.Metadata != nil {
			metadataJSON, _ = json.Marshal(msg.Metadata)
		}

		_, err := stmt.ExecContext(ctx,
			s.sessionID,
			msg.Role,
			msg.Content,
			metadataJSON,
			msg.Timestamp,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// PopItem removes and returns the most recent message
func (s *SQLiteSession) PopItem(ctx context.Context) (*Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var msg Message
	var metadataJSON sql.NullString

	err = tx.QueryRowContext(ctx, `
        SELECT id, role, content, metadata, created_at 
        FROM messages 
        WHERE session_id = ? 
        ORDER BY created_at DESC 
        LIMIT 1
    `, s.sessionID).Scan(&msg.ID, &msg.Role, &msg.Content, &metadataJSON, &msg.Timestamp)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &msg.Metadata)
	}

	// Delete the message
	_, err = tx.ExecContext(ctx, "DELETE FROM messages WHERE id = ?", msg.ID)
	if err != nil {
		return nil, err
	}

	return &msg, tx.Commit()
}

// Clear removes all messages from the session
func (s *SQLiteSession) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM messages WHERE session_id = ?", s.sessionID)
	return err
}

// Close closes the database connection
func (s *SQLiteSession) Close() error {
	return s.db.Close()
}
