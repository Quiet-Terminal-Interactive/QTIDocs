package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeEntry(t *testing.T, dir, name, subdomain string) {
	t.Helper()
	content := "subdomain: " + subdomain + "\nrepo: github.com/acme/widgets\nbranch: main\npath: qtidocs\ncontact: dev@example.com\nmaintainers:\n  - alice\n"
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestRun_ReportsRegisteredSubdomains(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "acme.yaml", "acme")
	writeEntry(t, dir, "other.yaml", "other")

	var gotBody struct {
		Registered []string `json:"registered"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"torn_down":1}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	err := run(dir, srv.URL, "sekret", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotBody.Registered) != 2 {
		t.Errorf("Registered = %v, want 2 entries", gotBody.Registered)
	}
}

func TestRun_EmptyRegistry(t *testing.T) {
	dir := t.TempDir()
	var gotBody struct {
		Registered []string `json:"registered"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"torn_down":0}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	if err := run(dir, srv.URL, "sekret", client); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotBody.Registered) != 0 {
		t.Errorf("Registered = %v, want empty", gotBody.Registered)
	}
}

func TestRun_AuthorizationHeaderSet(t *testing.T) {
	dir := t.TempDir()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	if err := run(dir, srv.URL, "my-secret", client); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer my-secret" {
		t.Errorf("Authorization = %q, want Bearer my-secret", gotAuth)
	}
}

func TestRun_PlatformErrorStatusPropagates(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	err := run(dir, srv.URL, "sekret", client)
	if err == nil {
		t.Fatal("expected error for platform 500 response, got nil")
	}
}

func TestRun_InvalidSitesDir(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	err := run(filepath.Join(t.TempDir(), "does-not-exist"), "http://example.com", "sekret", client)
	if err == nil {
		t.Fatal("expected error for nonexistent sites dir, got nil")
	}
}

func TestRun_UnreachablePlatform(t *testing.T) {
	dir := t.TempDir()
	client := &http.Client{Timeout: 1 * time.Second}
	err := run(dir, "http://127.0.0.1:0", "sekret", client)
	if err == nil {
		t.Fatal("expected error for unreachable platform, got nil")
	}
}
