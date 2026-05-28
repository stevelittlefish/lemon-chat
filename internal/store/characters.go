package store

import (
	"database/sql"
	"errors"
)

type Character struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Model        string  `json:"model"`
	SystemPrompt *string `json:"system_prompt"`
	FirstMessage *string `json:"first_message"`
	CreatedBy    int64   `json:"created_by"`
	Visibility   string  `json:"visibility"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// ListCharacters returns all characters visible to the given user.
// Private characters created by other users are excluded unless the user is an admin.
func (s *Store) ListCharacters(userID int64, isAdmin bool) ([]Character, error) {
	rows, err := s.db.Query(
		`SELECT id, name, model, system_prompt, first_message, created_by, visibility, created_at, updated_at
		 FROM character
		 WHERE visibility != 'private' OR created_by = ? OR ?
		 ORDER BY name`,
		userID, boolToInt(isAdmin),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chars []Character
	for rows.Next() {
		var c Character
		if err := rows.Scan(&c.ID, &c.Name, &c.Model, &c.SystemPrompt, &c.FirstMessage, &c.CreatedBy, &c.Visibility, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		chars = append(chars, c)
	}
	return chars, rows.Err()
}

func (s *Store) GetCharacter(id int64) (*Character, error) {
	c := &Character{}
	err := s.db.QueryRow(
		`SELECT id, name, model, system_prompt, first_message, created_by, visibility, created_at, updated_at
		 FROM character WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Model, &c.SystemPrompt, &c.FirstMessage, &c.CreatedBy, &c.Visibility, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) CreateCharacter(name, model string, systemPrompt, firstMessage *string, createdBy int64, visibility string) (*Character, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO character (name, model, system_prompt, first_message, created_by, visibility, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		name, model, systemPrompt, firstMessage, createdBy, visibility, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Character{
		ID: id, Name: name, Model: model, SystemPrompt: systemPrompt, FirstMessage: firstMessage,
		CreatedBy: createdBy, Visibility: visibility, CreatedAt: t, UpdatedAt: t,
	}, nil
}

func (s *Store) UpdateCharacter(id int64, name, model string, systemPrompt, firstMessage *string, visibility string) error {
	_, err := s.db.Exec(
		`UPDATE character SET name = ?, model = ?, system_prompt = ?, first_message = ?, visibility = ?, updated_at = ?
		 WHERE id = ?`,
		name, model, systemPrompt, firstMessage, visibility, now(), id,
	)
	return err
}

func (s *Store) DeleteCharacter(id int64) error {
	_, err := s.db.Exec(`DELETE FROM character WHERE id = ?`, id)
	return err
}
