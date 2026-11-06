package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const (
	historyListDefault = 20
	historyListMax     = 50
	historySearchMax   = 20
	// historyReadMaxChars caps one history_read page so a long conversation
	// is read in pieces rather than filling the context in one go.
	historyReadMaxChars = 24000
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

// historyTime formats a stored RFC 3339 timestamp in the configured timezone.
func historyTime(ts, timezone string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			t = t.In(loc)
		}
	}
	return t.Format("Mon 2 Jan 2006 15:04")
}

// writeSummaryLine writes one conversation as a list line.
func writeSummaryLine(b *strings.Builder, c *store.ConversationSummary, timezone string) {
	title := "(untitled)"
	if c.Title != nil && *c.Title != "" {
		title = *c.Title
	}
	fmt.Fprintf(b, "\n#%d: %q", c.ID, title)
	if c.CharacterName != nil {
		fmt.Fprintf(b, " with %s", *c.CharacterName)
	}
	fmt.Fprintf(b, " — %d messages, started %s, last active %s",
		c.MessageCount, historyTime(c.CreatedAt, timezone), historyTime(c.UpdatedAt, timezone))
}

func executeHistoryList(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
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
	owner, err := historyUser(tctx, args.User, "history_list")
	if err != nil {
		return "", err
	}
	total, err := tctx.Store.CountConversations(owner.ID, tctx.ConversationID)
	if err != nil {
		return "", err
	}
	convs, err := tctx.Store.ListConversationSummaries(owner.ID, tctx.ConversationID, args.Limit, args.Offset)
	if err != nil {
		return "", err
	}
	if len(convs) == 0 {
		if total == 0 {
			return fmt.Sprintf("%s has no other conversations.", userLabel(owner)), nil
		}
		return fmt.Sprintf("No conversations at offset %d; %s has %d.", args.Offset, userLabel(owner), total), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Conversations of %s, most recent first (%d–%d of %d):",
		userLabel(owner), args.Offset+1, args.Offset+len(convs), total)
	for i := range convs {
		writeSummaryLine(&b, &convs[i], tctx.Timezone)
	}
	if next := args.Offset + len(convs); next < total {
		fmt.Fprintf(&b, "\n\nThere are older conversations: call history_list with offset=%d to see them.", next)
	}
	b.WriteString("\n\nRead one with history_read.")
	return b.String(), nil
}

