package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// memoryMaxChars caps one memory, so the model saves short facts, not essays.
const memoryMaxChars = 500

// memoryToolID is the compound tool that gives a character the memory tools.
const memoryToolID = "memory"

// validMemoryText trims text and checks it is non-empty and within the cap.
func validMemoryText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("text is required")
	}
	if n := utf8.RuneCountInString(text); n > memoryMaxChars {
		return "", fmt.Errorf("text is %d characters; the limit is %d — keep each memory to one short fact", n, memoryMaxChars)
	}
	return text, nil
}

// protectedNote tells the model what protected means, under any list that
// has one.
const protectedNote = "Memories marked (protected) were set by an admin: they can't be changed or forgotten with the memory tools, and they take priority over anything that contradicts them."

// formatMemories renders a user's memories for the model, one per line with
// the id the update and forget tools take.
func formatMemories(st *store.Store, userID int64) (string, error) {
	memories, err := st.ListMemories(userID)
	if err != nil {
		return "", err
	}
	if len(memories) == 0 {
		return "No memories saved yet.", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Memories about the user (%d):", len(memories))
	if writeMemoryLines(&b, memories) {
		b.WriteString("\n\n" + protectedNote)
	}
	return b.String(), nil
}

// writeMemoryLines writes one line per memory and reports whether any is
// protected.
func writeMemoryLines(b *strings.Builder, memories []store.Memory) (anyProtected bool) {
	for _, m := range memories {
		if m.Protected {
			fmt.Fprintf(b, "\n#%d (protected): %s", m.ID, m.Text)
			anyProtected = true
		} else {
			fmt.Fprintf(b, "\n#%d: %s", m.ID, m.Text)
		}
	}
	return anyProtected
}

func executeMemoryList(_ string, tctx ToolContext) (string, error) {
	return formatMemories(tctx.Store, tctx.UserID)
}

func executeMemorySave(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	text, err := validMemoryText(args.Text)
	if err != nil {
		return "", err
	}
	convID := tctx.ConversationID
	m, err := tctx.Store.CreateMemory(tctx.UserID, text, &convID, false)
	if err != nil {
		return "", err
	}
	log.Printf("Saving memory id=%d user_id=%d conversation_id=%d", m.ID, tctx.UserID, tctx.ConversationID)
	result := fmt.Sprintf("Saved as #%d.", m.ID)
	// Without the list loaded at the start, show it now so the model can spot
	// a duplicate and merge it.
	if !tctx.MemoriesLoaded {
		list, err := formatMemories(tctx.Store, tctx.UserID)
		if err != nil {
			return "", err
		}
		result += "\n\n" + list
	}
	return result, nil
}

// ownMemory loads one of the user's own memories for the update and forget
// tools, refusing protected ones.
func ownMemory(tctx ToolContext, id int64) (*store.Memory, error) {
	m, err := tctx.Store.GetMemory(id)
	if err == nil && m.UserID != tctx.UserID {
		err = store.ErrNotFound
	}
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("no memory #%d", id)
	}
	if err != nil {
		return nil, err
	}
	if m.Protected {
		return nil, fmt.Errorf("memory #%d is protected: only an admin can change or forget it", id)
	}
	return m, nil
}

