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
	for _, m := range memories {
		fmt.Fprintf(&b, "\n#%d: %s", m.ID, m.Text)
	}
	return b.String(), nil
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
	m, err := tctx.Store.CreateMemory(tctx.UserID, text, &convID)
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
	if err := tctx.Store.UpdateMemory(args.ID, tctx.UserID, text); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", fmt.Errorf("no memory #%d", args.ID)
		}
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
	if err := tctx.Store.DeleteMemory(args.ID, tctx.UserID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", fmt.Errorf("no memory #%d", args.ID)
		}
		return "", err
	}
	log.Printf("Forgetting memory id=%d user_id=%d conversation_id=%d", args.ID, tctx.UserID, tctx.ConversationID)
	return fmt.Sprintf("Forgot #%d.", args.ID), nil
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
