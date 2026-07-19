package server

import (
	"encoding/json"
	"fmt"
)

func executeStateSet(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if len(args.Items) == 0 {
		return "", fmt.Errorf("items must not be empty")
	}
	for _, item := range args.Items {
		if item.Key == "" {
			return "", fmt.Errorf("each item must have a non-empty key")
		}
		if err := tctx.Store.SetState(tctx.ConversationID, item.Key, item.Value); err != nil {
			return "", err
		}
	}
	return "", nil
}

func executeStateModify(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Key   string  `json:"key"`
		Delta float64 `json:"delta"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Key == "" {
		return "", fmt.Errorf("key is required")
	}
	return tctx.Store.ModifyState(tctx.ConversationID, args.Key, args.Delta)
}

func executeStateUnset(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Key == "" {
		return "", fmt.Errorf("key is required")
	}
	return "", tctx.Store.UnsetState(tctx.ConversationID, args.Key)
}

func executeStateList(_ string, tctx ToolContext) (string, error) {
	return tctx.Store.ListState(tctx.ConversationID)
}

func executeStateClear(_ string, tctx ToolContext) (string, error) {
	n, err := tctx.Store.ClearState(tctx.ConversationID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Cleared %d key(s).", n), nil
}

func executeNoteToSelf(_ string, _ ToolContext) (string, error) {
	return "", nil
}

func executeNoteSave(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Key == "" {
		return "", fmt.Errorf("key is required")
	}
	if err := tctx.Store.SaveNote(args.Key, args.Value, tctx.UserID, tctx.ConversationID); err != nil {
		return "", err
	}
	return "saved", nil
}

func executeNoteLoad(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Key == "" {
		return "", fmt.Errorf("key is required")
	}
	n, err := tctx.Store.LoadNote(args.Key, tctx.UserID, tctx.ConversationID)
	if err != nil {
		return "", err
	}
	type noteResult struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		ReadOnly  bool   `json:"read_only"`
		UpdatedAt string `json:"updated_at"`
	}
	out, _ := json.Marshal(noteResult{Key: n.Key, Value: n.Value, ReadOnly: n.ReadOnly, UpdatedAt: n.UpdatedAt})
	return string(out), nil
}

func executeNoteList(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Prefix string `json:"prefix"`
	}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
	}
	return tctx.Store.ListNotes(args.Prefix, tctx.UserID, tctx.ConversationID)
}

func executeNoteDelete(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Key == "" {
		return "", fmt.Errorf("key is required")
	}
	if err := tctx.Store.DeleteNote(args.Key, tctx.UserID, tctx.ConversationID); err != nil {
		return "", err
	}
	return "deleted", nil
}

func executeNoteAppend(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Key  string `json:"key"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Key == "" {
		return "", fmt.Errorf("key is required")
	}
	return tctx.Store.AppendNote(args.Key, args.Text, tctx.UserID, tctx.ConversationID)
}
