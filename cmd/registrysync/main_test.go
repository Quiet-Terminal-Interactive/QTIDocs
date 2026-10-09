package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/registry/registrytest"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := registrytest.InitRepo(t)
	if err := os.MkdirAll(filepath.Join(dir, "sites"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeAndCommit(t, dir, "README.md", "registry repo\n")
	return dir
}

func writeAndCommit(t *testing.T, dir, path, content string) {
	registrytest.CommitFile(t, dir, path, content)
}

func removeAndCommit(t *testing.T, dir, path string) {
	registrytest.RemoveFile(t, dir, path)
}

func chdir(t *testing.T, dir string) {
	registrytest.Chdir(t, dir)
}

func headSHA(t *testing.T, dir string) string {
	return registrytest.HeadSHA(t, dir)
}

func validEntryYAML(subdomain string) string {
	return registrytest.ValidEntryYAML(subdomain)
}

func TestRun_RegistersAddedEntry(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	base := headSHA(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))

	var gotSubdomain string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Subdomain string `json:"subdomain"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotSubdomain = body.Subdomain
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted","subdomain":"acme"}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	err := run("sites", base, srv.URL, "sekret", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSubdomain != "acme" {
		t.Errorf("gotSubdomain = %q, want acme", gotSubdomain)
	}
}

func TestRun_SkipsDeletedEntry(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))
	writeAndCommit(t, dir, "sites/other.yaml", validEntryYAML("other"))
	base := headSHA(t, dir)
	removeAndCommit(t, dir, "sites/acme.yaml")

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	err := run("sites", base, srv.URL, "sekret", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected deleted entries to never call the platform endpoint")
	}
}

func TestRun_NoChangesIsOK(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))
	base := headSHA(t, dir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("platform endpoint should not be called when nothing changed")
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	if err := run("sites", base, srv.URL, "sekret", client); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_PlatformErrorStatusPropagates(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	base := headSHA(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	err := run("sites", base, srv.URL, "sekret", client)
	if err == nil {
		t.Fatal("expected error for platform 500 response, got nil")
	}
}

func TestRun_AuthorizationHeaderSet(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	base := headSHA(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml", validEntryYAML("acme"))

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	if err := run("sites", base, srv.URL, "my-secret", client); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer my-secret" {
		t.Errorf("Authorization = %q, want Bearer my-secret", gotAuth)
	}
}

func TestRun_IncludesVersionsInRequest(t *testing.T) {
	dir := gitRepo(t)
	chdir(t, dir)
	base := headSHA(t, dir)
	writeAndCommit(t, dir, "sites/acme.yaml",
		validEntryYAML("acme")+"versions:\n  - name: v1\n    ref: v1-branch\ndefault_version: v1\n")

	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	if err := run("sites", base, srv.URL, "sekret", client); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	versions, ok := gotBody["versions"].([]any)
	if !ok || len(versions) != 1 {
		t.Fatalf("gotBody[versions] = %v", gotBody["versions"])
	}
	if gotBody["default_version"] != "v1" {
		t.Errorf("default_version = %v, want v1", gotBody["default_version"])
	}
}
