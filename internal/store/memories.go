package store

import (
	"database/sql"
	"errors"
)

// Memory is one fact the model saved about a user, kept across conversations.
// ConversationID records where it was saved; it becomes nil if that
// conversation is deleted.
type Memory struct {
	ID             int64  `json:"id"`
	UserID         int64  `json:"user_id"`
	Text           string `json:"text"`
	ConversationID *int64 `json:"conversation_id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	// ConversationTitle is filled by ListMemories for display.
	ConversationTitle *string `json:"conversation_title,omitempty"`
}

// ListMemories returns a user's memories, oldest first.
func (s *Store) ListMemories(userID int64) ([]Memory, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.user_id, m.text, m.conversation_id, m.created_at, m.updated_at, c.title
		 FROM memory m
		 LEFT JOIN conversation c ON c.id = m.conversation_id
		 WHERE m.user_id = ?
		 ORDER BY m.id`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memories := []Memory{}
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.UserID, &m.Text, &m.ConversationID, &m.CreatedAt, &m.UpdatedAt, &m.ConversationTitle); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

// GetMemory returns a memory by id, whoever owns it.
func (s *Store) GetMemory(id int64) (*Memory, error) {
	m := &Memory{}
	err := s.db.QueryRow(
		`SELECT id, user_id, text, conversation_id, created_at, updated_at FROM memory WHERE id = ?`, id,
	).Scan(&m.ID, &m.UserID, &m.Text, &m.ConversationID, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

// CreateMemory saves a memory for a user. conversationID is nil for memories
// added by hand on the settings page.
func (s *Store) CreateMemory(userID int64, text string, conversationID *int64) (*Memory, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO memory (user_id, text, conversation_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		userID, text, conversationID, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Memory{ID: id, UserID: userID, Text: text, ConversationID: conversationID, CreatedAt: t, UpdatedAt: t}, nil
}

// UpdateMemory replaces a memory's text. It returns ErrNotFound unless the
// memory exists and belongs to userID.
func (s *Store) UpdateMemory(id, userID int64, text string) error {
	res, err := s.db.Exec(`UPDATE memory SET text = ?, updated_at = ? WHERE id = ? AND user_id = ?`, text, now(), id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteMemory removes a memory. It returns ErrNotFound unless the memory
// exists and belongs to userID.
func (s *Store) DeleteMemory(id, userID int64) error {
	res, err := s.db.Exec(`DELETE FROM memory WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
