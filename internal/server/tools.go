package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

type toolParam struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

type toolFunction struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Parameters  toolParam `json:"parameters"`
}

type toolDef struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

// ToolContext carries per-request model context to tool executors that need it.
type ToolContext struct {
	ModelName       string
	ModelServer     *config.ModelServer
	ResponseTimeout time.Duration
	SearXNGURL      string
}

var toolRegistry = map[string]toolDef{
	"web_search": {
		Type: "function",
		Function: toolFunction{
			Name:        "web_search",
			Description: "Search the web using SearXNG and return the top results.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query.",
					},
					"max_results": map[string]any{
						"type":        "integer",
						"description": "Number of results to return (1–10, default 5).",
					},
				},
				Required: []string{"query"},
			},
		},
	},
	"get_time": {
		Type: "function",
		Function: toolFunction{
			Name:        "get_time",
			Description: "Returns the current local date and time.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
	"roll_dice": {
		Type: "function",
		Function: toolFunction{
			Name:        "roll_dice",
			Description: "Rolls dice using standard notation (e.g. 2d6, d20, 2d6+4, d20-5). Returns each die result and the total.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"notation": map[string]any{
						"type":        "string",
						"description": "Dice notation, e.g. '2d6', 'd20', '2d6+4', 'd20-5'. Count is optional (omit for 1 die).",
					},
				},
				Required: []string{"notation"},
			},
		},
	},
	"fetch_url": {
		Type: "function",
		Function: toolFunction{
			Name:        "fetch_url",
			Description: "Fetches the content of a URL. Returns a clean markdown summary by default. Set source to true to get the raw HTML source instead.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch.",
					},
					"source": map[string]any{
						"type":        "boolean",
						"description": "If true, return the raw HTML source instead of a markdown summary.",
					},
				},
				Required: []string{"url"},
			},
		},
	},
}

// ToolDefsForCharacter returns tool definitions for the given tool IDs.
func ToolDefsForCharacter(toolIDs []string) []toolDef {
	var out []toolDef
	for _, id := range toolIDs {
		if def, ok := toolRegistry[id]; ok {
			out = append(out, def)
		}
	}
	return out
}

