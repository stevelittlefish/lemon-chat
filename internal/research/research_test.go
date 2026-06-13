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
