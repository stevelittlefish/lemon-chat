package store

import (
	"database/sql"
	"errors"
)

type Conversation struct {
	ID        int64
	UserID    int64
	PersonaID *int64
	Title     string
	CreatedAt string
	UpdatedAt string
}

func (s *Store) ListConversations(userID int64) ([]Conversation, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, persona_id, title, created_at, updated_at
		 FROM conversations WHERE user_id = ? ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.PersonaID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (s *Store) GetConversation(id, userID int64) (*Conversation, error) {
	c := &Conversation{}
	err := s.db.QueryRow(
		`SELECT id, user_id, persona_id, title, created_at, updated_at
		 FROM conversations WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&c.ID, &c.UserID, &c.PersonaID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (s *Store) CreateConversation(userID int64, title string, personaID *int64) (*Conversation, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO conversations (user_id, persona_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		userID, personaID, title, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Conversation{ID: id, UserID: userID, PersonaID: personaID, Title: title, CreatedAt: t, UpdatedAt: t}, nil
}

func (s *Store) DeleteConversation(id, userID int64) error {
	_, err := s.db.Exec(`DELETE FROM conversations WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) TouchConversation(id int64) error {
	_, err := s.db.Exec(`UPDATE conversations SET updated_at = ? WHERE id = ?`, now(), id)
	return err
}
