package server

import "testing"

func TestNormalizeReportHTML(t *testing.T) {
	input := "```html\n<!DOCTYPE html><html><body><h1>Report</h1></body></html>\n```"
	got, complete := normalizeReportHTML(input)
	if !complete {
		t.Fatal("normalizeReportHTML reported an incomplete document")
	}
	want := "<!DOCTYPE html><html><body><h1>Report</h1></body></html>"
	if got != want {
		t.Fatalf("normalizeReportHTML = %q, want %q", got, want)
	}
}

func TestNormalizeReportHTMLIncompleteFragment(t *testing.T) {
	// A fragment is not a complete document, but the partial is still returned
	// (not discarded) so it can be saved for recovery.
	got, complete := normalizeReportHTML("<section>Not a document</section>")
	if complete {
		t.Fatal("normalizeReportHTML accepted an HTML fragment as complete")
	}
	if got == "" {
		t.Fatal("normalizeReportHTML discarded the partial fragment")
	}
}

func TestNormalizeReportHTMLTruncated(t *testing.T) {
	// A document cut off mid-stream (no </html>) is incomplete but recoverable.
	input := "<!DOCTYPE html><html><body><h1>Report</h1><p>Half a sen"
	got, complete := normalizeReportHTML(input)
	if complete {
		t.Fatal("normalizeReportHTML reported a truncated document as complete")
	}
	if got != input {
		t.Fatalf("normalizeReportHTML = %q, want the partial preserved", got)
	}
}
