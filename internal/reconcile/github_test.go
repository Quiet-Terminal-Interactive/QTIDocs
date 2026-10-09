package reconcile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGitHub_ResolveRef(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widgets/commits/main" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github.sha" {
			t.Errorf("Accept = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", got)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("abc123\n"))
	}))
	defer srv.Close()

	g := HTTPGitHub{Token: "tok", BaseURL: srv.URL}
	sha, err := g.ResolveRef(context.Background(), "acme", "widgets", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "abc123" {
		t.Errorf("sha = %q, want abc123 (trimmed)", sha)
	}
}

func TestHTTPGitHub_ResolveRef_NoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("Authorization = %q, want empty", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("sha"))
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	if _, err := g.ResolveRef(context.Background(), "acme", "widgets", "main"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPGitHub_ResolveRef_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	_, err := g.ResolveRef(context.Background(), "acme", "widgets", "main")
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
}

func TestHTTPGitHub_Defaults(t *testing.T) {
	g := HTTPGitHub{}
	if g.baseURL() != "https://api.github.com" {
		t.Errorf("baseURL() = %q", g.baseURL())
	}
	if g.client() != http.DefaultClient {
		t.Error("client() should default to http.DefaultClient")
	}
}
