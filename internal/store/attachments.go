package store

type Attachment struct {
	ID             int64  `json:"id"`
	ToolCallID     string `json:"tool_call_id"`
	ConversationID int64  `json:"conversation_id"`
	Title          string `json:"title"`
	Filename       string `json:"filename"`
	MimeType       string `json:"mime_type"`
	DiskPath       string `json:"disk_path"`
	CreatedAt      string `json:"created_at"`
}

func (s *Store) CreateAttachment(toolCallID string, convID int64, title, filename, mimeType, diskPath string) (*Attachment, error) {
	a := &Attachment{
		ToolCallID:     toolCallID,
		ConversationID: convID,
		Title:          title,
		Filename:       filename,
		MimeType:       mimeType,
		DiskPath:       diskPath,
		CreatedAt:      now(),
	}
	res, err := s.db.Exec(
		`INSERT INTO attachment (tool_call_id, conversation_id, title, filename, mime_type, disk_path, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ToolCallID, a.ConversationID, a.Title, a.Filename, a.MimeType, a.DiskPath, a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.ID, err = res.LastInsertId()
	return a, err
}

func (s *Store) GetAttachment(id int64) (*Attachment, error) {
	a := &Attachment{}
	err := s.db.QueryRow(
		`SELECT id, tool_call_id, conversation_id, title, filename, mime_type, disk_path, created_at
		 FROM attachment WHERE id = ?`, id,
	).Scan(&a.ID, &a.ToolCallID, &a.ConversationID, &a.Title, &a.Filename, &a.MimeType, &a.DiskPath, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) ListAttachmentsByConversation(convID int64) ([]Attachment, error) {
	rows, err := s.db.Query(
		`SELECT id, tool_call_id, conversation_id, title, filename, mime_type, disk_path, created_at
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
		if err := rows.Scan(&a.ID, &a.ToolCallID, &a.ConversationID, &a.Title, &a.Filename, &a.MimeType, &a.DiskPath, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
