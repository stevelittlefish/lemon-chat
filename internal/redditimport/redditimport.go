// Package redditimport defines the browser-extension interchange format used
// to bring user-assisted Reddit captures into research jobs.
package redditimport

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const SchemaVersion = 1

const (
	MaxRequestName   = 40
	MaxPages         = 25
	MaxComments      = 2000
	MaxBodyChars     = 100_000
	MaxTotalChars    = 2_000_000
	MaxTitleChars    = 1000
	MaxAuthorChars   = 100
	MaxWarningChars  = 2000
	MaxDepth         = 50
	MaxPageSeconds   = 300
	MaxExpandActions = 500
)

var (
	redditPostPath  = regexp.MustCompile(`^/(?:r/[^/]+/)?comments/([a-z0-9]+)(?:/[^/]*)?(?:/([a-z0-9]+))?/?$`)
	redditShortPath = regexp.MustCompile(`^/([a-z0-9]+)/?$`)
)

// Request is copied from lemon-chat into the browser extension.
type Request struct {
	Version   int             `json:"version"`
	RequestID string          `json:"request_id"`
	Name      string          `json:"name,omitempty"`
	Pages     []RequestedPage `json:"pages"`
	Limits    CaptureLimits   `json:"limits"`
}

type RequestedPage struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

type CaptureLimits struct {
	MaxComments      int `json:"max_comments"`
	MaxSeconds       int `json:"max_seconds_per_page"`
	MaxExpandActions int `json:"max_expand_actions"`
}

// Response is copied from the browser extension back into lemon-chat.
type Response struct {
	Version    int            `json:"version"`
	RequestID  string         `json:"request_id"`
	CapturedAt string         `json:"captured_at,omitempty"`
	Pages      []CapturedPage `json:"pages"`
}

type CapturedPage struct {
	RequestedURL string            `json:"requested_url"`
	CanonicalURL string            `json:"canonical_url"`
	Title        string            `json:"title,omitempty"`
	Subreddit    string            `json:"subreddit,omitempty"`
	Post         CapturedPost      `json:"post"`
	Comments     []CapturedComment `json:"comments,omitempty"`
	Complete     bool              `json:"complete"`
	Warnings     []string          `json:"warnings,omitempty"`
	Failure      string            `json:"failure,omitempty"`
}

type CapturedPost struct {
	Author    string `json:"author,omitempty"`
	Body      string `json:"body,omitempty"`
	Permalink string `json:"permalink,omitempty"`
	Score     *int   `json:"score,omitempty"`
}

type CapturedComment struct {
	Author    string `json:"author,omitempty"`
	Body      string `json:"body,omitempty"`
	Permalink string `json:"permalink,omitempty"`
	Score     *int   `json:"score,omitempty"`
	Depth     int    `json:"depth"`
}

// NormalizedPage is validated, deduplicated content ready to be converted to
// the untrusted text passed through the research extractor.
type NormalizedPage struct {
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Complete bool     `json:"complete"`
	Warnings []string `json:"warnings,omitempty"`
	Failure  string   `json:"failure,omitempty"`
	Comments int      `json:"comments"`
}

// CanonicalizeURL returns a stable Reddit permalink and thread identity.
// Comment permalinks retain their comment ID, while ThreadURL identifies the
// containing post for grouping and repeat-request prevention.
func CanonicalizeURL(raw string) (canonical string, threadURL string, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", "", fmt.Errorf("invalid absolute URL %q", raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	cleanPath := strings.ToLower(path.Clean("/" + strings.TrimPrefix(u.EscapedPath(), "/")))
	if host == "redd.it" {
		m := redditShortPath.FindStringSubmatch(cleanPath)
		if m == nil {
			return "", "", fmt.Errorf("unsupported redd.it URL %q", raw)
		}
		threadURL = "https://www.reddit.com/comments/" + m[1] + "/"
		return threadURL, threadURL, nil
	}
	if host != "reddit.com" && host != "www.reddit.com" && host != "old.reddit.com" && host != "new.reddit.com" {
		return "", "", fmt.Errorf("not a Reddit URL %q", raw)
	}
	m := redditPostPath.FindStringSubmatch(cleanPath)
	if m == nil {
		return "", "", fmt.Errorf("not a Reddit post or comment URL %q", raw)
	}
	threadURL = "https://www.reddit.com/comments/" + m[1] + "/"
	canonical = threadURL
	if m[2] != "" {
		canonical += m[2] + "/"
	}
	return canonical, threadURL, nil
}

