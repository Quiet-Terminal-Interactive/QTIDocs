package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type fakeStore struct {
	entries map[string]storage.Entry
	listErr error
	getErr  error
	setErr  error
}

func newFakeStore() *fakeStore {
	return &fakeStore{entries: map[string]storage.Entry{}}
}
func (f *fakeStore) Get(subdomain string) (storage.Entry, bool, error) {
	if f.getErr != nil {
		return storage.Entry{}, false, f.getErr
	}
	e, ok := f.entries[subdomain]
	return e, ok, nil
}
func (f *fakeStore) Set(e storage.Entry) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.entries[e.Subdomain] = e
	return nil
}
func (f *fakeStore) List() ([]storage.Entry, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]storage.Entry, 0, len(f.entries))
	for _, e := range f.entries {
		out = append(out, e)
	}
	return out, nil
}
func (f *fakeStore) Delete(subdomain string) error {
	delete(f.entries, subdomain)
	return nil
}

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerify_ValidSignature(t *testing.T) {
	body := []byte(`{"repo":"acme/widgets","ref":"abc123"}`)
	sig := sign("sekret", body)
	if !Verify("sekret", body, sig) {
		t.Error("expected valid signature to verify")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	body := []byte(`{"repo":"acme/widgets"}`)
	sig := sign("sekret", body)
	if Verify("wrong-secret", body, sig) {
		t.Error("expected wrong secret to fail verification")
	}
}

func TestVerify_TamperedBody(t *testing.T) {
	body := []byte(`{"repo":"acme/widgets"}`)
	sig := sign("sekret", body)
	if Verify("sekret", []byte(`{"repo":"evil/widgets"}`), sig) {
		t.Error("expected tampered body to fail verification")
	}
}

func TestVerify_MissingPrefix(t *testing.T) {
	body := []byte("data")
	if Verify("sekret", body, "not-a-valid-header") {
		t.Error("expected malformed header to fail")
	}
}

func TestVerify_InvalidHex(t *testing.T) {
	if Verify("sekret", []byte("data"), "sha256=not-hex!!") {
		t.Error("expected invalid hex to fail")
	}
}

func TestMatchingEntries_EmptySecretNeverMatches(t *testing.T) {
	store := newFakeStore()
	store.entries["pr-1.acme"] = storage.Entry{Subdomain: "pr-1.acme", Repo: "acme/widgets", WebhookSecret: ""}

	body := []byte("data")
	sig := sign("", body)
	matches, err := matchingEntries(store, body, sig, "acme", "widgets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("matches = %+v, want none (empty-secret entry must never match)", matches)
	}
}

func doWebhookRequest(t *testing.T, h http.Handler, body []byte, sigHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/deploy", bytes.NewReader(body))
	if sigHeader != "" {
		req.Header.Set(SignatureHeader, sigHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_Success(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}

	var enqueued []build.Job
	h := Handler(store, func(j build.Job) { enqueued = append(enqueued, j) })

	body, _ := json.Marshal(deployRequest{Repo: "acme/widgets", Ref: "sha123"})
	rec := doWebhookRequest(t, h, body, sign("sekret", body))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(enqueued) != 1 || enqueued[0].Ref != "sha123" || enqueued[0].Subdomain != "acme" {
		t.Errorf("enqueued = %+v", enqueued)
	}
}

func TestHandler_RepoMatchIsCaseInsensitive(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "github.com/acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}

	var enqueued []build.Job
	h := Handler(store, func(j build.Job) { enqueued = append(enqueued, j) })

	body, _ := json.Marshal(deployRequest{Repo: "Acme/Widgets", Ref: "sha123"})
	rec := doWebhookRequest(t, h, body, sign("sekret", body))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(enqueued) != 1 {
		t.Errorf("enqueued = %+v", enqueued)
	}
}

func TestHandler_WrongSignatureUnauthorized(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}

	h := Handler(store, func(j build.Job) {})
	body, _ := json.Marshal(deployRequest{Repo: "acme/widgets", Ref: "sha123"})
	rec := doWebhookRequest(t, h, body, sign("wrong", body))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandler_UnregisteredRepoUnauthorized(t *testing.T) {
	store := newFakeStore()
	h := Handler(store, func(j build.Job) {})
	body, _ := json.Marshal(deployRequest{Repo: "ghost/repo", Ref: "sha123"})
	rec := doWebhookRequest(t, h, body, sign("anything", body))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandler_WrongMethod(t *testing.T) {
	store := newFakeStore()
	h := Handler(store, func(j build.Job) {})
	req := httptest.NewRequest(http.MethodGet, "/deploy", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	store := newFakeStore()
	h := Handler(store, func(j build.Job) {})
	rec := doWebhookRequest(t, h, []byte("not json"), "sha256=abcd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_MissingFields(t *testing.T) {
	store := newFakeStore()
	h := Handler(store, func(j build.Job) {})
	body, _ := json.Marshal(deployRequest{Repo: "acme/widgets"})
	rec := doWebhookRequest(t, h, body, "sha256=abcd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_InvalidRepoFormat(t *testing.T) {
	store := newFakeStore()
	h := Handler(store, func(j build.Job) {})
	body, _ := json.Marshal(deployRequest{Repo: "not-a-valid-repo", Ref: "sha"})
	rec := doWebhookRequest(t, h, body, "sha256=abcd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_BodyTooLarge(t *testing.T) {
	store := newFakeStore()
	h := Handler(store, func(j build.Job) {})
	big := bytes.Repeat([]byte("a"), maxBodyBytes+10)
	rec := doWebhookRequest(t, h, big, "sha256=abcd")
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}

func TestHandler_PreviewEntryCannotAuthenticate(t *testing.T) {
	store := newFakeStore()
	store.entries["pr-1.acme"] = storage.Entry{Subdomain: "pr-1.acme", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: ""}

	h := Handler(store, func(j build.Job) {})
	body, _ := json.Marshal(deployRequest{Repo: "acme/widgets", Ref: "sha123"})
	rec := doWebhookRequest(t, h, body, sign("", body))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandler_StoreListError(t *testing.T) {
	store := newFakeStore()
	store.listErr = errWebhookTest
	h := Handler(store, func(j build.Job) {})
	body, _ := json.Marshal(deployRequest{Repo: "acme/widgets", Ref: "sha"})
	rec := doWebhookRequest(t, h, body, "sha256=abcd")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestHandler_MultipleMatchingEntries(t *testing.T) {
	store := newFakeStore()
	store.entries["acme1"] = storage.Entry{Subdomain: "acme1", Repo: "acme/widgets", Path: "qtidocs", WebhookSecret: "sekret"}
	store.entries["acme2"] = storage.Entry{Subdomain: "acme2", Repo: "acme/widgets", Path: "docs", WebhookSecret: "sekret"}

	var enqueued []build.Job
	h := Handler(store, func(j build.Job) { enqueued = append(enqueued, j) })
	body, _ := json.Marshal(deployRequest{Repo: "acme/widgets", Ref: "sha123"})
	rec := doWebhookRequest(t, h, body, sign("sekret", body))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(enqueued) != 2 {
		t.Errorf("enqueued = %+v, want 2 jobs", enqueued)
	}
}

func TestJobFor_Unversioned(t *testing.T) {
	e := storage.Entry{Subdomain: "acme", Title: "Acme", Path: "qtidocs"}
	job := jobFor(e, "acme", "widgets", "sha123")
	if job.Ref != "sha123" || job.Owner != "acme" || job.Repo != "widgets" || job.Path != "qtidocs" {
		t.Errorf("job = %+v", job)
	}
	if len(job.Versions) != 0 {
		t.Errorf("Versions = %v, want empty", job.Versions)
	}
}

func TestJobFor_Versioned(t *testing.T) {
	e := storage.Entry{
		Subdomain: "acme", Path: "qtidocs",
		Versions:       []storage.Version{{Name: "v1", Ref: "v1-branch"}, {Name: "v2", Ref: "v2-branch"}},
		DefaultVersion: "v2",
	}
	job := jobFor(e, "acme", "widgets", "ignored-ref")
	if len(job.Versions) != 2 {
		t.Fatalf("Versions = %+v, want 2", job.Versions)
	}
	if job.DefaultVersion != "v2" {
		t.Errorf("DefaultVersion = %q, want v2", job.DefaultVersion)
	}
	if job.Versions[0].Name != "v1" || job.Versions[0].Ref != "v1-branch" {
		t.Errorf("Versions[0] = %+v", job.Versions[0])
	}
}

var errWebhookTest = errors.New("boom")
