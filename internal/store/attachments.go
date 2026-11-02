package store

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Attachment struct {
	ID             int64  `json:"id"`
	ToolCallID     string `json:"tool_call_id"`
	ConversationID int64  `json:"conversation_id"`
	MessageID      *int64 `json:"message_id,omitempty"`
	Source         string `json:"source"` // "tool" (model output) | "upload" (user input)
	Title          string `json:"title"`
	Filename       string `json:"filename"`
	MimeType       string `json:"mime_type"`
	DiskPath       string `json:"disk_path"`
	Status         string `json:"status"` // "pending" | "ready" | "error"
	Error          string `json:"error,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func (s *Store) CreateAttachment(toolCallID string, convID int64, title, filename, mimeType, diskPath string) (*Attachment, error) {
	a := &Attachment{
		ToolCallID:     toolCallID,
		ConversationID: convID,
		Source:         "tool",
		Title:          title,
		Filename:       filename,
		MimeType:       mimeType,
		DiskPath:       diskPath,
		Status:         "ready",
		CreatedAt:      now(),
	}
	res, err := s.db.Exec(
		`INSERT INTO attachment (tool_call_id, conversation_id, title, filename, mime_type, disk_path, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'ready', ?)`,
		a.ToolCallID, a.ConversationID, a.Title, a.Filename, a.MimeType, a.DiskPath, a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.ID, err = res.LastInsertId()
	return a, err
}

func (s *Store) CreatePendingAttachment(toolCallID string, convID int64, title, filename, mimeType string) (*Attachment, error) {
	ts := now()
	res, err := s.db.Exec(
		`INSERT INTO attachment (tool_call_id, conversation_id, title, filename, mime_type, disk_path, status, created_at)
		 VALUES (?, ?, ?, ?, ?, '', 'pending', ?)`,
		toolCallID, convID, title, filename, mimeType, ts,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Attachment{
		ID: id, ToolCallID: toolCallID, ConversationID: convID, Source: "tool",
		Title: title, Filename: filename, MimeType: mimeType,
		Status: "pending", CreatedAt: ts,
	}, nil
}

// CreateUploadAttachment records a user-uploaded file (model input). It has no
// tool call and is not yet bound to a message — message_id is backfilled by
// LinkAttachmentsToMessage when the composing message is sent.
func (s *Store) CreateUploadAttachment(convID int64, filename, mimeType, diskPath string) (*Attachment, error) {
	a := &Attachment{
		ConversationID: convID,
		Source:         "upload",
		Title:          filename,
		Filename:       filename,
		MimeType:       mimeType,
		DiskPath:       diskPath,
		Status:         "ready",
		CreatedAt:      now(),
	}
	res, err := s.db.Exec(
		`INSERT INTO attachment (tool_call_id, conversation_id, source, title, filename, mime_type, disk_path, status, created_at)
		 VALUES ('', ?, 'upload', ?, ?, ?, ?, 'ready', ?)`,
		a.ConversationID, a.Title, a.Filename, a.MimeType, a.DiskPath, a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.ID, err = res.LastInsertId()
	return a, err
}

// LinkAttachmentsToMessage binds previously-uploaded attachments to the user
// message they were sent with. Only unlinked uploads in the given conversation
// are affected, so an id from another conversation is silently ignored.
func (s *Store) LinkAttachmentsToMessage(messageID, convID int64, attachmentIDs []int64) error {
	for _, aid := range attachmentIDs {
		if _, err := s.db.Exec(
			`UPDATE attachment SET message_id = ?
			 WHERE id = ? AND conversation_id = ? AND source = 'upload' AND message_id IS NULL`,
			messageID, aid, convID,
		); err != nil {
			return err
		}
	}
	return nil
}

// ListAttachmentsByMessage returns the attachments bound to a single message,
// used to reconstruct multimodal history when re-sending a conversation.
func (s *Store) ListAttachmentsByMessage(messageID int64) ([]Attachment, error) {
	rows, err := s.db.Query(
		`SELECT id, tool_call_id, conversation_id, message_id, source, title, filename, mime_type, disk_path, status, COALESCE(error, ''), created_at
		 FROM attachment WHERE message_id = ? ORDER BY id`,
		messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.ToolCallID, &a.ConversationID, &a.MessageID, &a.Source, &a.Title, &a.Filename, &a.MimeType, &a.DiskPath, &a.Status, &a.Error, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) FinaliseAttachment(id int64, diskPath string) error {
	_, err := s.db.Exec(
		`UPDATE attachment SET status = 'ready', disk_path = ? WHERE id = ?`,
		diskPath, id,
	)
	return err
}

func (s *Store) FailAttachment(id int64, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE attachment SET status = 'error', error = ? WHERE id = ?`,
		errMsg, id,
	)
	return err
}