// NewRequest canonicalizes and groups URLs by thread while preserving the
// first search-result title and a specifically matched comment permalink.
func NewRequest(requestID string, pages []RequestedPage, limits CaptureLimits) (Request, error) {
	if strings.TrimSpace(requestID) == "" {
		return Request{}, errors.New("request ID is required")
	}
	if len(pages) == 0 {
		return Request{}, errors.New("at least one page is required")
	}
	if len(pages) > MaxPages {
		return Request{}, fmt.Errorf("too many pages: %d exceeds %d", len(pages), MaxPages)
	}
	seen := make(map[string]bool)
	out := make([]RequestedPage, 0, len(pages))
	for i, page := range pages {
		canonical, threadURL, err := CanonicalizeURL(page.URL)
		if err != nil {
			return Request{}, fmt.Errorf("page %d: %w", i+1, err)
		}
		key := threadURL
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, RequestedPage{URL: canonical, Title: clip(strings.TrimSpace(page.Title), MaxTitleChars)})
	}
	if limits.MaxComments <= 0 || limits.MaxComments > MaxComments {
		limits.MaxComments = MaxComments
	}
	if limits.MaxSeconds <= 0 || limits.MaxSeconds > MaxPageSeconds {
		limits.MaxSeconds = MaxPageSeconds
	}
	if limits.MaxExpandActions <= 0 || limits.MaxExpandActions > MaxExpandActions {
		limits.MaxExpandActions = MaxExpandActions
	}
	return Request{Version: SchemaVersion, RequestID: requestID, Pages: out, Limits: limits}, nil
}

// SetRequestName adds the human-readable filename stem used by the browser
// extension. RequestID remains the opaque value used for response matching.
func SetRequestName(req *Request, name string) {
	name = strings.Trim(strings.ToLower(strings.TrimSpace(name)), "_")
	if len(name) > MaxRequestName {
		name = strings.TrimRight(name[:MaxRequestName], "_")
	}
	valid := name != ""
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			valid = false
			break
		}
	}
	if valid {
		req.Name = name
	}
}

