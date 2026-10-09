package theme

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/nav"
)

func TestRenderPage_WithSiteTitle(t *testing.T) {
	var buf bytes.Buffer
	err := RenderPage(&buf, PageData{
		SiteTitle: "Acme Docs",
		Title:     "Getting Started",
		Content:   template.HTML("<p>hi</p>"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Acme Docs | QTIDocs") {
		t.Errorf("output missing combined title, got: %s", truncate(out))
	}
	if !strings.Contains(out, "<p>hi</p>") {
		t.Errorf("output missing content, got: %s", truncate(out))
	}
}

func TestRenderPage_WithoutSiteTitle(t *testing.T) {
	var buf bytes.Buffer
	err := RenderPage(&buf, PageData{Title: "Home", Content: template.HTML("body")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "QTIDocs") {
		t.Errorf("output missing brand name, got: %s", truncate(out))
	}
	if strings.Contains(out, "| QTIDocs") {
		t.Errorf("output should not combine with empty site title, got: %s", truncate(out))
	}
}

func TestRenderPage_WithNav(t *testing.T) {
	var buf bytes.Buffer
	entries := []*nav.Entry{
		{Title: "Guides", Path: "guides"},
	}
	err := RenderPage(&buf, PageData{Title: "Home", Nav: entries, Content: template.HTML("x")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Guides") {
		t.Errorf("output missing nav entry title, got: %s", truncate(buf.String()))
	}
}

func TestRenderPage_Description(t *testing.T) {
	var buf bytes.Buffer
	err := RenderPage(&buf, PageData{Title: "Home", Description: "A test description", Content: template.HTML("x")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "A test description") {
		t.Errorf("output missing description, got: %s", truncate(buf.String()))
	}
}

func TestRenderVersionRedirect_WithSiteTitle(t *testing.T) {
	var buf bytes.Buffer
	err := RenderVersionRedirect(&buf, "Acme Docs", "/v2/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Acme Docs | QTIDocs") {
		t.Errorf("output missing combined title, got: %s", truncate(out))
	}
	if !strings.Contains(out, "/v2/") {
		t.Errorf("output missing default path, got: %s", truncate(out))
	}
}

func TestRenderVersionRedirect_WithoutSiteTitle(t *testing.T) {
	var buf bytes.Buffer
	err := RenderVersionRedirect(&buf, "", "/v1/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "QTIDocs") {
		t.Errorf("output missing brand, got: %s", truncate(buf.String()))
	}
}

func TestRenderStats_WithSiteTitle(t *testing.T) {
	var buf bytes.Buffer
	err := RenderStats(&buf, StatsPageData{SiteTitle: "Acme"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Acme") {
		t.Errorf("output missing site title, got: %s", truncate(buf.String()))
	}
}

func TestRenderStats_WithoutSiteTitle(t *testing.T) {
	var buf bytes.Buffer
	err := RenderStats(&buf, StatsPageData{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "QTIDocs") {
		t.Errorf("output missing default brand, got: %s", truncate(buf.String()))
	}
}

func TestDefaultCSS_NonEmptyAndIncludesChroma(t *testing.T) {
	if len(DefaultCSS) == 0 {
		t.Fatal("DefaultCSS is empty")
	}
	if !bytes.Contains(DefaultCSS, []byte("@media (prefers-color-scheme: dark)")) {
		t.Error("DefaultCSS missing chroma dark-mode CSS")
	}
}

func TestStaticAssets_NonEmpty(t *testing.T) {
	assets := map[string][]byte{
		"SearchJS":    SearchJS,
		"ThemeJS":     ThemeJS,
		"AnalyticsJS": AnalyticsJS,
		"StatsJS":     StatsJS,
	}
	for name, data := range assets {
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestRenderPage_ContentNotEscaped(t *testing.T) {
	var buf bytes.Buffer
	err := RenderPage(&buf, PageData{Title: "T", Content: template.HTML("<em>raw</em>")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "<em>raw</em>") {
		t.Errorf("expected template.HTML content to render unescaped, got: %s", truncate(buf.String()))
	}
}

func TestRenderPage_TitleIsEscaped(t *testing.T) {
	var buf bytes.Buffer
	err := RenderPage(&buf, PageData{Title: "<script>alert(1)</script>", Content: template.HTML("x")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buf.String(), "<script>alert(1)</script>") {
		t.Errorf("expected plain-string Title to be escaped, got: %s", truncate(buf.String()))
	}
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}
