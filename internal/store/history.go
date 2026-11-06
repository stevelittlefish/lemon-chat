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

// SearchExcerpt is a matching message, with the matching words marked
// **like this**.
type SearchExcerpt struct {
	MessageID int64
	Role      string
	Name      *string
	CreatedAt string
	Snippet   string
}

// HistoryFilter narrows the history tools to a period and a character. Zero
// values don't filter.
type HistoryFilter struct {
	Since string // RFC 3339; only messages sent at or after this
	Until string // RFC 3339; only messages sent before this
	// Character keeps conversations whose character's name contains this
	// text, ignoring case.
	Character string
}

// historyMessageFilter selects the messages the history tools show: what the
// user and the assistant said, not tool calls or tool results.
const historyMessageFilter = `m.role IN ('user', 'assistant') AND m.content <> ''`

// messageTimeSQL limits message alias m to the filter's period.
func (f HistoryFilter) messageTimeSQL() (string, []any) {
	var sql string
	var args []any
	if f.Since != "" {
		sql += ` AND julianday(m.created_at) >= julianday(?)`
		args = append(args, f.Since)
	}
	if f.Until != "" {
		sql += ` AND julianday(m.created_at) < julianday(?)`
		args = append(args, f.Until)
	}
	return sql, args
}

// conversationSQL limits conversation alias c (joined to character ch) to the
// filter's character, and to conversations with a message in its period.
func (f HistoryFilter) conversationSQL() (string, []any) {
	var sql string
	var args []any
	if f.Character != "" {
		sql += ` AND ch.name LIKE ? ESCAPE '\'`
		args = append(args, likePattern(f.Character))
	}
	if timeSQL, timeArgs := f.messageTimeSQL(); timeSQL != "" {
		sql += ` AND EXISTS (SELECT 1 FROM message m WHERE m.conversation_id = c.id AND ` + historyMessageFilter + timeSQL + `)`
		args = append(args, timeArgs...)
	}
	return sql, args
}

// likePattern turns text into a LIKE pattern matching it anywhere, with the
// wildcards in it escaped (use with ESCAPE '\').
func likePattern(text string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(text) + "%"
}

const summaryColumns = `c.id, c.user_id, c.title, ch.name, c.model, c.created_at, c.updated_at,
	(SELECT COUNT(*) FROM message m WHERE m.conversation_id = c.id AND ` + historyMessageFilter + `)`

const summaryFrom = ` FROM conversation c LEFT JOIN character ch ON ch.id = c.character_id`

func scanSummary(row interface{ Scan(...any) error }, c *ConversationSummary) error {
	return row.Scan(&c.ID, &c.UserID, &c.Title, &c.CharacterName, &c.Model, &c.CreatedAt, &c.UpdatedAt, &c.MessageCount)
}

// ListConversationSummaries lists a user's conversations that pass f, most
// recently updated first, leaving out excludeID (the conversation asking).
func (s *Store) ListConversationSummaries(userID, excludeID int64, f HistoryFilter, limit, offset int) ([]ConversationSummary, error) {
	filterSQL, filterArgs := f.conversationSQL()
	args := append([]any{userID, excludeID}, filterArgs...)
	rows, err := s.db.Query(
		`SELECT `+summaryColumns+summaryFrom+`
		 WHERE c.user_id = ? AND c.id <> ?`+filterSQL+`
		 ORDER BY c.updated_at DESC, c.id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...,
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

// CountConversations returns how many of a user's conversations pass f,
// leaving out excludeID.
func (s *Store) CountConversations(userID, excludeID int64, f HistoryFilter) (int, error) {
	filterSQL, filterArgs := f.conversationSQL()
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*)`+summaryFrom+` WHERE c.user_id = ? AND c.id <> ?`+filterSQL,
		append([]any{userID, excludeID}, filterArgs...)...,
	).Scan(&n)
	return n, err
}

// UserCharacterNames lists the names of the characters a user has had
// conversations with, for telling the model which names it can filter on.
func (s *Store) UserCharacterNames(userID int64) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT ch.name`+summaryFrom+` WHERE c.user_id = ? AND ch.name IS NOT NULL ORDER BY ch.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
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

