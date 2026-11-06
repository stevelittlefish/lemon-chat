package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const (
	historyListDefault = 20
	historyListMax     = 50
	historySearchMax   = 20
	// historyInConversationMax caps the matching messages shown when
	// searching inside one conversation.
	historyInConversationMax = 15
	// historyReadMaxChars caps one history_read call so a long conversation
	// is read in pieces rather than filling the context in one go.
	historyReadMaxChars = 24000
	// historyReadAround is how many messages before a search match the
	// suggested history_read call starts.
	historyReadAround = 3
)

// historyUserParam is the argument that lets an admin's model read another
// user's history. It is removed from the schemas sent for non-admins.
const historyUserParam = "user"

// isHistoryTool reports whether name is one of the history tools.
func isHistoryTool(name string) bool {
	return strings.HasPrefix(name, "history_")
}

// withoutHistoryUserParam drops the admin-only user argument from the history
// tool schemas, so non-admins' models aren't offered it.
func withoutHistoryUserParam(defs []toolDef) []toolDef {
	out := make([]toolDef, len(defs))
	for i, d := range defs {
		if isHistoryTool(d.Function.Name) {
			props := make(map[string]any, len(d.Function.Parameters.Properties))
			for k, v := range d.Function.Parameters.Properties {
				if k != historyUserParam {
					props[k] = v
				}
			}
			d.Function.Parameters.Properties = props
		}
		out[i] = d
	}
	return out
}

// historyUser resolves whose history a tool call reads: the asking user when
// name is empty, otherwise (admins only) the user with that username or
// display name. It logs every read of another user's history.
func historyUser(tctx ToolContext, name, tool string) (*store.User, error) {
	self, err := tctx.Store.UserByID(tctx.UserID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || matchesUser(self, name) {
		return self, nil
	}
	if !self.IsAdmin {
		return nil, errors.New("you can only see this user's own conversations")
	}
	users, err := tctx.Store.ListUsers()
	if err != nil {
		return nil, err
	}
	for i := range users {
		if matchesUser(&users[i], name) {
			log.Printf("Reading another user's conversation history admin_id=%d user_id=%d tool=%s conversation_id=%d",
				self.ID, users[i].ID, tool, tctx.ConversationID)
			return &users[i], nil
		}
	}
	names := make([]string, len(users))
	for i := range users {
		names[i] = userLabel(&users[i])
	}
	return nil, fmt.Errorf("no user %q; the users are: %s", name, strings.Join(names, ", "))
}

// historyConversation loads a conversation a history tool asked for by id and
// its owner. Users can open their own conversations, admins anyone's (logged).
func historyConversation(tctx ToolContext, id int64, tool string) (*store.ConversationSummary, *store.User, error) {
	if id == tctx.ConversationID {
		return nil, nil, errors.New("that is the current conversation")
	}
	self, err := tctx.Store.UserByID(tctx.UserID)
	if err != nil {
		return nil, nil, err
	}
	conv, err := tctx.Store.ConversationSummaryByID(id)
	if err == nil && conv.UserID != self.ID && !self.IsAdmin {
		err = store.ErrNotFound
	}
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, fmt.Errorf("no conversation #%d", id)
	}
	if err != nil {
		return nil, nil, err
	}
	if conv.UserID == self.ID {
		return conv, self, nil
	}
	owner, err := tctx.Store.UserByID(conv.UserID)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Reading another user's conversation admin_id=%d user_id=%d tool=%s read_conversation_id=%d conversation_id=%d",
		self.ID, owner.ID, tool, conv.ID, tctx.ConversationID)
	return conv, owner, nil
}

func matchesUser(u *store.User, name string) bool {
	return strings.EqualFold(u.Username, name) || (u.DisplayName != nil && strings.EqualFold(*u.DisplayName, name))
}

// userLabel names a user for the model: "Display name (username)".
func userLabel(u *store.User) string {
	if u.DisplayName != nil && *u.DisplayName != "" && *u.DisplayName != u.Username {
		return fmt.Sprintf("%s (%s)", *u.DisplayName, u.Username)
	}
	return u.Username
}

func userSpeaker(u *store.User) string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return u.Username
}

// historySpeaker names who said a message.
func historySpeaker(role string, name *string, owner *store.User, conv *store.ConversationSummary) string {
	switch {
	case role == "user":
		return userSpeaker(owner)
	case name != nil && *name != "":
		return *name
	case conv.CharacterName != nil:
		return *conv.CharacterName
	}
	return "Assistant"
}

func historyLocation(timezone string) *time.Location {
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			return loc
		}
	}
	return time.Local
}

// historyTime formats a stored RFC 3339 timestamp in the configured timezone.
func historyTime(ts, timezone string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.In(historyLocation(timezone)).Format("Mon 2 Jan 2006 15:04")
}

