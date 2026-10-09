package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}

func TestBuild_SinglePage(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\n# Welcome\n")

	if err := Build(src, out, "Acme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(html, "Welcome") {
		t.Errorf("output missing content: %s", truncate(html))
	}
	if !strings.Contains(html, "Acme | QTIDocs") {
		t.Errorf("output missing site title: %s", truncate(html))
	}
}

func TestBuild_NestedPages(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nhome\n")
	writeFile(t, filepath.Join(src, "guides", "index.md"), "---\ntitle: Guides\n---\nguides home\n")
	writeFile(t, filepath.Join(src, "guides", "advanced.md"), "---\ntitle: Advanced\n---\nadvanced\n")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		t.Errorf("expected root index.html: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "guides", "index.html")); err != nil {
		t.Errorf("expected guides/index.html: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "guides", "advanced", "index.html")); err != nil {
		t.Errorf("expected guides/advanced/index.html: %v", err)
	}

	navHTML := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(navHTML, "Guides") {
		t.Errorf("expected nav to include Guides: %s", truncate(navHTML))
	}
}

func TestBuild_ZeroPagesWritesPlaceholder(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(html, "No pages yet") {
		t.Errorf("expected placeholder content: %s", truncate(html))
	}
}

func TestBuild_MissingTitleFails(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "# No frontmatter\n")

	err := Build(src, out, "")
	if err == nil {
		t.Fatal("expected error for missing frontmatter title, got nil")
	}
	if !strings.Contains(err.Error(), "index.md") {
		t.Errorf("error should name the offending file, got: %v", err)
	}
}

func TestBuild_WritesThemeAssets(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{"theme.css", "search.js", "theme.js", "analytics.js", "stats.js", "search-index.json"} {
		if _, err := os.Stat(filepath.Join(out, "_qtidocs", name)); err != nil {
			t.Errorf("expected _qtidocs/%s: %v", name, err)
		}
	}
}

func TestBuild_SearchIndexContainsPages(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\ndescription: desc\n---\nhello world\n")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := readFile(t, filepath.Join(out, "_qtidocs", "search-index.json"))
	if !strings.Contains(idx, "Home") || !strings.Contains(idx, "hello world") || !strings.Contains(idx, "desc") {
		t.Errorf("search index missing expected fields: %s", idx)
	}
}

func TestBuild_AssetsCopied(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "assets", "logo.png"), "fake-png-bytes")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readFile(t, filepath.Join(out, "assets", "logo.png"))
	if got != "fake-png-bytes" {
		t.Errorf("copied asset content = %q", got)
	}
}

func TestBuild_ValidOverrideCSSCopiedAndFlagged(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "assets", "qtidocs.css"), "body { color: red; }")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(html, "qtidocs.css") {
		t.Errorf("expected page to link qtidocs.css override, got: %s", truncate(html))
	}
}

func TestBuild_InvalidOverrideCSSFailsBuild(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "assets", "qtidocs.css"), `@import url("https://evil.example/x.css");`)

	err := Build(src, out, "")
	if err == nil {
		t.Fatal("expected error for invalid override CSS, got nil")
	}
}

func TestBuild_NavYamlOverridesDefaultNav(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "page-a.md"), "---\ntitle: Page A\n---\nx\n")
	writeFile(t, filepath.Join(src, "nav.yaml"), "- title: Custom Label\n  path: page-a\n")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(html, "Custom Label") {
		t.Errorf("expected nav.yaml's custom title in nav, got: %s", truncate(html))
	}
}

func TestBuild_SidebarYamlUsedWhenNoNavYaml(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "page-a.md"), "---\ntitle: Page A\n---\nx\n")
	writeFile(t, filepath.Join(src, "sidebar.yaml"), "- title: From Sidebar\n  path: page-a\n")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(html, "From Sidebar") {
		t.Errorf("expected sidebar.yaml's title in nav, got: %s", truncate(html))
	}
}

func TestBuild_InvalidNavYamlFails(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "nav.yaml"), "not: valid: yaml: [")

	err := Build(src, out, "")
	if err == nil {
		t.Fatal("expected error for invalid nav.yaml, got nil")
	}
}

func TestBuild_NavFalseHidesPage(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "hidden.md"), "---\ntitle: Hidden Page\nnav: false\n---\nx\n")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := readFile(t, filepath.Join(out, "index.html"))
	if strings.Contains(html, "Hidden Page") {
		t.Errorf("expected hidden page to be excluded from nav, got: %s", truncate(html))
	}
	if _, err := os.Stat(filepath.Join(out, "hidden", "index.html")); err != nil {
		t.Errorf("expected hidden page still written: %v", err)
	}
}

func TestBuildVersioned_PrefixesLinksAndWritesVersionSwitcher(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "guides.md"), "---\ntitle: Guides\n---\nx\n")

	err := BuildVersioned(src, out, "", VersionInfo{Current: "v2", Names: []string{"v1", "v2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(html, `href="/v2/guides"`) {
		t.Errorf("expected nav link prefixed with /v2/, got: %s", truncate(html))
	}
	if !strings.Contains(html, "v1") || !strings.Contains(html, "v2") {
		t.Errorf("expected version switcher listing v1 and v2, got: %s", truncate(html))
	}
}

func TestBuildVersioned_UnversionedZeroValueMatchesBuild(t *testing.T) {
	src := t.TempDir()
	outA := t.TempDir()
	outB := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")

	if err := Build(src, outA, "Acme"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if err := BuildVersioned(src, outB, "Acme", VersionInfo{}); err != nil {
		t.Fatalf("BuildVersioned error: %v", err)
	}

	a := readFile(t, filepath.Join(outA, "index.html"))
	b := readFile(t, filepath.Join(outB, "index.html"))
	if a != b {
		t.Errorf("Build and BuildVersioned(zero value) produced different output")
	}
}

func TestBuild_IgnoresAssetsFolderAsPages(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(src, "index.md"), "---\ntitle: Home\n---\nx\n")
	writeFile(t, filepath.Join(src, "assets", "notes.md"), "not a page")

	if err := Build(src, out, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "notes", "index.html")); !os.IsNotExist(err) {
		t.Errorf("expected assets/notes.md to never be rendered as a page")
	}
}

func truncate(s string) string {
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}