func executeHistorySearch(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Query string `json:"query"`
		User  string `json:"user"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	words := store.SearchWords(args.Query)
	if len(words) == 0 {
		return "", errors.New("query needs at least one word to search for (very common words like \"the\" are ignored)")
	}
	owner, err := historyUser(tctx, args.User, "history_search")
	if err != nil {
		return "", err
	}
	matches, err := tctx.Store.SearchConversations(owner.ID, tctx.ConversationID, args.Query, historySearchMax)
	if err != nil {
		return "", err
	}
	searched := strings.Join(words, ", ")
	if len(matches) == 0 {
		return fmt.Sprintf("No conversations of %s mention any of: %s. Try other words for the same thing, or broader ones.", userLabel(owner), searched), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Conversations of %s mentioning any of: %s. Those matching the most words come first.", userLabel(owner), searched)
	for i := range matches {
		m := &matches[i]
		writeSummaryLine(&b, &m.ConversationSummary, tctx.Timezone)
		fmt.Fprintf(&b, "\n  Matches %d of %d words (%s)", len(m.MatchedWords), len(words), strings.Join(m.MatchedWords, ", "))
		if m.TitleMatch {
			b.WriteString(", in the title")
		}
		if m.MessageHits > 0 {
			fmt.Fprintf(&b, "; %d matching messages", m.MessageHits)
		}
		if m.Best != nil {
			fmt.Fprintf(&b, "\n  Best match (%s, %s): %s", m.Best.Role, historyTime(m.Best.CreatedAt, tctx.Timezone), m.Best.Snippet)
		}
	}
	if len(matches) == historySearchMax {
		fmt.Fprintf(&b, "\n\nShowing the best %d matches; there may be more.", historySearchMax)
	}
	b.WriteString("\n\nRead one with history_read.")
	return b.String(), nil
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
		Offset         int   `json:"offset"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.ConversationID == tctx.ConversationID {
		return "", errors.New("that is the current conversation")
	}
	self, err := tctx.Store.UserByID(tctx.UserID)
	if err != nil {
		return "", err
	}
	conv, err := tctx.Store.ConversationSummaryByID(args.ConversationID)
	if err == nil && conv.UserID != self.ID && !self.IsAdmin {
		err = store.ErrNotFound
	}
	if errors.Is(err, store.ErrNotFound) {
		return "", fmt.Errorf("no conversation #%d", args.ConversationID)
	}
	if err != nil {
		return "", err
	}
	owner := self
	if conv.UserID != self.ID {
		if owner, err = tctx.Store.UserByID(conv.UserID); err != nil {
			return "", err
		}
		log.Printf("Reading another user's conversation admin_id=%d user_id=%d read_conversation_id=%d conversation_id=%d",
			self.ID, owner.ID, conv.ID, tctx.ConversationID)
	}

	all, err := tctx.Store.ListMessages(conv.ID)
	if err != nil {
		return "", err
	}
	var msgs []store.Message
	for _, m := range all {
		if (m.Role == "user" || m.Role == "assistant") && m.Content != "" {
			msgs = append(msgs, m)
		}
	}
	args.Offset = max(args.Offset, 0)
	if args.Offset >= len(msgs) && len(msgs) > 0 {
		return "", fmt.Errorf("offset %d is past the end; the conversation has %d messages", args.Offset, len(msgs))
	}

	var b strings.Builder
	title := "(untitled)"
	if conv.Title != nil && *conv.Title != "" {
		title = *conv.Title
	}
	fmt.Fprintf(&b, "Conversation #%d %q of %s", conv.ID, title, userLabel(owner))
	if conv.CharacterName != nil {
		fmt.Fprintf(&b, " with %s", *conv.CharacterName)
	}
	fmt.Fprintf(&b, ", started %s, last active %s, %d messages.",
		historyTime(conv.CreatedAt, tctx.Timezone), historyTime(conv.UpdatedAt, tctx.Timezone), len(msgs))
	if len(msgs) == 0 {
		return b.String(), nil
	}
	if args.Offset > 0 {
		fmt.Fprintf(&b, "\nStarting at message %d.", args.Offset+1)
	}

	used := 0
	next := len(msgs)
	for i := args.Offset; i < len(msgs); i++ {
		m := msgs[i]
		left := historyReadMaxChars - used
		if i > args.Offset && utf8.RuneCountInString(m.Content) > left {
			next = i
			break
		}
		speaker := userSpeaker(owner)
		if m.Role == "assistant" {
			switch {
			case m.Name != nil && *m.Name != "":
				speaker = *m.Name
			case conv.CharacterName != nil:
				speaker = *conv.CharacterName
			default:
				speaker = "Assistant"
			}
		}
		content := m.Content
		if n := utf8.RuneCountInString(content); n > left {
			// A single message bigger than a page: show its start.
			content = string([]rune(content)[:left]) + fmt.Sprintf("\n[message cut off: %d more characters]", n-left)
		}
		used += utf8.RuneCountInString(content)
		fmt.Fprintf(&b, "\n\n[%d] %s (%s, %s):\n%s", i+1, speaker, m.Role, historyTime(m.CreatedAt, tctx.Timezone), content)
		if names := toolNames(m.ToolCalls); len(names) > 0 {
			fmt.Fprintf(&b, "\n[used tools: %s]", strings.Join(names, ", "))
		}
	}
	if next < len(msgs) {
		fmt.Fprintf(&b, "\n\n[Showing messages %d–%d of %d. Call history_read with conversation_id=%d and offset=%d for the rest.]",
			args.Offset+1, next, len(msgs), conv.ID, next)
	} else {
		b.WriteString("\n\n[End of conversation.]")
	}
	return b.String(), nil
}
