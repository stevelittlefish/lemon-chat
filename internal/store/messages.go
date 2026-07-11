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
