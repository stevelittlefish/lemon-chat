package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// ListModels returns the sorted model IDs advertised by an OpenAI-compatible
// model server's /models endpoint.
func ListModels(ctx context.Context, client *http.Client, apiBase, apiKey string) ([]string, error) {
	url := strings.TrimRight(apiBase, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model server returned %d: %.300s", resp.StatusCode, body)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	models := make([]string, 0, len(result.Data))
	for _, model := range result.Data {
		if model.ID != "" {
			models = append(models, model.ID)
		}
	}
	slices.Sort(models)
	return models, nil
}