// ValidateAndNormalize validates a response against its request and returns
// deterministic, deduplicated text. Invalid input is rejected as a whole so a
// research job cannot resume from an ambiguous partial submission.
func ValidateAndNormalize(req Request, resp Response) ([]NormalizedPage, error) {
	if req.Version != SchemaVersion || resp.Version != SchemaVersion {
		return nil, fmt.Errorf("unsupported schema version: request=%d response=%d", req.Version, resp.Version)
	}
	if req.RequestID == "" || resp.RequestID != req.RequestID {
		return nil, errors.New("response request ID does not match")
	}
	if len(resp.Pages) == 0 {
		return nil, errors.New("response contains no pages")
	}
	if len(resp.Pages) != len(req.Pages) || len(resp.Pages) > MaxPages {
		return nil, errors.New("response must contain exactly one result for every requested page")
	}
	if resp.CapturedAt != "" {
		if _, err := time.Parse(time.RFC3339, resp.CapturedAt); err != nil {
			return nil, errors.New("captured_at must be RFC3339")
		}
	}

	requested := make(map[string]RequestedPage, len(req.Pages))
	for _, page := range req.Pages {
		_, threadURL, err := CanonicalizeURL(page.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid request page: %w", err)
		}
		requested[threadURL] = page
	}

	seenPages := make(map[string]bool)
	var totalChars int
	out := make([]NormalizedPage, 0, len(resp.Pages))
	for i, page := range resp.Pages {
		normalized, chars, err := normalizePage(page, requested, req.Limits.MaxComments)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", i+1, err)
		}
		_, threadURL, _ := CanonicalizeURL(normalized.URL)
		if seenPages[threadURL] {
			return nil, fmt.Errorf("page %d duplicates thread %s", i+1, threadURL)
		}
		seenPages[threadURL] = true
		totalChars += chars
		if totalChars > MaxTotalChars {
			return nil, fmt.Errorf("import text exceeds %d characters", MaxTotalChars)
		}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizePage(page CapturedPage, requested map[string]RequestedPage, maxComments int) (NormalizedPage, int, error) {
	requestedCanonical, requestedThread, err := CanonicalizeURL(page.RequestedURL)
	if err != nil {
		return NormalizedPage{}, 0, fmt.Errorf("invalid requested_url: %w", err)
	}
	req, ok := requested[requestedThread]
	if !ok {
		return NormalizedPage{}, 0, fmt.Errorf("URL was not requested: %s", requestedThread)
	}
	canonical := requestedCanonical
	if page.CanonicalURL != "" {
		canonical, _, err = CanonicalizeURL(page.CanonicalURL)
		if err != nil {
			return NormalizedPage{}, 0, fmt.Errorf("invalid canonical_url: %w", err)
		}
		_, canonicalThread, _ := CanonicalizeURL(canonical)
		if canonicalThread != requestedThread {
			return NormalizedPage{}, 0, errors.New("canonical_url does not match requested thread")
		}
	}
	if page.Failure != "" && (page.Post.Body != "" || len(page.Comments) > 0) {
		return NormalizedPage{}, 0, errors.New("failed page must not contain captured content")
	}
	if page.Failure == "" && strings.TrimSpace(page.Post.Body) == "" && len(page.Comments) == 0 {
		return NormalizedPage{}, 0, errors.New("page contains neither captured content nor a failure reason")
	}
	if maxComments <= 0 || maxComments > MaxComments {
		maxComments = MaxComments
	}
	if len(page.Comments) > maxComments {
		return NormalizedPage{}, 0, fmt.Errorf("too many comments: %d exceeds %d", len(page.Comments), maxComments)
	}

	title := strings.TrimSpace(page.Title)
	if title == "" {
		title = req.Title
	}
	if len(title) > MaxTitleChars {
		return NormalizedPage{}, 0, fmt.Errorf("title exceeds %d characters", MaxTitleChars)
	}
	if len(page.Post.Author) > MaxAuthorChars {
		return NormalizedPage{}, 0, errors.New("post author is too long")
	}
	if len(page.Post.Body) > MaxBodyChars {
		return NormalizedPage{}, 0, fmt.Errorf("post body exceeds %d characters", MaxBodyChars)
	}
	if page.Post.Permalink != "" {
		_, permalinkThread, permalinkErr := CanonicalizeURL(page.Post.Permalink)
		if permalinkErr != nil || permalinkThread != requestedThread {
			return NormalizedPage{}, 0, errors.New("post permalink does not match requested thread")
		}
	}

	warnings := make([]string, 0, len(page.Warnings))
	for _, warning := range page.Warnings {
		warning = strings.TrimSpace(warning)
		if len(warning) > MaxWarningChars {
			return NormalizedPage{}, 0, fmt.Errorf("warning exceeds %d characters", MaxWarningChars)
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}

	comments := make([]CapturedComment, 0, len(page.Comments))
	seenComments := make(map[string]bool)
	for j, comment := range page.Comments {
		if comment.Depth < 0 || comment.Depth > MaxDepth {
			return NormalizedPage{}, 0, fmt.Errorf("comment %d has invalid depth %d", j+1, comment.Depth)
		}
		if len(comment.Author) > MaxAuthorChars || len(comment.Body) > MaxBodyChars {
			return NormalizedPage{}, 0, fmt.Errorf("comment %d exceeds field limits", j+1)
		}
		comment.Body = strings.TrimSpace(comment.Body)
		if comment.Body == "" {
			continue
		}
		if comment.Permalink != "" {
			_, permalinkThread, permalinkErr := CanonicalizeURL(comment.Permalink)
			if permalinkErr != nil || permalinkThread != requestedThread {
				return NormalizedPage{}, 0, fmt.Errorf("comment %d permalink does not match requested thread", j+1)
			}
		}
		key := commentKey(comment)
		if seenComments[key] {
			continue
		}
		seenComments[key] = true
		comments = append(comments, comment)
	}

	content := formatPage(canonical, title, page, comments, warnings)
	return NormalizedPage{
		URL: canonical, Title: title, Content: content, Complete: page.Complete,
		Warnings: warnings, Failure: strings.TrimSpace(page.Failure), Comments: len(comments),
	}, len(content), nil
}

func commentKey(comment CapturedComment) string {
	if comment.Permalink != "" {
		if canonical, _, err := CanonicalizeURL(comment.Permalink); err == nil {
			return canonical
		}
	}
	return strconv.Itoa(comment.Depth) + "\x00" + strings.TrimSpace(comment.Author) + "\x00" + strings.TrimSpace(comment.Body)
}

func formatPage(canonical, title string, page CapturedPage, comments []CapturedComment, warnings []string) string {
	var b strings.Builder
	b.WriteString("# Reddit thread\n\n")
	writeField(&b, "Title", title)
	writeField(&b, "Subreddit", strings.TrimSpace(page.Subreddit))
	writeField(&b, "URL", canonical)
	writeField(&b, "Capture completeness", map[bool]string{true: "complete", false: "partial"}[page.Complete])
	if page.Failure != "" {
		writeField(&b, "Capture failure", strings.TrimSpace(page.Failure))
	}
	if len(warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range warnings {
			b.WriteString("- " + singleLine(warning) + "\n")
		}
		b.WriteByte('\n')
	}
	if page.Post.Body != "" || page.Post.Author != "" {
		b.WriteString("## Post\n\n")
		writeField(&b, "Author", strings.TrimSpace(page.Post.Author))
		writeScore(&b, page.Post.Score)
		b.WriteString(strings.TrimSpace(page.Post.Body) + "\n\n")
	}
	if len(comments) > 0 {
		b.WriteString("## Comments\n\n")
		for i, comment := range comments {
			fmt.Fprintf(&b, "### Comment %d (depth %d)\n\n", i+1, comment.Depth)
			writeField(&b, "Author", strings.TrimSpace(comment.Author))
			writeScore(&b, comment.Score)
			writeField(&b, "Permalink", strings.TrimSpace(comment.Permalink))
			b.WriteString(comment.Body + "\n\n")
		}
	}
	return strings.TrimSpace(b.String()) + "\n"
}

func writeField(b *strings.Builder, name, value string) {
	if value != "" {
		fmt.Fprintf(b, "%s: %s\n\n", name, singleLine(value))
	}
}

func writeScore(b *strings.Builder, score *int) {
	if score != nil {
		fmt.Fprintf(b, "Score: %d\n\n", *score)
	}
}

func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// SortPages makes diagnostic and fixture output stable without changing the
// extension's captured comment order.
func SortPages(pages []NormalizedPage) {
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].URL < pages[j].URL })
}
