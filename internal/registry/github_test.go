package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGitHub_HasWriteAccess_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widgets/collaborators/alice/permission" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"permission":"write"}`))
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.HasWriteAccess(context.Background(), "acme", "widgets", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected HasWriteAccess = true for write permission")
	}
}

func TestHTTPGitHub_HasWriteAccess_Admin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"permission":"admin"}`))
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.HasWriteAccess(context.Background(), "acme", "widgets", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected HasWriteAccess = true for admin permission")
	}
}

func TestHTTPGitHub_HasWriteAccess_ReadOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"permission":"read"}`))
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.HasWriteAccess(context.Background(), "acme", "widgets", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected HasWriteAccess = false for read permission")
	}
}

func TestHTTPGitHub_HasWriteAccess_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.HasWriteAccess(context.Background(), "acme", "widgets", "ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected HasWriteAccess = false for 404")
	}
}

func TestHTTPGitHub_HasWriteAccess_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	_, err := g.HasWriteAccess(context.Background(), "acme", "widgets", "alice")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}

func TestHTTPGitHub_OrgMember_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/qti/members/alice" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.OrgMember(context.Background(), "qti", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected OrgMember = true for 204")
	}
}

func TestHTTPGitHub_OrgMember_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.OrgMember(context.Background(), "qti", "ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected OrgMember = false for 404")
	}
}

func TestHTTPGitHub_OrgMember_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	_, err := g.OrgMember(context.Background(), "qti", "alice")
	if err == nil {
		t.Fatal("expected error for 403 status, got nil")
	}
}

func TestHTTPGitHub_PathExists_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widgets/contents/qtidocs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("ref"); got != "main" {
			t.Errorf("ref query = %q, want main", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.PathExists(context.Background(), "acme", "widgets", "main", "qtidocs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected PathExists = true")
	}
}

func TestHTTPGitHub_PathExists_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	ok, err := g.PathExists(context.Background(), "acme", "widgets", "main", "qtidocs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected PathExists = false for 404")
	}
}

func TestHTTPGitHub_PathExists_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	_, err := g.PathExists(context.Background(), "acme", "widgets", "main", "qtidocs")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
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
