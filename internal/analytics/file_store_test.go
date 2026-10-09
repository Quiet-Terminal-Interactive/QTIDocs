package analytics

import (
	"testing"
	"time"
)

func newTestFileStore(t *testing.T) *FileStore {
	t.Helper()
	return Open(t.TempDir())
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestFileStore_RecordAndSubdomains(t *testing.T) {
	s := newTestFileStore(t)
	err := s.Record("acme", Event{Path: "/", Timestamp: day(2026, 1, 1), VisitorHash: "h1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	subs, err := s.Subdomains()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 || subs[0] != "acme" {
		t.Errorf("subs = %v, want [acme]", subs)
	}
}

func TestFileStore_Subdomains_Empty(t *testing.T) {
	s := newTestFileStore(t)
	subs, err := s.Subdomains()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("subs = %v, want empty", subs)
	}
}

func TestFileStore_RollupComputesAggregate(t *testing.T) {
	s := newTestFileStore(t)
	d := day(2026, 1, 1)
	s.Record("acme", Event{Path: "/", Timestamp: d, VisitorHash: "h1", Referrer: "https://google.com/x"})
	s.Record("acme", Event{Path: "/", Timestamp: d, VisitorHash: "h2"})
	s.Record("acme", Event{Path: "/about", Timestamp: d, VisitorHash: "h1"})

	if err := s.Rollup("acme", d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sum, err := s.Aggregate("acme", nil, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 3 {
		t.Errorf("Views = %d, want 3", sum.Views)
	}
	if sum.Uniques != 2 {
		t.Errorf("Uniques = %d, want 2", sum.Uniques)
	}
	wantPages := map[string]int{"/": 2, "/about": 1}
	for _, p := range sum.TopPages {
		if wantPages[p.Path] != p.Views {
			t.Errorf("TopPages entry %+v doesn't match want %v", p, wantPages)
		}
	}
}

func TestFileStore_Rollup_NoEventsForDay(t *testing.T) {
	s := newTestFileStore(t)
	d := day(2026, 1, 1)
	if err := s.Rollup("acme", d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sum, err := s.Aggregate("acme", nil, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 0 {
		t.Errorf("Views = %d, want 0", sum.Views)
	}
}

func TestFileStore_Aggregate_FallsBackToRawWhenNotRolledUp(t *testing.T) {
	s := newTestFileStore(t)
	d := day(2026, 1, 1)
	s.Record("acme", Event{Path: "/", Timestamp: d, VisitorHash: "h1"})

	sum, err := s.Aggregate("acme", nil, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 1 {
		t.Errorf("Views = %d, want 1", sum.Views)
	}
}

func TestFileStore_Aggregate_DateRangeFiltering(t *testing.T) {
	s := newTestFileStore(t)
	d1, d2, d3 := day(2026, 1, 1), day(2026, 1, 2), day(2026, 1, 3)
	s.Record("acme", Event{Path: "/", Timestamp: d1, VisitorHash: "h1"})
	s.Record("acme", Event{Path: "/", Timestamp: d2, VisitorHash: "h1"})
	s.Record("acme", Event{Path: "/", Timestamp: d3, VisitorHash: "h1"})

	sum, err := s.Aggregate("acme", &d2, d3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 2 {
		t.Errorf("Views = %d, want 2 (d2 and d3 only)", sum.Views)
	}
}

func TestFileStore_Aggregate_NilFromMeansAllTime(t *testing.T) {
	s := newTestFileStore(t)
	d1, d2 := day(2026, 1, 1), day(2026, 1, 2)
	s.Record("acme", Event{Path: "/", Timestamp: d1, VisitorHash: "h1"})
	s.Record("acme", Event{Path: "/", Timestamp: d2, VisitorHash: "h1"})

	sum, err := s.Aggregate("acme", nil, d2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 2 {
		t.Errorf("Views = %d, want 2", sum.Views)
	}
}

func TestFileStore_Aggregate_UnknownSubdomain(t *testing.T) {
	s := newTestFileStore(t)
	sum, err := s.Aggregate("ghost", nil, day(2026, 1, 1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 0 {
		t.Errorf("Views = %d, want 0", sum.Views)
	}
}

func TestFileStore_Purge_RemovesOldRawLogsButKeepsAggregates(t *testing.T) {
	s := newTestFileStore(t)
	oldDay := day(2026, 1, 1)
	newDay := day(2026, 2, 1)
	s.Record("acme", Event{Path: "/", Timestamp: oldDay, VisitorHash: "h1"})
	s.Record("acme", Event{Path: "/", Timestamp: newDay, VisitorHash: "h1"})

	if err := s.Rollup("acme", oldDay); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cutoff := day(2026, 1, 15)
	if err := s.Purge("acme", cutoff); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sum, err := s.Aggregate("acme", nil, newDay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 2 {
		t.Errorf("Views = %d, want 2 (1 from surviving aggregate, 1 from new raw log)", sum.Views)
	}
}

func TestFileStore_Purge_MissingSubdomainIsNoop(t *testing.T) {
	s := newTestFileStore(t)
	if err := s.Purge("ghost", day(2026, 1, 1)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFileStore_Rollup_Idempotent(t *testing.T) {
	s := newTestFileStore(t)
	d := day(2026, 1, 1)
	s.Record("acme", Event{Path: "/", Timestamp: d, VisitorHash: "h1"})

	if err := s.Rollup("acme", d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Rollup("acme", d); err != nil {
		t.Fatalf("unexpected error on second rollup: %v", err)
	}

	sum, err := s.Aggregate("acme", nil, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 1 {
		t.Errorf("Views = %d, want 1 (rollup should not double-count)", sum.Views)
	}
}

func TestFileStore_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	d := day(2026, 1, 1)

	s1 := Open(dir)
	s1.Record("acme", Event{Path: "/", Timestamp: d, VisitorHash: "h1"})
	s1.Rollup("acme", d)

	s2 := Open(dir)
	sum, err := s2.Aggregate("acme", nil, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Views != 1 {
		t.Errorf("Views = %d, want 1", sum.Views)
	}
}

func TestFileStore_TopPagesCappedAndRanked(t *testing.T) {
	s := newTestFileStore(t)
	d := day(2026, 1, 1)
	for i := 0; i < 12; i++ {
		views := 12 - i
		for v := 0; v < views; v++ {
			s.Record("acme", Event{Path: pathN(i), Timestamp: d, VisitorHash: "h"})
		}
	}

	sum, err := s.Aggregate("acme", nil, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sum.TopPages) != 10 {
		t.Fatalf("len(TopPages) = %d, want 10 (capped)", len(sum.TopPages))
	}
	for i := 1; i < len(sum.TopPages); i++ {
		if sum.TopPages[i].Views > sum.TopPages[i-1].Views {
			t.Errorf("TopPages not sorted descending: %+v", sum.TopPages)
		}
	}
	if sum.TopPages[0].Path != pathN(0) {
		t.Errorf("TopPages[0].Path = %q, want %q (most views)", sum.TopPages[0].Path, pathN(0))
	}
}

func pathN(i int) string {
	return "/page" + string(rune('a'+i))
}
