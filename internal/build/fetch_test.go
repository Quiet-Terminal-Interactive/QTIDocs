package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type tarEntry struct {
	name     string
	content  string
	typeflag byte
	linkname string
}

func buildTarGz(t *testing.T, topDir string, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, e := range entries {
		name := e.name
		if topDir != "" {
			name = topDir + "/" + name
		}
		typeflag := e.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		hdr := &tar.Header{
			Name:     name,
			Typeflag: typeflag,
			Size:     int64(len(e.content)),
			Mode:     0o644,
			Linkname: e.linkname,
		}
		if typeflag == tar.TypeDir {
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader(%q): %v", name, err)
		}
		if typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatalf("Write(%q): %v", name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip Close: %v", err)
	}
	return buf.Bytes()
}

func testLimits() Limits {
	return Limits{MaxExtractedBytes: 1 << 20, MaxFiles: 1000}
}

func TestGitHubFetcher_Fetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widgets/tarball/main" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", got)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake tarball bytes"))
	}))
	defer srv.Close()

	f := GitHubFetcher{Token: "tok", BaseURL: srv.URL}
	rc, err := f.Fetch(context.Background(), "acme", "widgets", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "fake tarball bytes" {
		t.Errorf("body = %q", data)
	}
}

func TestGitHubFetcher_Fetch_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := GitHubFetcher{BaseURL: srv.URL}
	_, err := f.Fetch(context.Background(), "acme", "widgets", "main")
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
}

func TestGitHubFetcher_Defaults(t *testing.T) {
	f := GitHubFetcher{}
	if f.baseURL() != "https://api.github.com" {
		t.Errorf("baseURL() = %q", f.baseURL())
	}
	if f.client() != http.DefaultClient {
		t.Error("client() should default to http.DefaultClient")
	}
}

