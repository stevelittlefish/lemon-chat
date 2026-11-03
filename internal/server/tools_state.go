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
