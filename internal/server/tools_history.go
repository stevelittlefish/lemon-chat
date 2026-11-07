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

// historyScope is which conversations a set of history tools can see. The
// regular history_* tools see only the user's own; the admin_history_* tools
// see everyone's and refuse to run for non-admins. Both share one
// implementation.
type historyScope struct {
	admin  bool
	prefix string // tool name prefix, for "call X next" hints
}

var (
	ownHistory   = historyScope{prefix: "history_"}
	adminHistory = historyScope{admin: true, prefix: "admin_history_"}
)

func (sc historyScope) tool(name string) string { return sc.prefix + name }

// historySelf loads the asking user, refusing the admin tools to non-admins.
// The admin tools are withheld from non-admins' models, so this only trips if
// a model calls one it wasn't offered.
func (sc historyScope) historySelf(tctx ToolContext) (*store.User, error) {
	if sc.admin {
		return adminToolUser(tctx)
	}
	return tctx.Store.UserByID(tctx.UserID)
}

// adminToolUser loads the user an admin-only tool runs for, refusing
// non-admins.
func adminToolUser(tctx ToolContext) (*store.User, error) {
	self, err := tctx.Store.UserByID(tctx.UserID)
	if err != nil {
		return nil, err
	}
	if !self.IsAdmin {
		return nil, errors.New("this tool is for admins only")
	}
	return self, nil
}

// historyOwner resolves whose conversations a list or search covers: the
// asking user's own for the regular tools; for the admin tools, the user
// named (by username or display name), or everyone (nil) when name is empty.
// Admin calls are logged.
func (sc historyScope) historyOwner(tctx ToolContext, name, tool string) (self, owner *store.User, err error) {
	if self, err = sc.historySelf(tctx); err != nil {
		return nil, nil, err
	}
	if !sc.admin {
		return self, self, nil
	}
	if name = strings.TrimSpace(name); name != "" {
		if owner, err = findUser(tctx.Store, name); err != nil {
			return nil, nil, err
		}
	}
	log.Printf("Using admin history tool tool=%s admin_id=%d user_id=%d conversation_id=%d", tool, self.ID, ownerID(owner), tctx.ConversationID)
	return self, owner, nil
}

// findUser finds a user by username or display name, ignoring case; the error
// lists the users so the model can correct itself.
func findUser(st *store.Store, name string) (*store.User, error) {
	users, err := st.ListUsers()
	if err != nil {
		return nil, err
	}
	for i := range users {
		if matchesUser(&users[i], name) {
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
// its owner: the user's own for the regular tools, anyone's for the admin
// tools (logged).
func (sc historyScope) historyConversation(tctx ToolContext, id int64, tool string) (*store.ConversationSummary, *store.User, error) {
	if id == tctx.ConversationID {
		return nil, nil, errors.New("that is the current conversation")
	}
	self, err := sc.historySelf(tctx)
	if err != nil {
		return nil, nil, err
	}
	conv, err := tctx.Store.ConversationSummaryByID(id)
	if err == nil && conv.UserID != self.ID && !sc.admin {
		err = store.ErrNotFound
	}
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, fmt.Errorf("no conversation #%d", id)
	}
	if err != nil {
		return nil, nil, err
	}
	if sc.admin {
		log.Printf("Using admin history tool tool=%s admin_id=%d user_id=%d read_conversation_id=%d conversation_id=%d",
			tool, self.ID, conv.UserID, conv.ID, tctx.ConversationID)
	}
	if conv.UserID == self.ID {
		return conv, self, nil
	}
	owner, err := tctx.Store.UserByID(conv.UserID)
	if err != nil {
		return nil, nil, err
	}
	return conv, owner, nil
}

// whose says whose conversations a result covers.
func (sc historyScope) whose(owner *store.User) string {
	switch {
	case owner == nil:
		return "all users"
	case sc.admin:
		return userLabel(owner)
	}
	return userLabel(owner) + " (the person you are talking to)"
}

// ownerID is the store's owner argument: 0 for everyone.
func ownerID(owner *store.User) int64 {
	if owner == nil {
		return 0
	}
	return owner.ID
}

func matchesUser(u *store.User, name string) bool {
	return strings.EqualFold(u.Username, name) || (u.DisplayName != nil && strings.EqualFold(*u.DisplayName, name))
}

// nameLabel names a user for the model: "Display name (username)".
func nameLabel(username string, displayName *string) string {
	if displayName != nil && *displayName != "" && *displayName != username {
		return fmt.Sprintf("%s (%s)", *displayName, username)
	}
	return username
}

func userLabel(u *store.User) string { return nameLabel(u.Username, u.DisplayName) }

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

// historyFilterArgs are the filter arguments the list and search tools share.
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
// in the covered conversations matches the character filter.
func (sc historyScope) checkCharacter(st *store.Store, owner *store.User, character string) error {
	if character == "" {
		return nil
	}
	names, err := st.UserCharacterNames(ownerID(owner))
	if err != nil {
		return err
	}
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), strings.ToLower(character)) {
			return nil
		}
	}
	if len(names) == 0 {
		return fmt.Errorf("none of the conversations of %s are with a character", sc.whose(owner))
	}
	return fmt.Errorf("no character name contains %q; the conversations of %s are with: %s", character, sc.whose(owner), strings.Join(names, ", "))
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

