package server

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/analytics"
	"github.com/quiet-terminal-interactive/qtidocs/internal/theme"
)

const maxCollectBody = 4 << 10

const maxFieldLen = 300

type collectRequest struct {
	Path     string `json:"path"`
	Referrer string `json:"referrer"`
}

func (s *Server) handleCollect(w http.ResponseWriter, r *http.Request, subdomain string) {
	if s.Analytics == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req collectRequest
	body := http.MaxBytesReader(w, r.Body, maxCollectBody)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	event := analytics.Event{
		Path:        cleanPagePath(req.Path),
		Timestamp:   now,
		VisitorHash: analytics.HashVisitor(remoteIP(r), r.Header.Get("User-Agent"), now),
		Referrer:    truncate(req.Referrer, maxFieldLen),
	}

	if err := s.Analytics.Record(subdomain, event); err != nil {
		log.Printf("server: recording analytics event for %q: %v", subdomain, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func statsRangeFrom(rng string, now time.Time) (from *time.Time, ok bool) {
	today := truncateDay(now)
	switch rng {
	case "today":
		return &today, true
	case "7d":
		f := today.AddDate(0, 0, -6)
		return &f, true
	case "30d":
		f := today.AddDate(0, 0, -29)
		return &f, true
	case "all":
		return nil, true
	default:
		return nil, false
	}
}

func truncateDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

type statsResponse struct {
	Range     string                    `json:"range"`
	Views     int                       `json:"views"`
	Uniques   int                       `json:"uniques"`
	TopPages  []analytics.PageCount     `json:"top_pages"`
	Referrers []analytics.ReferrerCount `json:"referrers"`
}

func (s *Server) handleStatsData(w http.ResponseWriter, r *http.Request, subdomain string) {
	if s.Analytics == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rng := r.URL.Query().Get("range")
	if rng == "" {
		rng = "7d"
	}
	now := time.Now()
	from, ok := statsRangeFrom(rng, now)
	if !ok {
		http.Error(w, "unknown range", http.StatusBadRequest)
		return
	}

	summary, err := s.Analytics.Aggregate(subdomain, from, now)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(statsResponse{
		Range:     rng,
		Views:     summary.Views,
		Uniques:   summary.Uniques,
		TopPages:  summary.TopPages,
		Referrers: summary.Referrers,
	}); err != nil {
		log.Printf("server: encoding stats response for %q: %v", subdomain, err)
	}
}

func (s *Server) handleStatsPage(w http.ResponseWriter, r *http.Request, subdomain string) {
	if s.Analytics == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	title := subdomain
	if tr, ok := s.Resolver.(TitledResolver); ok {
		if t, ok := tr.Title(subdomain); ok {
			title = t
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := theme.RenderStats(w, theme.StatsPageData{SiteTitle: title}); err != nil {
		log.Printf("server: rendering stats page for %q: %v", subdomain, err)
	}
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func cleanPagePath(p string) string {
	p = truncate(p, maxFieldLen)
	if p == "" {
		return "/"
	}
	return path.Clean("/" + strings.TrimPrefix(p, "/"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
