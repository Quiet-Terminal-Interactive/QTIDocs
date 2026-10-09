package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/mailer"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type fakeStore struct {
	entries map[string]storage.Entry
	getErr  error
	setErr  error
	listErr error
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

type fakeBuilder struct {
	enqueued    []BuildJob
	tornDown    []string
	enqueueErr  error
	teardownErr error
}

func (f *fakeBuilder) Enqueue(ctx context.Context, job BuildJob) error {
	if f.enqueueErr != nil {
		return f.enqueueErr
	}
	f.enqueued = append(f.enqueued, job)
	return nil
}
func (f *fakeBuilder) Teardown(ctx context.Context, subdomain string) error {
	if f.teardownErr != nil {
		return f.teardownErr
	}
	f.tornDown = append(f.tornDown, subdomain)
	return nil
}

type fakeMailer struct {
	sent    []mailer.Message
	sendErr error
}

func (f *fakeMailer) Send(ctx context.Context, msg mailer.Message) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, msg)
	return nil
}

func doRegister(t *testing.T, store storage.Store, builder Builder, secret string, body any, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	return doRegisterWithMailer(t, store, builder, &fakeMailer{}, secret, body, authHeader)
}

func doRegisterWithMailer(t *testing.T, store storage.Store, builder Builder, mail mailer.Mailer, secret string, body any, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(data))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	RegisterHandler(store, builder, mail, secret).ServeHTTP(rec, req)
	return rec
}

