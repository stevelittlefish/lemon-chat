// Package searx provides a small client for querying a SearXNG instance.
package searx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Result is a single SearXNG search result.
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// searchTimeout bounds a single SearXNG request.
const searchTimeout = 15 * time.Second

// Search queries the SearXNG instance at baseURL for query and returns its
// results in order. page is 1-based; values <= 0 are treated as page 1.
// The caller is responsible for capping the result count.
func Search(ctx context.Context, baseURL, query string, page int) ([]Result, error) {
	if page <= 0 {
		page = 1
	}
	searchURL := strings.TrimRight(baseURL, "/") + "/search?q=" + url.QueryEscape(query) + fmt.Sprintf("&format=json&pageno=%d", page)

	callCtx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build SearXNG request: %w", err)
	}
	req.Header.Set("User-Agent", "lemon-chat/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach SearXNG: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read SearXNG response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SearXNG returned HTTP %d", resp.StatusCode)
	}

	var data struct {
		Results []Result `json:"results"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("SearXNG response was not valid JSON: %w", err)
	}
	return data.Results, nil
}