// writeSummaryLine writes one conversation as a list line. The admin tools
// say whose conversation it is.
func (sc historyScope) writeSummaryLine(b *strings.Builder, c *store.ConversationSummary, timezone string) {
	fmt.Fprintf(b, "\n#%d", c.ID)
	if sc.admin {
		fmt.Fprintf(b, " (%s)", nameLabel(c.OwnerUsername, c.OwnerDisplayName))
	}
	fmt.Fprintf(b, ": %q", historyTitle(c))
	if c.CharacterName != nil {
		fmt.Fprintf(b, " with %s", *c.CharacterName)
	}
	fmt.Fprintf(b, " — %s, started %s, last active %s",
		countNoun(c.MessageCount, "message"), historyTime(c.CreatedAt, timezone), historyTime(c.UpdatedAt, timezone))
}

func executeHistoryList(argsJSON string, tctx ToolContext) (string, error) {
	return ownHistory.list(argsJSON, tctx)
}

func executeHistorySearch(argsJSON string, tctx ToolContext) (string, error) {
	return ownHistory.search(argsJSON, tctx)
}

func executeHistoryRead(argsJSON string, tctx ToolContext) (string, error) {
	return ownHistory.read(argsJSON, tctx)
}

func executeAdminHistoryList(argsJSON string, tctx ToolContext) (string, error) {
	return adminHistory.list(argsJSON, tctx)
}

func executeAdminHistorySearch(argsJSON string, tctx ToolContext) (string, error) {
	return adminHistory.search(argsJSON, tctx)
}

func executeAdminHistoryRead(argsJSON string, tctx ToolContext) (string, error) {
	return adminHistory.read(argsJSON, tctx)
}

func (sc historyScope) list(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		historyFilterArgs
		User   string `json:"user"` // admin tools only
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
	_, owner, err := sc.historyOwner(tctx, args.User, sc.tool("list"))
	if err != nil {
		return "", err
	}
	if err := sc.checkCharacter(tctx.Store, owner, f.Character); err != nil {
		return "", err
	}
	total, err := tctx.Store.CountConversations(ownerID(owner), tctx.ConversationID, f)
	if err != nil {
		return "", err
	}
	convs, err := tctx.Store.ListConversationSummaries(ownerID(owner), tctx.ConversationID, f, args.Limit, args.Offset)
	if err != nil {
		return "", err
	}
	which := fmt.Sprintf("conversations of %s%s", sc.whose(owner), args.describe())
	if len(convs) == 0 {
		if total == 0 {
			return fmt.Sprintf("%s There are no other %s.", historyToday(tctx.Timezone), which), nil
		}
		return fmt.Sprintf("No conversations at offset %d; there are %d %s.", args.Offset, total, which), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s There are %d %s. Most recent first (%d–%d):",
		historyToday(tctx.Timezone), total, which, args.Offset+1, args.Offset+len(convs))
	for i := range convs {
		sc.writeSummaryLine(&b, &convs[i], tctx.Timezone)
	}
	if next := args.Offset + len(convs); next < total {
		fmt.Fprintf(&b, "\n\nThere are older ones: call %s with offset=%d to see them.", sc.tool("list"), next)
	}
	fmt.Fprintf(&b, "\n\nNext: to find something inside a conversation, use %s with its conversation_id. To read one, use %s.",
		sc.tool("search"), sc.tool("read"))
	return b.String(), nil
}

