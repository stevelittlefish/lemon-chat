package store

import (
	"database/sql"
	"errors"
)

type CharacterHiddenMessage struct {
	ID          int64  `json:"id"`
	CharacterID int64  `json:"character_id"`
	Role        string `json:"role"`
	Content     string `json:"content"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
}

type Character struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Model          string  `json:"model"`
	SystemPrompt   *string `json:"system_prompt"`
	FirstMessage   *string `json:"first_message"`
	TitlePrompt    *string `json:"title_prompt"`
	CreatedBy      int64   `json:"created_by"`
	Visibility     string  `json:"visibility"`
	AutoTitle      bool    `json:"auto_title"`
	AvatarFilename *string `json:"-"`
	HasAvatar      bool    `json:"has_avatar"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// ListCharacters returns all characters visible to the given user.
// Private characters created by other users are excluded unless the user is an admin.
func (s *Store) ListCharacters(userID int64, isAdmin bool) ([]Character, error) {
	rows, err := s.db.Query(
		`SELECT id, name, model, system_prompt, first_message, title_prompt, created_by, visibility, auto_title, avatar_filename, created_at, updated_at
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
		var autoTitle int
		if err := rows.Scan(&c.ID, &c.Name, &c.Model, &c.SystemPrompt, &c.FirstMessage, &c.TitlePrompt, &c.CreatedBy, &c.Visibility, &autoTitle, &c.AvatarFilename, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.AutoTitle = autoTitle != 0
		c.HasAvatar = c.AvatarFilename != nil
		chars = append(chars, c)
	}
	return chars, rows.Err()
}

func (s *Store) GetCharacter(id int64) (*Character, error) {
	c := &Character{}
	var autoTitle int
	err := s.db.QueryRow(
		`SELECT id, name, model, system_prompt, first_message, title_prompt, created_by, visibility, auto_title, avatar_filename, created_at, updated_at
		 FROM character WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Model, &c.SystemPrompt, &c.FirstMessage, &c.TitlePrompt, &c.CreatedBy, &c.Visibility, &autoTitle, &c.AvatarFilename, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.AutoTitle = autoTitle != 0
	c.HasAvatar = c.AvatarFilename != nil
	return c, nil
}

func (s *Store) CreateCharacter(name, model string, systemPrompt, firstMessage, titlePrompt *string, createdBy int64, visibility string, autoTitle bool) (*Character, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO character (name, model, system_prompt, first_message, title_prompt, created_by, visibility, auto_title, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name, model, systemPrompt, firstMessage, titlePrompt, createdBy, visibility, boolToInt(autoTitle), t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Character{
		ID: id, Name: name, Model: model, SystemPrompt: systemPrompt, FirstMessage: firstMessage,
		TitlePrompt: titlePrompt, CreatedBy: createdBy, Visibility: visibility, AutoTitle: autoTitle,
		CreatedAt: t, UpdatedAt: t,
	}, nil
}

func (s *Store) UpdateCharacter(id int64, name, model string, systemPrompt, firstMessage, titlePrompt *string, visibility string, autoTitle bool) error {
	_, err := s.db.Exec(
		`UPDATE character SET name = ?, model = ?, system_prompt = ?, first_message = ?, title_prompt = ?, visibility = ?, auto_title = ?, updated_at = ?
		 WHERE id = ?`,
		name, model, systemPrompt, firstMessage, titlePrompt, visibility, boolToInt(autoTitle), now(), id,
	)
	return err
}

func (s *Store) DeleteCharacter(id int64) error {
	_, err := s.db.Exec(`DELETE FROM character WHERE id = ?`, id)
	return err
}

func (s *Store) SetCharacterAvatar(id int64, filename string) error {
	_, err := s.db.Exec(`UPDATE character SET avatar_filename = ? WHERE id = ?`, filename, id)
	return err
}

func (s *Store) ClearCharacterAvatar(id int64) error {
	_, err := s.db.Exec(`UPDATE character SET avatar_filename = NULL WHERE id = ?`, id)
	return err
}

func (s *Store) ListCharacterHiddenMessages(characterID int64) ([]CharacterHiddenMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, character_id, role, content, sort_order, created_at
		 FROM character_hidden_message
		 WHERE character_id = ?
		 ORDER BY sort_order ASC`,
		characterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []CharacterHiddenMessage
	for rows.Next() {
		var m CharacterHiddenMessage
		if err := rows.Scan(&m.ID, &m.CharacterID, &m.Role, &m.Content, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *Store) ReplaceCharacterHiddenMessages(characterID int64, msgs []CharacterHiddenMessage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM character_hidden_message WHERE character_id = ?`, characterID); err != nil {
		return err
	}
	t := now()
	for i, m := range msgs {
		if _, err := tx.Exec(
			`INSERT INTO character_hidden_message (character_id, role, content, sort_order, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			characterID, m.Role, m.Content, i, t,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
