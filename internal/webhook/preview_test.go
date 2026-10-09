package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

func TestPreviewSubdomain(t *testing.T) {
	got := PreviewSubdomain("acme", 42)
	if got != "pr-42.acme" {
		t.Errorf("PreviewSubdomain() = %q, want pr-42.acme", got)
	}
}

func TestPreviewDeployHandler_Success(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}

	var enqueued []build.Job
	h := PreviewDeployHandler(store, func(j build.Job) { enqueued = append(enqueued, j) })

	body, _ := json.Marshal(previewDeployRequest{Repo: "acme/widgets", PR: 7, Ref: "sha-pr7"})
	rec := doWebhookRequest(t, h, body, sign("sekret", body))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(enqueued) != 1 || enqueued[0].Subdomain != "pr-7.acme" || enqueued[0].Ref != "sha-pr7" {
		t.Errorf("enqueued = %+v", enqueued)
	}
	previewEntry, ok := store.entries["pr-7.acme"]
	if !ok {
		t.Fatal("expected preview entry to be created in the store")
	}
	if previewEntry.Branch != "" || previewEntry.WebhookSecret != "" {
		t.Errorf("preview entry = %+v, want empty Branch and WebhookSecret", previewEntry)
	}
}

func TestPreviewDeployHandler_PreservesExistingOutputDir(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}
	store.entries["pr-7.acme"] = storage.Entry{Subdomain: "pr-7.acme", OutputDir: "/old/preview/output"}

	h := PreviewDeployHandler(store, func(j build.Job) {})
	body, _ := json.Marshal(previewDeployRequest{Repo: "acme/widgets", PR: 7, Ref: "sha-pr7"})
	rec := doWebhookRequest(t, h, body, sign("sekret", body))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
	if store.entries["pr-7.acme"].OutputDir != "/old/preview/output" {
		t.Errorf("OutputDir = %q, want preserved", store.entries["pr-7.acme"].OutputDir)
	}
}

func TestPreviewDeployHandler_Unauthorized(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}
	h := PreviewDeployHandler(store, func(j build.Job) {})
	body, _ := json.Marshal(previewDeployRequest{Repo: "acme/widgets", PR: 7, Ref: "sha"})
	rec := doWebhookRequest(t, h, body, sign("wrong", body))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestPreviewDeployHandler_InvalidPR(t *testing.T) {
	store := newFakeStore()
	h := PreviewDeployHandler(store, func(j build.Job) {})
	body, _ := json.Marshal(previewDeployRequest{Repo: "acme/widgets", PR: 0, Ref: "sha"})
	rec := doWebhookRequest(t, h, body, "sha256=abcd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestPreviewDeployHandler_WrongMethod(t *testing.T) {
	store := newFakeStore()
	h := PreviewDeployHandler(store, func(j build.Job) {})
	req := httptest.NewRequest(http.MethodGet, "/preview/deploy", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestPreviewDeployHandler_InvalidBody(t *testing.T) {
	store := newFakeStore()
	h := PreviewDeployHandler(store, func(j build.Job) {})
	rec := doWebhookRequest(t, h, []byte("not json"), "sha256=abcd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestPreviewTeardownHandler_Success(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}
	store.entries["pr-7.acme"] = storage.Entry{Subdomain: "pr-7.acme"}

	h := PreviewTeardownHandler(store)
	body, _ := json.Marshal(previewTeardownRequest{Repo: "acme/widgets", PR: 7})
	rec := doWebhookRequest(t, h, body, sign("sekret", body))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if _, ok := store.entries["pr-7.acme"]; ok {
		t.Error("expected preview entry to be torn down")
	}
}

func TestPreviewTeardownHandler_Unauthorized(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}
	h := PreviewTeardownHandler(store)
	body, _ := json.Marshal(previewTeardownRequest{Repo: "acme/widgets", PR: 7})
	rec := doWebhookRequest(t, h, body, sign("wrong", body))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestPreviewTeardownHandler_InvalidPR(t *testing.T) {
	store := newFakeStore()
	h := PreviewTeardownHandler(store)
	body, _ := json.Marshal(previewTeardownRequest{Repo: "acme/widgets", PR: -1})
	rec := doWebhookRequest(t, h, body, "sha256=abcd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestPreviewTeardownHandler_WrongMethod(t *testing.T) {
	store := newFakeStore()
	h := PreviewTeardownHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/preview/teardown", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestPreviewTeardownHandler_MissingRepo(t *testing.T) {
	store := newFakeStore()
	h := PreviewTeardownHandler(store)
	body, _ := json.Marshal(previewTeardownRequest{PR: 7})
	rec := doWebhookRequest(t, h, body, "sha256=abcd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