func TestRegisterHandler_Success_NewRegistration(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	mail := &fakeMailer{}
	body := registerRequest{Subdomain: "acme", Repo: "github.com/acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}

	rec := doRegisterWithMailer(t, store, builder, mail, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	entry := store.entries["acme"]
	if entry.Repo != "github.com/acme/widgets" || entry.WebhookSecret == "" {
		t.Fatalf("entry = %+v", entry)
	}
	if strings.Contains(rec.Body.String(), entry.WebhookSecret) {
		t.Errorf("response leaks the webhook secret: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"secret_emailed":true`) {
		t.Errorf("expected secret_emailed true, got: %s", rec.Body.String())
	}
	if len(mail.sent) != 1 {
		t.Fatalf("sent %d emails, want 1", len(mail.sent))
	}
	msg := mail.sent[0]
	if msg.To != "dev@example.com" {
		t.Errorf("To = %q", msg.To)
	}
	for _, want := range []string{entry.WebhookSecret, "QTIDOCS_DEPLOY_SECRET", "gh secret set QTIDOCS_DEPLOY_SECRET --repo acme/widgets", deployTemplateURL} {
		if !strings.Contains(msg.Body, want) {
			t.Errorf("email body missing %q:\n%s", want, msg.Body)
		}
	}
	if len(builder.enqueued) != 1 || builder.enqueued[0].Subdomain != "acme" {
		t.Errorf("enqueued = %+v", builder.enqueued)
	}
}

func TestRegisterHandler_NewRegistration_NonMainBranchInstructions(t *testing.T) {
	mail := &fakeMailer{}
	body := registerRequest{Subdomain: "acme", Repo: "github.com/acme/widgets", Branch: "docs", Path: "qtidocs", Contact: "dev@example.com"}
	rec := doRegisterWithMailer(t, newFakeStore(), &fakeBuilder{}, mail, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(mail.sent) != 1 || !strings.Contains(mail.sent[0].Body, "from main to docs") {
		t.Errorf("expected branch-change instructions, got: %+v", mail.sent)
	}
}

func TestRegisterHandler_MailFailure_DoesNotPersist(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	mail := &fakeMailer{sendErr: errors.New("smtp down")}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}

	rec := doRegisterWithMailer(t, store, builder, mail, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
	if _, ok := store.entries["acme"]; ok {
		t.Error("entry was stored even though the secret was never delivered")
	}
	if len(builder.enqueued) != 0 {
		t.Errorf("enqueued = %+v, want none", builder.enqueued)
	}
}

func TestRegisterHandler_InvalidContact(t *testing.T) {
	mail := &fakeMailer{}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com\r\nBcc: evil@example.com"}
	rec := doRegisterWithMailer(t, newFakeStore(), &fakeBuilder{}, mail, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if len(mail.sent) != 0 {
		t.Errorf("sent = %+v, want none", mail.sent)
	}
}

func TestRegisterHandler_Reregistration_PreservesSecretAndOutputDir(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{
		Subdomain: "acme", Repo: "github.com/acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com",
		OutputDir: "/old/output", WebhookSecret: "existing-secret",
	}
	store.entries["pr-3.acme"] = storage.Entry{Subdomain: "pr-3.acme"}
	builder := &fakeBuilder{}
	mail := &fakeMailer{}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "develop", Path: "docs", Contact: "DEV@example.com"}

	rec := doRegisterWithMailer(t, store, builder, mail, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(mail.sent) != 0 {
		t.Errorf("re-registration emailed the existing secret: %+v", mail.sent)
	}
	if !strings.Contains(rec.Body.String(), `"secret_emailed":false`) {
		t.Errorf("expected secret_emailed false for re-registration, got: %s", rec.Body.String())
	}
	if len(builder.tornDown) != 0 {
		t.Errorf("tornDown = %v, want previews kept", builder.tornDown)
	}

	entry := store.entries["acme"]
	if entry.WebhookSecret != "existing-secret" {
		t.Errorf("WebhookSecret = %q, want preserved existing-secret", entry.WebhookSecret)
	}
	if entry.OutputDir != "/old/output" {
		t.Errorf("OutputDir = %q, want preserved", entry.OutputDir)
	}
	if entry.Branch != "develop" || entry.Path != "docs" {
		t.Errorf("entry = %+v, want updated branch/path", entry)
	}
}

func TestRegisterHandler_Reregistration_RotatesSecretOnOwnerChange(t *testing.T) {
	cases := map[string]struct {
		existingContact string
		req             registerRequest
	}{
		"repo changed": {
			existingContact: "dev@example.com",
			req:             registerRequest{Subdomain: "acme", Repo: "github.com/someone/else", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"},
		},
		"contact changed": {
			existingContact: "dev@example.com",
			req:             registerRequest{Subdomain: "acme", Repo: "github.com/acme/widgets", Branch: "main", Path: "qtidocs", Contact: "new@example.com"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store := newFakeStore()
			store.entries["acme"] = storage.Entry{
				Subdomain: "acme", Repo: "github.com/acme/widgets", Branch: "main", Path: "qtidocs", Contact: tc.existingContact,
				OutputDir: "/old/output", WebhookSecret: "existing-secret",
			}
			store.entries["pr-3.acme"] = storage.Entry{Subdomain: "pr-3.acme"}
			store.entries["pr-3.acmeco"] = storage.Entry{Subdomain: "pr-3.acmeco"}
			builder := &fakeBuilder{}
			mail := &fakeMailer{}

			rec := doRegisterWithMailer(t, store, builder, mail, "sekret", tc.req, "Bearer sekret")
			if rec.Code != http.StatusAccepted {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), `"secret_emailed":true`) {
				t.Errorf("expected secret_emailed true, got: %s", rec.Body.String())
			}

			entry := store.entries["acme"]
			if entry.WebhookSecret == "" || entry.WebhookSecret == "existing-secret" {
				t.Errorf("WebhookSecret = %q, want a new secret", entry.WebhookSecret)
			}
			if len(mail.sent) != 1 || mail.sent[0].To != tc.req.Contact || !strings.Contains(mail.sent[0].Body, entry.WebhookSecret) {
				t.Errorf("sent = %+v, want the new secret emailed to %s", mail.sent, tc.req.Contact)
			}
			if len(builder.tornDown) != 1 || builder.tornDown[0] != "pr-3.acme" {
				t.Errorf("tornDown = %v, want [pr-3.acme]", builder.tornDown)
			}
		})
	}
}

func TestRegisterHandler_Reregistration_LegacyEntryWithoutContactKeepsSecret(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{
		Subdomain: "acme", Repo: "github.com/acme/widgets", Branch: "main", Path: "qtidocs", WebhookSecret: "existing-secret",
	}
	mail := &fakeMailer{}
	body := registerRequest{Subdomain: "acme", Repo: "github.com/acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}

	rec := doRegisterWithMailer(t, store, &fakeBuilder{}, mail, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	entry := store.entries["acme"]
	if entry.WebhookSecret != "existing-secret" || len(mail.sent) != 0 {
		t.Errorf("WebhookSecret = %q, sent = %+v, want secret kept and nothing emailed", entry.WebhookSecret, mail.sent)
	}
	if entry.Contact != "dev@example.com" {
		t.Errorf("Contact = %q, want recorded", entry.Contact)
	}
}

func TestRegisterHandler_WrongMethod(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()
	RegisterHandler(store, builder, &fakeMailer{}, "sekret").ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestRegisterHandler_Unauthorized_WrongSecret(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}
	rec := doRegister(t, store, builder, "sekret", body, "Bearer wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRegisterHandler_Unauthorized_NoHeader(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}
	rec := doRegister(t, store, builder, "sekret", body, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRegisterHandler_Unauthorized_EmptyConfiguredSecret(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}
	rec := doRegister(t, store, builder, "", body, "Bearer ")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRegisterHandler_InvalidBody(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer sekret")
	rec := httptest.NewRecorder()
	RegisterHandler(store, builder, &fakeMailer{}, "sekret").ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestRegisterHandler_MissingRequiredFields(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{}
	body := registerRequest{Subdomain: "acme"}
	rec := doRegister(t, store, builder, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestRegisterHandler_StoreGetError(t *testing.T) {
	store := newFakeStore()
	store.getErr = errors.New("disk error")
	builder := &fakeBuilder{}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}
	rec := doRegister(t, store, builder, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestRegisterHandler_StoreSetError(t *testing.T) {
	store := newFakeStore()
	store.setErr = errors.New("disk full")
	builder := &fakeBuilder{}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}
	rec := doRegister(t, store, builder, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestRegisterHandler_BuilderEnqueueError(t *testing.T) {
	store := newFakeStore()
	builder := &fakeBuilder{enqueueErr: errors.New("worker down")}
	body := registerRequest{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", Contact: "dev@example.com"}
	rec := doRegister(t, store, builder, "sekret", body, "Bearer sekret")
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
	if _, ok := store.entries["acme"]; !ok {
		t.Error("expected entry to still be written even though enqueue failed")
	}
}

func TestValidBearer(t *testing.T) {
	if !validBearer("Bearer sekret", "sekret") {
		t.Error("expected valid bearer token to match")
	}
	if validBearer("Bearer wrong", "sekret") {
		t.Error("expected mismatched token to fail")
	}
	if validBearer("sekret", "sekret") {
		t.Error("expected missing Bearer prefix to fail")
	}
	if validBearer("", "sekret") {
		t.Error("expected empty header to fail")
	}
}