func (sc historyScope) search(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		historyFilterArgs
		Query          string `json:"query"`
		ConversationID int64  `json:"conversation_id"`
		User           string `json:"user"` // admin tools only
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
		return sc.searchInConversation(tctx, args.ConversationID, args.Query, words, f)
	}
	_, owner, err := sc.historyOwner(tctx, args.User, sc.tool("search"))
	if err != nil {
		return "", err
	}
	if err := sc.checkCharacter(tctx.Store, owner, f.Character); err != nil {
		return "", err
	}
	matches, err := tctx.Store.SearchConversations(ownerID(owner), tctx.ConversationID, args.Query, f, historySearchMax)
	if err != nil {
		return "", err
	}
	searched := strings.Join(words, ", ")
	which := fmt.Sprintf("conversations of %s%s", sc.whose(owner), args.describe())
	if len(matches) == 0 {
		return fmt.Sprintf("No %s mention any of: %s. Try other words for the same thing, or broader ones.", which, searched), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s Showing %s mentioning any of: %s. Those matching the most words come first.", historyToday(tctx.Timezone), which, searched)
	for i := range matches {
		m := &matches[i]
		sc.writeSummaryLine(&b, &m.ConversationSummary, tctx.Timezone)
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
	fmt.Fprintf(&b, "\n\nNext: to see every matching message in one of these, call %s again with the same query and its conversation_id. To read one, use %s.",
		sc.tool("search"), sc.tool("read"))
	return b.String(), nil
}

// searchInConversation lists the best-matching messages in one conversation,
// in conversation order, numbered as the read tool numbers them.
func (sc historyScope) searchInConversation(tctx ToolContext, convID int64, query string, words []string, f store.HistoryFilter) (string, error) {
	conv, owner, err := sc.historyConversation(tctx, convID, sc.tool("search"))
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
	fmt.Fprintf(&b, "\n\nNext: read a match with the messages around it using %s. For example, for message %d: conversation_id=%d, start=%d, count=8.",
		sc.tool("read"), first, conv.ID, max(first-historyReadAround, 1))
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

func (sc historyScope) read(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		ConversationID int64 `json:"conversation_id"`
		Start          int   `json:"start"`
		Count          int   `json:"count"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	conv, owner, err := sc.historyConversation(tctx, args.ConversationID, sc.tool("read"))
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
		fmt.Fprintf(&b, "\n\n[Showing messages %d–%d of %d. To read on, call %s with conversation_id=%d and start=%d.]",
			start, shown, len(msgs), sc.tool("read"), conv.ID, shown+1)
	}
	return b.String(), nil
}

func executeAdminListUsers(_ string, tctx ToolContext) (string, error) {
	self, err := adminHistory.historySelf(tctx)
	if err != nil {
		return "", err
	}
	log.Printf("Using admin history tool tool=admin_list_users admin_id=%d conversation_id=%d", self.ID, tctx.ConversationID)
	users, err := tctx.Store.ListUserActivity()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s There are %s:", historyToday(tctx.Timezone), countNoun(len(users), "user"))
	for _, u := range users {
		fmt.Fprintf(&b, "\n- username: %s", u.Username)
		if u.DisplayName != nil && *u.DisplayName != "" && *u.DisplayName != u.Username {
			fmt.Fprintf(&b, " | display name: %s", *u.DisplayName)
		}
		if u.IsAdmin {
			b.WriteString(" | role: admin")
		} else {
			b.WriteString(" | role: user")
		}
		if u.ID == self.ID {
			b.WriteString(" | the person you are talking to")
		}
		if u.Conversations == 0 || u.LastActive == nil {
			b.WriteString(" | no conversations")
			continue
		}
		fmt.Fprintf(&b, " | %s, last active %s", countNoun(u.Conversations, "conversation"), historyTime(*u.LastActive, tctx.Timezone))
	}
	b.WriteString("\n\nNext: use admin_history_list or admin_history_search with user set to a username to look at one person's conversations, or leave user out to cover everyone's.")
	return b.String(), nil
}