// historyToday is a sentence giving today's date, so the model can work out
// since/until for "last week" and the like.
func historyToday(timezone string) string {
	return "Today is " + time.Now().In(historyLocation(timezone)).Format("Mon 2 Jan 2006") + "."
}

// historyFilterArgs are the filter arguments history_list and history_search
// share.
type historyFilterArgs struct {
	Since     string `json:"since"`
	Until     string `json:"until"`
	Character string `json:"character"`
}

// filter turns the arguments into a store filter: since and until are whole
// days (YYYY-MM-DD) in the configured timezone, both included.
func (a historyFilterArgs) filter(timezone string) (store.HistoryFilter, error) {
	loc := historyLocation(timezone)
	day := func(name, value string) (time.Time, error) {
		t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), loc)
		if err != nil {
			return t, fmt.Errorf("%s must be a date written YYYY-MM-DD, e.g. 2026-03-14; got %q", name, value)
		}
		return t, nil
	}
	var f store.HistoryFilter
	var since, until time.Time
	var err error
	if strings.TrimSpace(a.Since) != "" {
		if since, err = day("since", a.Since); err != nil {
			return f, err
		}
		f.Since = since.UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(a.Until) != "" {
		if until, err = day("until", a.Until); err != nil {
			return f, err
		}
		if !since.IsZero() && until.Before(since) {
			return f, errors.New("until is before since")
		}
		f.Until = until.AddDate(0, 0, 1).UTC().Format(time.RFC3339)
	}
	f.Character = strings.TrimSpace(a.Character)
	return f, nil
}

// describe says what a filter keeps, for result headers ("" when nothing).
func (a historyFilterArgs) describe() string {
	var parts []string
	if c := strings.TrimSpace(a.Character); c != "" {
		parts = append(parts, fmt.Sprintf("with a character matching %q", c))
	}
	since, until := strings.TrimSpace(a.Since), strings.TrimSpace(a.Until)
	switch {
	case since != "" && until != "":
		parts = append(parts, fmt.Sprintf("with messages from %s to %s", since, until))
	case since != "":
		parts = append(parts, "with messages since "+since)
	case until != "":
		parts = append(parts, "with messages up to "+until)
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, ", ")
}

// checkCharacter fails with the names the model could use when no character
// the owner has talked to matches the character filter.
func checkCharacter(st *store.Store, owner *store.User, character string) error {
	if character == "" {
		return nil
	}
	names, err := st.UserCharacterNames(owner.ID)
	if err != nil {
		return err
	}
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), strings.ToLower(character)) {
			return nil
		}
	}
	if len(names) == 0 {
		return fmt.Errorf("%s has no conversations with a character", userLabel(owner))
	}
	return fmt.Errorf("no character name contains %q; %s has talked to: %s", character, userLabel(owner), strings.Join(names, ", "))
}

// countNoun writes n and the singular, or the plural form (default: singular
// plus "s") when n isn't 1.
func countNoun(n int, singular string, pluralForm ...string) string {
	if n == 1 {
		return "1 " + singular
	}
	if len(pluralForm) > 0 {
		return fmt.Sprintf("%d %s", n, pluralForm[0])
	}
	return fmt.Sprintf("%d %ss", n, singular)
}

func historyTitle(c *store.ConversationSummary) string {
	if c.Title != nil && *c.Title != "" {
		return *c.Title
	}
	return "(untitled)"
}

// writeSummaryLine writes one conversation as a list line.
func writeSummaryLine(b *strings.Builder, c *store.ConversationSummary, timezone string) {
	fmt.Fprintf(b, "\n#%d: %q", c.ID, historyTitle(c))
	if c.CharacterName != nil {
		fmt.Fprintf(b, " with %s", *c.CharacterName)
	}
	fmt.Fprintf(b, " — %s, started %s, last active %s",
		countNoun(c.MessageCount, "message"), historyTime(c.CreatedAt, timezone), historyTime(c.UpdatedAt, timezone))
}

