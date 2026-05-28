package store

import (
	"database/sql"
	"errors"
	"time"
)

type Conversation struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"user_id"`
	Model       *string `json:"model"`
	CharacterID *int64  `json:"character_id"`
	Title       *string `json:"title"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func (s *Store) ListConversations(userID int64) ([]Conversation, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, model, character_id, title, created_at, updated_at
		 FROM conversation WHERE user_id = ? ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Model, &c.CharacterID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (s *Store) GetConversation(id, userID int64) (*Conversation, error) {
	c := &Conversation{}
	err := s.db.QueryRow(
		`SELECT id, user_id, model, character_id, title, created_at, updated_at
		 FROM conversation WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&c.ID, &c.UserID, &c.Model, &c.CharacterID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (s *Store) CreateConversation(userID int64, title *string, model *string, characterID *int64) (*Conversation, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO conversation (user_id, model, character_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, model, characterID, title, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Conversation{ID: id, UserID: userID, Model: model, CharacterID: characterID, Title: title, CreatedAt: t, UpdatedAt: t}, nil
}

func (s *Store) DeleteConversation(id, userID int64) error {
	_, err := s.db.Exec(`DELETE FROM conversation WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) TouchConversation(id int64) error {
	_, err := s.db.Exec(`UPDATE conversation SET updated_at = ? WHERE id = ?`, now(), id)
	return err
}

func (s *Store) UpdateConversationTitle(id int64, title string) error {
	// Deliberately does not touch updated_at so sidebar sort order is unaffected.
	_, err := s.db.Exec(`UPDATE conversation SET title = ? WHERE id = ?`, title, id)
	return err
}

func (s *Store) ListUntitledEligible() ([]int64, error) {
	oneMinAgo := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	fiveMinAgo := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT id FROM conversation
		WHERE title IS NULL AND (
			(created_at < ? AND (SELECT COUNT(*) FROM message WHERE conversation_id = conversation.id) >= 6)
			OR
			(created_at < ? AND (SELECT COUNT(*) FROM message WHERE conversation_id = conversation.id) >= 2)
		)
	`, oneMinAgo, fiveMinAgo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
