package store

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
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

// ConversationMatch is a search hit: the conversation, which of the search
// words it mentions, how many of its messages match, and the best-matching
// message's extract (nil when only the title matched).
type ConversationMatch struct {
	ConversationSummary
	MatchedWords []string
	TitleMatch   bool
	MessageHits  int
	Best         *SearchExcerpt
}

// SearchExcerpt is the best-matching message in a conversation, with the
// matching words marked **like this**.
type SearchExcerpt struct {
	MessageID int64
	Role      string
	CreatedAt string
	Snippet   string
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

// searchStopWords are left out of searches: they appear in nearly every
// conversation, so they'd match everything.
var searchStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "but": true,
	"by": true, "did": true, "do": true, "for": true, "from": true, "had": true, "has": true, "have": true,
	"he": true, "her": true, "him": true, "his": true, "i": true, "if": true, "in": true, "is": true,
	"it": true, "its": true, "me": true, "my": true, "of": true, "on": true, "or": true, "she": true,
	"so": true, "that": true, "the": true, "their": true, "them": true, "they": true, "this": true,
	"to": true, "was": true, "we": true, "were": true, "what": true, "when": true, "which": true,
	"who": true, "with": true, "you": true, "your": true, "about": true, "talk": true, "talked": true,
}

// maxSearchWords caps how many words one search uses.
const maxSearchWords = 12

// SearchWords splits a search into the distinct words it will look for:
// lower-cased, without punctuation or stop words.
func SearchWords(query string) []string {
	var words []string
	seen := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if searchStopWords[w] || seen[w] {
			continue
		}
		seen[w] = true
		words = append(words, w)
		if len(words) == maxSearchWords {
			break
		}
	}
	return words
}

// ftsTerm is the FTS5 query for one word: quoted, so it is never read as
// query syntax, and a prefix match for words of four letters or more, so a
// partial word like "guit" still matches.
func ftsTerm(word string) string {
	t := `"` + word + `"`
	if utf8.RuneCountInString(word) >= 4 {
		t += "*"
	}
	return t
}

// SearchConversations finds a user's conversations, leaving out excludeID,
// that mention any of the words in query, in their title or in what the user
// and the assistant said. Words match whatever their ending ("tuned" finds
// "tuning"). Conversations mentioning the most different words come first,
// then those with the words in the title, then the most matching messages,
// then the most recent.
func (s *Store) SearchConversations(userID, excludeID int64, query string, limit int) ([]ConversationMatch, error) {
	words := SearchWords(query)
	if len(words) == 0 {
		return nil, nil
	}
	type hit struct {
		words    []string
		title    bool
		messages int
	}
	hits := map[int64]*hit{}
	get := func(id int64) *hit {
		if hits[id] == nil {
			hits[id] = &hit{}
		}
		return hits[id]
	}
	for _, w := range words {
		matched := map[int64]bool{}
		rows, err := s.db.Query(
			`SELECT m.conversation_id, COUNT(*) FROM message_fts
			 JOIN message m ON m.id = message_fts.rowid
			 JOIN conversation c ON c.id = m.conversation_id
			 WHERE message_fts MATCH ? AND c.user_id = ? AND c.id <> ?
			 GROUP BY m.conversation_id`,
			ftsTerm(w), userID, excludeID,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int64
			var n int
			if err := rows.Scan(&id, &n); err != nil {
				rows.Close()
				return nil, err
			}
			h := get(id)
			h.messages += n
			matched[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		rows, err = s.db.Query(
			`SELECT c.id FROM conversation_title_fts
			 JOIN conversation c ON c.id = conversation_title_fts.rowid
			 WHERE conversation_title_fts MATCH ? AND c.user_id = ? AND c.id <> ?`,
			ftsTerm(w), userID, excludeID,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			get(id).title = true
			matched[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		for id := range matched {
			hits[id].words = append(hits[id].words, w)
		}
	}
	if len(hits) == 0 {
		return nil, nil
	}

	var out []ConversationMatch
	for id, h := range hits {
		c, err := s.ConversationSummaryByID(id)
		if err != nil {
			return nil, err
		}
		out = append(out, ConversationMatch{ConversationSummary: *c, MatchedWords: h.words, TitleMatch: h.title, MessageHits: h.messages})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if len(a.MatchedWords) != len(b.MatchedWords) {
			return len(a.MatchedWords) > len(b.MatchedWords)
		}
		if a.TitleMatch != b.TitleMatch {
			return a.TitleMatch
		}
		if a.MessageHits != b.MessageHits {
			return a.MessageHits > b.MessageHits
		}
		if a.UpdatedAt != b.UpdatedAt {
			return a.UpdatedAt > b.UpdatedAt
		}
		return a.ID > b.ID
	})
	if len(out) > limit {
		out = out[:limit]
	}

	terms := make([]string, len(words))
	for i, w := range words {
		terms[i] = ftsTerm(w)
	}
	anyWord := strings.Join(terms, " OR ")
	for i := range out {
		if out[i].MessageHits == 0 {
			continue
		}
		e := &SearchExcerpt{}
		err := s.db.QueryRow(
			`SELECT m.id, m.role, m.created_at, snippet(message_fts, 0, '**', '**', '…', 40)
			 FROM message_fts JOIN message m ON m.id = message_fts.rowid
			 WHERE message_fts MATCH ? AND m.conversation_id = ?
			 ORDER BY rank LIMIT 1`,
			anyWord, out[i].ID,
		).Scan(&e.MessageID, &e.Role, &e.CreatedAt, &e.Snippet)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		e.Snippet = strings.Join(strings.Fields(e.Snippet), " ")
		out[i].Best = e
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
