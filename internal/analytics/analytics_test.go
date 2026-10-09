package analytics

import (
	"errors"
	"testing"
	"time"
)

func TestHashVisitor_Deterministic(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := HashVisitor("1.2.3.4", "ua", day)
	b := HashVisitor("1.2.3.4", "ua", day)
	if a != b {
		t.Errorf("expected deterministic hash, got %q != %q", a, b)
	}
	if len(a) != 64 {
		t.Errorf("len(hash) = %d, want 64 (sha256 hex)", len(a))
	}
}

func TestHashVisitor_RotatesDaily(t *testing.T) {
	day1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	a := HashVisitor("1.2.3.4", "ua", day1)
	b := HashVisitor("1.2.3.4", "ua", day2)
	if a == b {
		t.Error("expected hash to differ across calendar days")
	}
}

func TestHashVisitor_SameCalendarDayDifferentTime(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	a := HashVisitor("1.2.3.4", "ua", t1)
	b := HashVisitor("1.2.3.4", "ua", t2)
	if a != b {
		t.Error("expected same hash within the same UTC calendar day")
	}
}

func TestHashVisitor_DifferentIPOrUA(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base := HashVisitor("1.2.3.4", "ua", day)
	diffIP := HashVisitor("5.6.7.8", "ua", day)
	diffUA := HashVisitor("1.2.3.4", "other-ua", day)
	if base == diffIP || base == diffUA {
		t.Error("expected hash to depend on both ip and ua")
	}
}

func TestReferrerLabel_Empty(t *testing.T) {
	if got := ReferrerLabel(""); got != "direct" {
		t.Errorf("ReferrerLabel(\"\") = %q, want direct", got)
	}
}

func TestReferrerLabel_InvalidURL(t *testing.T) {
	if got := ReferrerLabel("://not a url"); got != "direct" {
		t.Errorf("ReferrerLabel(invalid) = %q, want direct", got)
	}
}

func TestReferrerLabel_NoHost(t *testing.T) {
	if got := ReferrerLabel("/relative/path"); got != "direct" {
		t.Errorf("ReferrerLabel(relative) = %q, want direct", got)
	}
}

func TestReferrerLabel_ExtractsHost(t *testing.T) {
	if got := ReferrerLabel("https://www.google.com/search?q=x"); got != "www.google.com" {
		t.Errorf("ReferrerLabel = %q, want www.google.com", got)
	}
}

type fakeStore struct {
	subdomains  []string
	subsErr     error
	rollupCalls []string
	rollupErr   map[string]error
	purgeCalls  []string
	purgeErr    map[string]error
}

func (f *fakeStore) Record(subdomain string, e Event) error { return nil }
func (f *fakeStore) Rollup(subdomain string, day time.Time) error {
	f.rollupCalls = append(f.rollupCalls, subdomain)
	if f.rollupErr != nil {
		if err := f.rollupErr[subdomain]; err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeStore) Purge(subdomain string, before time.Time) error {
	f.purgeCalls = append(f.purgeCalls, subdomain)
	if f.purgeErr != nil {
		if err := f.purgeErr[subdomain]; err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeStore) Aggregate(subdomain string, from *time.Time, to time.Time) (Summary, error) {
	return Summary{}, nil
}
func (f *fakeStore) Subdomains() ([]string, error) {
	if f.subsErr != nil {
		return nil, f.subsErr
	}
	return f.subdomains, nil
}

func TestSweep_RollsUpAndPurgesEverySubdomain(t *testing.T) {
	s := &fakeStore{subdomains: []string{"a", "b"}}
	err := Sweep(s, 30*24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.rollupCalls) != 2 || len(s.purgeCalls) != 2 {
		t.Errorf("rollupCalls = %v, purgeCalls = %v", s.rollupCalls, s.purgeCalls)
	}
}

func TestSweep_SubdomainsError(t *testing.T) {
	s := &fakeStore{subsErr: errors.New("boom")}
	err := Sweep(s, time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSweep_RollupErrorStopsSweep(t *testing.T) {
	s := &fakeStore{
		subdomains: []string{"a", "b"},
		rollupErr:  map[string]error{"a": errors.New("boom")},
	}
	err := Sweep(s, time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(s.purgeCalls) != 0 {
		t.Errorf("purgeCalls = %v, want none (rollup failed first)", s.purgeCalls)
	}
}

func TestSweep_PurgeErrorPropagates(t *testing.T) {
	s := &fakeStore{
		subdomains: []string{"a"},
		purgeErr:   map[string]error{"a": errors.New("boom")},
	}
	err := Sweep(s, time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSweep_NoSubdomainsIsNoop(t *testing.T) {
	s := &fakeStore{}
	if err := Sweep(s, time.Hour, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
