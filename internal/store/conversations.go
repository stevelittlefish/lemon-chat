package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Conversation struct {
	ID                     int64   `json:"id"`
	UserID                 int64   `json:"user_id"`
	Model                  *string `json:"model"`
	CharacterID            *int64  `json:"character_id"`
	Title                  *string `json:"title"`
	BackgroundAttachmentID *int64  `json:"background_attachment_id"`
	CreatedAt              string  `json:"created_at"`
	UpdatedAt              string  `json:"updated_at"`
}

func (s *Store) ListConversations(userID int64, limit, offset int) ([]Conversation, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, model, character_id, title, background_attachment_id, created_at, updated_at
		 FROM conversation WHERE user_id = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Model, &c.CharacterID, &c.Title, &c.BackgroundAttachmentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (s *Store) GetConversation(id, userID int64) (*Conversation, error) {
	c := &Conversation{}
	err := s.db.QueryRow(
		`SELECT id, user_id, model, character_id, title, background_attachment_id, created_at, updated_at
		 FROM conversation WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&c.ID, &c.UserID, &c.Model, &c.CharacterID, &c.Title, &c.BackgroundAttachmentID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// ConversationOwner returns the user_id that owns a conversation. Does not
// enforce ownership — used to route WebSocket events to the right user.
func (s *Store) ConversationOwner(id int64) (int64, error) {
	var userID int64
	err := s.db.QueryRow(`SELECT user_id FROM conversation WHERE id = ?`, id).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return userID, err
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

// DeleteConversation removes the conversation and all its messages and attachment
// records. It returns the relative attachment disk paths (relative to DataDir) so
// the caller can remove the files from disk after a successful return.
func (s *Store) DeleteConversation(id, userID int64) (attachmentPaths []string, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check ownership before touching anything.
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM conversation WHERE id = ? AND user_id = ?`, id, userID).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrNotFound
	}

	attachmentPaths, _, err = deleteConversationsTx(tx, []int64{id})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return attachmentPaths, nil
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

func (s *Store) SetConversationBackground(convID, attachmentID int64) error {
	_, err := s.db.Exec(`UPDATE conversation SET background_attachment_id = ? WHERE id = ?`, attachmentID, convID)
	return err
}

// deleteConversationsTx deletes the given conversations together with their
// messages and attachment rows. It returns the attachment disk paths (relative
// to DataDir) that are no longer referenced by any remaining attachment —
// forks share files by disk_path — so the caller can remove them after commit.
func deleteConversationsTx(tx *sql.Tx, ids []int64) (attachmentPaths []string, messages int64, err error) {
	if len(ids) == 0 {
		return nil, 0, nil
	}
	in := "?" + strings.Repeat(",?", len(ids)-1)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	attRows, err := tx.Query(`
		SELECT DISTINCT disk_path FROM attachment WHERE conversation_id IN (`+in+`)
		AND disk_path NOT IN (
			SELECT disk_path FROM attachment WHERE conversation_id NOT IN (`+in+`)
		)`, append(append([]any{}, args...), args...)...)
	if err != nil {
		return nil, 0, err
	}
	for attRows.Next() {
		var p string
		if err := attRows.Scan(&p); err != nil {
			attRows.Close()
			return nil, 0, err
		}
		attachmentPaths = append(attachmentPaths, p)
	}
	attRows.Close()
	if err := attRows.Err(); err != nil {
		return nil, 0, err
	}

	// Order matters for the FKs: attachment references message and
	// conversation; message references conversation.
	if _, err := tx.Exec(`DELETE FROM attachment WHERE conversation_id IN (`+in+`)`, args...); err != nil {
		return nil, 0, err
	}
	res, err := tx.Exec(`DELETE FROM message WHERE conversation_id IN (`+in+`)`, args...)
	if err != nil {
		return nil, 0, err
	}
	messages, _ = res.RowsAffected()
	if _, err := tx.Exec(`DELETE FROM conversation WHERE id IN (`+in+`)`, args...); err != nil {
		return nil, 0, err
	}
	return attachmentPaths, messages, nil
}

// DeleteStaleConversations removes conversations older than 30 minutes that
// never got a reply (fewer than two messages), including any attachments the
// user uploaded to them. It returns the attachment disk paths to remove.
func (s *Store) DeleteStaleConversations() (conversations, messages int64, attachmentPaths []string, err error) {
	threshold := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id FROM conversation
		WHERE created_at < ?
		AND (SELECT COUNT(*) FROM message WHERE conversation_id = conversation.id) < 2`, threshold)
	if err != nil {
		return 0, 0, nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, nil, err
	}
	if len(ids) == 0 {
		return 0, 0, nil, nil
	}

	attachmentPaths, messages, err = deleteConversationsTx(tx, ids)
	if err != nil {
		return 0, 0, nil, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, nil, err
	}
	return int64(len(ids)), messages, attachmentPaths, nil
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
		id               int64
		role, content    string
		name             *string
		charID           *int64
		promptTokens     *int64
		completionTokens *int64
		totalTimeMS      *int64
		toolCalls        *string
		toolCallID       *string
	}

	rows, err := s.db.Query(
		`SELECT id, role, content, name, character_id, prompt_tokens, completion_tokens, total_time_ms, tool_calls, tool_call_id
		 FROM message WHERE conversation_id = ? AND id <= ? ORDER BY created_at ASC, id ASC`,
		sourceConvID, untilMessageID,
	)
	if err != nil {
		return nil, err
	}
	var msgRows []msgRow
	for rows.Next() {
		var m msgRow
		if err := rows.Scan(&m.id, &m.role, &m.content, &m.name, &m.charID, &m.promptTokens, &m.completionTokens, &m.totalTimeMS, &m.toolCalls, &m.toolCallID); err != nil {
			rows.Close()
			return nil, err
		}
		msgRows = append(msgRows, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Tool attachments are tied to the fork range by tool_call_id; user
	// uploads by message_id.
	type attRow struct {
		id                                            int64
		toolCallID, source, title, filename, mimeType string
		diskPath, status, createdAt                   string
		messageID                                     *int64
		errMsg                                        *string
	}
	attRows, err := s.db.Query(
		`SELECT id, tool_call_id, message_id, source, title, filename, mime_type, disk_path, status, error, created_at
		 FROM attachment
		 WHERE conversation_id = ?
		 AND (
		     tool_call_id IN (
		         SELECT tool_call_id FROM message
		         WHERE conversation_id = ? AND id <= ? AND tool_call_id IS NOT NULL AND tool_call_id != ''
		     )
		     OR message_id IN (SELECT id FROM message WHERE conversation_id = ? AND id <= ?)
		 )
		 ORDER BY id`,
		sourceConvID, sourceConvID, untilMessageID, sourceConvID, untilMessageID,
	)
	if err != nil {
		return nil, err
	}
	var atts []attRow
	for attRows.Next() {
		var a attRow
		if err := attRows.Scan(&a.id, &a.toolCallID, &a.messageID, &a.source, &a.title, &a.filename, &a.mimeType, &a.diskPath, &a.status, &a.errMsg, &a.createdAt); err != nil {
			attRows.Close()
			return nil, err
		}
		atts = append(atts, a)
	}
	attRows.Close()
	if err := attRows.Err(); err != nil {
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

	newMsgID := make(map[int64]int64, len(msgRows))
	for _, m := range msgRows {
		res, err := tx.Exec(
			`INSERT INTO message (conversation_id, role, content, name, character_id, created_at, prompt_tokens, completion_tokens, total_time_ms, tool_calls, tool_call_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			newConv.ID, m.role, m.content, m.name, m.charID, t, m.promptTokens, m.completionTokens, m.totalTimeMS, m.toolCalls, m.toolCallID,
		)
		if err != nil {
			return nil, err
		}
		newMsgID[m.id], _ = res.LastInsertId()
	}

	newAttID := make(map[int64]int64, len(atts))
	for _, a := range atts {
		var msgID *int64
		if a.messageID != nil {
			if id, ok := newMsgID[*a.messageID]; ok {
				msgID = &id
			}
		}
		// A pending attachment is finalised by id by the generating goroutine,
		// which only knows the source row, so the copy would never complete.
		status, errMsg := a.status, a.errMsg
		if status == "pending" {
			status = "error"
			msg := "Still generating when the conversation was forked"
			errMsg = &msg
		}
		res, err := tx.Exec(
			`INSERT INTO attachment (tool_call_id, conversation_id, message_id, source, title, filename, mime_type, disk_path, status, error, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a.toolCallID, newConv.ID, msgID, a.source, a.title, a.filename, a.mimeType, a.diskPath, status, errMsg, a.createdAt,
		)
		if err != nil {
			return nil, err
		}
		newAttID[a.id], _ = res.LastInsertId()
	}

	if src.BackgroundAttachmentID != nil {
		if id, ok := newAttID[*src.BackgroundAttachmentID]; ok {
			if _, err := tx.Exec(`UPDATE conversation SET background_attachment_id = ? WHERE id = ?`, id, newConv.ID); err != nil {
				return nil, err
			}
			newConv.BackgroundAttachmentID = &id
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return newConv, nil
}

type ImportMessage struct {
	Role       string
	Content    string
	ToolCalls  *string // JSON array string, nil if none
	ToolCallID string  // empty means NULL
}

func (s *Store) ImportConversation(userID int64, model *string, characterID *int64, msgs []ImportMessage) (*Conversation, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	t := now()
	res, err := tx.Exec(
		`INSERT INTO conversation (user_id, model, character_id, title, created_at, updated_at) VALUES (?, ?, ?, NULL, ?, ?)`,
		userID, model, characterID, t, t,
	)
	if err != nil {
		return nil, err
	}
	convID, _ := res.LastInsertId()

	base := time.Now().UTC()
	n := len(msgs)
	for i, m := range msgs {
		ts := base.Add(-time.Duration(n-1-i) * time.Second).Format(time.RFC3339)
		var tcID *string
		if m.ToolCallID != "" {
			tcID = &m.ToolCallID
		}
		if _, err := tx.Exec(
			`INSERT INTO message (conversation_id, role, content, name, character_id, created_at,
			                      prompt_tokens, completion_tokens, total_time_ms, tool_calls, tool_call_id)
			 VALUES (?, ?, ?, NULL, NULL, ?, NULL, NULL, NULL, ?, ?)`,
			convID, m.Role, m.Content, ts, m.ToolCalls, tcID,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Conversation{
		ID: convID, UserID: userID, Model: model, CharacterID: characterID,
		CreatedAt: t, UpdatedAt: t,
	}, nil
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
