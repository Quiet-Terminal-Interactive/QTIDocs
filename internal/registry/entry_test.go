package registry

import (
	"strings"
	"testing"
)

func TestParseEntry_Basic(t *testing.T) {
	src := []byte(`
subdomain: acme
title: Acme Docs
repo: github.com/acme/widgets
branch: main
path: qtidocs
maintainers:
  - alice
  - bob
`)
	e, err := ParseEntry(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Subdomain != "acme" || e.Title != "Acme Docs" || e.Repo != "github.com/acme/widgets" {
		t.Errorf("e = %+v", e)
	}
	if len(e.Maintainers) != 2 {
		t.Errorf("Maintainers = %v", e.Maintainers)
	}
}

func TestParseEntry_InvalidYAML(t *testing.T) {
	_, err := ParseEntry([]byte("not: valid: yaml: ["))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEntry_Filename(t *testing.T) {
	e := Entry{Subdomain: "acme"}
	if e.Filename() != "acme.yaml" {
		t.Errorf("Filename() = %q, want acme.yaml", e.Filename())
	}
}

func validEntry() Entry {
	return Entry{
		Subdomain:   "acme",
		Repo:        "github.com/acme/widgets",
		Branch:      "main",
		Path:        "qtidocs",
		Contact:     "dev@example.com",
		Maintainers: []string{"alice"},
	}
}

func TestEntry_Validate_Valid(t *testing.T) {
	if err := validEntry().Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEntry_Validate_InvalidSubdomain(t *testing.T) {
	e := validEntry()
	e.Subdomain = "Not_Valid!"
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "subdomain") {
		t.Errorf("err = %v, want subdomain complaint", err)
	}
}

func TestEntry_Validate_InvalidRepo(t *testing.T) {
	e := validEntry()
	e.Repo = "not-a-repo"
	err := e.Validate()
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

func TestEntry_Validate_MissingBranch(t *testing.T) {
	e := validEntry()
	e.Branch = ""
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "branch") {
		t.Errorf("err = %v, want branch complaint", err)
	}
}

func TestEntry_Validate_MissingPath(t *testing.T) {
	e := validEntry()
	e.Path = ""
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "path") {
		t.Errorf("err = %v, want path complaint", err)
	}
}

func TestEntry_Validate_MissingContact(t *testing.T) {
	e := validEntry()
	e.Contact = ""
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "contact is required") {
		t.Errorf("err = %v, want contact complaint", err)
	}
}

func TestEntry_Validate_InvalidContact(t *testing.T) {
	for _, c := range []string{"not-an-email", "Jane <jane@example.com>", "a@b.com, c@d.com", "jane@example.com\nBcc: x@y.com"} {
		e := validEntry()
		e.Contact = c
		err := e.Validate()
		if err == nil || !strings.Contains(err.Error(), "contact") {
			t.Errorf("contact %q: err = %v, want contact complaint", c, err)
		}
	}
}

func TestEntry_Validate_NoMaintainers(t *testing.T) {
	e := validEntry()
	e.Maintainers = nil
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "maintainers") {
		t.Errorf("err = %v, want maintainers complaint", err)
	}
}

func TestEntry_Validate_BlankMaintainer(t *testing.T) {
	e := validEntry()
	e.Maintainers = []string{"  "}
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "blank") {
		t.Errorf("err = %v, want blank-maintainer complaint", err)
	}
}

func TestEntry_Validate_MultipleErrorsJoined(t *testing.T) {
	e := Entry{}
	err := e.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"subdomain", "branch", "path", "maintainers"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q: %s", want, msg)
		}
	}
}

func TestEntry_Validate_VersionsValid(t *testing.T) {
	e := validEntry()
	e.Versions = []Version{{Name: "v1", Ref: "v1-branch"}, {Name: "v2", Ref: "v2-branch"}}
	e.DefaultVersion = "v2"
	if err := e.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEntry_Validate_VersionMissingField(t *testing.T) {
	e := validEntry()
	e.Versions = []Version{{Name: "v1"}}
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "name and ref") {
		t.Errorf("err = %v, want name/ref complaint", err)
	}
}

func TestEntry_Validate_UnsafeVersionName(t *testing.T) {
	for _, name := range []string{"..", ".", "../../escaped", "v1/v2", `v1\v2`, "_qtidocs", "-v1", "v 1"} {
		e := validEntry()
		e.Versions = []Version{{Name: name, Ref: "a"}}
		err := e.Validate()
		if err == nil || !strings.Contains(err.Error(), "version name") {
			t.Errorf("name %q: err = %v, want version name complaint", name, err)
		}
	}
}

func TestEntry_Validate_DuplicateVersionName(t *testing.T) {
	e := validEntry()
	e.Versions = []Version{{Name: "v1", Ref: "a"}, {Name: "v1", Ref: "b"}}
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("err = %v, want duplicate complaint", err)
	}
}

func TestEntry_Validate_DefaultVersionNotListed(t *testing.T) {
	e := validEntry()
	e.Versions = []Version{{Name: "v1", Ref: "a"}}
	e.DefaultVersion = "v2"
	err := e.Validate()
	if err == nil || !strings.Contains(err.Error(), "default_version") {
		t.Errorf("err = %v, want default_version complaint", err)
	}
}

func TestParseRepo(t *testing.T) {
	cases := []struct {
		in, wantOwner, wantName string
		wantErr                 bool
	}{
		{"github.com/acme/widgets", "acme", "widgets", false},
		{"https://github.com/acme/widgets", "acme", "widgets", false},
		{"https://github.com/acme/widgets.git", "acme", "widgets", false},
		{"http://github.com/acme/widgets", "acme", "widgets", false},
		{"acme/widgets", "acme", "widgets", false},
		{"not-a-repo", "", "", true},
		{"github.com/acme", "", "", true},
		{"github.com//widgets", "", "", true},
		{"", "", "", true},
	}
	for _, c := range cases {
		owner, name, err := ParseRepo(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseRepo(%q): expected error, got nil", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRepo(%q): unexpected error: %v", c.in, err)
			continue
		}
		if owner != c.wantOwner || name != c.wantName {
			t.Errorf("ParseRepo(%q) = (%q, %q), want (%q, %q)", c.in, owner, name, c.wantOwner, c.wantName)
		}
	}
}
