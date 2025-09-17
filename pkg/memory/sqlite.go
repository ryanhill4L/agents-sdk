package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type SQLiteSession struct {
	sessionID string
	db        *sql.DB
	dbPath    string
}

func NewSQLiteSession(dbPath, sessionID string) (*SQLiteSession, error) {
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			dbPath = "sessions.db"
		} else {
			dbPath = filepath.Join(homeDir, ".claude-code", "sessions.db")
			os.MkdirAll(filepath.Dir(dbPath), 0755)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	session := &SQLiteSession{
		sessionID: sessionID,
		db:        db,
		dbPath:    dbPath,
	}

	if err := session.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	return session, nil
}

func (s *SQLiteSession) initialize() error {
	query := `
    CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id TEXT NOT NULL,
        role TEXT NOT NULL,
        content TEXT NOT NULL,
        metadata TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_session_id ON messages (session_id);
    CREATE INDEX IF NOT EXISTS idx_created_at ON messages (created_at);`

	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteSession) Load() ([]types.Message, error) {
	query := `
    SELECT role, content, metadata, created_at
    FROM messages
    WHERE session_id = ?
    ORDER BY created_at ASC`

	rows, err := s.db.Query(query, s.sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var role, content string
		var metadataJSON sql.NullString
		var timestamp time.Time

		err := rows.Scan(&role, &content, &metadataJSON, &timestamp)
		if err != nil {
			return nil, err
		}

		var metadata map[string]any
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &metadata)
		}

		var msg types.Message
		switch types.Role(role) {
		case types.RoleUser:
			msg = &types.UserMessage{
				BaseMessage: types.BaseMessage{
					Role: types.RoleUser,
					Content: []types.ContentBlock{
						&types.TextBlock{Text: content},
					},
					Timestamp: timestamp,
					Metadata:  metadata,
				},
			}
		case types.RoleAssistant:
			msg = &types.AssistantMessage{
				BaseMessage: types.BaseMessage{
					Role: types.RoleAssistant,
					Content: []types.ContentBlock{
						&types.TextBlock{Text: content},
					},
					Timestamp: timestamp,
					Metadata:  metadata,
				},
			}
		case types.RoleSystem:
			msg = &types.SystemMessage{
				BaseMessage: types.BaseMessage{
					Role: types.RoleSystem,
					Content: []types.ContentBlock{
						&types.TextBlock{Text: content},
					},
					Timestamp: timestamp,
					Metadata:  metadata,
				},
			}
		case types.RoleTool:
			toolMsg := &types.ToolMessage{
				BaseMessage: types.BaseMessage{
					Role: types.RoleTool,
					Content: []types.ContentBlock{
						&types.TextBlock{Text: content},
					},
					Timestamp: timestamp,
					Metadata:  metadata,
				},
			}
			if toolCallID, ok := metadata["tool_call_id"].(string); ok {
				toolMsg.ToolCallID = toolCallID
			}
			if toolName, ok := metadata["tool_name"].(string); ok {
				toolMsg.ToolName = toolName
			}
			if isError, ok := metadata["is_error"].(bool); ok {
				toolMsg.IsError = isError
			}
			msg = toolMsg
		}

		if msg != nil {
			messages = append(messages, msg)
		}
	}

	if len(messages) == 0 {
		return nil, ErrSessionNotFound
	}

	return messages, rows.Err()
}

func (s *SQLiteSession) Save(messages []types.Message) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM messages WHERE session_id = ?", s.sessionID)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
        INSERT INTO messages (session_id, role, content, metadata, created_at)
        VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, msg := range messages {
		var content string
		for _, block := range msg.GetContent() {
			if textBlock, ok := block.(*types.TextBlock); ok {
				if content != "" {
					content += "\n"
				}
				content += textBlock.Text
			}
		}

		metadata := make(map[string]any)

		switch m := msg.(type) {
		case *types.UserMessage:
			if m.Metadata != nil {
				metadata = m.Metadata
			}
		case *types.AssistantMessage:
			if m.Metadata != nil {
				metadata = m.Metadata
			}
		case *types.SystemMessage:
			if m.Metadata != nil {
				metadata = m.Metadata
			}
		case *types.ToolMessage:
			metadata["tool_call_id"] = m.ToolCallID
			metadata["tool_name"] = m.ToolName
			metadata["is_error"] = m.IsError
			if m.Metadata != nil {
				for k, v := range m.Metadata {
					metadata[k] = v
				}
			}
		}

		var metadataJSON []byte
		if len(metadata) > 0 {
			metadataJSON, _ = json.Marshal(metadata)
		}

		_, err := stmt.Exec(
			s.sessionID,
			string(msg.GetRole()),
			content,
			metadataJSON,
			msg.GetTimestamp(),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteSession) Clear() error {
	_, err := s.db.Exec("DELETE FROM messages WHERE session_id = ?", s.sessionID)
	return err
}

func (s *SQLiteSession) Exists() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ?", s.sessionID).Scan(&count)
	return err == nil && count > 0
}

func (s *SQLiteSession) Close() error {
	return s.db.Close()
}

func (s *SQLiteSession) GetID() string {
	return s.sessionID
}