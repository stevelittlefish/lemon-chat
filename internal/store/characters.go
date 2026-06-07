package store

import (
	"database/sql"
	"encoding/json"
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
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Model          string   `json:"model"`
	SystemPrompt   *string  `json:"system_prompt"`
	FirstMessage   *string  `json:"first_message"`
	TitlePrompt    *string  `json:"title_prompt"`
	CreatedBy      int64    `json:"created_by"`
	Visibility     string   `json:"visibility"`
	AutoTitle      bool     `json:"auto_title"`
	IsDefault      bool     `json:"is_default"`
	Tools          []string `json:"tools"`
	AvatarFilename *string  `json:"-"`
	HasAvatar      bool     `json:"has_avatar"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// ListCharacters returns all characters visible to the given user.
// Private characters created by other users are excluded unless the user is an admin.
func (s *Store) ListCharacters(userID int64, isAdmin bool) ([]Character, error) {
	rows, err := s.db.Query(
		`SELECT id, name, model, system_prompt, first_message, title_prompt, created_by, visibility, auto_title, is_default, tools, avatar_filename, created_at, updated_at
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
		var autoTitle, isDefault int
		var toolsJSON sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Model, &c.SystemPrompt, &c.FirstMessage, &c.TitlePrompt, &c.CreatedBy, &c.Visibility, &autoTitle, &isDefault, &toolsJSON, &c.AvatarFilename, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.AutoTitle = autoTitle != 0
		c.IsDefault = isDefault != 0
		c.HasAvatar = c.AvatarFilename != nil
		if toolsJSON.Valid && toolsJSON.String != "" {
			json.Unmarshal([]byte(toolsJSON.String), &c.Tools) //nolint:errcheck
		}
		if c.Tools == nil {
			c.Tools = []string{}
		}
		chars = append(chars, c)
	}
	return chars, rows.Err()
}

func (s *Store) GetCharacter(id int64) (*Character, error) {
	c := &Character{}
	var autoTitle, isDefault int
	var toolsJSON sql.NullString
	err := s.db.QueryRow(
		`SELECT id, name, model, system_prompt, first_message, title_prompt, created_by, visibility, auto_title, is_default, tools, avatar_filename, created_at, updated_at
		 FROM character WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Model, &c.SystemPrompt, &c.FirstMessage, &c.TitlePrompt, &c.CreatedBy, &c.Visibility, &autoTitle, &isDefault, &toolsJSON, &c.AvatarFilename, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.AutoTitle = autoTitle != 0
	c.IsDefault = isDefault != 0
	c.HasAvatar = c.AvatarFilename != nil
	if toolsJSON.Valid && toolsJSON.String != "" {
		json.Unmarshal([]byte(toolsJSON.String), &c.Tools) //nolint:errcheck
	}
	if c.Tools == nil {
		c.Tools = []string{}
	}
	return c, nil
}

func (s *Store) CreateCharacter(name, model string, systemPrompt, firstMessage, titlePrompt *string, createdBy int64, visibility string, autoTitle bool, tools []string) (*Character, error) {
	t := now()
	var toolsJSON *string
	if len(tools) > 0 {
		b, _ := json.Marshal(tools)
		s := string(b)
		toolsJSON = &s
	}
	res, err := s.db.Exec(
		`INSERT INTO character (name, model, system_prompt, first_message, title_prompt, created_by, visibility, auto_title, tools, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name, model, systemPrompt, firstMessage, titlePrompt, createdBy, visibility, boolToInt(autoTitle), toolsJSON, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if tools == nil {
		tools = []string{}
	}
	return &Character{
		ID: id, Name: name, Model: model, SystemPrompt: systemPrompt, FirstMessage: firstMessage,
		TitlePrompt: titlePrompt, CreatedBy: createdBy, Visibility: visibility, AutoTitle: autoTitle,
		Tools: tools, CreatedAt: t, UpdatedAt: t,
	}, nil
}

func (s *Store) UpdateCharacter(id int64, name, model string, systemPrompt, firstMessage, titlePrompt *string, visibility string, autoTitle bool, tools []string) error {
	var toolsJSON *string
	if len(tools) > 0 {
		b, _ := json.Marshal(tools)
		s := string(b)
		toolsJSON = &s
	}
	_, err := s.db.Exec(
		`UPDATE character SET name = ?, model = ?, system_prompt = ?, first_message = ?, title_prompt = ?, visibility = ?, auto_title = ?, tools = ?, updated_at = ?
		 WHERE id = ?`,
		name, model, systemPrompt, firstMessage, titlePrompt, visibility, boolToInt(autoTitle), toolsJSON, now(), id,
	)
	return err
}

func (s *Store) DeleteCharacter(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Conversations that used this character revert to the character's model so
	// they still satisfy the CHECK (model IS NOT NULL) != (character_id IS NOT NULL).
	if _, err := tx.Exec(`
		UPDATE conversation
		SET model        = (SELECT model FROM character WHERE id = ?),
		    character_id = NULL
		WHERE character_id = ?`, id, id); err != nil {
		return err
	}

	// Null out character attribution on individual messages (nullable FK).
	if _, err := tx.Exec(`UPDATE message SET character_id = NULL WHERE character_id = ?`, id); err != nil {
		return err
	}

	// character_hidden_message has ON DELETE CASCADE, so it cleans itself up.
	if _, err := tx.Exec(`DELETE FROM character WHERE id = ?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) SetCharacterAvatar(id int64, filename string) error {
	_, err := s.db.Exec(`UPDATE character SET avatar_filename = ? WHERE id = ?`, filename, id)
	return err
}

func (s *Store) ClearCharacterAvatar(id int64) error {
	_, err := s.db.Exec(`UPDATE character SET avatar_filename = NULL WHERE id = ?`, id)
	return err
}

func (s *Store) SetDefaultCharacter(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE character SET is_default = 0 WHERE is_default = 1`); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE character SET is_default = 1 WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ClearDefaultCharacter() error {
	_, err := s.db.Exec(`UPDATE character SET is_default = 0 WHERE is_default = 1`)
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
