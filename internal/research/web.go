package research

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/htmltext"
)

type webpage struct {
	Content string
	Title   string
	OGImage string
}

var (
	reWebTitle   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reWebOGImage = regexp.MustCompile(`(?is)<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']|<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)
)

// fetchWebpage fetches a URL and returns cleaned text content, preserving
// paragraph boundaries as blank lines (the truncation logic relies on them).
func (r *Researcher) fetchWebpage(ctx context.Context, pageURL string) (*webpage, error) {
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, "GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36 lemon-chat/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	raw := string(body)

	page := &webpage{}
	if m := reWebTitle.FindStringSubmatch(raw); m != nil {
		page.Title = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := reWebOGImage.FindStringSubmatch(raw); m != nil {
		if m[1] != "" {
			page.OGImage = m[1]
		} else {
			page.OGImage = m[2]
		}
	}

	page.Content = htmltext.StripStructured(raw)
	return page, nil
}

// truncateContent cuts content to maxChars, preferring the last paragraph
// break that falls after 80% of the limit.
func truncateContent(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}
	cut := content[:maxChars]
	if idx := strings.LastIndex(cut, "\n\n"); idx > maxChars*8/10 {
		return cut[:idx]
	}
	return cut
}

// isLowQuality reports whether an extraction summary indicates the finding
// should be discarded.
func isLowQuality(summary string) bool {
	if strings.TrimSpace(summary) == "" {
		return true
	}
	lower := strings.ToLower(summary)
	for _, marker := range lowQualityMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// fetchAndExtract fetches a URL, runs the goal-based LLM extraction over its
// content, and returns a finding — or nil if the page failed to fetch or the
// extraction was low quality.
func (r *Researcher) fetchAndExtract(ctx context.Context, pageURL, searchTitle string) *Finding {
	page, err := r.fetchWebpage(ctx, pageURL)
	if err != nil || page.Content == "" {
		return nil
	}

	title := page.Title
	if title == "" {
		title = searchTitle
	}

	content := truncateContent(page.Content, r.cfg.MaxContentChars)
	finding := r.ExtractText(ctx, pageURL, title, content)
	if finding != nil {
		finding.OGImage = page.OGImage
	}
	return finding
}

// ExtractText runs already-fetched text through the production goal-based
// extractor. Browser-imported content uses this entry point so it receives the
// same prompt-injection guard and quality filtering as ordinary webpages.
func (r *Researcher) ExtractText(ctx context.Context, pageURL, title, content string) *Finding {
	content = truncateContent(content, r.cfg.MaxContentChars)

	// Two-message structure: extraction prompt, then the page content wrapped
	// in prompt-injection guard markers.
	msgs := []chatMsg{
		{Role: "user", Content: fmt.Sprintf(extractorSystem, r.cfg.Query)},
		{Role: "user", Content: untrustedContextMessage("webpage", content)},
	}
	out, err := r.llmCallWorker(ctx, msgs, 0.2, 2048, extractionTimeout)
	if err != nil {
		return nil
	}

	finding := &Finding{URL: pageURL, Title: title}
	var extracted struct {
		Rational string `json:"rational"`
		Evidence string `json:"evidence"`
		Summary  string `json:"summary"`
	}
	if parseJSONObject(out, &extracted) == nil && extracted.Summary != "" {
		finding.Rational = extracted.Rational
		finding.Evidence = extracted.Evidence
		finding.Summary = extracted.Summary
	} else {
		// Fallback: raw response as evidence, first 500 chars as summary.
		finding.Evidence = out
		if len(out) > 500 {
			finding.Summary = out[:500]
		} else {
			finding.Summary = out
		}
	}

	if isLowQuality(finding.Summary) {
		return nil
	}
	return finding
}