func executeHistoryList(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		historyFilterArgs
		User   string `json:"user"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Limit <= 0 {
		args.Limit = historyListDefault
	}
	args.Limit = min(args.Limit, historyListMax)
	args.Offset = max(args.Offset, 0)
	f, err := args.filter(tctx.Timezone)
	if err != nil {
		return "", err
	}
	owner, err := historyUser(tctx, args.User, "history_list")
	if err != nil {
		return "", err
	}
	if err := checkCharacter(tctx.Store, owner, f.Character); err != nil {
		return "", err
	}
	total, err := tctx.Store.CountConversations(owner.ID, tctx.ConversationID, f)
	if err != nil {
		return "", err
	}
	convs, err := tctx.Store.ListConversationSummaries(owner.ID, tctx.ConversationID, f, args.Limit, args.Offset)
	if err != nil {
		return "", err
	}
	which := fmt.Sprintf("conversations of %s%s", userLabel(owner), args.describe())
	if len(convs) == 0 {
		if total == 0 {
			return fmt.Sprintf("There are no other %s. %s", which, historyToday(tctx.Timezone)), nil
		}
		return fmt.Sprintf("No conversations at offset %d; there are %d %s.", args.Offset, total, which), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s There are %d %s. Most recent first (%d–%d):",
		historyToday(tctx.Timezone), total, which, args.Offset+1, args.Offset+len(convs))
	for i := range convs {
		writeSummaryLine(&b, &convs[i], tctx.Timezone)
	}
	if next := args.Offset + len(convs); next < total {
		fmt.Fprintf(&b, "\n\nThere are older ones: call history_list with offset=%d to see them.", next)
	}
	b.WriteString("\n\nNext: to find something inside a conversation, use history_search with its conversation_id. To read one, use history_read.")
	return b.String(), nil
}

func executeHistorySearch(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		historyFilterArgs
		Query          string `json:"query"`
		ConversationID int64  `json:"conversation_id"`
		User           string `json:"user"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	words := store.SearchWords(args.Query)
	if len(words) == 0 {
		return "", errors.New("query needs at least one word to search for (very common words like \"the\" are ignored)")
	}
	f, err := args.filter(tctx.Timezone)
	if err != nil {
		return "", err
	}
	if args.ConversationID != 0 {
		return searchInConversation(tctx, args.ConversationID, args.Query, words, f)
	}
	owner, err := historyUser(tctx, args.User, "history_search")
	if err != nil {
		return "", err
	}
	if err := checkCharacter(tctx.Store, owner, f.Character); err != nil {
		return "", err
	}
	matches, err := tctx.Store.SearchConversations(owner.ID, tctx.ConversationID, args.Query, f, historySearchMax)
	if err != nil {
		return "", err
	}
	searched := strings.Join(words, ", ")
	which := fmt.Sprintf("conversations of %s%s", userLabel(owner), args.describe())
	if len(matches) == 0 {
		return fmt.Sprintf("No %s mention any of: %s. Try other words for the same thing, or broader ones.", which, searched), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s Showing %s mentioning any of: %s. Those matching the most words come first.", historyToday(tctx.Timezone), which, searched)
	for i := range matches {
		m := &matches[i]
		writeSummaryLine(&b, &m.ConversationSummary, tctx.Timezone)
		fmt.Fprintf(&b, "\n  Matches %d of %d words (%s)", len(m.MatchedWords), len(words), strings.Join(m.MatchedWords, ", "))
		if m.TitleMatch {
			b.WriteString(", in the title")
		}
		if m.MessageHits > 0 {
			fmt.Fprintf(&b, "; %s", countNoun(m.MessageHits, "matching message"))
		}
		if m.Best != nil {
			fmt.Fprintf(&b, "\n  Best match (%s, %s): %s", m.Best.Role, historyTime(m.Best.CreatedAt, tctx.Timezone), m.Best.Snippet)
		}
	}
	if len(matches) == historySearchMax {
		fmt.Fprintf(&b, "\n\nShowing the best %d; there may be more. Add since, until or character to narrow it down.", historySearchMax)
	}
	b.WriteString("\n\nNext: to see every matching message in one of these, call history_search again with the same query and its conversation_id. To read one, use history_read.")
	return b.String(), nil
}

// searchInConversation lists the best-matching messages in one conversation,
// in conversation order, numbered as history_read numbers them.
func searchInConversation(tctx ToolContext, convID int64, query string, words []string, f store.HistoryFilter) (string, error) {
	conv, owner, err := historyConversation(tctx, convID, "history_search")
	if err != nil {
		return "", err
	}
	msgs, err := historyMessages(tctx.Store, conv.ID)
	if err != nil {
		return "", err
	}
	number := make(map[int64]int, len(msgs))
	for i, m := range msgs {
		number[m.ID] = i + 1
	}
	hits, err := tctx.Store.SearchConversation(conv.ID, query, f, historyInConversationMax)
	if err != nil {
		return "", err
	}
	sort.Slice(hits, func(i, j int) bool { return number[hits[i].MessageID] < number[hits[j].MessageID] })

	var b strings.Builder
	fmt.Fprintf(&b, "Conversation #%d %q of %s", conv.ID, historyTitle(conv), userLabel(owner))
	if conv.CharacterName != nil {
		fmt.Fprintf(&b, " with %s", *conv.CharacterName)
	}
	fmt.Fprintf(&b, " (%s).", countNoun(len(msgs), "message"))
	searched := strings.Join(words, ", ")
	if len(hits) == 0 {
		fmt.Fprintf(&b, " No messages in it mention any of: %s.", searched)
		return b.String(), nil
	}
	if len(hits) == historyInConversationMax {
		fmt.Fprintf(&b, " The %d best messages mentioning any of: %s, in conversation order:", len(hits), searched)
	} else {
		fmt.Fprintf(&b, " %s any of: %s, in conversation order:", countNoun(len(hits), "message mentions", "messages mention"), searched)
	}
	for _, h := range hits {
		fmt.Fprintf(&b, "\n\n[message %d] %s (%s, %s): %s", number[h.MessageID],
			historySpeaker(h.Role, h.Name, owner, conv), h.Role, historyTime(h.CreatedAt, tctx.Timezone), h.Snippet)
	}
	first := number[hits[0].MessageID]
	fmt.Fprintf(&b, "\n\nNext: read a match with the messages around it using history_read. For example, for message %d: conversation_id=%d, start=%d, count=8.",
		first, conv.ID, max(first-historyReadAround, 1))
	return b.String(), nil
}

// historyMessages returns what the user and the assistant said in a
// conversation, in order. Message numbers in the history tools are positions
// in this list, counting from 1.
func historyMessages(st *store.Store, convID int64) ([]store.Message, error) {
	all, err := st.ListMessages(convID)
	if err != nil {
		return nil, err
	}
	var msgs []store.Message
	for _, m := range all {
		if (m.Role == "user" || m.Role == "assistant") && m.Content != "" {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

// toolNames lists the tools an assistant message called, from its raw JSON.
func toolNames(toolCalls *string) []string {
	if toolCalls == nil {
		return nil
	}
	var calls []struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	json.Unmarshal([]byte(*toolCalls), &calls) //nolint:errcheck
	var names []string
	for _, c := range calls {
		if c.Function.Name != "" {
			names = append(names, c.Function.Name)
		}
	}
	return names
}

func executeHistoryRead(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		ConversationID int64 `json:"conversation_id"`
		Start          int   `json:"start"`
		Count          int   `json:"count"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	conv, owner, err := historyConversation(tctx, args.ConversationID, "history_read")
	if err != nil {
		return "", err
	}
	msgs, err := historyMessages(tctx.Store, conv.ID)
	if err != nil {
		return "", err
	}
	start := max(args.Start, 1)
	if start > len(msgs) && len(msgs) > 0 {
		return "", fmt.Errorf("start=%d is past the end; the conversation has %d messages", start, len(msgs))
	}
	end := len(msgs) // the last message number to show
	if args.Count > 0 {
		end = min(start-1+args.Count, len(msgs))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Conversation #%d %q of %s", conv.ID, historyTitle(conv), userLabel(owner))
	if conv.CharacterName != nil {
		fmt.Fprintf(&b, " with %s", *conv.CharacterName)
	}
	fmt.Fprintf(&b, ", started %s, last active %s, %s.",
		historyTime(conv.CreatedAt, tctx.Timezone), historyTime(conv.UpdatedAt, tctx.Timezone), countNoun(len(msgs), "message"))
	if len(msgs) == 0 {
		return b.String(), nil
	}

	used := 0
	shown := start - 1 // the last message number written
	for i := start - 1; i < end; i++ {
		m := msgs[i]
		left := historyReadMaxChars - used
		if i > start-1 && utf8.RuneCountInString(m.Content) > left {
			break
		}
		content := m.Content
		if n := utf8.RuneCountInString(content); n > left {
			// A single message bigger than the limit: show its start.
			content = string([]rune(content)[:left]) + fmt.Sprintf("\n[message cut off: %d more characters]", n-left)
		}
		used += utf8.RuneCountInString(content)
		fmt.Fprintf(&b, "\n\n[message %d] %s (%s, %s):\n%s", i+1,
			historySpeaker(m.Role, m.Name, owner, conv), m.Role, historyTime(m.CreatedAt, tctx.Timezone), content)
		if names := toolNames(m.ToolCalls); len(names) > 0 {
			fmt.Fprintf(&b, "\n[used tools: %s]", strings.Join(names, ", "))
		}
		shown = i + 1
	}
	switch {
	case shown == len(msgs) && start == 1:
		b.WriteString("\n\n[End of conversation. You have read all of it.]")
	case shown == len(msgs):
		fmt.Fprintf(&b, "\n\n[Showing messages %d–%d of %d; that is the end of the conversation.]", start, shown, len(msgs))
	default:
		fmt.Fprintf(&b, "\n\n[Showing messages %d–%d of %d. To read on, call history_read with conversation_id=%d and start=%d.]",
			start, shown, len(msgs), conv.ID, shown+1)
	}
	return b.String(), nil
}
