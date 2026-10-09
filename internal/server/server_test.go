package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSubdomainOf(t *testing.T) {
	cases := []struct{ host, want string }{
		{"acme.qtidocs.dev", "acme"},
		{"acme.qtidocs.dev:8080", "acme"},
		{"ACME.qtidocs.dev", "acme"},
		{"acme", "acme"},
		{"pr-42.acme.qtidocs.dev", "pr-42.acme"},
		{"pr-42.acme.qtidocs.dev:8080", "pr-42.acme"},
		{"pr-42.acme", "pr-42.acme"},
	}
	for _, c := range cases {
		if got := subdomainOf(c.host); got != c.want {
			t.Errorf("subdomainOf(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}

func TestValidSubdomainKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"acme", true},
		{"a", true},
		{"acme-widgets", true},
		{"pr-42.acme", true},
		{"pr-42.acme-widgets", true},
		{"", false},
		{"-acme", false},
		{"acme-", false},
		{"Acme", false},
		{"acme..widgets", false},
		{"pr-abc.acme", false},
		{"pr-42.", false},
	}
	for _, c := range cases {
		if got := validSubdomainKey(c.key); got != c.want {
			t.Errorf("validSubdomainKey(%q) = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestFixtureResolver_Resolve(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "acme"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	r := FixtureResolver{Root: root}

	dir, ok := r.Resolve("acme")
	if !ok || dir != filepath.Join(root, "acme") {
		t.Errorf("Resolve(acme) = (%q, %v)", dir, ok)
	}

	_, ok = r.Resolve("ghost")
	if ok {
		t.Error("expected Resolve(ghost) = false")
	}
}

func TestFixtureResolver_Resolve_FileNotDir(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "notadir"), []byte("x"), 0o644)
	r := FixtureResolver{Root: root}

	_, ok := r.Resolve("notadir")
	if ok {
		t.Error("expected Resolve to fail when the entry is a file, not a dir")
	}
}

func TestStripVersionSegment(t *testing.T) {
	cases := []struct {
		rel         string
		names       []string
		wantSeg     string
		wantRest    string
		wantMatched bool
	}{
		{"v1/guides/advanced", []string{"v1", "v2"}, "v1", "guides/advanced", true},
		{"v2", []string{"v1", "v2"}, "v2", "", true},
		{"other/page", []string{"v1", "v2"}, "", "other/page", false},
		{"", []string{"v1", "v2"}, "", "", false},
	}
	for _, c := range cases {
		seg, rest, matched := stripVersionSegment(c.rel, c.names)
		if seg != c.wantSeg || rest != c.wantRest || matched != c.wantMatched {
			t.Errorf("stripVersionSegment(%q, %v) = (%q, %q, %v), want (%q, %q, %v)",
				c.rel, c.names, seg, rest, matched, c.wantSeg, c.wantRest, c.wantMatched)
		}
	}
}

func writeSiteFile(t *testing.T, root, subdomain, rel, content string) {
	t.Helper()
	full := filepath.Join(root, subdomain, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestServeHTTP_PageRoute(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "index.html", "<h1>Home</h1>")
	s := New(FixtureResolver{Root: root})

	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "<h1>Home</h1>" {
		t.Errorf("body = %q", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("expected CSP header to be set")
	}
}

func TestServeHTTP_NestedPageRoute(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "guides/advanced/index.html", "<p>Advanced</p>")
	s := New(FixtureResolver{Root: root})

	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/guides/advanced", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "<p>Advanced</p>" {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestServeHTTP_UnregisteredSubdomain404(t *testing.T) {
	root := t.TempDir()
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://ghost.qtidocs.dev/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestServeHTTP_InvalidHostHeader404(t *testing.T) {
	root := t.TempDir()
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://Not_Valid..Host/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestServeHTTP_MissingPage404(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "index.html", "home")
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/does-not-exist", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestServeHTTP_AssetServedWithSanitizedContentType(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "assets/logo.png", "fake-png-bytes")
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/assets/logo.png", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
}

func TestServeHTTP_AssetJSNeverServedAsScript(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "assets/evil.js", "alert(1)")
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/assets/evil.js", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream (never executable)", ct)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff (else browsers run it as script anyway)", got)
	}
}

func TestServeHTTP_SVGAssetSandboxed(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "assets/logo.svg", `<svg xmlns="http://www.w3.org/2000/svg"><script href="/assets/x.txt"/></svg>`)
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/assets/logo.svg", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got != svgCSP {
		t.Errorf("Content-Security-Policy = %q, want %q", got, svgCSP)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

func TestServeHTTP_ThemeJSServedAsScript(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "_qtidocs/search.js", "console.log(1)")
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/_qtidocs/search.js", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/javascript", ct)
	}
}

func TestServeHTTP_DirectoryTraversalBlocked(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "acme", "index.html", "home")
	os.WriteFile(filepath.Join(root, "secret.txt"), []byte("top secret"), 0o644)

	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/../secret.txt", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK && rec.Body.String() == "top secret" {
		t.Fatal("path traversal escaped the site directory")
	}
}

type versionedResolver struct {
	dir      string
	versions []string
}

func (v versionedResolver) Resolve(subdomain string) (string, bool) { return v.dir, true }
func (v versionedResolver) Versions(subdomain string) ([]string, bool) {
	return v.versions, true
}

func TestServeHTTP_VersionedSite(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "v1"), 0o755)
	os.WriteFile(filepath.Join(root, "v1", "index.html"), []byte("v1 home"), 0o644)
	os.MkdirAll(filepath.Join(root, "v2"), 0o755)
	os.WriteFile(filepath.Join(root, "v2", "index.html"), []byte("v2 home"), 0o644)

	s := New(versionedResolver{dir: root, versions: []string{"v1", "v2"}})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/v2", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "v2 home" {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestServeHTTP_VersionedSiteRootNotStripped(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "index.html"), []byte("redirect page"), 0o644)

	s := New(versionedResolver{dir: root, versions: []string{"v1", "v2"}})
	req := httptest.NewRequest(http.MethodGet, "http://acme.qtidocs.dev/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "redirect page" {
		t.Errorf("status = %d, body = %q, want root redirect page served unversioned", rec.Code, rec.Body.String())
	}
}

func TestServeHTTP_PreviewSubdomainRouting(t *testing.T) {
	root := t.TempDir()
	writeSiteFile(t, root, "pr-7.acme", "index.html", "preview home")
	s := New(FixtureResolver{Root: root})
	req := httptest.NewRequest(http.MethodGet, "http://pr-7.acme.qtidocs.dev/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "preview home" {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}
