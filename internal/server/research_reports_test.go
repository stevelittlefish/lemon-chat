package server

import "testing"

func TestNormalizeReportHTML(t *testing.T) {
	input := "```html\n<!DOCTYPE html><html><body><h1>Report</h1></body></html>\n```"
	got, err := normalizeReportHTML(input)
	if err != nil {
		t.Fatalf("normalizeReportHTML returned error: %v", err)
	}
	want := "<!DOCTYPE html><html><body><h1>Report</h1></body></html>"
	if got != want {
		t.Fatalf("normalizeReportHTML = %q, want %q", got, want)
	}
}

func TestNormalizeReportHTMLRejectsFragment(t *testing.T) {
	if _, err := normalizeReportHTML("<section>Not a document</section>"); err == nil {
		t.Fatal("normalizeReportHTML accepted an HTML fragment")
	}
}
