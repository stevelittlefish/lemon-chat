// Package htmltext converts raw HTML into plain text. It offers a fast
// snippet-oriented strip and a structure-preserving variant for long-form
// page content.
package htmltext

import (
	"html"
	"regexp"
	"strings"
)

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reChrome = regexp.MustCompile(`(?is)<(nav|header|footer|aside)[^>]*>.*?</(nav|header|footer|aside)>`)
	reBlock  = regexp.MustCompile(`(?i)</?(p|div|section|article|h[1-6]|li|ul|ol|tr|table|blockquote|pre)[^>]*>|<br[^>]*>`)
	reTag    = regexp.MustCompile(`<[^>]+>`)
	reSpaces = regexp.MustCompile(`[ \t]+`)
	reBlank  = regexp.MustCompile(`\n\s*\n\s*\n+`)
	reSpace  = regexp.MustCompile(`\s+`)
)

// Strip removes scripts, styles, and all tags, collapsing every run of
// whitespace to a single space. Suitable for short snippets and titles where
// layout does not matter.
func Strip(s string) string {
	s = reScript.ReplaceAllString(s, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = reTag.ReplaceAllString(s, " ")
	s = reSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// StripStructured removes scripts, styles, and navigational chrome, converts
// block-level elements to blank-line paragraph breaks, unescapes HTML
// entities, and collapses excess blank lines. Paragraph boundaries are
// preserved as blank lines so length-based truncation can cut on them.
func StripStructured(s string) string {
	s = reScript.ReplaceAllString(s, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = reChrome.ReplaceAllString(s, " ")
	s = reBlock.ReplaceAllString(s, "\n\n")
	s = reTag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = reSpaces.ReplaceAllString(s, " ")
	s = reBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
