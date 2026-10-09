package search

import (
	"encoding/json"
	"testing"
)

func TestBuildIndex_Basic(t *testing.T) {
	docs := []Doc{
		{Path: "", Title: "Home", Content: "welcome"},
		{Path: "guides/advanced", Title: "Advanced", Description: "desc", Content: "advanced content"},
	}
	out, err := BuildIndex(docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []Doc
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("failed to unmarshal index: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Path != "" || got[0].Title != "Home" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].Description != "desc" {
		t.Errorf("got[1].Description = %q, want %q", got[1].Description, "desc")
	}
}

func TestBuildIndex_NilDocsProducesEmptyArray(t *testing.T) {
	out, err := BuildIndex(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "[]" {
		t.Errorf("BuildIndex(nil) = %q, want %q", out, "[]")
	}
}

func TestBuildIndex_EmptySliceProducesEmptyArray(t *testing.T) {
	out, err := BuildIndex([]Doc{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "[]" {
		t.Errorf("BuildIndex([]Doc{}) = %q, want %q", out, "[]")
	}
}

func TestBuildIndex_OmitsEmptyDescription(t *testing.T) {
	out, err := BuildIndex([]Doc{{Path: "p", Title: "T", Content: "c"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var raw []map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if _, ok := raw[0]["description"]; ok {
		t.Errorf("expected description to be omitted when empty, got %v", raw[0])
	}
}

func TestPlainText_StripsTags(t *testing.T) {
	got := PlainText("<p>Hello <strong>World</strong></p>")
	want := "Hello World"
	if got != want {
		t.Errorf("PlainText() = %q, want %q", got, want)
	}
}

func TestPlainText_CollapsesWhitespace(t *testing.T) {
	got := PlainText("<p>Hello   \n\n  World  </p>\n<p>Again</p>")
	want := "Hello World Again"
	if got != want {
		t.Errorf("PlainText() = %q, want %q", got, want)
	}
}

func TestPlainText_EmptyInput(t *testing.T) {
	got := PlainText("")
	if got != "" {
		t.Errorf("PlainText(\"\") = %q, want empty", got)
	}
}

func TestPlainText_OnlyTagsNoText(t *testing.T) {
	got := PlainText("<div><span></span></div>")
	if got != "" {
		t.Errorf("PlainText() = %q, want empty", got)
	}
}

func TestPlainText_NestedStructure(t *testing.T) {
	got := PlainText("<table><tr><td>A</td><td>B</td></tr></table>")
	want := "A B"
	if got != want {
		t.Errorf("PlainText() = %q, want %q", got, want)
	}
}

func TestPlainText_HandlesEntities(t *testing.T) {
	got := PlainText("<p>A &amp; B</p>")
	want := "A & B"
	if got != want {
		t.Errorf("PlainText() = %q, want %q", got, want)
	}
}
