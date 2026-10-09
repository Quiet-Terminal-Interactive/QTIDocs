package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
)

func TestReportBuildFailure_NoTokenIsNoop(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	r := GitHubReporter{BaseURL: srv.URL}
	err := r.ReportBuildFailure(context.Background(), build.Job{Owner: "o", Repo: "r"}, errors.New("boom"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected no HTTP call when Token is empty")
	}
}

func TestReportBuildFailure_OpensIssue(t *testing.T) {
	var gotPath string
	var gotBody struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	r := GitHubReporter{Token: "tok", BaseURL: srv.URL}
	job := build.Job{Owner: "acme", Repo: "widgets", Subdomain: "widgets", Ref: "main", Path: "qtidocs"}
	err := r.ReportBuildFailure(context.Background(), job, errors.New("compile error"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/repos/acme/widgets/issues" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
	}
	if !strings.Contains(gotBody.Title, "widgets") {
		t.Errorf("title = %q, want to mention subdomain", gotBody.Title)
	}
	if !strings.Contains(gotBody.Body, "compile error") {
		t.Errorf("body = %q, want to contain the build error", gotBody.Body)
	}
	if !strings.Contains(gotBody.Body, "qtidocs") || !strings.Contains(gotBody.Body, "main") {
		t.Errorf("body = %q, want to mention path and ref", gotBody.Body)
	}
}

func TestReportBuildFailure_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	r := GitHubReporter{Token: "tok", BaseURL: srv.URL}
	err := r.ReportBuildFailure(context.Background(), build.Job{Owner: "o", Repo: "r"}, errors.New("boom"))
	if err == nil {
		t.Fatal("expected error for 403 status, got nil")
	}
}

func TestReportBuildFailure_NetworkError(t *testing.T) {
	r := GitHubReporter{Token: "tok", BaseURL: "http://127.0.0.1:0"}
	err := r.ReportBuildFailure(context.Background(), build.Job{Owner: "o", Repo: "r"}, errors.New("boom"))
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

func TestGitHubReporter_Defaults(t *testing.T) {
	r := GitHubReporter{}
	if r.baseURL() != "https://api.github.com" {
		t.Errorf("baseURL() = %q, want default", r.baseURL())
	}
	if r.client() != http.DefaultClient {
		t.Error("client() should default to http.DefaultClient")
	}
}

func TestGitHubReporter_Overrides(t *testing.T) {
	custom := &http.Client{}
	r := GitHubReporter{BaseURL: "https://example.com", Client: custom}
	if r.baseURL() != "https://example.com" {
		t.Errorf("baseURL() = %q", r.baseURL())
	}
	if r.client() != custom {
		t.Error("client() should return the overridden client")
	}
}
