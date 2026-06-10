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
	Timezone        string
	// for tools that create attachments
	ToolCallID     string
	ConversationID int64
	Store          *store.Store
	DataDir        string
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
	"searxng": {
		Type: "function",
		Function: toolFunction{
			Name:        "searxng",
			Description: "Search the web via SearXNG and return the top results.",
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
					"page": map[string]any{
						"type":        "integer",
						"description": "Page number of results (default 1). Use higher values to retrieve more results beyond the first page.",
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
	"pick_random": {
		Type: "function",
		Function: toolFunction{
			Name:        "pick_random",
			Description: "Picks one item at random from a list of options. Use when you want a random outcome from a defined set — encounter type, weather, NPC mood, loot, etc. Define all options before calling; the result is server-side random and cannot be influenced.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"options": map[string]any{
						"type":        "array",
						"description": "The options to pick from. Must contain at least 2 items.",
						"items":       map[string]any{"type": "string"},
					},
				},
				Required: []string{"options"},
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
	"generate_image_sdxl": {
		Type: "function",
		Function: toolFunction{
			Name:        "generate_image_sdxl",
			Description: "Generates an image using Stable Diffusion XL via ComfyUI. Use to illustrate scenes, characters, or objects. Be descriptive — include art style, lighting, and mood. The generated image is automatically displayed in the chat — do not describe or embed it in your text response.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "Detailed visual description of the image. Include subject, setting, art style, lighting, and mood.",
					},
					"negative_prompt": map[string]any{
						"type":        "string",
						"description": "Things to exclude from the image (e.g. 'blurry, low quality, extra limbs').",
					},
					"seed": map[string]any{
						"type":        "integer",
						"description": "Seed for reproducible results. Omit or set to 0 for a random seed.",
					},
					"cfg": map[string]any{
						"type":        "number",
						"description": "Classifier-free guidance scale. Higher values follow the prompt more strictly. Default: 7.0. Typical range: 5–12.",
					},
					"width": map[string]any{
						"type":        "integer",
						"description": "Image width in pixels. Default: 1024. Must be a multiple of 64. SDXL works best at ~1 megapixel total — recommended sizes: 1024×1024 (square), 1344×768 (16:9), 768×1344 (9:16), 1216×832 (3:2), 832×1216 (2:3), 1152×896 (4:3), 896×1152 (3:4).",
					},
					"height": map[string]any{
						"type":        "integer",
						"description": "Image height in pixels. Default: 1024. Must be a multiple of 64. See width description for recommended size combinations.",
					},
					"steps": map[string]any{
						"type":        "integer",
						"description": "Number of diffusion steps. Default: 30. More steps can improve quality but take longer. Typical range: 20–50.",
					},
				},
				Required: []string{"prompt"},
			},
		},
	},
	"generate_image_flux": {
		Type: "function",
		Function: toolFunction{
			Name:        "generate_image_flux",
			Description: "Generates an image using Flux Schnell via ComfyUI. Fast, high-quality image generation. Use 1–4 steps for best results — no negative prompt needed. Be descriptive about subject, art style, lighting, and mood. The generated image is automatically displayed in the chat — do not describe or embed it in your text response.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "Detailed visual description of the image. Include subject, setting, art style, lighting, and mood.",
					},
					"seed": map[string]any{
						"type":        "integer",
						"description": "Seed for reproducible results. Omit or set to 0 for a random seed.",
					},
					"width": map[string]any{
						"type":        "integer",
						"description": "Image width in pixels. Default: 1024. Must be a multiple of 64. Recommended sizes: 1024×1024 (square), 1360×768 (16:9), 768×1360 (9:16), 1024×768 (4:3), 768×1024 (3:4).",
					},
					"height": map[string]any{
						"type":        "integer",
						"description": "Image height in pixels. Default: 1024. Must be a multiple of 64. See width description for recommended size combinations.",
					},
					"steps": map[string]any{
						"type":        "integer",
						"description": "Number of diffusion steps. Default: 4. Range: 1–8. Flux Schnell is optimised for very few steps.",
					},
				},
				Required: []string{"prompt"},
			},
		},
	},
	"wikipedia_search": {
		Type: "function",
		Function: toolFunction{
			Name:        "wikipedia_search",
			Description: "Searches Wikipedia and returns matching article titles and snippets. Use this to find the correct article title before calling wikipedia_get_page.",
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
					"lang": map[string]any{
						"type":        "string",
						"description": "Wikipedia language code (e.g. \"en\", \"fr\", \"de\"). Defaults to \"en\".",
					},
				},
				Required: []string{"query"},
			},
		},
	},
	"wikipedia_get_page": {
		Type: "function",
		Function: toolFunction{
			Name:        "wikipedia_get_page",
			Description: "Fetches a Wikipedia article. Without a section argument, returns the intro paragraph and a table of contents listing all sections. With a section argument, returns the full text of that section. Use iteratively to read an article section by section.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "The Wikipedia article title, exactly as returned by wikipedia_search.",
					},
					"section": map[string]any{
						"type":        "string",
						"description": "Section name to retrieve. Omit to get the intro and table of contents.",
					},
					"lang": map[string]any{
						"type":        "string",
						"description": "Wikipedia language code (e.g. \"en\", \"fr\", \"de\"). Defaults to \"en\".",
					},
				},
				Required: []string{"title"},
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
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
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
			return "", fmt.Errorf("title, filename, and content are all required and must be non-empty")
		}
		// Sanitise filename: strip any path components.
		args.Filename = filepath.Base(args.Filename)

		relDir := filepath.Join("attachments", randomID())
		absDir := filepath.Join(tctx.DataDir, relDir)
		if err := os.MkdirAll(absDir, 0755); err != nil {
			return "", fmt.Errorf("server error: could not create storage directory (%w) — tell the user the document could not be saved due to a server storage problem", err)
		}
		if err := os.WriteFile(filepath.Join(absDir, args.Filename), []byte(args.Content), 0644); err != nil {
			return "", fmt.Errorf("server error: could not write document to disk (%w) — tell the user the document could not be saved due to a server storage problem", err)
		}
		diskPath := filepath.Join(relDir, args.Filename)

		mimeType := mimeTypeForFilename(args.Filename)
		log.Printf("Creating document attachment title=%q filename=%q conversation_id=%d", args.Title, args.Filename, tctx.ConversationID)

		att, err := tctx.Store.CreateAttachment(tctx.ToolCallID, tctx.ConversationID, args.Title, args.Filename, mimeType, diskPath)
		if err != nil {
			return "", fmt.Errorf("server error: could not record document in database (%w) — tell the user the document could not be saved due to a server error", err)
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
	"get_time": func(_ string, tctx ToolContext) (string, error) {
		now := time.Now()
		if tctx.Timezone != "" {
			loc, err := time.LoadLocation(tctx.Timezone)
			if err == nil {
				now = now.In(loc)
			} else {
				return "", fmt.Errorf("invalid timezone configuration: %w", err)
			}
		}
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
				return "", fmt.Errorf("invalid dice count %q: must be a whole number between 1 and 100", halves[0])
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
			return "", fmt.Errorf("invalid die sides %q: must be a whole number between 2 and 10000", sidesStr)
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
	"pick_random": func(argsJSON string, _ ToolContext) (string, error) {
		var args struct {
			Options []string `json:"options"`
			Label   string   `json:"label"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if len(args.Options) < 2 {
			return "", fmt.Errorf("options must contain at least 2 items")
		}
		return args.Options[rand.Intn(len(args.Options))], nil
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
			return "", fmt.Errorf("invalid URL %q: %w", args.URL, err)
		}
		req.Header.Set("User-Agent", "lemon-chat/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("could not fetch %q: %w — tell the user the page could not be retrieved and suggest checking the URL", args.URL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("server returned HTTP %d for %q — tell the user the page could not be retrieved (HTTP %d)", resp.StatusCode, args.URL, resp.StatusCode)
		}

		const maxBody = 5 * 1024 * 1024
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
		if err != nil {
			return "", fmt.Errorf("could not read response from %q: %w", args.URL, err)
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
	"searxng": func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
			Page       int    `json:"page"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Query == "" {
			return "", fmt.Errorf("query is required")
		}
		if tctx.SearXNGURL == "" {
			return "", fmt.Errorf("searxng is not configured (add [searxng] url to lemon.toml)")
		}
		n := args.MaxResults
		if n <= 0 {
			n = 5
		}
		if n > 10 {
			n = 10
		}
		page := args.Page
		if page <= 0 {
			page = 1
		}

		log.Printf("Searching web query=%q max_results=%d page=%d", args.Query, n, page)

		searchURL := strings.TrimRight(tctx.SearXNGURL, "/") + "/search?q=" + url.QueryEscape(args.Query) + fmt.Sprintf("&format=json&pageno=%d", page)
		fetchCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(fetchCtx, "GET", searchURL, nil)
		if err != nil {
			return "", fmt.Errorf("the SearXNG URL in lemon.toml appears to be malformed: %w", err)
		}
		req.Header.Set("User-Agent", "lemon-chat/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("could not reach SearXNG at %q: %w — tell the user that the search service is currently unreachable and they may want to check that SearXNG is running", tctx.SearXNGURL, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
		if err != nil {
			return "", fmt.Errorf("could not read response from SearXNG: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("SearXNG returned HTTP %d — tell the user the search service returned an error", resp.StatusCode)
		}

		var data searxngResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return "", fmt.Errorf("SearXNG response was not valid JSON (got: %.200s): %w — SearXNG may need to have the JSON format enabled", string(body), err)
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
	"wikipedia_search": func(argsJSON string, _ ToolContext) (string, error) {
		var args struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
			Lang       string `json:"lang"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Query == "" {
			return "", fmt.Errorf("query is required")
		}
		lang := args.Lang
		if lang == "" {
			lang = "en"
		}
		n := args.MaxResults
		if n <= 0 {
			n = 5
		}
		if n > 10 {
			n = 10
		}

		log.Printf("Searching Wikipedia query=%q lang=%q", args.Query, lang)

		searchURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&format=json&srlimit=%d",
			lang, url.QueryEscape(args.Query), n)

		fetchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(fetchCtx, "GET", searchURL, nil)
		if err != nil {
			return "", fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", "lemon-chat/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("could not reach Wikipedia: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
		if err != nil {
			return "", fmt.Errorf("could not read response: %w", err)
		}

		var data struct {
			Query struct {
				Search []struct {
					Title   string `json:"title"`
					Snippet string `json:"snippet"`
				} `json:"search"`
			} `json:"query"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", fmt.Errorf("could not parse Wikipedia response: %w", err)
		}

		if len(data.Query.Search) == 0 {
			return "No results found for: " + args.Query, nil
		}

		var sb strings.Builder
		for i, r := range data.Query.Search {
			snippet := strings.TrimSpace(stripHTML(r.Snippet))
			fmt.Fprintf(&sb, "%d. **%s** — %s\n", i+1, r.Title, snippet)
		}
		return strings.TrimSpace(sb.String()), nil
	},
	"wikipedia_get_page": func(argsJSON string, _ ToolContext) (string, error) {
		var args struct {
			Title   string `json:"title"`
			Section string `json:"section"`
			Lang    string `json:"lang"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Title == "" {
			return "", fmt.Errorf("title is required")
		}
		lang := args.Lang
		if lang == "" {
			lang = "en"
		}

		log.Printf("Fetching Wikipedia page title=%q section=%q lang=%q", args.Title, args.Section, lang)

		fetchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		doGet := func(u string) ([]byte, int, error) {
			req, err := http.NewRequestWithContext(fetchCtx, "GET", u, nil)
			if err != nil {
				return nil, 0, err
			}
			req.Header.Set("User-Agent", "lemon-chat/1.0")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, 0, err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
			return body, resp.StatusCode, err
		}

		// Fetch TOC (needed for both the intro and section cases).
		tocURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php?action=parse&page=%s&prop=sections&format=json",
			lang, url.QueryEscape(args.Title))
		tocBody, tocStatus, err := doGet(tocURL)
		if err != nil {
			return "", fmt.Errorf("could not reach Wikipedia: %w", err)
		}
		if tocStatus != http.StatusOK {
			return "", fmt.Errorf("Wikipedia returned HTTP %d for %q", tocStatus, args.Title)
		}

		var tocData struct {
			Parse struct {
				Sections []struct {
					Number string `json:"number"`
					Line   string `json:"line"`
					Index  string `json:"index"`
				} `json:"sections"`
			} `json:"parse"`
			Error struct {
				Info string `json:"info"`
			} `json:"error"`
		}
		if err := json.Unmarshal(tocBody, &tocData); err != nil {
			return "", fmt.Errorf("could not parse Wikipedia response: %w", err)
		}
		if tocData.Error.Info != "" {
			return "", fmt.Errorf("Wikipedia error for %q: %s — try wikipedia_search to find the correct article title", args.Title, tocData.Error.Info)
		}

		if args.Section == "" {
			// Fetch intro via the REST summary endpoint.
			summaryURL := fmt.Sprintf("https://%s.wikipedia.org/api/rest_v1/page/summary/%s",
				lang, url.PathEscape(args.Title))
			summaryBody, summaryStatus, err := doGet(summaryURL)
			if err != nil {
				return "", fmt.Errorf("could not reach Wikipedia: %w", err)
			}
			if summaryStatus == http.StatusNotFound {
				return "", fmt.Errorf("Wikipedia article %q not found — try wikipedia_search to find the correct article title", args.Title)
			}
			if summaryStatus != http.StatusOK {
				return "", fmt.Errorf("Wikipedia returned HTTP %d for %q", summaryStatus, args.Title)
			}

			var summary struct {
				Extract string `json:"extract"`
			}
			if err := json.Unmarshal(summaryBody, &summary); err != nil {
				return "", fmt.Errorf("could not parse Wikipedia summary: %w", err)
			}

			var sb strings.Builder
			sb.WriteString(summary.Extract)
			if len(tocData.Parse.Sections) > 0 {
				sb.WriteString("\n\nSections:\n")
				for _, s := range tocData.Parse.Sections {
					title := strings.TrimSpace(stripHTML(s.Line))
					fmt.Fprintf(&sb, "%s. %s\n", s.Number, title)
				}
			}
			return strings.TrimSpace(sb.String()), nil
		}

		// Section requested — find its index.
		sectionLower := strings.ToLower(args.Section)
		sectionIndex := ""
		for _, s := range tocData.Parse.Sections {
			title := strings.TrimSpace(stripHTML(s.Line))
			if strings.ToLower(title) == sectionLower {
				sectionIndex = s.Index
				break
			}
		}
		if sectionIndex == "" {
			var names []string
			for _, s := range tocData.Parse.Sections {
				names = append(names, strings.TrimSpace(stripHTML(s.Line)))
			}
			return "", fmt.Errorf("section %q not found in %q — available sections: %s", args.Section, args.Title, strings.Join(names, ", "))
		}

		contentURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php?action=parse&page=%s&prop=text&section=%s&format=json",
			lang, url.QueryEscape(args.Title), sectionIndex)
		contentBody, contentStatus, err := doGet(contentURL)
		if err != nil {
			return "", fmt.Errorf("could not reach Wikipedia: %w", err)
		}
		if contentStatus != http.StatusOK {
			return "", fmt.Errorf("Wikipedia returned HTTP %d fetching section %q", contentStatus, args.Section)
		}

		var contentData struct {
			Parse struct {
				Text map[string]string `json:"text"`
			} `json:"parse"`
		}
		if err := json.Unmarshal(contentBody, &contentData); err != nil {
			return "", fmt.Errorf("could not parse Wikipedia section response: %w", err)
		}
		html := contentData.Parse.Text["*"]
		if html == "" {
			return fmt.Sprintf("Section %q appears to be empty.", args.Section), nil
		}
		return strings.TrimSpace(stripHTML(html)), nil
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
	Configured  bool   `json:"configured"`
	ConfigHint  string `json:"config_hint,omitempty"`
}

var allTools []ToolMeta

// makeImageExecutor returns an executor for a ComfyUI workflow. comfyURL,
// workflowFile, defaultSteps, and defaultCFG are captured at startup.
func makeImageExecutor(comfyURL, workflowFile string, defaultSteps int, defaultCFG float64) func(string, ToolContext) (string, error) {
	return func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			Prompt         string  `json:"prompt"`
			NegativePrompt string  `json:"negative_prompt"`
			Seed           int64   `json:"seed"`
			CFG            float64 `json:"cfg"`
			Width          int     `json:"width"`
			Height         int     `json:"height"`
			Steps          int     `json:"steps"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Prompt == "" {
			return "", fmt.Errorf("prompt is required")
		}

		seed := args.Seed
		if seed <= 0 {
			seed = rand.Int63()
		}
		cfg := args.CFG
		if cfg <= 0 {
			cfg = defaultCFG
		}
		width := args.Width
		if width <= 0 {
			width = 1024
		}
		height := args.Height
		if height <= 0 {
			height = 1024
		}
		steps := args.Steps
		if steps <= 0 {
			steps = defaultSteps
		}

		workflowData, err := os.ReadFile(workflowFile)
		if err != nil {
			return "", fmt.Errorf("generate_image: cannot read workflow file %q — check the workflow path in lemon.toml: %w", workflowFile, err)
		}

		// Replace placeholders: prompts use JSON-encoded strings; numeric values are bare.
		// Placeholders absent from the workflow JSON are replaced as no-ops.
		promptJSON, _ := json.Marshal(args.Prompt)
		negJSON, _ := json.Marshal(args.NegativePrompt)
		workflowStr := strings.ReplaceAll(string(workflowData), `"__PROMPT__"`, string(promptJSON))
		workflowStr = strings.ReplaceAll(workflowStr, `"__NEGATIVE_PROMPT__"`, string(negJSON))
		workflowStr = strings.ReplaceAll(workflowStr, `__SEED__`, strconv.FormatInt(seed, 10))
		workflowStr = strings.ReplaceAll(workflowStr, `__CFG__`, strconv.FormatFloat(cfg, 'f', -1, 64))
		workflowStr = strings.ReplaceAll(workflowStr, `__WIDTH__`, strconv.Itoa(width))
		workflowStr = strings.ReplaceAll(workflowStr, `__HEIGHT__`, strconv.Itoa(height))
		workflowStr = strings.ReplaceAll(workflowStr, `__STEPS__`, strconv.Itoa(steps))

		var workflow map[string]any
		if err := json.Unmarshal([]byte(workflowStr), &workflow); err != nil {
			return "", fmt.Errorf("workflow file %q contains invalid JSON: %w — tell the user the image could not be generated due to an invalid workflow file", workflowFile, err)
		}

		clientID := randomID()
		promptPayload, _ := json.Marshal(map[string]any{
			"prompt":    workflow,
			"client_id": clientID,
		})

		log.Printf("Generating image prompt=%q conversation_id=%d", args.Prompt, tctx.ConversationID)

		comfyBase := strings.TrimRight(comfyURL, "/")

		submitCtx, submitCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer submitCancel()

		req, err := http.NewRequestWithContext(submitCtx, "POST", comfyBase+"/prompt", bytes.NewReader(promptPayload))
		if err != nil {
			return "", fmt.Errorf("build prompt request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("could not reach ComfyUI at %q: %w — tell the user the image could not be generated because ComfyUI is unreachable and they should check that it is running", comfyURL, err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("ComfyUI returned HTTP %d: %s — tell the user the image could not be generated because ComfyUI returned an error", resp.StatusCode, respBody)
		}

		var promptResp struct {
			PromptID string `json:"prompt_id"`
		}
		if err := json.Unmarshal(respBody, &promptResp); err != nil || promptResp.PromptID == "" {
			return "", fmt.Errorf("ComfyUI returned an unexpected response (expected a prompt_id): %s — tell the user image generation failed due to an unexpected ComfyUI response", string(respBody))
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
			return "", fmt.Errorf("image generation timed out after 120 seconds — tell the user the image was not produced in time and suggest checking that ComfyUI is processing jobs correctly")
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
			return "", fmt.Errorf("image was generated but could not be downloaded from ComfyUI: %w — tell the user the image was produced but could not be retrieved", err)
		}
		imgData, _ := io.ReadAll(dlResp.Body)
		dlResp.Body.Close()

		relDir := filepath.Join("attachments", randomID())
		absDir := filepath.Join(tctx.DataDir, relDir)
		if err := os.MkdirAll(absDir, 0755); err != nil {
			return "", fmt.Errorf("server error: could not create storage directory (%w) — tell the user the image was generated but could not be saved due to a server storage problem", err)
		}
		if err := os.WriteFile(filepath.Join(absDir, "image.png"), imgData, 0644); err != nil {
			return "", fmt.Errorf("server error: could not write image to disk (%w) — tell the user the image was generated but could not be saved due to a server storage problem", err)
		}
		diskPath := filepath.Join(relDir, "image.png")

		att, err := tctx.Store.CreateAttachment(tctx.ToolCallID, tctx.ConversationID, "Generated image", "image.png", "image/png", diskPath)
		if err != nil {
			return "", fmt.Errorf("server error: could not record image in database (%w) — tell the user the image was generated but could not be saved due to a server error", err)
		}

		result := AttachmentResult{
			AttachmentID: att.ID,
			Title:        "Generated image",
			Filename:     "image.png",
			MimeType:     "image/png",
		}
		out, _ := json.Marshal(result)
		return string(out), nil
	}
}

// InitTools wires up executors for configured image tools and builds the list
// returned by GET /api/tools. Call once at server startup.
func InitTools(cfg *config.Config) {
	allTools = []ToolMeta{
		{"get_time", "Get current time", "Returns the current local date and time.", true, ""},
		{"roll_dice", "Roll dice", "Rolls dice using standard notation (e.g. 2d6, 1d20).", true, ""},
		{"pick_random", "Pick random", "Picks one item at random from a list of options.", true, ""},
		{"fetch_url", "Fetch URL", "Fetches a URL and returns its content as markdown, or raw HTML if source is true.", true, ""},
		{"create_document", "Create document", "Saves a file (report, script, notes, etc.) the user can download.", true, ""},
		{"wikipedia_search", "Wikipedia search", "Searches Wikipedia and returns matching article titles and snippets.", true, ""},
		{"wikipedia_get_page", "Wikipedia get page", "Fetches a Wikipedia article intro + TOC, or a specific section by name.", true, ""},
	}

	searxngConfigured := cfg.SearXNG.URL != ""
	searxngHint := ""
	if !searxngConfigured {
		searxngHint = "Add [searxng] url = \"http://…\" to lemon.toml to enable SearXNG search."
	}
	allTools = append(allTools, ToolMeta{"searxng", "SearXNG", "Searches the web via SearXNG and returns the top results.", searxngConfigured, searxngHint})

	sdxlConfigured := cfg.ComfyUI.URL != "" && cfg.ComfyUI.SDXLFile != ""
	sdxlHint := ""
	if !sdxlConfigured {
		sdxlHint = "Add [comfyui] url and sdxl_file to lemon.toml to enable SDXL image generation."
	} else {
		executors["generate_image_sdxl"] = makeImageExecutor(cfg.ComfyUI.URL, cfg.ComfyUI.SDXLFile, 30, 7.0)
	}
	allTools = append(allTools, ToolMeta{"generate_image_sdxl", "Generate image (SDXL)", toolRegistry["generate_image_sdxl"].Function.Description, sdxlConfigured, sdxlHint})

	fluxConfigured := cfg.ComfyUI.URL != "" && cfg.ComfyUI.FluxFile != ""
	fluxHint := ""
	if !fluxConfigured {
		fluxHint = "Add [comfyui] url and flux_file to lemon.toml to enable Flux Schnell image generation."
	} else {
		executors["generate_image_flux"] = makeImageExecutor(cfg.ComfyUI.URL, cfg.ComfyUI.FluxFile, 4, 1.0)
	}
	allTools = append(allTools, ToolMeta{"generate_image_flux", "Generate image (Flux Schnell)", toolRegistry["generate_image_flux"].Function.Description, fluxConfigured, fluxHint})
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, allTools)
}
