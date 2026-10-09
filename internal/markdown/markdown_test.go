package markdown

import (
	"strings"
	"testing"
)

func TestRender_BasicHeading(t *testing.T) {
	r := New()
	out, err := r.Render([]byte("# Hello\n\nWorld\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), "<h1") {
		t.Errorf("output = %q, want an <h1>", out)
	}
	if !strings.Contains(string(out), "World") {
		t.Errorf("output = %q, want to contain World", out)
	}
}

func TestRender_GFMTable(t *testing.T) {
	r := New()
	src := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	out, err := r.Render([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"<table>", "<th>A</th>", "<td>1</td>"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output = %q, want to contain %q", out, want)
		}
	}
}

func TestRender_GFMTaskList(t *testing.T) {
	r := New()
	src := "- [x] done\n- [ ] todo\n"
	out, err := r.Render([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), `type="checkbox"`) {
		t.Errorf("output = %q, want a checkbox input", out)
	}
	if !strings.Contains(string(out), "checked") {
		t.Errorf("output = %q, want the done item checked", out)
	}
}

func TestRender_Strikethrough(t *testing.T) {
	r := New()
	out, err := r.Render([]byte("~~gone~~\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), "<del>") {
		t.Errorf("output = %q, want <del>", out)
	}
}

func TestRender_FencedCodeHighlighting(t *testing.T) {
	r := New()
	src := "```go\nfunc main() {}\n```\n"
	out, err := r.Render([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), wrapperClass) {
		t.Errorf("output = %q, want chroma wrapper class %q", out, wrapperClass)
	}
}

func TestRender_RawHTMLIsSanitized(t *testing.T) {
	r := New()
	src := "Hello <script>alert(1)</script> World\n"
	out, err := r.Render([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(out), "<script") {
		t.Errorf("output = %q, want <script> stripped by sanitize", out)
	}
	if !strings.Contains(string(out), "Hello") || !strings.Contains(string(out), "World") {
		t.Errorf("output = %q, want surrounding text to survive", out)
	}
}

func TestRender_RawHTMLAllowedElementSurvives(t *testing.T) {
	r := New()
	src := "<div class=\"note\">Note</div>\n"
	out, err := r.Render([]byte(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), `<div class="note">Note</div>`) {
		t.Errorf("output = %q, want allowed raw div to survive", out)
	}
}

func TestRender_AssetImageRewritten(t *testing.T) {
	r := New()
	out, err := r.Render([]byte("![alt](assets/logo.png)\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), `src="/assets/logo.png"`) {
		t.Errorf("output = %q, want src rewritten to /assets/logo.png", out)
	}
}

func TestRender_AssetLinkRewritten(t *testing.T) {
	r := New()
	out, err := r.Render([]byte("[doc](assets/file.pdf)\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), `href="/assets/file.pdf"`) {
		t.Errorf("output = %q, want href rewritten to /assets/file.pdf", out)
	}
}

func TestRender_NonAssetLinkUnchanged(t *testing.T) {
	r := New()
	out, err := r.Render([]byte("[other page](guides/advanced)\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), `href="guides/advanced"`) {
		t.Errorf("output = %q, want non-assets link left unrewritten", out)
	}
}

func TestRender_ExternalLinkUnchanged(t *testing.T) {
	r := New()
	out, err := r.Render([]byte("[ext](https://example.com/assets/x)\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), `href="https://example.com/assets/x"`) {
		t.Errorf("output = %q, want external link left unrewritten", out)
	}
}

func TestResolveAssetDestination(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"assets/logo.png", "/assets/logo.png"},
		{"guides/advanced", "guides/advanced"},
		{"https://example.com/assets/x", "https://example.com/assets/x"},
		{"", ""},
	}
	for _, c := range cases {
		got := string(resolveAssetDestination([]byte(c.in)))
		if got != c.want {
			t.Errorf("resolveAssetDestination(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRender_EmptyInput(t *testing.T) {
	r := New()
	out, err := r.Render([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "" {
		t.Errorf("output = %q, want empty", out)
	}
}
