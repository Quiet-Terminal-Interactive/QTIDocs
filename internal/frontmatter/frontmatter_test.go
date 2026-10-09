package frontmatter

import (
	"errors"
	"testing"
)

func TestParse_Basic(t *testing.T) {
	src := []byte("---\ntitle: Hello World\norder: 3\ndescription: A desc\nauthor: Jane\n---\n# Body\n\nContent here.\n")
	data, body, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Title != "Hello World" {
		t.Errorf("Title = %q, want %q", data.Title, "Hello World")
	}
	if data.Order != 3 {
		t.Errorf("Order = %d, want 3", data.Order)
	}
	if data.Description != "A desc" {
		t.Errorf("Description = %q, want %q", data.Description, "A desc")
	}
	if data.Author != "Jane" {
		t.Errorf("Author = %q, want %q", data.Author, "Jane")
	}
	if !data.Nav {
		t.Errorf("Nav = false, want true (default)")
	}
	wantBody := "# Body\n\nContent here.\n"
	if string(body) != wantBody {
		t.Errorf("body = %q, want %q", body, wantBody)
	}
}

func TestParse_NavExplicitFalse(t *testing.T) {
	src := []byte("---\ntitle: Hidden\nnav: false\n---\nbody\n")
	data, _, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Nav {
		t.Errorf("Nav = true, want false")
	}
}

func TestParse_NavExplicitTrue(t *testing.T) {
	src := []byte("---\ntitle: Shown\nnav: true\n---\nbody\n")
	data, _, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !data.Nav {
		t.Errorf("Nav = false, want true")
	}
}

func TestParse_DefaultsOmitted(t *testing.T) {
	src := []byte("---\ntitle: Minimal\n---\nbody\n")
	data, _, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Order != 0 {
		t.Errorf("Order = %d, want 0", data.Order)
	}
	if data.Description != "" {
		t.Errorf("Description = %q, want empty", data.Description)
	}
	if data.Author != "" {
		t.Errorf("Author = %q, want empty", data.Author)
	}
}

func TestParse_UnknownFieldsIgnored(t *testing.T) {
	src := []byte("---\ntitle: Has Extra\nsomethingElse: 42\nweird: [1,2,3]\n---\nbody\n")
	data, _, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Title != "Has Extra" {
		t.Errorf("Title = %q, want %q", data.Title, "Has Extra")
	}
}

func TestParse_MissingTitle(t *testing.T) {
	src := []byte("---\norder: 1\n---\nbody\n")
	_, _, err := Parse(src)
	if !errors.Is(err, ErrMissingTitle) {
		t.Errorf("err = %v, want ErrMissingTitle", err)
	}
}

func TestParse_EmptyTitle(t *testing.T) {
	src := []byte("---\ntitle: \"\"\n---\nbody\n")
	_, _, err := Parse(src)
	if !errors.Is(err, ErrMissingTitle) {
		t.Errorf("err = %v, want ErrMissingTitle", err)
	}
}

func TestParse_NoFrontmatterFence(t *testing.T) {
	src := []byte("# Just a markdown file\n\nNo frontmatter here.\n")
	_, _, err := Parse(src)
	if !errors.Is(err, ErrMissingTitle) {
		t.Errorf("err = %v, want ErrMissingTitle", err)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	_, _, err := Parse([]byte(""))
	if !errors.Is(err, ErrMissingTitle) {
		t.Errorf("err = %v, want ErrMissingTitle", err)
	}
}

func TestParse_UnterminatedBlock(t *testing.T) {
	src := []byte("---\ntitle: Oops\nno closing fence here\n")
	_, _, err := Parse(src)
	if err == nil {
		t.Fatal("expected error for unterminated block, got nil")
	}
	if errors.Is(err, ErrMissingTitle) {
		t.Errorf("err = %v, want a non-ErrMissingTitle unterminated-block error", err)
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	src := []byte("---\ntitle: [unclosed\n---\nbody\n")
	_, _, err := Parse(src)
	if err == nil {
		t.Fatal("expected YAML parse error, got nil")
	}
}

func TestParse_CRLFLineEndings(t *testing.T) {
	src := []byte("---\r\ntitle: CRLF Test\r\n---\r\nbody\r\n")
	data, _, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Title != "CRLF Test" {
		t.Errorf("Title = %q, want %q", data.Title, "CRLF Test")
	}
}

func TestParse_EmptyBody(t *testing.T) {
	src := []byte("---\ntitle: No Body\n---\n")
	data, body, err := Parse(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Title != "No Body" {
		t.Errorf("Title = %q", data.Title)
	}
	if string(body) != "" {
		t.Errorf("body = %q, want empty", body)
	}
}
