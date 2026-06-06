package server

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
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

// ToolContext carries per-request context to tool executors.
type ToolContext struct {
	ModelName       string
	ModelServer     *config.ModelServer
	ResponseTimeout time.Duration
	SearXNGURL      string
	ComfyUIURL      string
	ComfyUIWorkflow string
	// for tools that create attachments
	ToolCallID     string
	ConversationID int64
	Store          *store.Store
}

var toolRegistry = map[string]toolDef{
	"create_document": {
		Type: "function",
		Function: toolFunction{
			Name:        "create_document",
			Description: "Creates a downloadable file. Use for reports, plans, scripts, code, or any content the user will want to save. Choose the filename extension to match the content type (e.g. report.md, script.py, notes.txt).",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Human-readable title shown in the chat.",
					},
					"filename": map[string]any{
						"type":        "string",
						"description": "Suggested filename including extension, e.g. report.md or analysis.py.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Full text content of the document.",
					},
				},
				Required: []string{"title", "filename", "content"},
			},
		},
	},
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
	"generate_image": {
		Type: "function",
		Function: toolFunction{
			Name:        "generate_image",
			Description: "Generates an image using Stable Diffusion. Use to illustrate scenes, characters, or objects described in the story. Be descriptive — include art style, lighting, and mood. The generated image is automatically displayed in the chat — do not describe or embed the image in your text response.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "Detailed visual description of the image. Include subject, setting, art style, lighting, mood.",
					},
					"negative_prompt": map[string]any{
						"type":        "string",
						"description": "Things to exclude from the image (optional).",
					},
				},
				Required: []string{"prompt"},
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

