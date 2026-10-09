package storage

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestGenerateSecret_ReturnsHex64(t *testing.T) {
	s, err := GenerateSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 64 {
		t.Errorf("len(secret) = %d, want 64 (32 bytes hex-encoded)", len(s))
	}
}

func TestGenerateSecret_Unique(t *testing.T) {
	a, _ := GenerateSecret()
	b, _ := GenerateSecret()
	if a == b {
		t.Error("expected two generated secrets to differ")
	}
}

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	dir := t.TempDir()
	return Open(filepath.Join(dir, "routing.json"))
}

func TestFileStore_GetMissingReturnsNotOK(t *testing.T) {
	s := newTestStore(t)
	_, ok, err := s.Get("nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for missing entry")
	}
}

func TestFileStore_SetAndGet(t *testing.T) {
	s := newTestStore(t)
	entry := Entry{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"}
	if err := s.Set(entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok, err := s.Get("acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.Subdomain != entry.Subdomain || got.Repo != entry.Repo || got.Branch != entry.Branch || got.Path != entry.Path {
		t.Errorf("got = %+v, want %+v", got, entry)
	}
}

func TestFileStore_SetOverwrites(t *testing.T) {
	s := newTestStore(t)
	_ = s.Set(Entry{Subdomain: "acme", Branch: "main"})
	_ = s.Set(Entry{Subdomain: "acme", Branch: "develop"})

	got, _, err := s.Get("acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Branch != "develop" {
		t.Errorf("got.Branch = %q, want develop", got.Branch)
	}
}

func TestFileStore_List(t *testing.T) {
	s := newTestStore(t)
	_ = s.Set(Entry{Subdomain: "a"})
	_ = s.Set(Entry{Subdomain: "b"})

	entries, err := s.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Subdomain] = true
	}
	if !names["a"] || !names["b"] {
		t.Errorf("names = %v, want both a and b", names)
	}
}

func TestFileStore_ListEmpty(t *testing.T) {
	s := newTestStore(t)
	entries, err := s.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestFileStore_Delete(t *testing.T) {
	s := newTestStore(t)
	_ = s.Set(Entry{Subdomain: "acme"})

	if err := s.Delete("acme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok, err := s.Get("acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected entry to be gone after Delete")
	}
}

func TestFileStore_DeleteMissingIsNoop(t *testing.T) {
	s := newTestStore(t)
	if err := s.Delete("nope"); err != nil {
		t.Errorf("unexpected error deleting missing entry: %v", err)
	}
}

func TestFileStore_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routing.json")

	s1 := Open(path)
	_ = s1.Set(Entry{Subdomain: "acme", Repo: "acme/widgets"})

	s2 := Open(path)
	got, ok, err := s2.Get("acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok || got.Repo != "acme/widgets" {
		t.Errorf("got = %+v, ok = %v", got, ok)
	}
}

func TestFileStore_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routing.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	s := Open(path)
	_, _, err := s.Get("acme")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestFileStore_ConcurrentSetsAreSerialized(t *testing.T) {
	s := newTestStore(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.Set(Entry{Subdomain: "site", Branch: "b" + string(rune('a'+i))})
		}(i)
	}
	wg.Wait()

	entries, err := s.List()
	if err != nil {
		t.Fatalf("unexpected error after concurrent writes: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}

func TestEntry_OmitsEmptyOptionalFields(t *testing.T) {
	s := newTestStore(t)
	_ = s.Set(Entry{Subdomain: "bare"})

	raw, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, field := range []string{"title", "output_dir", "webhook_secret", "deployed_sha", "versions", "default_version", "deployed_versions"} {
		if strings.Contains(string(raw), field) {
			t.Errorf("expected empty field %q to be omitted, got: %s", field, raw)
		}
	}
}
