package server

import "testing"

func TestNormalizeRemixHTML(t *testing.T) {
	input := "```html\n<!DOCTYPE html><html><body><h1>Report</h1></body></html>\n```"
	got, err := normalizeRemixHTML(input)
	if err != nil {
		t.Fatalf("normalizeRemixHTML returned error: %v", err)
	}
	want := "<!DOCTYPE html><html><body><h1>Report</h1></body></html>"
	if got != want {
		t.Fatalf("normalizeRemixHTML = %q, want %q", got, want)
	}
}

func TestNormalizeRemixHTMLRejectsFragment(t *testing.T) {
	if _, err := normalizeRemixHTML("<section>Not a document</section>"); err == nil {
		t.Fatal("normalizeRemixHTML accepted an HTML fragment")
	}
}