func TestExtract_BasicFilesUnderConfiguredPath(t *testing.T) {
	data := buildTarGz(t, "acme-widgets-abc123", []tarEntry{
		{name: "qtidocs/index.md", content: "# Home"},
		{name: "qtidocs/guides/advanced.md", content: "# Advanced"},
		{name: "README.md", content: "not included"},
	})

	res, err := Extract(bytes.NewReader(data), "qtidocs", t.TempDir(), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(res.Dir)

	if res.Files != 2 {
		t.Errorf("Files = %d, want 2", res.Files)
	}
	indexContent, err := os.ReadFile(filepath.Join(res.Dir, "index.md"))
	if err != nil {
		t.Fatalf("unexpected error reading index.md: %v", err)
	}
	if string(indexContent) != "# Home" {
		t.Errorf("index.md content = %q", indexContent)
	}
	if _, err := os.Stat(filepath.Join(res.Dir, "guides", "advanced.md")); err != nil {
		t.Errorf("expected guides/advanced.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res.Dir, "README.md")); !os.IsNotExist(err) {
		t.Errorf("expected README.md to be excluded, stat err = %v", err)
	}
}

func TestExtract_EmptyConfiguredPathTakesWholeTree(t *testing.T) {
	data := buildTarGz(t, "acme-widgets-abc123", []tarEntry{
		{name: "index.md", content: "# Home"},
	})

	res, err := Extract(bytes.NewReader(data), "", t.TempDir(), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(res.Dir)

	if res.Files != 1 {
		t.Errorf("Files = %d, want 1", res.Files)
	}
}

func TestExtract_DiscardsSymlinks(t *testing.T) {
	data := buildTarGz(t, "top", []tarEntry{
		{name: "qtidocs/index.md", content: "# Home"},
		{name: "qtidocs/evil-link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
	})

	res, err := Extract(bytes.NewReader(data), "qtidocs", t.TempDir(), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(res.Dir)

	if res.Files != 1 {
		t.Errorf("Files = %d, want 1 (symlink discarded)", res.Files)
	}
	if _, err := os.Stat(filepath.Join(res.Dir, "evil-link")); !os.IsNotExist(err) {
		t.Errorf("expected symlink entry to never be written, stat err = %v", err)
	}
}

func TestExtract_PathTraversalAttemptDiscarded(t *testing.T) {
	data := buildTarGz(t, "top", []tarEntry{
		{name: "qtidocs/index.md", content: "# Home"},
		{name: "qtidocs/../../../etc/passwd", content: "evil"},
	})

	res, err := Extract(bytes.NewReader(data), "qtidocs", t.TempDir(), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(res.Dir)

	if res.Files != 1 {
		t.Errorf("Files = %d, want 1 (traversal entry discarded, only index.md written)", res.Files)
	}
}

func TestExtract_ConfiguredDirectoryItselfNotWritten(t *testing.T) {
	data := buildTarGz(t, "top", []tarEntry{
		{name: "qtidocs", typeflag: tar.TypeDir},
		{name: "qtidocs/index.md", content: "# Home"},
	})

	res, err := Extract(bytes.NewReader(data), "qtidocs", t.TempDir(), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(res.Dir)

	if res.Files != 1 {
		t.Errorf("Files = %d, want 1", res.Files)
	}
}

func TestExtract_FileCountLimitExceeded(t *testing.T) {
	entries := make([]tarEntry, 5)
	for i := range entries {
		entries[i] = tarEntry{name: "qtidocs/file" + string(rune('a'+i)) + ".md", content: "x"}
	}
	data := buildTarGz(t, "top", entries)

	limits := Limits{MaxExtractedBytes: 1 << 20, MaxFiles: 3}
	parent := t.TempDir()
	_, err := Extract(bytes.NewReader(data), "qtidocs", parent, limits)
	if err == nil {
		t.Fatal("expected error for exceeding MaxFiles, got nil")
	}

	leftover, _ := os.ReadDir(parent)
	if len(leftover) != 0 {
		t.Errorf("expected no leftover dirs after failed Extract, found %v", leftover)
	}
}

func TestExtract_ByteLimitExceeded(t *testing.T) {
	data := buildTarGz(t, "top", []tarEntry{
		{name: "qtidocs/big.md", content: "0123456789"},
	})

	limits := Limits{MaxExtractedBytes: 5, MaxFiles: 1000}
	parent := t.TempDir()
	_, err := Extract(bytes.NewReader(data), "qtidocs", parent, limits)
	if err == nil {
		t.Fatal("expected error for exceeding MaxExtractedBytes, got nil")
	}

	leftover, _ := os.ReadDir(parent)
	if len(leftover) != 0 {
		t.Errorf("expected no leftover dirs after failed Extract, found %v", leftover)
	}
}

func TestExtract_ByteLimitExactlyAtLimit(t *testing.T) {
	data := buildTarGz(t, "top", []tarEntry{
		{name: "qtidocs/exact.md", content: "01234"},
	})

	limits := Limits{MaxExtractedBytes: 5, MaxFiles: 1000}
	res, err := Extract(bytes.NewReader(data), "qtidocs", t.TempDir(), limits)
	if err != nil {
		t.Fatalf("unexpected error at exact limit: %v", err)
	}
	defer os.RemoveAll(res.Dir)
	if res.Bytes != 5 {
		t.Errorf("Bytes = %d, want 5", res.Bytes)
	}
}

func TestExtract_CumulativeByteLimitAcrossFiles(t *testing.T) {
	data := buildTarGz(t, "top", []tarEntry{
		{name: "qtidocs/a.md", content: "01234"},
		{name: "qtidocs/b.md", content: "01234"},
	})

	limits := Limits{MaxExtractedBytes: 8, MaxFiles: 1000}
	parent := t.TempDir()
	_, err := Extract(bytes.NewReader(data), "qtidocs", parent, limits)
	if err == nil {
		t.Fatal("expected error: second file should push cumulative bytes over the limit")
	}
}

func TestExtract_InvalidGzip(t *testing.T) {
	_, err := Extract(bytes.NewReader([]byte("not gzip data")), "qtidocs", t.TempDir(), testLimits())
	if err == nil {
		t.Fatal("expected error for invalid gzip data, got nil")
	}
}

func TestExtract_EmptyTarball(t *testing.T) {
	data := buildTarGz(t, "top", nil)
	res, err := Extract(bytes.NewReader(data), "qtidocs", t.TempDir(), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(res.Dir)
	if res.Files != 0 {
		t.Errorf("Files = %d, want 0", res.Files)
	}
}

func TestExtract_NoMatchingPath(t *testing.T) {
	data := buildTarGz(t, "top", []tarEntry{
		{name: "other/file.md", content: "x"},
	})
	res, err := Extract(bytes.NewReader(data), "qtidocs", t.TempDir(), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(res.Dir)
	if res.Files != 0 {
		t.Errorf("Files = %d, want 0", res.Files)
	}
}

func TestRelativeToPath(t *testing.T) {
	cases := []struct {
		name           string
		entry          string
		configuredPath string
		wantRel        string
		wantOK         bool
	}{
		{"file inside path", "top/qtidocs/index.md", "qtidocs", "index.md", true},
		{"nested file inside path", "top/qtidocs/guides/a.md", "qtidocs", "guides/a.md", true},
		{"file outside path", "top/README.md", "qtidocs", "", false},
		{"top-level dir entry alone", "top", "qtidocs", "", false},
		{"configured dir itself", "top/qtidocs", "qtidocs", "", false},
		{"empty configured path takes everything", "top/anything.md", "", "anything.md", true},
		{"dot configured path takes everything", "top/anything.md", ".", "anything.md", true},
		{"similarly prefixed sibling dir excluded", "top/qtidocs-other/x.md", "qtidocs", "", false},
		{"traversal collapses safely", "top/qtidocs/../other/x.md", "qtidocs", "", false},
		{"leading slash normalized", "top/qtidocs/./index.md", "qtidocs", "index.md", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rel, ok := relativeToPath(c.entry, c.configuredPath)
			if rel != c.wantRel || ok != c.wantOK {
				t.Errorf("relativeToPath(%q, %q) = (%q, %v), want (%q, %v)",
					c.entry, c.configuredPath, rel, ok, c.wantRel, c.wantOK)
			}
		})
	}
}
