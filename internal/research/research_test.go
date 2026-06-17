package research

import (
	"strings"
	"testing"
)

func TestParseJSONStringArray(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"plain array", `["a", "b", "c"]`, []string{"a", "b", "c"}},
		{"code fence", "```json\n[\"a\", \"b\"]\n```", []string{"a", "b"}},
		{"surrounding prose", `Here are the queries: ["one", "two"]`, []string{"one", "two"}},
		{"truncated array", `["first query", "second query", "thi`, []string{"first query", "second query"}},
		{"echoed example then answer", `["query one", "query two", "query three"] is the format. ["real a", "real b"]`, []string{"real a", "real b"}},
		// A tool-eager model may wrap the queries in tool-call markup; the array
		// must still be harvested so the round actually searches.
		{"tool-call wrapped queries", `<|tool_call>call:google_search:search{queries: ["a", "b"]}<tool_call|>`, []string{"a", "b"}},
		{"no array", `no json here`, nil},
	}
	for _, c := range cases {
		got := parseJSONStringArray(c.in)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: got %v, want %v", c.name, got, c.want)
				break
			}
		}
	}
}

func TestParseJSONObject(t *testing.T) {
	var v struct {
		Summary string `json:"summary"`
	}
	if err := parseJSONObject("Sure! Here you go:\n```json\n{\"summary\": \"hi\"}\n```", &v); err != nil || v.Summary != "hi" {
		t.Errorf("fenced object: err=%v summary=%q", err, v.Summary)
	}
	if err := parseJSONObject(`prose {"summary": "x"} more prose`, &v); err != nil || v.Summary != "x" {
		t.Errorf("embedded object: err=%v summary=%q", err, v.Summary)
	}
	if err := parseJSONObject("nothing here", &v); err == nil {
		t.Error("expected error for non-JSON input")
	}
}

func TestParseOutlineJSON(t *testing.T) {
	// The deep-report outline is parsed as {"sections": [{title, intent}]} with
	// the same tolerant extraction as other object parsing (fences, prose).
	var v struct {
		Sections []reportSection `json:"sections"`
	}
	in := "Here is the outline:\n```json\n{\"sections\": [{\"title\": \"Background\", \"intent\": \"set the scene\"}, {\"title\": \"Analysis\", \"intent\": \"dig in\"}]}\n```"
	if err := parseJSONObject(in, &v); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(v.Sections) != 2 || v.Sections[0].Title != "Background" || v.Sections[1].Intent != "dig in" {
		t.Errorf("unexpected sections: %+v", v.Sections)
	}

	// Round-trips through marshalOutline cleanly.
	if got := marshalOutline(v.Sections); !strings.Contains(got, `"title":"Background"`) {
		t.Errorf("marshalOutline: %q", got)
	}
}

func TestStripThinking(t *testing.T) {
	if got := stripThinking("<think>internal monologue</think>YES — looks complete"); got != "YES — looks complete" {
		t.Errorf("closed block: got %q", got)
	}
	if got := stripThinking("answer first <thinking>unterminated reasoning"); got != "answer first" {
		t.Errorf("unterminated block: got %q", got)
	}
	if got := stripThinking("no tags at all"); got != "no tags at all" {
		t.Errorf("no tags: got %q", got)
	}
}

func TestStripToolCalls(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"asymmetric markers", `<|tool_call>call:search{queries: ["a"]}<tool_call|>`, ""},
		{"angle markers", "framing text <tool_call>foo</tool_call> more", "framing text  more"},
		{"unterminated open", "the framing <|tool_call|>call:search{...", "the framing"},
		{"no markup", "a clean framing paragraph", "a clean framing paragraph"},
	}
	for _, c := range cases {
		if got := stripToolCalls(c.in); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTruncateContent(t *testing.T) {
	para := strings.Repeat("x", 90)
	content := para + "\n\n" + para + "\n\n" + para
	got := truncateContent(content, 200)
	if len(got) > 200 {
		t.Errorf("truncated content too long: %d", len(got))
	}
	// Should cut at the paragraph break after 80% of the limit (index 182).
	if !strings.HasSuffix(got, para) || strings.HasSuffix(got, "\n\n") {
		t.Errorf("expected cut at paragraph boundary, got %q...", got[len(got)-10:])
	}
	if truncateContent("short", 200) != "short" {
		t.Error("short content should be unchanged")
	}
}

func TestIsLowQuality(t *testing.T) {
	if !isLowQuality("") {
		t.Error("empty summary should be low quality")
	}
	if !isLowQuality("The page is mostly Cookie Consent boilerplate") {
		t.Error("marker match should be case-insensitive")
	}
	if isLowQuality("Detailed statistics about the topic with sources") {
		t.Error("good summary flagged as low quality")
	}
}

func TestFormatFindingsUsesStableSourceIDs(t *testing.T) {
	st := State{Findings: []Finding{
		{URL: "https://example.test/a", Title: "First source", Summary: "Alpha facts."},
		{URL: "https://example.test/b", Title: "Second source", Summary: "Beta facts."},
	}}
	r := New(Config{}, st, nil, nil)

	got := r.formatFindings([]Finding{st.Findings[1]})
	for _, want := range []string{
		"[S2] Second source",
		"URL: https://example.test/b",
		"Summary: Beta facts.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("formatFindings missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "[S1]") {
		t.Errorf("formatFindings renumbered subset instead of preserving stable ID:\n%s", got)
	}
}

func TestCompositeReportIncludesSourceIDMap(t *testing.T) {
	st := State{
		Round:       1,
		Findings:    []Finding{{URL: "https://example.test/a", Title: "First source", Summary: "Alpha facts."}},
		QueriesUsed: []string{"alpha"},
		AnalyzedURLs: []AnalyzedURL{
			{URL: "https://example.test/a", Title: "First source"},
		},
	}
	r := New(Config{Model: "test-model"}, st, nil, nil)

	got := r.formatCompositeReport("The answer cites a source [S1].")
	for _, want := range []string{
		"The answer cites a source [S1].",
		"- [S1] [First source](https://example.test/a)",
		"[S1]: https://example.test/a",
		"**1. [S1] [First source](https://example.test/a)**",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("formatCompositeReport missing %q in:\n%s", want, got)
		}
	}
}

