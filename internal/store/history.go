package store

import (
	"database/sql"
	"errors"
	"strings"
)

// ConversationSummary is one conversation as the history tools list it.
type ConversationSummary struct {
	ID            int64
	UserID        int64
	Title         *string
	CharacterName *string
	Model         *string
	CreatedAt     string
	UpdatedAt     string
	MessageCount  int // user and assistant messages with text
}

// ConversationMatch is a search hit: the conversation, how many of its
// messages matched, and the first matching message (nil when only the title
// matched).
type ConversationMatch struct {
	ConversationSummary
	MatchCount int
	FirstMatch *Message
}

// historyMessageFilter selects the messages the history tools show: what the
// user and the assistant said, not tool calls or tool results.
const historyMessageFilter = `m.role IN ('user', 'assistant') AND m.content <> ''`

const summaryColumns = `c.id, c.user_id, c.title, ch.name, c.model, c.created_at, c.updated_at,
	(SELECT COUNT(*) FROM message m WHERE m.conversation_id = c.id AND ` + historyMessageFilter + `)`

func scanSummary(row interface{ Scan(...any) error }, c *ConversationSummary, extra ...any) error {
	return row.Scan(append([]any{&c.ID, &c.UserID, &c.Title, &c.CharacterName, &c.Model, &c.CreatedAt, &c.UpdatedAt, &c.MessageCount}, extra...)...)
}

// ListConversationSummaries lists a user's conversations, most recently
// updated first, leaving out excludeID (the conversation asking).
func (s *Store) ListConversationSummaries(userID, excludeID int64, limit, offset int) ([]ConversationSummary, error) {
	rows, err := s.db.Query(
		`SELECT `+summaryColumns+`
		 FROM conversation c LEFT JOIN character ch ON ch.id = c.character_id
		 WHERE c.user_id = ? AND c.id <> ?
		 ORDER BY c.updated_at DESC, c.id DESC LIMIT ? OFFSET ?`,
		userID, excludeID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ConversationSummary
	for rows.Next() {
		var c ConversationSummary
		if err := scanSummary(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CountConversations returns how many conversations a user has, leaving out
// excludeID.
func (s *Store) CountConversations(userID, excludeID int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM conversation WHERE user_id = ? AND id <> ?`, userID, excludeID).Scan(&n)
	return n, err
}

// likePattern turns text into a LIKE pattern matching it anywhere, with the
// wildcards in it escaped (use with ESCAPE '\').
func likePattern(text string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(text) + "%"
}

// SearchConversations finds a user's conversations whose title or messages
// contain query (case-insensitive for ASCII), most recently updated first,
// leaving out excludeID.
func (s *Store) SearchConversations(userID, excludeID int64, query string, limit int) ([]ConversationMatch, error) {
	pattern := likePattern(query)
	rows, err := s.db.Query(
		`SELECT * FROM (
		   SELECT `+summaryColumns+`,
		     (SELECT COUNT(*) FROM message m WHERE m.conversation_id = c.id AND `+historyMessageFilter+`
		        AND m.content LIKE ? ESCAPE '\') AS hits
		   FROM conversation c LEFT JOIN character ch ON ch.id = c.character_id
		   WHERE c.user_id = ? AND c.id <> ?
		 ) WHERE hits > 0 OR title LIKE ? ESCAPE '\'
		 ORDER BY updated_at DESC, id DESC LIMIT ?`,
		pattern, userID, excludeID, pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	var out []ConversationMatch
	for rows.Next() {
		var c ConversationMatch
		if err := scanSummary(rows, &c.ConversationSummary, &c.MatchCount); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].MatchCount == 0 {
			continue
		}
		m := &Message{}
		err := s.db.QueryRow(
			`SELECT m.id, m.role, m.content, m.name, m.created_at FROM message m
			 WHERE m.conversation_id = ? AND `+historyMessageFilter+` AND m.content LIKE ? ESCAPE '\'
			 ORDER BY m.created_at ASC, m.id ASC LIMIT 1`,
			out[i].ID, pattern,
		).Scan(&m.ID, &m.Role, &m.Content, &m.Name, &m.CreatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		m.ConversationID = out[i].ID
		out[i].FirstMatch = m
	}
	return out, nil
}

// ConversationSummaryByID returns one conversation's summary whoever owns it;
// callers check ownership.
func (s *Store) ConversationSummaryByID(id int64) (*ConversationSummary, error) {
	c := &ConversationSummary{}
	err := scanSummary(s.db.QueryRow(
		`SELECT `+summaryColumns+`
		 FROM conversation c LEFT JOIN character ch ON ch.id = c.character_id
		 WHERE c.id = ?`, id,
	), c)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}
