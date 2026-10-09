package nav

import (
	"testing"
)

func TestParseConfig_Basic(t *testing.T) {
	src := []byte(`
- title: Home
  path: ""
- title: Guides
  items:
    - path: guides/advanced
`)
	cfg, err := ParseConfig(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg) != 2 {
		t.Fatalf("len(cfg) = %d, want 2", len(cfg))
	}
	if cfg[0].Title != "Home" {
		t.Errorf("cfg[0].Title = %q, want Home", cfg[0].Title)
	}
	if cfg[1].Title != "Guides" || len(cfg[1].Items) != 1 {
		t.Errorf("cfg[1] = %+v", cfg[1])
	}
	if cfg[1].Items[0].Path != "guides/advanced" {
		t.Errorf("cfg[1].Items[0].Path = %q", cfg[1].Items[0].Path)
	}
}

func TestParseConfig_Empty(t *testing.T) {
	cfg, err := ParseConfig([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg) != 0 {
		t.Errorf("len(cfg) = %d, want 0", len(cfg))
	}
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	_, err := ParseConfig([]byte("not: a: list: [unterminated"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestFromConfig_ExplicitTitle(t *testing.T) {
	cfg := []ConfigEntry{
		{Title: "Custom Title", Path: "page"},
	}
	pages := []Page{{URLPath: "page", Title: "Real Title"}}
	got := FromConfig(cfg, pages)
	if len(got) != 1 || got[0].Title != "Custom Title" {
		t.Errorf("got = %+v, want Custom Title to win", dumpEntries(got))
	}
}

func TestFromConfig_FallsBackToPageTitle(t *testing.T) {
	cfg := []ConfigEntry{
		{Path: "page"},
	}
	pages := []Page{{URLPath: "page", Title: "Real Title"}}
	got := FromConfig(cfg, pages)
	if len(got) != 1 || got[0].Title != "Real Title" {
		t.Errorf("got = %+v, want fallback to Real Title", dumpEntries(got))
	}
}

func TestFromConfig_FallbackMissingPageLeavesTitleEmpty(t *testing.T) {
	cfg := []ConfigEntry{
		{Path: "does-not-exist"},
	}
	got := FromConfig(cfg, nil)
	if len(got) != 1 || got[0].Title != "" {
		t.Errorf("got = %+v, want empty title", dumpEntries(got))
	}
}

func TestFromConfig_PreservesExplicitOrderNoSorting(t *testing.T) {
	cfg := []ConfigEntry{
		{Title: "Zeta", Path: "z"},
		{Title: "Alpha", Path: "a"},
	}
	got := FromConfig(cfg, nil)
	if len(got) != 2 || got[0].Title != "Zeta" || got[1].Title != "Alpha" {
		t.Errorf("got = %+v, want order preserved (Zeta, Alpha)", dumpEntries(got))
	}
}

func TestFromConfig_NestedItems(t *testing.T) {
	cfg := []ConfigEntry{
		{
			Title: "Section",
			Items: []ConfigEntry{
				{Title: "Child A", Path: "a"},
				{Title: "Child B", Path: "b"},
			},
		},
	}
	got := FromConfig(cfg, nil)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	section := got[0]
	if section.Path != "" {
		t.Errorf("section.Path = %q, want empty (unlinked header)", section.Path)
	}
	if len(section.Children) != 2 {
		t.Fatalf("children = %+v, want 2", dumpEntries(section.Children))
	}
	if section.Children[0].Title != "Child A" || section.Children[1].Title != "Child B" {
		t.Errorf("children = %+v", dumpEntries(section.Children))
	}
}

func TestFromConfig_EmptyConfig(t *testing.T) {
	got := FromConfig(nil, nil)
	if got != nil {
		t.Errorf("FromConfig(nil, nil) = %+v, want nil", dumpEntries(got))
	}
}

func TestFromConfig_PathTrimmedForLookup(t *testing.T) {
	cfg := []ConfigEntry{
		{Path: "/page/"},
	}
	pages := []Page{{URLPath: "page", Title: "Real Title"}}
	got := FromConfig(cfg, pages)
	if len(got) != 1 || got[0].Title != "Real Title" {
		t.Errorf("got = %+v, want trimmed path to match page", dumpEntries(got))
	}
}
