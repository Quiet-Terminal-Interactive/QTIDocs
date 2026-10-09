package sanitize

import (
	"strings"
	"testing"
)

func TestHTML_StripsScriptTags(t *testing.T) {
	out := string(HTML([]byte(`<p>hello</p><script>alert(1)</script>`)))
	if strings.Contains(out, "script") {
		t.Errorf("expected <script> to be stripped, got %q", out)
	}
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("expected <p>hello</p> to survive, got %q", out)
	}
}

func TestHTML_StripsEventHandlers(t *testing.T) {
	out := string(HTML([]byte(`<p onclick="alert(1)">hi</p>`)))
	if strings.Contains(out, "onclick") {
		t.Errorf("expected onclick attribute to be stripped, got %q", out)
	}
}

func TestHTML_StripsJavascriptURLs(t *testing.T) {
	out := string(HTML([]byte(`<a href="javascript:alert(1)">click</a>`)))
	if strings.Contains(out, "javascript:") {
		t.Errorf("expected javascript: URL to be stripped, got %q", out)
	}
}

func TestHTML_StripsIframe(t *testing.T) {
	out := string(HTML([]byte(`<iframe src="https://evil.example"></iframe>`)))
	if strings.Contains(out, "iframe") {
		t.Errorf("expected <iframe> to be stripped, got %q", out)
	}
}

func TestHTML_StripsForm(t *testing.T) {
	out := string(HTML([]byte(`<form action="https://evil.example"><input type="text" name="x"></form>`)))
	if strings.Contains(out, "<form") {
		t.Errorf("expected <form> to be stripped, got %q", out)
	}
	if strings.Contains(out, `type="text"`) {
		t.Errorf("expected text input to be stripped, got %q", out)
	}
}

func TestHTML_AllowsAllowlistedElements(t *testing.T) {
	src := `<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table>`
	out := string(HTML([]byte(src)))
	for _, want := range []string{"<table>", "<th>A</th>", "<td>1</td>"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got %q", want, out)
		}
	}
}

func TestHTML_AllowsTaskListCheckbox(t *testing.T) {
	src := `<input type="checkbox" checked disabled>`
	out := string(HTML([]byte(src)))
	if !strings.Contains(out, `type="checkbox"`) {
		t.Errorf("expected checkbox input to survive, got %q", out)
	}
}

func TestHTML_RejectsNonAllowlistedURLScheme(t *testing.T) {
	out := string(HTML([]byte(`<a href="data:text/html,<script>alert(1)</script>">click</a>`)))
	if strings.Contains(out, "data:") {
		t.Errorf("expected data: href to be stripped, got %q", out)
	}
}

func TestHTML_AllowsMailtoHref(t *testing.T) {
	out := string(HTML([]byte(`<a href="mailto:a@b.com">mail</a>`)))
	if !strings.Contains(out, `href="mailto:a@b.com"`) {
		t.Errorf("expected mailto: href to survive, got %q", out)
	}
}

func TestHTML_AllowsImgAttrs(t *testing.T) {
	out := string(HTML([]byte(`<img src="/assets/x.png" alt="desc" width="10" height="10" title="t">`)))
	for _, want := range []string{`src="/assets/x.png"`, `alt="desc"`} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got %q", want, out)
		}
	}
}

func TestHTML_StripsDisallowedElement(t *testing.T) {
	out := string(HTML([]byte(`<video src="x.mp4"></video>text`)))
	if strings.Contains(out, "<video") {
		t.Errorf("expected <video> to be stripped, got %q", out)
	}
	if !strings.Contains(out, "text") {
		t.Errorf("expected text content to survive, got %q", out)
	}
}

func TestCSS_ValidSimpleStylesheet(t *testing.T) {
	err := CSS([]byte(`body { color: red; background-color: #fff; }`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCSS_RejectsImport(t *testing.T) {
	err := CSS([]byte(`@import url("https://evil.example/style.css");`))
	if err == nil {
		t.Fatal("expected error for @import, got nil")
	}
	if !strings.Contains(err.Error(), "@import") {
		t.Errorf("error = %v, want mention of @import", err)
	}
}

func TestCSS_RejectsExternalURL(t *testing.T) {
	err := CSS([]byte(`.x { background-image: url("https://evil.example/leak.png"); }`))
	if err == nil {
		t.Fatal("expected error for external url(), got nil")
	}
}

func TestCSS_RejectsProtocolRelativeURL(t *testing.T) {
	err := CSS([]byte(`.x { background-image: url("//evil.example/leak.png"); }`))
	if err == nil {
		t.Fatal("expected error for protocol-relative url(), got nil")
	}
}

func TestCSS_AllowsRelativeURL(t *testing.T) {
	err := CSS([]byte(`.x { background-image: url("/assets/bg.png"); }`))
	if err != nil {
		t.Errorf("unexpected error for relative url(): %v", err)
	}
}

func TestCSS_AllowsDataURI(t *testing.T) {
	err := CSS([]byte(`.x { background-image: url("data:image/png;base64,abcd"); }`))
	if err != nil {
		t.Errorf("unexpected error for data: url(): %v", err)
	}
}

func TestCSS_AllowsEmptyURL(t *testing.T) {
	err := CSS([]byte(`.x { background-image: url(); }`))
	if err != nil {
		t.Errorf("unexpected error for empty url(): %v", err)
	}
}

func TestCheckCSSURL_Malformed(t *testing.T) {
	err := checkCSSURL([]byte(`url(`))
	if err == nil {
		t.Fatal("expected error for malformed url() token, got nil")
	}
}

func TestAssetContentType(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"photo.png", "image/png"},
		{"photo.PNG", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"anim.gif", "image/gif"},
		{"pic.webp", "image/webp"},
		{"pic.avif", "image/avif"},
		{"favicon.ico", "image/x-icon"},
		{"logo.svg", "image/svg+xml"},
		{"doc.pdf", "application/pdf"},
		{"style.css", "text/css; charset=utf-8"},
		{"notes.txt", "text/plain; charset=utf-8"},
		{"data.json", "application/json"},
		{"font.woff", "font/woff"},
		{"font.woff2", "font/woff2"},
		{"font.ttf", "font/ttf"},
		{"font.otf", "font/otf"},
		{"script.js", "application/octet-stream"},
		{"page.html", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}
	for _, c := range cases {
		if got := AssetContentType(c.name); got != c.want {
			t.Errorf("AssetContentType(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestThemeAssetContentType_JS(t *testing.T) {
	got := ThemeAssetContentType("search.js")
	want := "text/javascript; charset=utf-8"
	if got != want {
		t.Errorf("ThemeAssetContentType(search.js) = %q, want %q", got, want)
	}
}

func TestThemeAssetContentType_JSCaseInsensitive(t *testing.T) {
	got := ThemeAssetContentType("search.JS")
	want := "text/javascript; charset=utf-8"
	if got != want {
		t.Errorf("ThemeAssetContentType(search.JS) = %q, want %q", got, want)
	}
}

func TestThemeAssetContentType_FallsThroughToAssetContentType(t *testing.T) {
	got := ThemeAssetContentType("theme.css")
	want := "text/css; charset=utf-8"
	if got != want {
		t.Errorf("ThemeAssetContentType(theme.css) = %q, want %q", got, want)
	}
}
