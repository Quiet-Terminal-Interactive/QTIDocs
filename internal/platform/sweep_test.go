package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

func doSweep(t *testing.T, store storage.Store, builder Builder, secret string, body any, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/sweep", bytes.NewReader(data))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	SweepHandler(store, builder, secret).ServeHTTP(rec, req)
	return rec
}

func TestSweepHandler_TearsDownUnregisteredEntry(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	store.entries["stale"] = storage.Entry{Subdomain: "stale"}
	builder := &fakeBuilder{}

	rec := doSweep(t, store, builder, "sekret", sweepRequest{Registered: []string{"acme"}}, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(builder.tornDown) != 1 || builder.tornDown[0] != "stale" {
		t.Errorf("tornDown = %v, want [stale]", builder.tornDown)
	}
}

func TestSweepHandler_KeepsPreviewOfRegisteredSite(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	store.entries["pr-42.acme"] = storage.Entry{Subdomain: "pr-42.acme"}
	builder := &fakeBuilder{}

	rec := doSweep(t, store, builder, "sekret", sweepRequest{Registered: []string{"acme"}}, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(builder.tornDown) != 0 {
		t.Errorf("tornDown = %v, want none (preview of a registered site)", builder.tornDown)
	}
}

func TestSweepHandler_TearsDownPreviewOfUnregisteredSite(t *testing.T) {
	store := newFakeStore()
	store.entries["pr-42.acme"] = storage.Entry{Subdomain: "pr-42.acme"}
	builder := &fakeBuilder{}

	rec := doSweep(t, store, builder, "sekret", sweepRequest{Registered: []string{}}, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(builder.tornDown) != 1 || builder.tornDown[0] != "pr-42.acme" {
		t.Errorf("tornDown = %v, want [pr-42.acme]", builder.tornDown)
	}
}

func TestSweepHandler_RegisteredEntriesUntouched(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	builder := &fakeBuilder{}

	rec := doSweep(t, store, builder, "sekret", sweepRequest{Registered: []string{"acme"}}, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(builder.tornDown) != 0 {
		t.Errorf("tornDown = %v, want none", builder.tornDown)
	}
}

func TestSweepHandler_WrongMethod(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	req := httptest.NewRequest(http.MethodGet, "/sweep", nil)
	rec := httptest.NewRecorder()
	SweepHandler(store, builder, "sekret").ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestSweepHandler_Unauthorized(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	rec := doSweep(t, store, builder, "sekret", sweepRequest{}, "Bearer wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestSweepHandler_InvalidBody(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	req := httptest.NewRequest(http.MethodPost, "/sweep", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer sekret")
	rec := httptest.NewRecorder()
	SweepHandler(store, builder, "sekret").ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestSweepHandler_StoreListError(t *testing.T) {
	store := newFakeStore()
	store.listErr = errors.New("disk error")
	builder := &fakeBuilder{}
	rec := doSweep(t, store, builder, "sekret", sweepRequest{}, "Bearer sekret")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestSweepHandler_TeardownErrorContinuesOtherEntries(t *testing.T) {
	store := newFakeStore()
	store.entries["bad"] = storage.Entry{Subdomain: "bad"}
	store.entries["good"] = storage.Entry{Subdomain: "good"}
	builder := &fakeBuilder{teardownErr: errors.New("fail")}

	rec := doSweep(t, store, builder, "sekret", sweepRequest{Registered: []string{}}, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(builder.tornDown) != 0 {
		t.Errorf("tornDown = %v, want none since teardownErr always fails", builder.tornDown)
	}
}

func TestSweepHandler_ReportsCorrectTornDownCount(t *testing.T) {
	store := newFakeStore()
	store.entries["a"] = storage.Entry{Subdomain: "a"}
	store.entries["b"] = storage.Entry{Subdomain: "b"}
	builder := &fakeBuilder{}

	rec := doSweep(t, store, builder, "sekret", sweepRequest{Registered: []string{}}, "Bearer sekret")
	var resp struct {
		TornDown int `json:"torn_down"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TornDown != 2 {
		t.Errorf("torn_down = %d, want 2", resp.TornDown)
	}
}
