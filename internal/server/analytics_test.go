package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/analytics"
)

func resolvableFixture(t *testing.T) FixtureResolver {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "acme"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return FixtureResolver{Root: root}
}

type fakeAnalyticsStore struct {
	recorded  []analytics.Event
	recordErr error
	aggregate analytics.Summary
	aggErr    error
	lastFrom  *time.Time
	lastTo    time.Time
}

func (f *fakeAnalyticsStore) Record(subdomain string, e analytics.Event) error {
	if f.recordErr != nil {
		return f.recordErr
	}
	f.recorded = append(f.recorded, e)
	return nil
}
func (f *fakeAnalyticsStore) Rollup(subdomain string, day time.Time) error   { return nil }
func (f *fakeAnalyticsStore) Purge(subdomain string, before time.Time) error { return nil }
func (f *fakeAnalyticsStore) Aggregate(subdomain string, from *time.Time, to time.Time) (analytics.Summary, error) {
	f.lastFrom, f.lastTo = from, to
	if f.aggErr != nil {
		return analytics.Summary{}, f.aggErr
	}
	return f.aggregate, nil
}
func (f *fakeAnalyticsStore) Subdomains() ([]string, error) { return nil, nil }

func TestHandleCollect_RecordsEvent(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}

	body, _ := json.Marshal(collectRequest{Path: "/guides", Referrer: "https://google.com"})
	req := httptest.NewRequest(http.MethodPost, "http://acme.qtidocs.dev/_qtidocs/collect", bytes.NewReader(body))
	req.RemoteAddr = "1.2.3.4:5555"
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(store.recorded) != 1 {
		t.Fatalf("recorded = %+v, want 1 event", store.recorded)
	}
	if store.recorded[0].Path != "/guides" {
		t.Errorf("Path = %q", store.recorded[0].Path)
	}
}

func TestHandleCollect_DisabledWithoutAnalytics(t *testing.T) {
	s := &Server{Resolver: resolvableFixture(t)}
	req := httptest.NewRequest(http.MethodPost, "http://acme.qtidocs.dev/_qtidocs/collect", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when Analytics is disabled", rec.Code)
	}
}

func TestHandleCollect_WrongMethod(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_qtidocs/collect", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandleCollect_InvalidJSON(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	req := httptest.NewRequest(http.MethodPost, "http://acme.qtidocs.dev/_qtidocs/collect", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCollect_BodyTooLarge(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	big := bytes.Repeat([]byte("a"), maxCollectBody+100)
	req := httptest.NewRequest(http.MethodPost, "http://acme.qtidocs.dev/_qtidocs/collect", bytes.NewReader(big))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (MaxBytesReader rejects oversized body)", rec.Code)
	}
}

func TestHandleCollect_NeverFailsVisitorOnStoreError(t *testing.T) {
	store := &fakeAnalyticsStore{recordErr: errWant}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	body, _ := json.Marshal(collectRequest{Path: "/x"})
	req := httptest.NewRequest(http.MethodPost, "http://acme.qtidocs.dev/_qtidocs/collect", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204 even on store error (best-effort)", rec.Code)
	}
}

func TestHandleStatsData_Success(t *testing.T) {
	store := &fakeAnalyticsStore{aggregate: analytics.Summary{Views: 42, Uniques: 10}}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}

	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_qtidocs/stats.json?range=30d", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp statsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Views != 42 || resp.Uniques != 10 || resp.Range != "30d" {
		t.Errorf("resp = %+v", resp)
	}
}

func TestHandleStatsData_DefaultRange(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_qtidocs/stats.json", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	var resp statsResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Range != "7d" {
		t.Errorf("Range = %q, want 7d default", resp.Range)
	}
}

func TestHandleStatsData_UnknownRange(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_qtidocs/stats.json?range=bogus", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleStatsData_DisabledWithoutAnalytics(t *testing.T) {
	s := &Server{Resolver: resolvableFixture(t)}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_qtidocs/stats.json", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleStatsData_AggregateError(t *testing.T) {
	store := &fakeAnalyticsStore{aggErr: errWant}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_qtidocs/stats.json", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestHandleStatsData_WrongMethod(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	req := httptest.NewRequest(http.MethodPost, "http://acme.qtidocs.dev/_qtidocs/stats.json", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandleStatsPage_Success(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: resolvableFixture(t), Analytics: store}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_stats", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

type titledResolver struct{ title string }

func (t titledResolver) Resolve(subdomain string) (string, bool) { return "", true }
func (t titledResolver) Title(subdomain string) (string, bool)   { return t.title, true }

func TestHandleStatsPage_UsesResolvedTitle(t *testing.T) {
	store := &fakeAnalyticsStore{}
	s := &Server{Resolver: titledResolver{title: "Acme Docs"}, Analytics: store}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_stats", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Acme Docs")) {
		t.Errorf("expected page to show resolved title, got: %s", rec.Body.String())
	}
}

func TestHandleStatsPage_DisabledWithoutAnalytics(t *testing.T) {
	s := &Server{Resolver: resolvableFixture(t)}
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_stats", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestStatsRangeFrom(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		rng     string
		wantOK  bool
		wantNil bool
	}{
		{"today", true, false},
		{"7d", true, false},
		{"30d", true, false},
		{"all", true, true},
		{"bogus", false, true},
	}
	for _, c := range cases {
		from, ok := statsRangeFrom(c.rng, now)
		if ok != c.wantOK {
			t.Errorf("statsRangeFrom(%q) ok = %v, want %v", c.rng, ok, c.wantOK)
		}
		if (from == nil) != c.wantNil {
			t.Errorf("statsRangeFrom(%q) from=%v, wantNil=%v", c.rng, from, c.wantNil)
		}
	}
}

func TestStatsRangeFrom_7dIsSixDaysBeforeToday(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	from, _ := statsRangeFrom("7d", now)
	want := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	if !from.Equal(want) {
		t.Errorf("from = %v, want %v", from, want)
	}
}

func TestCleanPagePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "/"},
		{"/guides", "/guides"},
		{"guides", "/guides"},
		{"/guides/../secret", "/secret"},
		{"//double-slash", "/double-slash"},
	}
	for _, c := range cases {
		if got := cleanPagePath(c.in); got != c.want {
			t.Errorf("cleanPagePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCleanPagePath_Truncates(t *testing.T) {
	long := "/" + string(bytes.Repeat([]byte("a"), maxFieldLen+50))
	got := cleanPagePath(long)
	if len(got) > maxFieldLen+1 {
		t.Errorf("len(got) = %d, want truncated to around maxFieldLen", len(got))
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short string = %q", got)
	}
	if got := truncate("hello world", 5); got != "hello" {
		t.Errorf("truncate long string = %q, want hello", got)
	}
}

func TestRemoteIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	if got := remoteIP(req); got != "1.2.3.4" {
		t.Errorf("remoteIP() = %q, want 1.2.3.4", got)
	}
}

func TestRemoteIP_NoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-a-host-port"
	if got := remoteIP(req); got != "not-a-host-port" {
		t.Errorf("remoteIP() = %q, want raw fallback", got)
	}
}

var errWant = errors.New("boom")
