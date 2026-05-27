package store

type Message struct {
	ID             int64
	ConversationID int64
	Role           string
	Content        string
	CreatedAt      string
}

func (s *Store) ListMessages(conversationID int64) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *Store) CreateMessage(conversationID int64, role, content string) (*Message, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO messages (conversation_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		conversationID, role, content, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Message{ID: id, ConversationID: conversationID, Role: role, Content: content, CreatedAt: t}, nil
}
