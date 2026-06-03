package store

type Message struct {
	ID               int64   `json:"id"`
	ConversationID   int64   `json:"conversation_id"`
	Role             string  `json:"role"`
	Content          string  `json:"content"`
	Name             *string `json:"name"`
	CharacterID      *int64  `json:"character_id,omitempty"`
	CreatedAt        string  `json:"created_at"`
	PromptTokens     *int64  `json:"prompt_tokens,omitempty"`
	CompletionTokens *int64  `json:"completion_tokens,omitempty"`
	TotalTimeMS      *int64  `json:"total_time_ms,omitempty"`
}

type MessageStats struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTimeMS      int64
}

func (s *Store) DeleteOrphanedMessages() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM message WHERE conversation_id NOT IN (SELECT id FROM conversation)`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *Store) ListMessages(conversationID int64) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, name, character_id, created_at,
		        prompt_tokens, completion_tokens, total_time_ms
		 FROM message WHERE conversation_id = ? ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.Name, &m.CharacterID, &m.CreatedAt,
			&m.PromptTokens, &m.CompletionTokens, &m.TotalTimeMS); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *Store) CreateMessage(conversationID int64, role, content string, characterID *int64, assistantName *string, stats *MessageStats) (*Message, error) {
	t := now()
	var promptTokens, completionTokens, totalTimeMS *int64
	if stats != nil {
		promptTokens = &stats.PromptTokens
		completionTokens = &stats.CompletionTokens
		totalTimeMS = &stats.TotalTimeMS
	}
	res, err := s.db.Exec(
		`INSERT INTO message (conversation_id, role, content, character_id, name, created_at,
		                      prompt_tokens, completion_tokens, total_time_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		conversationID, role, content, characterID, assistantName, t,
		promptTokens, completionTokens, totalTimeMS,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Message{
		ID: id, ConversationID: conversationID, Role: role, Content: content,
		CharacterID: characterID, Name: assistantName, CreatedAt: t,
		PromptTokens: promptTokens, CompletionTokens: completionTokens, TotalTimeMS: totalTimeMS,
	}, nil
}