// AttachmentResult is the JSON structure returned by tools that create attachments.
// The messages layer detects this shape to emit the attachment SSE event.
type AttachmentResult struct {
	AttachmentID int64  `json:"attachment_id"`
	Title        string `json:"title"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
}

func mimeTypeForFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".md":
		return "text/markdown"
	case ".py":
		return "text/x-python"
	case ".js":
		return "text/javascript"
	case ".ts":
		return "text/typescript"
	case ".go":
		return "text/x-go"
	case ".sh":
		return "text/x-sh"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".json":
		return "application/json"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = cryptorand.Read(b)
	return hex.EncodeToString(b)
}

var executors = map[string]func(string, ToolContext) (string, error){
	"create_document": func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			Title    string `json:"title"`
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Title == "" || args.Filename == "" || args.Content == "" {
			return "", fmt.Errorf("title, filename, and content are required")
		}
		// Sanitise filename: strip any path components.
		args.Filename = filepath.Base(args.Filename)

		dir := filepath.Join("data", "attachments", randomID())
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("create attachment dir: %w", err)
		}
		diskPath := filepath.Join(dir, args.Filename)
		if err := os.WriteFile(diskPath, []byte(args.Content), 0644); err != nil {
			return "", fmt.Errorf("write attachment: %w", err)
		}

		mimeType := mimeTypeForFilename(args.Filename)
		log.Printf("Creating document attachment title=%q filename=%q conversation_id=%d", args.Title, args.Filename, tctx.ConversationID)

		att, err := tctx.Store.CreateAttachment(tctx.ToolCallID, tctx.ConversationID, args.Title, args.Filename, mimeType, diskPath)
		if err != nil {
			return "", fmt.Errorf("store attachment: %w", err)
		}

		result := AttachmentResult{
			AttachmentID: att.ID,
			Title:        att.Title,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
		}
		out, _ := json.Marshal(result)
		return string(out), nil
	},
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
	"generate_image": func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			Prompt         string `json:"prompt"`
			NegativePrompt string `json:"negative_prompt"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Prompt == "" {
			return "", fmt.Errorf("prompt is required")
		}
		if tctx.ComfyUIURL == "" {
			return "", fmt.Errorf("generate_image is not configured (add [comfyui] url to lemon.toml)")
		}

		workflowData, err := os.ReadFile(tctx.ComfyUIWorkflow)
		if err != nil {
			return "", fmt.Errorf("read workflow: %w", err)
		}

		// Replace placeholders: prompts use JSON-encoded strings; seed is a bare integer.
		promptJSON, _ := json.Marshal(args.Prompt)
		negJSON, _ := json.Marshal(args.NegativePrompt)
		seed := rand.Int63()
		workflowStr := strings.ReplaceAll(string(workflowData), `"__PROMPT__"`, string(promptJSON))
		workflowStr = strings.ReplaceAll(workflowStr, `"__NEGATIVE_PROMPT__"`, string(negJSON))
		workflowStr = strings.ReplaceAll(workflowStr, `__SEED__`, strconv.FormatInt(seed, 10))

		var workflow map[string]any
		if err := json.Unmarshal([]byte(workflowStr), &workflow); err != nil {
			return "", fmt.Errorf("parse workflow after substitution: %w", err)
		}

		clientID := randomID()
		promptPayload, _ := json.Marshal(map[string]any{
			"prompt":    workflow,
			"client_id": clientID,
		})

		log.Printf("Generating image prompt=%q conversation_id=%d", args.Prompt, tctx.ConversationID)

		comfyBase := strings.TrimRight(tctx.ComfyUIURL, "/")

		submitCtx, submitCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer submitCancel()

		req, err := http.NewRequestWithContext(submitCtx, "POST", comfyBase+"/prompt", bytes.NewReader(promptPayload))
		if err != nil {
			return "", fmt.Errorf("build prompt request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("submit prompt to ComfyUI: %w", err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("ComfyUI returned %d: %s", resp.StatusCode, respBody)
		}

		var promptResp struct {
			PromptID string `json:"prompt_id"`
		}
		if err := json.Unmarshal(respBody, &promptResp); err != nil || promptResp.PromptID == "" {
			return "", fmt.Errorf("parse ComfyUI prompt response: %w", err)
		}

		// Poll /history/{prompt_id} until the job completes or we time out.
		type comfyImage struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		}
		type comfyOutput struct {
			Images []comfyImage `json:"images"`
		}
		type comfyJob struct {
			Outputs map[string]comfyOutput `json:"outputs"`
		}

		deadline := time.Now().Add(120 * time.Second)
		var found *comfyImage
		for time.Now().Before(deadline) {
			time.Sleep(1 * time.Second)

			pollCtx, pollCancel := context.WithTimeout(context.Background(), 5*time.Second)
			pollReq, _ := http.NewRequestWithContext(pollCtx, "GET", comfyBase+"/history/"+promptResp.PromptID, nil)
			pollResp, pollErr := http.DefaultClient.Do(pollReq)
			pollCancel()
			if pollErr != nil {
				continue
			}
			pollBody, _ := io.ReadAll(pollResp.Body)
			pollResp.Body.Close()

			var history map[string]comfyJob
			if jsonErr := json.Unmarshal(pollBody, &history); jsonErr != nil {
				continue
			}
			if job, ok := history[promptResp.PromptID]; ok {
				for _, output := range job.Outputs {
					if len(output.Images) > 0 {
						img := output.Images[0]
						found = &img
						break
					}
				}
				if found != nil {
					break
				}
			}
		}

		if found == nil {
			return "", fmt.Errorf("image generation timed out after 120 seconds")
		}

		// Download the generated image.
		viewURL := comfyBase + "/view?filename=" + url.QueryEscape(found.Filename) +
			"&subfolder=" + url.QueryEscape(found.Subfolder) +
			"&type=" + url.QueryEscape(found.Type)

		dlCtx, dlCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dlCancel()

		dlReq, err := http.NewRequestWithContext(dlCtx, "GET", viewURL, nil)
		if err != nil {
			return "", fmt.Errorf("build download request: %w", err)
		}
		dlResp, err := http.DefaultClient.Do(dlReq)
		if err != nil {
			return "", fmt.Errorf("download image from ComfyUI: %w", err)
		}
		imgData, _ := io.ReadAll(dlResp.Body)
		dlResp.Body.Close()

		dir := filepath.Join("data", "attachments", randomID())
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("create attachment dir: %w", err)
		}
		diskPath := filepath.Join(dir, "image.png")
		if err := os.WriteFile(diskPath, imgData, 0644); err != nil {
			return "", fmt.Errorf("write image: %w", err)
		}

		att, err := tctx.Store.CreateAttachment(tctx.ToolCallID, tctx.ConversationID, "Generated image", "image.png", "image/png", diskPath)
		if err != nil {
			return "", fmt.Errorf("store attachment: %w", err)
		}

		result := AttachmentResult{
			AttachmentID: att.ID,
			Title:        "Generated image",
			Filename:     "image.png",
			MimeType:     "image/png",
		}
		out, _ := json.Marshal(result)
		return string(out), nil
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
// generate_image is included only when cfg.ComfyUI.URL is non-empty.
func InitTools(cfg *config.Config) {
	allTools = []ToolMeta{
		{"get_time", "Get current time", "Returns the current local date and time."},
		{"roll_dice", "Roll dice", "Rolls dice using standard notation (e.g. 2d6, 1d20)."},
		{"fetch_url", "Fetch URL", "Fetches a URL and returns its content as markdown, or raw HTML if source is true."},
		{"create_document", "Create document", "Saves a file (report, script, notes, etc.) the user can download."},
	}
	if cfg.SearXNG.URL != "" {
		allTools = append(allTools, ToolMeta{"web_search", "Web search", "Searches the web using SearXNG and returns the top results."})
	}
	if cfg.ComfyUI.URL != "" {
		allTools = append(allTools, ToolMeta{"generate_image", "Generate image", "Generates an image using Stable Diffusion via ComfyUI."})
	}
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, allTools)
}
