package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

func doInternalRequest(t *testing.T, h http.Handler, body []byte, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/internal/build", bytes.NewReader(body))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestInternalBuildHandler_Success(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"}

	var enqueued []build.Job
	h := InternalBuildHandler(store, func(j build.Job) { enqueued = append(enqueued, j) }, "sekret")

	body, _ := json.Marshal(internalBuildRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(enqueued) != 1 || enqueued[0].Ref != "main" {
		t.Errorf("enqueued = %+v", enqueued)
	}
}

func TestInternalBuildHandler_Unauthorized(t *testing.T) {
	store := newFakeStore()
	h := InternalBuildHandler(store, func(j build.Job) {}, "sekret")
	body, _ := json.Marshal(internalBuildRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"})
	rec := doInternalRequest(t, h, body, "Bearer wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestInternalBuildHandler_EmptySecretAlwaysUnauthorized(t *testing.T) {
	store := newFakeStore()
	h := InternalBuildHandler(store, func(j build.Job) {}, "")
	body, _ := json.Marshal(internalBuildRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"})
	rec := doInternalRequest(t, h, body, "Bearer ")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestInternalBuildHandler_WrongMethod(t *testing.T) {
	store := newFakeStore()
	h := InternalBuildHandler(store, func(j build.Job) {}, "sekret")
	req := httptest.NewRequest(http.MethodGet, "/internal/build", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestInternalBuildHandler_InvalidBody(t *testing.T) {
	store := newFakeStore()
	h := InternalBuildHandler(store, func(j build.Job) {}, "sekret")
	rec := doInternalRequest(t, h, []byte("not json"), "Bearer sekret")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestInternalBuildHandler_MissingFields(t *testing.T) {
	store := newFakeStore()
	h := InternalBuildHandler(store, func(j build.Job) {}, "sekret")
	body, _ := json.Marshal(internalBuildRequest{Subdomain: "acme"})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestInternalBuildHandler_InvalidRepo(t *testing.T) {
	store := newFakeStore()
	h := InternalBuildHandler(store, func(j build.Job) {}, "sekret")
	body, _ := json.Marshal(internalBuildRequest{Subdomain: "acme", Repo: "not-valid", Branch: "main", Path: "qtidocs"})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestInternalBuildHandler_UnregisteredSubdomain(t *testing.T) {
	store := newFakeStore()
	h := InternalBuildHandler(store, func(j build.Job) {}, "sekret")
	body, _ := json.Marshal(internalBuildRequest{Subdomain: "ghost", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestInternalBuildHandler_StoreGetError(t *testing.T) {
	store := newFakeStore()
	store.getErr = errWebhookTest
	h := InternalBuildHandler(store, func(j build.Job) {}, "sekret")
	body, _ := json.Marshal(internalBuildRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestInternalTeardownHandler_Success(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	h := InternalTeardownHandler(store, "sekret")

	body, _ := json.Marshal(internalTeardownRequest{Subdomain: "acme"})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if _, ok := store.entries["acme"]; ok {
		t.Error("expected entry to be removed")
	}
}

func TestInternalTeardownHandler_Unauthorized(t *testing.T) {
	store := newFakeStore()
	h := InternalTeardownHandler(store, "sekret")
	body, _ := json.Marshal(internalTeardownRequest{Subdomain: "acme"})
	rec := doInternalRequest(t, h, body, "Bearer wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestInternalTeardownHandler_WrongMethod(t *testing.T) {
	store := newFakeStore()
	h := InternalTeardownHandler(store, "sekret")
	req := httptest.NewRequest(http.MethodGet, "/internal/teardown", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestInternalTeardownHandler_MissingSubdomain(t *testing.T) {
	store := newFakeStore()
	h := InternalTeardownHandler(store, "sekret")
	body, _ := json.Marshal(internalTeardownRequest{})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestInternalTeardownHandler_MissingEntryIsStillAccepted(t *testing.T) {
	store := newFakeStore()
	h := InternalTeardownHandler(store, "sekret")
	body, _ := json.Marshal(internalTeardownRequest{Subdomain: "ghost"})
	rec := doInternalRequest(t, h, body, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202 (idempotent teardown)", rec.Code)
	}
}
