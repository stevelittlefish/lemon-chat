package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/htmltext"
	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/searx"
)

const webToolRequestTimeout = 30 * time.Second

func executeFetchURL(argsJSON string, tctx ToolContext) (string, error) {
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

	fetchCtx, cancel := context.WithTimeout(context.Background(), webToolRequestTimeout)
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

	stripped := htmltext.Strip(content)
	const maxStripped = 50_000
	if len(stripped) > maxStripped {
		stripped = stripped[:maxStripped] + "\n\n[truncated — page content exceeded 50,000 characters]"
	}

	if tctx.ModelServer == nil {
		return stripped, nil
	}
	return summariseHTML(stripped, tctx.ModelName, tctx.ModelServer, tctx.ResponseTimeout)
}

func executeSearXNG(argsJSON string, tctx ToolContext) (string, error) {
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
		n = 10
	}
	if n > 40 {
		n = 40
	}
	page := args.Page
	if page <= 0 {
		page = 1
	}

	log.Printf("Searching web query=%q max_results=%d page=%d", args.Query, n, page)

	results, err := searx.Search(context.Background(), tctx.SearXNGURL, args.Query, page)
	if err != nil {
		return "", fmt.Errorf("%w — tell the user the web search service is currently unavailable; they may want to check that SearXNG is running at %q and has the JSON output format enabled", err, tctx.SearXNGURL)
	}

	if len(results) == 0 {
		return "No results found for: " + args.Query, nil
	}

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
}

func executeWikipediaSearch(argsJSON string, _ ToolContext) (string, error) {
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

	fetchCtx, cancel := context.WithTimeout(context.Background(), webToolRequestTimeout)
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
		snippet := strings.TrimSpace(htmltext.Strip(r.Snippet))
		fmt.Fprintf(&sb, "%d. **%s** — %s\n", i+1, r.Title, snippet)
	}
	return strings.TrimSpace(sb.String()), nil
}

func executeWikipediaGetPage(argsJSON string, _ ToolContext) (string, error) {
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

	fetchCtx, cancel := context.WithTimeout(context.Background(), webToolRequestTimeout)
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
				title := strings.TrimSpace(htmltext.Strip(s.Line))
				fmt.Fprintf(&sb, "%s. %s\n", s.Number, title)
			}
		}
		return strings.TrimSpace(sb.String()), nil
	}

	// Section requested — find its index.
	sectionLower := strings.ToLower(args.Section)
	sectionIndex := ""
	for _, s := range tocData.Parse.Sections {
		title := strings.TrimSpace(htmltext.Strip(s.Line))
		if strings.ToLower(title) == sectionLower {
			sectionIndex = s.Index
			break
		}
	}
	if sectionIndex == "" {
		var names []string
		for _, s := range tocData.Parse.Sections {
			names = append(names, strings.TrimSpace(htmltext.Strip(s.Line)))
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
	return strings.TrimSpace(htmltext.Strip(html)), nil
}

func summariseHTML(text, modelName string, srv *config.ModelServer, timeout time.Duration) (string, error) {
	msgs := []llm.Message{
		{Role: "system", Content: "Convert the following web page text to clean, well-structured markdown. Preserve headings, lists, links, and code blocks. Remove navigation, footer, and boilerplate text."},
		{Role: "user", Content: text},
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	content, err := llm.ChatComplete(ctx, http.DefaultClient, srv.APIBase+"/chat/completions", srv.APIKey, modelName, msgs, nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(content), nil
}
