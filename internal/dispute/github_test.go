package dispute

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGitHub_ListOpenIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/issues" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("labels"); got != "dispute" {
			t.Errorf("labels query = %q, want dispute", got)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", auth)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"number": 1,
				"body":   "Deadline: 2026-01-01",
				"labels": []map[string]string{{"name": "dispute"}},
			},
		})
	}))
	defer srv.Close()

	g := HTTPGitHub{Token: "tok", BaseURL: srv.URL}
	issues, err := g.ListOpenIssues(context.Background(), "owner", "repo", "dispute")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 || issues[0].Number != 1 || issues[0].Body != "Deadline: 2026-01-01" {
		t.Errorf("issues = %+v", issues)
	}
	if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "dispute" {
		t.Errorf("issues[0].Labels = %v", issues[0].Labels)
	}
}

func TestHTTPGitHub_ListOpenIssues_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	_, err := g.ListOpenIssues(context.Background(), "owner", "repo", "dispute")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}

func TestHTTPGitHub_CommentOnIssue(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/issues/5/comments" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	err := g.CommentOnIssue(context.Background(), "owner", "repo", 5, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody["body"] != "hello" {
		t.Errorf("gotBody = %v, want body=hello", gotBody)
	}
}

func TestHTTPGitHub_CommentOnIssue_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	err := g.CommentOnIssue(context.Background(), "owner", "repo", 5, "hello")
	if err == nil {
		t.Fatal("expected error for 403 status, got nil")
	}
}

func TestHTTPGitHub_AddLabel(t *testing.T) {
	var gotBody map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues/7/labels" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	err := g.AddLabel(context.Background(), "owner", "repo", 7, "deadline-passed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotBody["labels"]) != 1 || gotBody["labels"][0] != "deadline-passed" {
		t.Errorf("gotBody = %v", gotBody)
	}
}

func TestHTTPGitHub_AddLabel_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	err := g.AddLabel(context.Background(), "owner", "repo", 7, "x")
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
}

func TestHTTPGitHub_CloseIssue(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/issues/9" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	err := g.CloseIssue(context.Background(), "owner", "repo", 9, "completed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody["state"] != "closed" || gotBody["state_reason"] != "completed" {
		t.Errorf("gotBody = %v", gotBody)
	}
}

func TestHTTPGitHub_CloseIssue_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	g := HTTPGitHub{BaseURL: srv.URL}
	err := g.CloseIssue(context.Background(), "owner", "repo", 9, "completed")
	if err == nil {
		t.Fatal("expected error for 400 status, got nil")
	}
}

func TestHTTPGitHub_DefaultBaseURLAndClient(t *testing.T) {
	g := HTTPGitHub{}
	if g.baseURL() != "https://api.github.com" {
		t.Errorf("baseURL() = %q, want default", g.baseURL())
	}
	if g.client() != http.DefaultClient {
		t.Error("client() should default to http.DefaultClient")
	}
}
