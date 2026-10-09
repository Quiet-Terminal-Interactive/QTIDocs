package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEntryFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestLoadAll_Basic(t *testing.T) {
	dir := t.TempDir()
	writeEntryFile(t, dir, "acme.yaml", "subdomain: acme\nrepo: github.com/acme/widgets\nbranch: main\npath: qtidocs\ncontact: dev@example.com\nmaintainers: [alice]\n")
	writeEntryFile(t, dir, "other.yaml", "subdomain: other\nrepo: github.com/other/widgets\nbranch: main\npath: qtidocs\ncontact: dev@example.com\nmaintainers: [bob]\n")

	entries, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries["acme"].Repo != "github.com/acme/widgets" {
		t.Errorf("entries[acme] = %+v", entries["acme"])
	}
}

func TestLoadAll_IgnoresNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	writeEntryFile(t, dir, "acme.yaml", "subdomain: acme\nrepo: github.com/acme/widgets\nbranch: main\npath: qtidocs\ncontact: dev@example.com\nmaintainers: [alice]\n")
	writeEntryFile(t, dir, "README.md", "not a registry entry")

	entries, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1 (README.md should be ignored)", len(entries))
	}
}

func TestLoadAll_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()
	writeEntryFile(t, dir, "acme.yaml", "subdomain: acme\nrepo: github.com/acme/widgets\nbranch: main\npath: qtidocs\ncontact: dev@example.com\nmaintainers: [alice]\n")
	if err := os.MkdirAll(filepath.Join(dir, "subdir.yaml"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	entries, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1 (directory should be ignored)", len(entries))
	}
}

func TestLoadAll_FilenameSubdomainMismatch(t *testing.T) {
	dir := t.TempDir()
	writeEntryFile(t, dir, "acme.yaml", "subdomain: other-name\nrepo: github.com/acme/widgets\nbranch: main\npath: qtidocs\ncontact: dev@example.com\nmaintainers: [alice]\n")

	_, err := LoadAll(dir)
	if err == nil || !strings.Contains(err.Error(), "does not match filename") {
		t.Errorf("err = %v, want filename-mismatch complaint", err)
	}
}

func TestLoadAll_InvalidYAMLFile(t *testing.T) {
	dir := t.TempDir()
	writeEntryFile(t, dir, "acme.yaml", "not: valid: yaml: [")

	_, err := LoadAll(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	entries, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestLoadAll_NonexistentDir(t *testing.T) {
	_, err := LoadAll(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for nonexistent dir, got nil")
	}
}