func TestNormalizeSourceCitations(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "comma list",
			in:   "Freshness matters [S13, S18].",
			want: "Freshness matters [S13] [S18].",
		},
		{
			name: "semicolon list",
			in:   "Freshness matters [S1; S2; S3].",
			want: "Freshness matters [S1] [S2] [S3].",
		},
		{
			name: "and list",
			in:   "Freshness matters [S1 and S2].",
			want: "Freshness matters [S1] [S2].",
		},
		{
			name: "single citation unchanged",
			in:   "Freshness matters [S1].",
			want: "Freshness matters [S1].",
		},
		{
			name: "ordinary bracket unchanged",
			in:   "Freshness matters [see appendix].",
			want: "Freshness matters [see appendix].",
		},
	}
	for _, c := range cases {
		if got := normalizeSourceCitations(c.in); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCompositeReportNormalizesGroupedSourceIDs(t *testing.T) {
	st := State{
		Findings: []Finding{
			{URL: "https://example.test/a", Title: "First source", Summary: "Alpha facts."},
			{URL: "https://example.test/b", Title: "Second source", Summary: "Beta facts."},
		},
	}
	r := New(Config{Model: "test-model"}, st, nil, nil)

	got := r.formatCompositeReport("The answer cites grouped sources [S1, S2].")
	if !strings.Contains(got, "The answer cites grouped sources [S1] [S2].") {
		t.Errorf("grouped citations were not normalized:\n%s", got)
	}
}

func TestUntrustedContextMessage(t *testing.T) {
	out := untrustedContextMessage("webpage", "content with "+guardOpen+" breakout attempt")
	if strings.Count(out, guardOpen) != 1 {
		t.Error("guard marker in content was not escaped")
	}
	if !strings.HasPrefix(out, "UNTRUSTED SOURCE DATA") {
		t.Error("missing hardcoded header")
	}
}

func TestModeDefaulting(t *testing.T) {
	// An empty mode defaults to research.
	r := New(Config{}, State{}, nil, nil)
	if r.cfg.Mode != ModeResearch || r.brainstorm() {
		t.Errorf("empty mode: got mode=%q brainstorm=%v, want research/false", r.cfg.Mode, r.brainstorm())
	}
	// An explicit brainstorm mode is preserved and reported.
	rb := New(Config{Mode: ModeBrainstorm}, State{}, nil, nil)
	if rb.cfg.Mode != ModeBrainstorm || !rb.brainstorm() {
		t.Errorf("brainstorm mode: got mode=%q brainstorm=%v, want brainstorm/true", rb.cfg.Mode, rb.brainstorm())
	}
}

func TestStateRoundTrip(t *testing.T) {
	st := State{
		Round:        3,
		EmptyRounds:  1,
		ElapsedMS:    1234,
		Category:     "howto",
		Plan:         "the plan",
		Report:       "the report",
		Findings:     []Finding{{URL: "https://a", Title: "A", Summary: "s"}},
		QueriesUsed:  []string{"q1", "q2"},
		AnalyzedURLs: []AnalyzedURL{{URL: "https://a", Title: "A"}},
	}
	findings, queries, urls := MarshalState(st)
	got := UnmarshalState(st.Round, st.EmptyRounds, st.ElapsedMS, &st.Category, &st.Plan, &st.Report, &findings, &queries, &urls)
	if got.Round != 3 || got.Category != "howto" || len(got.Findings) != 1 || len(got.QueriesUsed) != 2 || got.Findings[0].URL != "https://a" {
		t.Errorf("round trip mismatch: %+v", got)
	}

	// Nil pointers (fresh job) must yield a clean zero state.
	zero := UnmarshalState(0, 0, 0, nil, nil, nil, nil, nil, nil)
	if zero.Round != 0 || zero.Findings != nil || zero.Plan != "" {
		t.Errorf("zero state mismatch: %+v", zero)
	}
}
