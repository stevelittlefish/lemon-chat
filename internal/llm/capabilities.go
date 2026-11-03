package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

// ImageInputSupport reports, for whichever of ids the server describes,
// whether the model accepts image input. IDs the server says nothing about are
// absent from the result. It reads the Codex catalog's input_modalities, an
// OpenAI-compatible /models listing's architecture.input_modalities
// (OpenRouter), or Ollama's /api/show capabilities; plain vLLM reports nothing.
// It is a diagnostic for --list-models, not used at runtime.
func ImageInputSupport(ctx context.Context, client *http.Client, srv *config.ModelServer, token, accountID string, ids []string) (map[string]bool, error) {
	base := strings.TrimRight(srv.APIBase, "/")
	out := map[string]bool{}
	if srv.UsesResponses() {
		var catalog struct {
			Models []struct {
				Slug            string   `json:"slug"`
				InputModalities []string `json:"input_modalities"`
			} `json:"models"`
		}
		if err := getJSON(ctx, client, base+"/models?client_version="+codexClientVersion, token, accountID, &catalog); err != nil {
			return nil, err
		}
		for _, m := range catalog.Models {
			if m.InputModalities != nil {
				out[m.Slug] = slices.Contains(m.InputModalities, "image")
			}
		}
		return out, nil
	}

	var listing struct {
		Data []struct {
			ID           string `json:"id"`
			Architecture *struct {
				InputModalities []string `json:"input_modalities"`
			} `json:"architecture"`
		} `json:"data"`
	}
	if err := getJSON(ctx, client, base+"/models", token, "", &listing); err != nil {
		return nil, err
	}
	for _, m := range listing.Data {
		if m.Architecture != nil && m.Architecture.InputModalities != nil {
			out[m.ID] = slices.Contains(m.Architecture.InputModalities, "image")
		}
	}

	// Ollama's OpenAI-compatible listing has no capabilities; its native
	// /api/show does. Other servers 404 here, which just leaves the ID unknown.
	root := strings.TrimSuffix(base, "/v1")
	for _, id := range ids {
		if _, known := out[id]; known {
			continue
		}
		if vision, ok := ollamaVision(ctx, client, root, id); ok {
			out[id] = vision
		}
	}
	return out, nil
}

func ollamaVision(ctx context.Context, client *http.Client, root, id string) (vision, ok bool) {
	body, _ := json.Marshal(map[string]string{"model": id})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, root+"/api/show", bytes.NewReader(body))
	if err != nil {
		return false, false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, false
	}
	var show struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&show); err != nil || show.Capabilities == nil {
		return false, false
	}
	return slices.Contains(show.Capabilities, "vision"), true
}

func getJSON(ctx context.Context, client *http.Client, url, token, accountID string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if accountID != "" {
		req.Header.Set("chatgpt-account-id", accountID)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("model request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("model server returned %d: %.300s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
