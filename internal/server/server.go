package server

import (
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/quiet-terminal-interactive/qtidocs/internal/analytics"
	"github.com/quiet-terminal-interactive/qtidocs/internal/sanitize"
)

const (
	collectPath   = "_qtidocs/collect"
	statsDataPath = "_qtidocs/stats.json"
	statsPath     = "_stats"
)

const csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

const svgCSP = "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; font-src 'self' data:; frame-ancestors 'none'; sandbox"

type Resolver interface {
	Resolve(subdomain string) (dir string, ok bool)
}

type VersionedResolver interface {
	Versions(subdomain string) (names []string, ok bool)
}

type FixtureResolver struct {
	Root string
}

func (f FixtureResolver) Resolve(subdomain string) (string, bool) {
	dir := filepath.Join(f.Root, subdomain)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return dir, true
}

var subdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

var previewLabelPattern = regexp.MustCompile(`^pr-[0-9]+$`)

func validSubdomainKey(s string) bool {
	if subdomainPattern.MatchString(s) {
		return true
	}
	first, rest, ok := strings.Cut(s, ".")
	return ok && previewLabelPattern.MatchString(first) && subdomainPattern.MatchString(rest)
}

type Server struct {
	Resolver  Resolver
	Analytics analytics.Store
}

func New(resolver Resolver) *Server {
	return &Server{Resolver: resolver}
}

type TitledResolver interface {
	Title(subdomain string) (title string, ok bool)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", csp)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	subdomain := subdomainOf(r.Host)
	if !validSubdomainKey(subdomain) {
		http.NotFound(w, r)
		return
	}

	dir, ok := s.Resolver.Resolve(subdomain)
	if !ok {
		http.NotFound(w, r)
		return
	}

	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

	switch rel {
	case collectPath:
		s.handleCollect(w, r, subdomain)
		return
	case statsDataPath:
		s.handleStatsData(w, r, subdomain)
		return
	case statsPath:
		s.handleStatsPage(w, r, subdomain)
		return
	}

	if vr, ok := s.Resolver.(VersionedResolver); ok {
		if names, ok := vr.Versions(subdomain); ok {
			if versioned, rest, matched := stripVersionSegment(rel, names); matched {
				dir, rel = filepath.Join(dir, versioned), rest
			}
		}
	}

	serveSite(w, r, dir, rel)
}

func stripVersionSegment(rel string, names []string) (segment, rest string, matched bool) {
	first, after, _ := strings.Cut(rel, "/")
	if first == "" {
		return "", rel, false
	}
	for _, name := range names {
		if first == name {
			return first, after, true
		}
	}
	return "", rel, false
}

func subdomainOf(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.ToLower(host)

	first, rest, ok := strings.Cut(host, ".")
	if !ok {
		return host
	}
	if previewLabelPattern.MatchString(first) {
		if second, _, ok := strings.Cut(rest, "."); ok {
			return first + "." + second
		}
		return first + "." + rest
	}
	return first
}

func serveSite(w http.ResponseWriter, r *http.Request, siteDir, rel string) {
	switch {
	case strings.HasPrefix(rel, "assets/"):
		serveStatic(w, r, siteDir, rel, sanitize.AssetContentType)
	case strings.HasPrefix(rel, "_qtidocs/"):
		serveStatic(w, r, siteDir, rel, sanitize.ThemeAssetContentType)
	default:
		servePage(w, r, siteDir, rel)
	}
}

func servePage(w http.ResponseWriter, r *http.Request, siteDir, rel string) {
	file := filepath.Join(siteDir, filepath.FromSlash(rel), "index.html")
	data, err := os.ReadFile(file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(data); err != nil {
		log.Printf("server: writing page response for %q: %v", rel, err)
	}
}

func serveStatic(w http.ResponseWriter, r *http.Request, siteDir, rel string, contentType func(name string) string) {
	file := filepath.Join(siteDir, filepath.FromSlash(rel))
	info, err := os.Stat(file)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := contentType(rel)
	if strings.HasPrefix(ct, "image/svg+xml") {
		w.Header().Set("Content-Security-Policy", svgCSP)
	}
	w.Header().Set("Content-Type", ct)
	if _, err := w.Write(data); err != nil {
		log.Printf("server: writing static response for %q: %v", rel, err)
	}
}
