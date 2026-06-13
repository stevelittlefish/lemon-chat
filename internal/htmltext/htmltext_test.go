package htmltext

import "testing"

func TestStripCollapsesWhitespace(t *testing.T) {
	in := "<p>Hello   <b>world</b></p>\n\n<p>line\ntwo</p>"
	got := Strip(in)
	want := "Hello world line two"
	if got != want {
		t.Errorf("Strip = %q, want %q", got, want)
	}
}

func TestStripRemovesScriptAndStyle(t *testing.T) {
	in := `<style>.a{color:red}</style><script>alert(1)</script>Visible`
	if got := Strip(in); got != "Visible" {
		t.Errorf("Strip = %q, want %q", got, "Visible")
	}
}

func TestStripDoesNotUnescapeEntities(t *testing.T) {
	// Snippet stripping leaves entities as-is (matches prior behaviour).
	if got := Strip("Tom &amp; Jerry"); got != "Tom &amp; Jerry" {
		t.Errorf("Strip = %q, want %q", got, "Tom &amp; Jerry")
	}
}

func TestStripStructuredPreservesParagraphs(t *testing.T) {
	in := "<h1>Title</h1><p>First para.</p><p>Second para.</p>"
	got := StripStructured(in)
	want := "Title\n\nFirst para.\n\nSecond para."
	if got != want {
		t.Errorf("StripStructured = %q, want %q", got, want)
	}
}

func TestStripStructuredUnescapesEntities(t *testing.T) {
	if got := StripStructured("<p>Tom &amp; Jerry</p>"); got != "Tom & Jerry" {
		t.Errorf("StripStructured = %q, want %q", got, "Tom & Jerry")
	}
}

func TestStripStructuredDropsChromeAndCollapsesBlankRuns(t *testing.T) {
	in := "<nav>menu menu</nav><p>Body</p>\n\n\n\n<footer>foot</footer>"
	got := StripStructured(in)
	want := "Body"
	if got != want {
		t.Errorf("StripStructured = %q, want %q", got, want)
	}
}
