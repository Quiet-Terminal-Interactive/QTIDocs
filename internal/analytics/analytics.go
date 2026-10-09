package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"time"
)

const dateLayout = "2006-01-02"

type Event struct {
	Path        string    `json:"path"`
	Timestamp   time.Time `json:"ts"`
	VisitorHash string    `json:"visitor_hash"`
	Referrer    string    `json:"referrer,omitempty"`
}

func HashVisitor(ip, ua string, day time.Time) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s", day.UTC().Format(dateLayout), ip, ua)
	return hex.EncodeToString(h.Sum(nil))
}

func ReferrerLabel(raw string) string {
	if raw == "" {
		return "direct"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "direct"
	}
	return u.Host
}

type DailyAggregate struct {
	Views     int            `json:"views"`
	ByPath    map[string]int `json:"by_path,omitempty"`
	Referrers map[string]int `json:"referrers,omitempty"`
	Uniques   int            `json:"uniques"`
}

func computeDaily(events []Event) DailyAggregate {
	agg := DailyAggregate{Views: len(events)}
	if len(events) == 0 {
		return agg
	}
	agg.ByPath = map[string]int{}
	agg.Referrers = map[string]int{}
	seen := map[string]bool{}
	for _, e := range events {
		agg.ByPath[e.Path]++
		agg.Referrers[ReferrerLabel(e.Referrer)]++
		if !seen[e.VisitorHash] {
			seen[e.VisitorHash] = true
			agg.Uniques++
		}
	}
	return agg
}

type PageCount struct {
	Path  string `json:"path"`
	Views int    `json:"views"`
}

type ReferrerCount struct {
	Referrer string `json:"referrer"`
	Views    int    `json:"views"`
}

const topN = 10

type Summary struct {
	Views     int
	Uniques   int
	TopPages  []PageCount
	Referrers []ReferrerCount
}

func merge(byPath, byReferrer map[string]int, views, uniques *int, day DailyAggregate) {
	*views += day.Views
	*uniques += day.Uniques
	for p, c := range day.ByPath {
		byPath[p] += c
	}
	for r, c := range day.Referrers {
		byReferrer[r] += c
	}
}

func rankPages(m map[string]int) []PageCount {
	out := make([]PageCount, 0, len(m))
	for p, c := range m {
		out = append(out, PageCount{Path: p, Views: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Views != out[j].Views {
			return out[i].Views > out[j].Views
		}
		return out[i].Path < out[j].Path
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out
}

func rankReferrers(m map[string]int) []ReferrerCount {
	out := make([]ReferrerCount, 0, len(m))
	for r, c := range m {
		out = append(out, ReferrerCount{Referrer: r, Views: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Views != out[j].Views {
			return out[i].Views > out[j].Views
		}
		return out[i].Referrer < out[j].Referrer
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out
}

type Store interface {
	Record(subdomain string, e Event) error
	Rollup(subdomain string, day time.Time) error
	Purge(subdomain string, before time.Time) error
	Aggregate(subdomain string, from *time.Time, to time.Time) (Summary, error)
	Subdomains() ([]string, error)
}

func Sweep(s Store, retention time.Duration, now time.Time) error {
	subdomains, err := s.Subdomains()
	if err != nil {
		return fmt.Errorf("analytics: listing subdomains: %w", err)
	}

	yesterday := now.UTC().AddDate(0, 0, -1)
	cutoff := now.UTC().Add(-retention)

	for _, sub := range subdomains {
		if err := s.Rollup(sub, yesterday); err != nil {
			return fmt.Errorf("analytics: rolling up %q: %w", sub, err)
		}
		if err := s.Purge(sub, cutoff); err != nil {
			return fmt.Errorf("analytics: purging %q: %w", sub, err)
		}
	}
	return nil
}
