package nav

import (
	"reflect"
	"testing"
)

func TestBuild_FlatPages(t *testing.T) {
	pages := []Page{
		{URLPath: "b-page", Title: "B Page", Order: 2, Nav: true},
		{URLPath: "a-page", Title: "A Page", Order: 1, Nav: true},
	}
	got := Build(pages)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Title != "A Page" || got[1].Title != "B Page" {
		t.Errorf("order mismatch: got[0]=%q got[1]=%q", got[0].Title, got[1].Title)
	}
}

func TestBuild_RootPageOmitted(t *testing.T) {
	pages := []Page{
		{URLPath: "", Title: "Home", Nav: true},
		{URLPath: "page", Title: "Page", Nav: true},
	}
	got := Build(pages)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1 (root omitted)", len(got))
	}
	if got[0].Title != "Page" {
		t.Errorf("got[0].Title = %q, want Page", got[0].Title)
	}
}

func TestBuild_NavFalseOmitted(t *testing.T) {
	pages := []Page{
		{URLPath: "visible", Title: "Visible", Nav: true},
		{URLPath: "hidden", Title: "Hidden", Nav: false},
	}
	got := Build(pages)
	if len(got) != 1 || got[0].Title != "Visible" {
		t.Errorf("got = %+v, want only Visible", dumpEntries(got))
	}
}

func TestBuild_NestedFolderWithIndexPage(t *testing.T) {
	pages := []Page{
		{URLPath: "guides", Title: "Guide Home", Order: 0, Nav: true},
		{URLPath: "guides/advanced", Title: "Advanced", Order: 1, Nav: true},
	}
	got := Build(pages)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	top := got[0]
	if top.Title != "Guide Home" || top.Path != "guides" {
		t.Errorf("top = %+v, want Title=Guide Home Path=guides", top)
	}
	if len(top.Children) != 1 || top.Children[0].Title != "Advanced" || top.Children[0].Path != "guides/advanced" {
		t.Errorf("children = %+v", dumpEntries(top.Children))
	}
}

func TestBuild_FolderWithoutIndexPageGetsDerivedTitle(t *testing.T) {
	pages := []Page{
		{URLPath: "getting-started/install", Title: "Install", Order: 0, Nav: true},
	}
	got := Build(pages)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	top := got[0]
	if top.Title != "Getting Started" {
		t.Errorf("top.Title = %q, want %q", top.Title, "Getting Started")
	}
	if top.Path != "" {
		t.Errorf("top.Path = %q, want empty (unlinked section header)", top.Path)
	}
	if len(top.Children) != 1 || top.Children[0].Title != "Install" {
		t.Errorf("children = %+v", dumpEntries(top.Children))
	}
}

func TestBuild_SortsByOrderThenTitleCaseInsensitive(t *testing.T) {
	pages := []Page{
		{URLPath: "zeta", Title: "zeta", Order: 0, Nav: true},
		{URLPath: "Alpha", Title: "Alpha", Order: 0, Nav: true},
		{URLPath: "beta", Title: "beta", Order: 0, Nav: true},
	}
	got := Build(pages)
	var titles []string
	for _, e := range got {
		titles = append(titles, e.Title)
	}
	want := []string{"Alpha", "beta", "zeta"}
	if !reflect.DeepEqual(titles, want) {
		t.Errorf("titles = %v, want %v", titles, want)
	}
}

func TestBuild_ExplicitOrderBeatsTitle(t *testing.T) {
	pages := []Page{
		{URLPath: "z-page", Title: "Z Page", Order: 1, Nav: true},
		{URLPath: "a-page", Title: "A Page", Order: 2, Nav: true},
	}
	got := Build(pages)
	if got[0].Title != "Z Page" || got[1].Title != "A Page" {
		t.Errorf("titles = [%q, %q], want order to win over alphabetical", got[0].Title, got[1].Title)
	}
}

func TestBuild_EmptyPages(t *testing.T) {
	got := Build(nil)
	if got != nil {
		t.Errorf("Build(nil) = %+v, want nil", dumpEntries(got))
	}
}

func TestBuild_DeeplyNestedPath(t *testing.T) {
	pages := []Page{
		{URLPath: "a/b/c", Title: "Deep", Nav: true},
	}
	got := Build(pages)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	a := got[0]
	if len(a.Children) != 1 {
		t.Fatalf("a.Children = %+v, want 1", dumpEntries(a.Children))
	}
	b := a.Children[0]
	if len(b.Children) != 1 || b.Children[0].Title != "Deep" {
		t.Fatalf("b.Children = %+v", dumpEntries(b.Children))
	}
}

func TestTitleFromSlug(t *testing.T) {
	cases := map[string]string{
		"getting-started": "Getting Started",
		"advanced":        "Advanced",
		"a-b-c":           "A B C",
		"":                "",
		"école":           "École",
		"日本語-guide":       "日本語 Guide",
	}
	for slug, want := range cases {
		if got := titleFromSlug(slug); got != want {
			t.Errorf("titleFromSlug(%q) = %q, want %q", slug, got, want)
		}
	}
}

func dumpEntries(entries []*Entry) []Entry {
	out := make([]Entry, len(entries))
	for i, e := range entries {
		out[i] = *e
	}
	return out
}