// anyWordQuery is the FTS5 query matching any of words.
func anyWordQuery(words []string) string {
	terms := make([]string, len(words))
	for i, w := range words {
		terms[i] = ftsTerm(w)
	}
	return strings.Join(terms, " OR ")
}

// SearchConversations finds a user's conversations that pass f, leaving out
// excludeID, that mention any of the words in query, in their title or in
// what the user and the assistant said. Words match whatever their ending
// ("tuned" finds "tuning"). With a period set, only messages sent in it
// count. Conversations mentioning the most different words come first, then
// those with the words in the title, then the most matching messages, then
// the most recent.
func (s *Store) SearchConversations(userID, excludeID int64, query string, f HistoryFilter, limit int) ([]ConversationMatch, error) {
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
	convSQL, convArgs := f.conversationSQL()
	timeSQL, timeArgs := f.messageTimeSQL()
	for _, w := range words {
		matched := map[int64]bool{}
		args := append([]any{ftsTerm(w), userID, excludeID}, timeArgs...)
		rows, err := s.db.Query(
			`SELECT m.conversation_id, COUNT(*) FROM message_fts
			 JOIN message m ON m.id = message_fts.rowid
			 JOIN conversation c ON c.id = m.conversation_id
			 LEFT JOIN character ch ON ch.id = c.character_id
			 WHERE message_fts MATCH ? AND c.user_id = ? AND c.id <> ?`+timeSQL+convSQL+`
			 GROUP BY m.conversation_id`,
			append(args, convArgs...)...,
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
			get(id).messages += n
			matched[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		rows, err = s.db.Query(
			`SELECT c.id FROM conversation_title_fts
			 JOIN conversation c ON c.id = conversation_title_fts.rowid
			 LEFT JOIN character ch ON ch.id = c.character_id
			 WHERE conversation_title_fts MATCH ? AND c.user_id = ? AND c.id <> ?`+convSQL,
			append([]any{ftsTerm(w), userID, excludeID}, convArgs...)...,
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

	for i := range out {
		if out[i].MessageHits == 0 {
			continue
		}
		excerpts, err := s.SearchConversation(out[i].ID, query, f, 1)
		if err != nil {
			return nil, err
		}
		if len(excerpts) > 0 {
			out[i].Best = &excerpts[0]
		}
	}
	return out, nil
}

// SearchConversation finds the messages in one conversation that mention any
// of the words in query and were sent in f's period (f's character is
// ignored), best matches first, at most limit of them.
func (s *Store) SearchConversation(conversationID int64, query string, f HistoryFilter, limit int) ([]SearchExcerpt, error) {
	words := SearchWords(query)
	if len(words) == 0 {
		return nil, nil
	}
	timeSQL, timeArgs := f.messageTimeSQL()
	args := append([]any{anyWordQuery(words), conversationID}, timeArgs...)
	rows, err := s.db.Query(
		`SELECT m.id, m.role, m.name, m.created_at, snippet(message_fts, 0, '**', '**', '…', 40)
		 FROM message_fts JOIN message m ON m.id = message_fts.rowid
		 WHERE message_fts MATCH ? AND m.conversation_id = ?`+timeSQL+`
		 ORDER BY rank LIMIT ?`,
		append(args, limit)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchExcerpt
	for rows.Next() {
		var e SearchExcerpt
		if err := rows.Scan(&e.MessageID, &e.Role, &e.Name, &e.CreatedAt, &e.Snippet); err != nil {
			return nil, err
		}
		e.Snippet = strings.Join(strings.Fields(e.Snippet), " ")
		out = append(out, e)
	}
	return out, rows.Err()
}

// ConversationSummaryByID returns one conversation's summary whoever owns it;
// callers check ownership.
func (s *Store) ConversationSummaryByID(id int64) (*ConversationSummary, error) {
	c := &ConversationSummary{}
	err := scanSummary(s.db.QueryRow(`SELECT `+summaryColumns+summaryFrom+` WHERE c.id = ?`, id), c)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}
