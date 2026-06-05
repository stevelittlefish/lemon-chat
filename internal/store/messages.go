package store

import "log"

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
	// Tool call support: ToolCalls is set on assistant messages that triggered tool calls (raw JSON array).
	// ToolCallID is set on tool-result messages (role="tool").
	ToolCalls  *string `json:"tool_calls,omitempty"`
	ToolCallID string  `json:"tool_call_id,omitempty"`
}

type MessageStats struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTimeMS      int64
}

func (s *Store) DeleteOrphanedMessages() (int64, error) {
	log.Println("store: DeleteOrphanedMessages — scanning for orphaned messages")

	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, substr(content, 1, 80)
		 FROM message
		 WHERE conversation_id NOT IN (SELECT id FROM conversation)
		 ORDER BY conversation_id, id`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type orphan struct {
		id, convID    int64
		role, snippet string
	}
	var orphans []orphan
	for rows.Next() {
		var o orphan
		if err := rows.Scan(&o.id, &o.convID, &o.role, &o.snippet); err != nil {
			return 0, err
		}
		orphans = append(orphans, o)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(orphans) == 0 {
		log.Println("store: DeleteOrphanedMessages — no orphaned messages found, nothing to do")
		return 0, nil
	}

	log.Printf("store: DeleteOrphanedMessages — found %d orphaned message(s):", len(orphans))
	for _, o := range orphans {
		log.Printf("store:   message id=%d conversation_id=%d role=%s content=%q", o.id, o.convID, o.role, o.snippet)
	}

	res, err := s.db.Exec(`DELETE FROM message WHERE conversation_id NOT IN (SELECT id FROM conversation)`)
	if err != nil {
		log.Printf("store: DeleteOrphanedMessages — delete failed: %v", err)
		return 0, err
	}
	n, _ := res.RowsAffected()
	log.Printf("store: DeleteOrphanedMessages — deleted %d message(s) successfully", n)
	return n, nil
}

func (s *Store) ListMessages(conversationID int64) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, name, character_id, created_at,
		        prompt_tokens, completion_tokens, total_time_ms, tool_calls, tool_call_id
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
		var toolCallID *string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.Name, &m.CharacterID, &m.CreatedAt,
			&m.PromptTokens, &m.CompletionTokens, &m.TotalTimeMS, &m.ToolCalls, &toolCallID); err != nil {
			return nil, err
		}
		if toolCallID != nil {
			m.ToolCallID = *toolCallID
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *Store) CreateMessage(conversationID int64, role, content string, characterID *int64, assistantName *string, stats *MessageStats, toolCalls *string, toolCallID string) (*Message, error) {
	t := now()
	var promptTokens, completionTokens, totalTimeMS *int64
	if stats != nil {
		promptTokens = &stats.PromptTokens
		completionTokens = &stats.CompletionTokens
		totalTimeMS = &stats.TotalTimeMS
	}
	var tcID *string
	if toolCallID != "" {
		tcID = &toolCallID
	}
	res, err := s.db.Exec(
		`INSERT INTO message (conversation_id, role, content, character_id, name, created_at,
		                      prompt_tokens, completion_tokens, total_time_ms, tool_calls, tool_call_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		conversationID, role, content, characterID, assistantName, t,
		promptTokens, completionTokens, totalTimeMS, toolCalls, tcID,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Message{
		ID: id, ConversationID: conversationID, Role: role, Content: content,
		CharacterID: characterID, Name: assistantName, CreatedAt: t,
		PromptTokens: promptTokens, CompletionTokens: completionTokens, TotalTimeMS: totalTimeMS,
		ToolCalls: toolCalls, ToolCallID: toolCallID,
	}, nil
}