var executors = map[string]func(string, ToolContext) (string, error){
	"get_time": func(_ string, _ ToolContext) (string, error) {
		now := time.Now()
		return now.Weekday().String() + ", " + now.Format(time.RFC3339), nil
	},
	"roll_dice": func(argsJSON string, _ ToolContext) (string, error) {
		var args struct {
			Notation string `json:"notation"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		halves := strings.SplitN(strings.ToLower(args.Notation), "d", 2)
		if len(halves) != 2 {
			return "", fmt.Errorf("invalid dice notation: %q", args.Notation)
		}

		// Empty prefix means 1 die (e.g. "d20")
		count := 1
		if halves[0] != "" {
			var err error
			count, err = strconv.Atoi(halves[0])
			if err != nil || count < 1 || count > 100 {
				return "", fmt.Errorf("invalid dice count: %q", halves[0])
			}
		}

		// Right part may include a modifier: "6", "6+4", "20-5"
		right := halves[1]
		modifier := 0
		sidesStr := right
		if idx := strings.IndexAny(right, "+-"); idx != -1 {
			var err error
			modifier, err = strconv.Atoi(right[idx:])
			if err != nil {
				return "", fmt.Errorf("invalid modifier: %q", right[idx:])
			}
			sidesStr = right[:idx]
		}
		sides, err := strconv.Atoi(sidesStr)
		if err != nil || sides < 2 || sides > 10000 {
			return "", fmt.Errorf("invalid die sides: %q", sidesStr)
		}

		rolls := make([]int, count)
		diceTotal := 0
		for i := range rolls {
			rolls[i] = rand.Intn(sides) + 1
			diceTotal += rolls[i]
		}
		total := diceTotal + modifier

		rollStrs := make([]string, count)
		for i, v := range rolls {
			rollStrs[i] = strconv.Itoa(v)
		}
		rollExpr := strings.Join(rollStrs, " + ")

		if modifier > 0 {
			return fmt.Sprintf("Rolled %s: %s + %d = %d", args.Notation, rollExpr, modifier, total), nil
		} else if modifier < 0 {
			return fmt.Sprintf("Rolled %s: %s - %d = %d", args.Notation, rollExpr, -modifier, total), nil
		} else if count == 1 {
			return fmt.Sprintf("Rolled %s: %d", args.Notation, rolls[0]), nil
		}
		return fmt.Sprintf("Rolled %s: %s = %d", args.Notation, rollExpr, total), nil
	},
	"fetch_url": func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			URL    string `json:"url"`
			Source bool   `json:"source"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.URL == "" {
			return "", fmt.Errorf("url is required")
		}

		log.Printf("Fetching URL url=%q source=%v", args.URL, args.Source)

		fetchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(fetchCtx, "GET", args.URL, nil)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		req.Header.Set("User-Agent", "lemon-chat/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("fetch failed: %w", err)
		}
		defer resp.Body.Close()

		const maxBody = 5 * 1024 * 1024
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
		if err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}

		content := string(body)

		if args.Source {
			const maxSource = 100_000
			if len(content) > maxSource {
				content = content[:maxSource] + "\n\n[truncated — content exceeded 100,000 characters]"
			}
			return content, nil
		}

		stripped := stripHTML(content)
		const maxStripped = 50_000
		if len(stripped) > maxStripped {
			stripped = stripped[:maxStripped] + "\n\n[truncated — page content exceeded 50,000 characters]"
		}

		if tctx.ModelServer == nil {
			return stripped, nil
		}
		return summariseHTML(stripped, tctx.ModelName, tctx.ModelServer, tctx.ResponseTimeout)
	},
	"web_search": func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Query == "" {
			return "", fmt.Errorf("query is required")
		}
		if tctx.SearXNGURL == "" {
			return "", fmt.Errorf("web_search is not configured (add [searxng] url to lemon.toml)")
		}
		n := args.MaxResults
		if n <= 0 {
			n = 5
		}
		if n > 10 {
			n = 10
		}

		log.Printf("Searching web query=%q max_results=%d", args.Query, n)

		searchURL := strings.TrimRight(tctx.SearXNGURL, "/") + "/search?q=" + url.QueryEscape(args.Query) + "&format=json&pageno=1"
		fetchCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(fetchCtx, "GET", searchURL, nil)
		if err != nil {
			return "", fmt.Errorf("build search request: %w", err)
		}
		req.Header.Set("User-Agent", "lemon-chat/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("search request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
		if err != nil {
			return "", fmt.Errorf("read search response: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("SearXNG returned %d", resp.StatusCode)
		}

		var data searxngResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return "", fmt.Errorf("decode search response: %w", err)
		}

		if len(data.Results) == 0 {
			return "No results found for: " + args.Query, nil
		}

		results := data.Results
		if len(results) > n {
			results = results[:n]
		}

		var sb strings.Builder
		for i, r := range results {
			fmt.Fprintf(&sb, "%d. **%s** — %s\n", i+1, r.Title, r.URL)
			if r.Content != "" {
				fmt.Fprintf(&sb, "   %s\n", r.Content)
			}
			sb.WriteString("\n")
		}
		return strings.TrimSpace(sb.String()), nil
	},
}

type searxngResponse struct {
	Results []searxngResult `json:"results"`
}

type searxngResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reTag    = regexp.MustCompile(`<[^>]+>`)
	reSpace  = regexp.MustCompile(`\s+`)
)

func stripHTML(html string) string {
	s := reScript.ReplaceAllString(html, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = reTag.ReplaceAllString(s, " ")
	s = reSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func summariseHTML(text, modelName string, srv *config.ModelServer, timeout time.Duration) (string, error) {
	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	msgs := []chatMsg{
		{Role: "system", Content: "Convert the following web page text to clean, well-structured markdown. Preserve headings, lists, links, and code blocks. Remove navigation, footer, and boilerplate text."},
		{Role: "user", Content: text},
	}
	payload, _ := json.Marshal(map[string]any{
		"model":    modelName,
		"messages": msgs,
		"stream":   false,
	})

	chatURL := srv.APIBase + "/chat/completions"
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build summarise request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if srv.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+srv.APIKey)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("summarise request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("read summarise response: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("model server returned %d: %s", httpResp.StatusCode, respBody)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode summarise response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in summarise response")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// ExecuteTool runs a tool by name and returns its result string.
func ExecuteTool(name, argsJSON string, tctx ToolContext) (string, error) {
	fn, ok := executors[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return fn(argsJSON, tctx)
}

// ToolMeta is the metadata shape returned to the frontend.
type ToolMeta struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

var allTools []ToolMeta

// InitTools builds the available tools list. Call once at server startup.
// web_search is included only when cfg.SearXNG.URL is non-empty.
func InitTools(cfg *config.Config) {
	allTools = []ToolMeta{
		{"get_time", "Get current time", "Returns the current local date and time."},
		{"roll_dice", "Roll dice", "Rolls dice using standard notation (e.g. 2d6, 1d20)."},
		{"fetch_url", "Fetch URL", "Fetches a URL and returns its content as markdown, or raw HTML if source is true."},
	}
	if cfg.SearXNG.URL != "" {
		allTools = append(allTools, ToolMeta{"web_search", "Web search", "Searches the web using SearXNG and returns the top results."})
	}
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, allTools)
}
