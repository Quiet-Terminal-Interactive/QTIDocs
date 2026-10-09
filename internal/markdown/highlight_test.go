package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func TestChromaCSS_Succeeds(t *testing.T) {
	css, err := ChromaCSS()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(css)
	if !strings.Contains(s, "@media (prefers-color-scheme: dark)") {
		t.Errorf("css missing dark media query: %q", truncate(s))
	}
	if !strings.Contains(s, `:root:not([data-theme="light"])`) {
		t.Errorf("css missing light-not selector: %q", truncate(s))
	}
	if !strings.Contains(s, `:root[data-theme="dark"]`) {
		t.Errorf("css missing dark-theme selector: %q", truncate(s))
	}
	if strings.Contains(s, "/* Background */") {
		t.Errorf("css should have Background rule stripped: %q", truncate(s))
	}
}

func TestFixWrapperRule_DropsBackgroundRule(t *testing.T) {
	in := []byte("/* Background */ .chroma-chroma { background-color: #fff; color: #000 }\n.other { color: red }\n")
	out := fixWrapperRule(in)
	if bytes.Contains(out, []byte("Background")) {
		t.Errorf("expected Background rule dropped, got %q", out)
	}
	if !bytes.Contains(out, []byte(".other { color: red }")) {
		t.Errorf("expected unrelated line preserved, got %q", out)
	}
}

func TestFixWrapperRule_StripsBackgroundColorFromPreWrapper(t *testing.T) {
	in := []byte("/* PreWrapper */ .chroma-chroma { background-color: #111; color: #eee }\n")
	out := fixWrapperRule(in)
	if bytes.Contains(out, []byte("background-color")) {
		t.Errorf("expected background-color stripped, got %q", out)
	}
	if !bytes.Contains(out, []byte("color: #eee")) {
		t.Errorf("expected color declaration kept, got %q", out)
	}
}

func TestFixWrapperRule_LeavesOtherLinesAlone(t *testing.T) {
	in := []byte(".chroma-foo { color: blue }\n")
	out := fixWrapperRule(in)
	if !bytes.Equal(in, out) {
		t.Errorf("fixWrapperRule(%q) = %q, want unchanged", in, out)
	}
}

func TestTokenClasses_ExtractsClasses(t *testing.T) {
	css := []byte(".chroma-chroma .chroma-kw { color: red }\n.chroma-chroma .chroma-str { color: green }\n.unrelated { color: blue }\n")
	got := tokenClasses(css)
	want := map[string]bool{"chroma-kw": true, "chroma-str": true}
	if len(got) != len(want) {
		t.Fatalf("tokenClasses() = %v, want %v", got, want)
	}
	for k := range want {
		if !got[k] {
			t.Errorf("tokenClasses() missing %q", k)
		}
	}
}

func TestTokenClasses_EmptyInput(t *testing.T) {
	got := tokenClasses([]byte(""))
	if len(got) != 0 {
		t.Errorf("tokenClasses(\"\") = %v, want empty", got)
	}
}

func TestFillMissingTokenColors_AddsFallbackForMissingClass(t *testing.T) {
	light := []byte(".chroma-chroma .chroma-kw { color: red }\n.chroma-chroma .chroma-other { color: cyan }\n")
	dark := []byte("/* PreWrapper */ .chroma-chroma { color: #eee; }\n.chroma-chroma .chroma-kw { color: pink }\n")

	out := fillMissingTokenColors(light, dark)
	if !bytes.Contains(out, []byte(".chroma-chroma .chroma-other { color: #eee; }")) {
		t.Errorf("expected fallback rule for chroma-other, got %q", out)
	}
	count := bytes.Count(out, []byte(".chroma-chroma .chroma-kw"))
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of chroma-kw rule, got %d in %q", count, out)
	}
}

func TestFillMissingTokenColors_NoMissingClassesReturnsUnchanged(t *testing.T) {
	light := []byte(".chroma-chroma .chroma-kw { color: red }\n")
	dark := []byte("/* PreWrapper */ .chroma-chroma { color: #eee; }\n.chroma-chroma .chroma-kw { color: pink }\n")

	out := fillMissingTokenColors(light, dark)
	if !bytes.Equal(out, dark) {
		t.Errorf("fillMissingTokenColors() = %q, want unchanged dark %q", out, dark)
	}
}

func TestFillMissingTokenColors_NoBaseColorReturnsDarkUnchanged(t *testing.T) {
	light := []byte(".chroma-chroma .chroma-kw { color: red }\n")
	dark := []byte(".chroma-chroma .chroma-other { color: blue }\n")

	out := fillMissingTokenColors(light, dark)
	if !bytes.Equal(out, dark) {
		t.Errorf("fillMissingTokenColors() = %q, want unchanged dark %q (no base color to fall back to)", out, dark)
	}
}

func TestScopeChromaCSS_PrefixesOnlyFirstOccurrence(t *testing.T) {
	in := []byte(".chroma-chroma .chroma-err { color: red }\n")
	out := scopeChromaCSS(in, `:root[data-theme="dark"]`)
	want := `:root[data-theme="dark"] .chroma-chroma .chroma-err { color: red }`
	if string(out) != want+"\n" {
		t.Errorf("scopeChromaCSS() = %q, want %q", out, want)
	}
}

func TestScopeChromaCSS_MultipleLines(t *testing.T) {
	in := []byte(".chroma-chroma { color: red }\n.chroma-chroma .chroma-kw { color: blue }\n")
	out := scopeChromaCSS(in, "PREFIX")
	lines := bytes.Split(out, []byte("\n"))
	if !bytes.HasPrefix(lines[0], []byte("PREFIX .chroma-chroma")) {
		t.Errorf("line 0 = %q, want PREFIX-scoped", lines[0])
	}
	if !bytes.HasPrefix(lines[1], []byte("PREFIX .chroma-chroma .chroma-kw")) {
		t.Errorf("line 1 = %q, want PREFIX-scoped", lines[1])
	}
}

func TestScopeChromaCSS_LineWithoutChromaClassUnaffected(t *testing.T) {
	in := []byte("body { margin: 0 }\n")
	out := scopeChromaCSS(in, "PREFIX")
	if !bytes.Equal(in, out) {
		t.Errorf("scopeChromaCSS() = %q, want unchanged %q", out, in)
	}
}

func truncate(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
