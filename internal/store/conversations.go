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
	res, err := s.db.Exec(`DELETE FROM conversation WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) TouchConversation(id int64) error {
	_, err := s.db.Exec(`UPDATE conversation SET updated_at = ? WHERE id = ?`, now(), id)
	return err
}

// UpdateConversationAfterMessage updates the conversation's model/character and touches updated_at.
// Exactly one of model or characterID must be non-nil (enforced by DB constraint).
func (s *Store) UpdateConversationAfterMessage(id int64, model *string, characterID *int64) error {
	_, err := s.db.Exec(
		`UPDATE conversation SET model = ?, character_id = ?, updated_at = ? WHERE id = ?`,
		model, characterID, now(), id,
	)
	return err
}

func (s *Store) UpdateConversationTitle(id int64, title string) error {
	// Deliberately does not touch updated_at so sidebar sort order is unaffected.
	_, err := s.db.Exec(`UPDATE conversation SET title = ? WHERE id = ?`, title, id)
	return err
}

func (s *Store) DeleteStaleConversations() (conversations, messages int64, err error) {
	threshold := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)
	staleSubquery := `
		SELECT id FROM conversation
		WHERE created_at < ?
		AND (SELECT COUNT(*) FROM message WHERE conversation_id = conversation.id) < 2`
	res, err := s.db.Exec(`DELETE FROM message WHERE conversation_id IN (`+staleSubquery+`)`, threshold)
	if err != nil {
		return 0, 0, err
	}
	messages, _ = res.RowsAffected()
	res, err = s.db.Exec(`DELETE FROM conversation WHERE id IN (`+staleSubquery+`)`, threshold)
	if err != nil {
		return 0, 0, err
	}
	conversations, _ = res.RowsAffected()
	return conversations, messages, nil
}

func (s *Store) ForkConversation(sourceConvID, userID int64, untilMessageID int64) (*Conversation, error) {
	src, err := s.GetConversation(sourceConvID, userID)
	if err != nil {
		return nil, err
	}

	title := "copy: untitled"
	if src.Title != nil {
		title = "copy: " + *src.Title
	}

	// Verify untilMessageID belongs to this conversation.
	var check int
	err = s.db.QueryRow(
		`SELECT COUNT(*) FROM message WHERE id = ? AND conversation_id = ?`,
		untilMessageID, sourceConvID,
	).Scan(&check)
	if err != nil || check == 0 {
		return nil, ErrNotFound
	}

	type msgRow struct {
		role, content    string
		name             *string
		charID           *int64
		promptTokens     *int64
		completionTokens *int64
		totalTimeMS      *int64
	}

	rows, err := s.db.Query(
		`SELECT role, content, name, character_id, prompt_tokens, completion_tokens, total_time_ms
		 FROM message WHERE conversation_id = ? AND id <= ? ORDER BY created_at ASC`,
		sourceConvID, untilMessageID,
	)
	if err != nil {
		return nil, err
	}
	var msgRows []msgRow
	for rows.Next() {
		var m msgRow
		if err := rows.Scan(&m.role, &m.content, &m.name, &m.charID, &m.promptTokens, &m.completionTokens, &m.totalTimeMS); err != nil {
			rows.Close()
			return nil, err
		}
		msgRows = append(msgRows, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	t := now()
	res, err := tx.Exec(
		`INSERT INTO conversation (user_id, model, character_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, src.Model, src.CharacterID, &title, t, t,
	)
	if err != nil {
		return nil, err
	}
	newID, _ := res.LastInsertId()
	newConv := &Conversation{ID: newID, UserID: userID, Model: src.Model, CharacterID: src.CharacterID, Title: &title, CreatedAt: t, UpdatedAt: t}

	for _, m := range msgRows {
		if _, err := tx.Exec(
			`INSERT INTO message (conversation_id, role, content, name, character_id, created_at, prompt_tokens, completion_tokens, total_time_ms)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			newConv.ID, m.role, m.content, m.name, m.charID, t, m.promptTokens, m.completionTokens, m.totalTimeMS,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return newConv, nil
}

func (s *Store) GetConversationTitlePrompt(convID int64) (string, error) {
	var prompt string
	err := s.db.QueryRow(
		`SELECT COALESCE(ch.title_prompt, '')
		 FROM conversation conv
		 JOIN character ch ON ch.id = conv.character_id
		 WHERE conv.id = ?`, convID,
	).Scan(&prompt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return prompt, err
}

func (s *Store) ListUntitledEligible() ([]int64, error) {
	fiveMinAgo := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT id FROM conversation
		WHERE title IS NULL
		AND created_at < ?
		AND (SELECT COUNT(*) FROM message WHERE conversation_id = conversation.id) >= 2
	`, fiveMinAgo)
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
