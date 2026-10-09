package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPBuilder_Enqueue(t *testing.T) {
	var gotAuth string
	var gotJob BuildJob
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotJob)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	b := HTTPBuilder{URL: srv.URL, Secret: "sekret"}
	job := BuildJob{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"}
	if err := b.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer sekret" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotJob.Subdomain != "acme" {
		t.Errorf("gotJob = %+v", gotJob)
	}
}

func TestHTTPBuilder_Enqueue_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	b := HTTPBuilder{URL: srv.URL, Secret: "sekret"}
	err := b.Enqueue(context.Background(), BuildJob{Subdomain: "acme"})
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}

func TestHTTPBuilder_Teardown(t *testing.T) {
	var gotAuth string
	var gotBody struct {
		Subdomain string `json:"subdomain"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	b := HTTPBuilder{TeardownURL: srv.URL, Secret: "sekret"}
	if err := b.Teardown(context.Background(), "acme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer sekret" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotBody.Subdomain != "acme" {
		t.Errorf("gotBody.Subdomain = %q", gotBody.Subdomain)
	}
}

func TestHTTPBuilder_Teardown_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	b := HTTPBuilder{TeardownURL: srv.URL, Secret: "sekret"}
	err := b.Teardown(context.Background(), "acme")
	if err == nil {
		t.Fatal("expected error for 403 status, got nil")
	}
}

func TestHTTPBuilder_DefaultClient(t *testing.T) {
	b := HTTPBuilder{}
	if b.client() != http.DefaultClient {
		t.Error("client() should default to http.DefaultClient")
	}
}