func (s *Store) ClearPendingAttachments() (int, error) {
	res, err := s.db.Exec(
		`UPDATE attachment SET status = 'error', error = 'Server restarted while image was generating'
		 WHERE status = 'pending'`,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *Store) GetAttachment(id int64) (*Attachment, error) {
	a := &Attachment{}
	err := s.db.QueryRow(
		`SELECT id, tool_call_id, conversation_id, message_id, source, title, filename, mime_type, disk_path, status, COALESCE(error, ''), created_at
		 FROM attachment WHERE id = ?`, id,
	).Scan(&a.ID, &a.ToolCallID, &a.ConversationID, &a.MessageID, &a.Source, &a.Title, &a.Filename, &a.MimeType, &a.DiskPath, &a.Status, &a.Error, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// GetAttachmentForUser fetches an attachment only if its conversation is owned
// by userID. Returns ErrNotFound if the attachment doesn't exist or belongs
// to a different user, so the caller cannot distinguish the two cases.
func (s *Store) GetAttachmentForUser(id, userID int64) (*Attachment, error) {
	a := &Attachment{}
	err := s.db.QueryRow(`
		SELECT a.id, a.tool_call_id, a.conversation_id, a.message_id, a.source, a.title, a.filename, a.mime_type, a.disk_path, a.status, COALESCE(a.error, ''), a.created_at
		FROM attachment a
		JOIN conversation c ON c.id = a.conversation_id
		WHERE a.id = ? AND c.user_id = ?
	`, id, userID).Scan(&a.ID, &a.ToolCallID, &a.ConversationID, &a.MessageID, &a.Source, &a.Title, &a.Filename, &a.MimeType, &a.DiskPath, &a.Status, &a.Error, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) ListAttachmentsByConversation(convID int64) ([]Attachment, error) {
	rows, err := s.db.Query(
		`SELECT id, tool_call_id, conversation_id, message_id, source, title, filename, mime_type, disk_path, status, COALESCE(error, ''), created_at
		 FROM attachment WHERE conversation_id = ? ORDER BY id`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.ToolCallID, &a.ConversationID, &a.MessageID, &a.Source, &a.Title, &a.Filename, &a.MimeType, &a.DiskPath, &a.Status, &a.Error, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// RemoveAttachmentFiles deletes the per-attachment directories for the given
// disk paths (relative to dataDir, as returned by the Delete* functions).
// Every attachment lives in its own attachments/<random-id>/ directory; any
// path that doesn't have that shape is skipped. In particular a pending or
// failed attachment has an empty disk_path and filepath.Dir("") is ".", which would
// otherwise resolve to dataDir itself.
func RemoveAttachmentFiles(dataDir string, paths []string) {
	for _, p := range paths {
		dir := filepath.Dir(filepath.Clean(p))
		parent, name := filepath.Split(dir)
		if filepath.Clean(parent) != "attachments" || name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
			if p != "" {
				log.Printf("store: RemoveAttachmentFiles — refusing to remove unexpected path %q", p)
			}
			continue
		}
		if err := os.RemoveAll(filepath.Join(dataDir, dir)); err != nil {
			log.Printf("store: RemoveAttachmentFiles — removing %q: %v", dir, err)
		}
	}
}

// DeleteUnlinkedUploads removes user uploads that were never attached to a
// sent message (message_id IS NULL) and are older than olderThan — the user
// pasted an image and then abandoned the draft. It returns the disk paths to
// remove that no other attachment row still references.
func (s *Store) DeleteUnlinkedUploads(olderThan time.Duration) (deleted int64, attachmentPaths []string, err error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	const stale = `source = 'upload' AND message_id IS NULL AND created_at < ?`
	rows, err := tx.Query(`
		SELECT DISTINCT disk_path FROM attachment WHERE `+stale+`
		AND disk_path NOT IN (SELECT disk_path FROM attachment WHERE NOT (`+stale+`))`, cutoff, cutoff)
	if err != nil {
		return 0, nil, err
	}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return 0, nil, err
		}
		attachmentPaths = append(attachmentPaths, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	// A conversation may have had one of these set as its background.
	if _, err := tx.Exec(`UPDATE conversation SET background_attachment_id = NULL
		WHERE background_attachment_id IN (SELECT id FROM attachment WHERE `+stale+`)`, cutoff); err != nil {
		return 0, nil, err
	}
	res, err := tx.Exec(`DELETE FROM attachment WHERE `+stale, cutoff)
	if err != nil {
		return 0, nil, err
	}
	deleted, _ = res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}
	return deleted, attachmentPaths, nil
}