func executeMemoryUpdate(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		ID   int64  `json:"id"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	text, err := validMemoryText(args.Text)
	if err != nil {
		return "", err
	}
	if _, err := ownMemory(tctx, args.ID); err != nil {
		return "", err
	}
	if err := tctx.Store.UpdateMemory(args.ID, tctx.UserID, text); err != nil {
		return "", err
	}
	log.Printf("Updating memory id=%d user_id=%d conversation_id=%d", args.ID, tctx.UserID, tctx.ConversationID)
	return fmt.Sprintf("Updated #%d.", args.ID), nil
}

func executeMemoryForget(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if _, err := ownMemory(tctx, args.ID); err != nil {
		return "", err
	}
	if err := tctx.Store.DeleteMemory(args.ID, tctx.UserID); err != nil {
		return "", err
	}
	log.Printf("Forgetting memory id=%d user_id=%d conversation_id=%d", args.ID, tctx.UserID, tctx.ConversationID)
	return fmt.Sprintf("Forgot #%d.", args.ID), nil
}

// ── Admin memory tools ───────────────────────────────────────────────

// The admin_memory_* tools manage anyone's memories, protected ones
// included, and set protection. Like the admin history tools they are
// withheld from non-admins and refuse to run for them; every call is logged.

func logAdminMemoryTool(tool string, self *store.User, ownerID, memoryID int64, tctx ToolContext) {
	log.Printf("Using admin memory tool tool=%s admin_id=%d user_id=%d memory_id=%d conversation_id=%d",
		tool, self.ID, ownerID, memoryID, tctx.ConversationID)
}

// adminMemory loads any user's memory by id for the admin tools.
func adminMemory(tctx ToolContext, id int64) (*store.Memory, *store.User, error) {
	m, err := tctx.Store.GetMemory(id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, fmt.Errorf("no memory #%d", id)
	}
	if err != nil {
		return nil, nil, err
	}
	owner, err := tctx.Store.UserByID(m.UserID)
	if err != nil {
		return nil, nil, err
	}
	return m, owner, nil
}

// memoryOwnerLabel names whose memories a list is: the admin's own are
// marked as the person they are talking to.
func memoryOwnerLabel(owner, self *store.User) string {
	if owner.ID == self.ID {
		return userLabel(owner) + " (the person you are talking to)"
	}
	return userLabel(owner)
}

func executeAdminMemoryList(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		User string `json:"user"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	self, err := adminToolUser(tctx)
	if err != nil {
		return "", err
	}
	// One named user, or everyone.
	var owner *store.User
	var owners []store.User
	if name := strings.TrimSpace(args.User); name != "" {
		if owner, err = findUser(tctx.Store, name); err != nil {
			return "", err
		}
		owners = []store.User{*owner}
	} else if owners, err = tctx.Store.ListUsers(); err != nil {
		return "", err
	}
	logAdminMemoryTool("admin_memory_list", self, ownerID(owner), 0, tctx)

	var b strings.Builder
	anyProtected := false
	for i := range owners {
		memories, err := tctx.Store.ListMemories(owners[i].ID)
		if err != nil {
			return "", err
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		label := memoryOwnerLabel(&owners[i], self)
		if len(memories) == 0 {
			fmt.Fprintf(&b, "Memories of %s: none.", label)
			continue
		}
		fmt.Fprintf(&b, "Memories of %s (%d):", label, len(memories))
		if writeMemoryLines(&b, memories) {
			anyProtected = true
		}
	}
	if anyProtected {
		b.WriteString("\n\nMemories marked (protected) can't be changed or forgotten by the user or their models; only admins can, with these admin tools.")
	}
	b.WriteString("\n\nNext: admin_memory_save adds a memory for a user, admin_memory_update changes one's text or protection by id, admin_memory_forget deletes one by id.")
	return b.String(), nil
}

func executeAdminMemorySave(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		User      string `json:"user"`
		Text      string `json:"text"`
		Protected bool   `json:"protected"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	self, err := adminToolUser(tctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(args.User) == "" {
		return "", errors.New("user is required: the username of whose memory this is, from admin_list_users or admin_memory_list")
	}
	owner, err := findUser(tctx.Store, strings.TrimSpace(args.User))
	if err != nil {
		return "", err
	}
	text, err := validMemoryText(args.Text)
	if err != nil {
		return "", err
	}
	// No source conversation: the admin's conversation isn't the owner's.
	m, err := tctx.Store.CreateMemory(owner.ID, text, nil, args.Protected)
	if err != nil {
		return "", err
	}
	logAdminMemoryTool("admin_memory_save", self, owner.ID, m.ID, tctx)
	if m.Protected {
		return fmt.Sprintf("Saved as #%d for %s, protected.", m.ID, userLabel(owner)), nil
	}
	return fmt.Sprintf("Saved as #%d for %s.", m.ID, userLabel(owner)), nil
}

func executeAdminMemoryUpdate(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		ID        int64   `json:"id"`
		Text      *string `json:"text"`
		Protected *bool   `json:"protected"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	self, err := adminToolUser(tctx)
	if err != nil {
		return "", err
	}
	if args.Text == nil && args.Protected == nil {
		return "", errors.New("give text to change what the memory says, protected to protect or unprotect it, or both")
	}
	m, owner, err := adminMemory(tctx, args.ID)
	if err != nil {
		return "", err
	}
	logAdminMemoryTool("admin_memory_update", self, owner.ID, m.ID, tctx)
	if args.Text != nil {
		text, err := validMemoryText(*args.Text)
		if err != nil {
			return "", err
		}
		if err := tctx.Store.UpdateMemory(m.ID, m.UserID, text); err != nil {
			return "", err
		}
	}
	if args.Protected != nil {
		if err := tctx.Store.SetMemoryProtected(m.ID, *args.Protected); err != nil {
			return "", err
		}
	}
	if m, err = tctx.Store.GetMemory(m.ID); err != nil {
		return "", err
	}
	state := "not protected"
	if m.Protected {
		state = "protected"
	}
	return fmt.Sprintf("Updated #%d for %s (%s): %s", m.ID, userLabel(owner), state, m.Text), nil
}

func executeAdminMemoryForget(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	self, err := adminToolUser(tctx)
	if err != nil {
		return "", err
	}
	m, owner, err := adminMemory(tctx, args.ID)
	if err != nil {
		return "", err
	}
	logAdminMemoryTool("admin_memory_forget", self, owner.ID, m.ID, tctx)
	if err := tctx.Store.DeleteMemory(m.ID, m.UserID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Forgot #%d for %s: %s", m.ID, userLabel(owner), m.Text), nil
}

// memoryAutoloadCallID is the tool call id of the memory_list call stored at
// the start of a conversation whose character has "Load memories at start".
func memoryAutoloadCallID(convID int64) string {
	return fmt.Sprintf("memory_autoload_%d", convID)
}

// loadsMemories reports whether a conversation with char should open with
// the user's memories: the memory tool is on and so is the toggle.
func loadsMemories(char *store.Character) bool {
	if char == nil || !char.LoadMemories {
		return false
	}
	for _, id := range char.Tools {
		if id == memoryToolID {
			return true
		}
	}
	return false
}
