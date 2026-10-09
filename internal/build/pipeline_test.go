package build

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type fakeStore struct {
	entries map[string]storage.Entry
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

type fakeFetcher struct {
	tarballs map[string][]byte
	errs     map[string]error
	delay    time.Duration
}

func (f *fakeFetcher) Fetch(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if err, ok := f.errs[ref]; ok {
		return nil, err
	}
	data, ok := f.tarballs[ref]
	if !ok {
		return nil, errors.New("fakeFetcher: no tarball registered for ref " + ref)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func simpleTarball(t *testing.T) []byte {
	t.Helper()
	return buildTarGz(t, "acme-widgets-abc", []tarEntry{
		{name: "qtidocs/index.md", content: "---\ntitle: Home\n---\n# Hello\n"},
	})
}

func basePipeline(t *testing.T, fetcher Fetcher, store storage.Store) Pipeline {
	t.Helper()
	return Pipeline{
		Fetcher:   fetcher,
		Store:     store,
		Limits:    Limits{MaxExtractedBytes: 1 << 20, MaxFiles: 1000, MaxBuildTime: 10 * time.Second, MaxOutputBytes: 1 << 20},
		SitesRoot: t.TempDir(),
	}
}

func TestPipeline_Run_SuccessfulBuild(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"}

	fetcher := &fakeFetcher{tarballs: map[string][]byte{"main": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	if err := p.Run(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := store.entries["acme"]
	if entry.OutputDir == "" {
		t.Fatal("expected OutputDir to be set after a successful build")
	}
	if entry.DeployedSHA != "main" {
		t.Errorf("DeployedSHA = %q, want main", entry.DeployedSHA)
	}
	if _, err := os.Stat(filepath.Join(entry.OutputDir, "index.html")); err != nil {
		t.Errorf("expected index.html in output dir: %v", err)
	}
}

func TestPipeline_Run_UnregisteredSubdomainFails(t *testing.T) {
	store := newFakeStore()
	fetcher := &fakeFetcher{tarballs: map[string][]byte{"main": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	err := p.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for unregistered subdomain, got nil")
	}
}

func TestPipeline_Run_FetchFailure(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{errs: map[string]error{"main": errors.New("network error")}}
	p := basePipeline(t, fetcher, store)

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	err := p.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for fetch failure, got nil")
	}
}

func TestPipeline_Run_ExtractLimitExceeded(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{tarballs: map[string][]byte{"main": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)
	p.Limits.MaxExtractedBytes = 1

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	err := p.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for extraction byte limit, got nil")
	}
	if store.entries["acme"].OutputDir != "" {
		t.Error("expected OutputDir to remain unset after a failed build")
	}
}

func TestPipeline_Run_OutputByteLimitExceeded(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{tarballs: map[string][]byte{"main": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)
	p.Limits.MaxOutputBytes = 1

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	err := p.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for output byte limit, got nil")
	}
	if store.entries["acme"].OutputDir != "" {
		t.Error("expected OutputDir to remain unset after a failed build")
	}
}

func TestPipeline_Run_BuildTimeLimitExceeded(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{
		tarballs: map[string][]byte{"main": simpleTarball(t)},
		delay:    20 * time.Millisecond,
	}
	p := basePipeline(t, fetcher, store)
	p.Limits.MaxBuildTime = 1 * time.Millisecond

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	err := p.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for build-time limit exceeded, got nil")
	}
}

func TestPipeline_Run_CleansUpPreviousOutput(t *testing.T) {
	store := newFakeStore()
	oldOutput := t.TempDir()
	marker := filepath.Join(oldOutput, "marker.txt")
	_ = os.WriteFile(marker, []byte("old"), 0o644)
	store.entries["acme"] = storage.Entry{Subdomain: "acme", OutputDir: oldOutput}

	fetcher := &fakeFetcher{tarballs: map[string][]byte{"main": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	if err := p.Run(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(oldOutput); !os.IsNotExist(err) {
		t.Errorf("expected previous output dir to be removed, stat err = %v", err)
	}
}

func TestPipeline_Run_StoreSetError(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	store.setErr = errors.New("disk full")
	fetcher := &fakeFetcher{tarballs: map[string][]byte{"main": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)

	job := Job{Subdomain: "acme", Owner: "acme", Repo: "widgets", Ref: "main", Path: "qtidocs"}
	err := p.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error when Store.Set fails, got nil")
	}
}

func TestPipeline_Run_Versioned_Success(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{tarballs: map[string][]byte{
		"v1-ref": simpleTarball(t),
		"v2-ref": simpleTarball(t),
	}}
	p := basePipeline(t, fetcher, store)

	job := Job{
		Subdomain: "acme", Owner: "acme", Repo: "widgets", Path: "qtidocs",
		Versions:       []Version{{Name: "v1", Ref: "v1-ref"}, {Name: "v2", Ref: "v2-ref"}},
		DefaultVersion: "v2",
	}
	if err := p.Run(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := store.entries["acme"]
	if entry.DeployedVersions["v1"] != "v1-ref" || entry.DeployedVersions["v2"] != "v2-ref" {
		t.Errorf("DeployedVersions = %v", entry.DeployedVersions)
	}
	if entry.DeployedSHA != "" {
		t.Errorf("DeployedSHA = %q, want empty for a versioned entry", entry.DeployedSHA)
	}
	if _, err := os.Stat(filepath.Join(entry.OutputDir, "v1", "index.html")); err != nil {
		t.Errorf("expected v1/index.html: %v", err)
	}
	if _, err := os.Stat(filepath.Join(entry.OutputDir, "v2", "index.html")); err != nil {
		t.Errorf("expected v2/index.html: %v", err)
	}
	if _, err := os.Stat(filepath.Join(entry.OutputDir, "index.html")); err != nil {
		t.Errorf("expected root redirect index.html: %v", err)
	}
}

func TestPipeline_Run_Versioned_DefaultVersionFallback(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{tarballs: map[string][]byte{"v1-ref": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)

	job := Job{
		Subdomain: "acme", Owner: "acme", Repo: "widgets", Path: "qtidocs",
		Versions:       []Version{{Name: "v1", Ref: "v1-ref"}},
		DefaultVersion: "does-not-exist",
	}
	if err := p.Run(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry := store.entries["acme"]
	content, err := os.ReadFile(filepath.Join(entry.OutputDir, "index.html"))
	if err != nil {
		t.Fatalf("unexpected error reading redirect page: %v", err)
	}
	if !bytes.Contains(content, []byte("/v1/")) {
		t.Errorf("expected redirect to /v1/, got: %s", content)
	}
}

func TestPipeline_Run_Versioned_OneRefFailsAbortsWholeBuild(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{
		tarballs: map[string][]byte{"v1-ref": simpleTarball(t)},
		errs:     map[string]error{"v2-ref": errors.New("404")},
	}
	p := basePipeline(t, fetcher, store)

	job := Job{
		Subdomain: "acme", Owner: "acme", Repo: "widgets", Path: "qtidocs",
		Versions: []Version{{Name: "v1", Ref: "v1-ref"}, {Name: "v2", Ref: "v2-ref"}},
	}
	err := p.Run(context.Background(), job)
	if err == nil {
		t.Fatal("expected error when one version's fetch fails, got nil")
	}
	if store.entries["acme"].OutputDir != "" {
		t.Error("expected no publish when any version fails")
	}
}

func TestPipeline_Run_Versioned_RejectsTraversingVersionName(t *testing.T) {
	store := newFakeStore()
	store.entries["acme"] = storage.Entry{Subdomain: "acme"}
	fetcher := &fakeFetcher{tarballs: map[string][]byte{"v1-ref": simpleTarball(t)}}
	p := basePipeline(t, fetcher, store)

	job := Job{
		Subdomain: "acme", Owner: "acme", Repo: "widgets", Path: "qtidocs",
		Versions: []Version{{Name: "../../escaped", Ref: "v1-ref"}},
	}
	if err := p.Run(context.Background(), job); err == nil {
		t.Fatal("expected error for a traversing version name, got nil")
	}
	if _, err := os.Stat(filepath.Join(p.SitesRoot, "escaped")); !os.IsNotExist(err) {
		t.Errorf("expected nothing written outside the site dir, stat err = %v", err)
	}
	if store.entries["acme"].OutputDir != "" {
		t.Error("expected no publish")
	}
}

func TestTeardown_RemovesEntryAndOutput(t *testing.T) {
	store := newFakeStore()
	outDir := t.TempDir()
	store.entries["acme"] = storage.Entry{Subdomain: "acme", OutputDir: outDir}

	if err := Teardown(store, "acme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := store.entries["acme"]; ok {
		t.Error("expected entry to be removed")
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("expected output dir removed, stat err = %v", err)
	}
}

func TestTeardown_MissingEntryIsNoop(t *testing.T) {
	store := newFakeStore()
	if err := Teardown(store, "ghost"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTeardown_GetError(t *testing.T) {
	store := newFakeStore()
	store.getErr = errors.New("disk error")
	if err := Teardown(store, "acme"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHasVersion(t *testing.T) {
	versions := []Version{{Name: "v1"}, {Name: "v2"}}
	if !hasVersion(versions, "v1") {
		t.Error("expected hasVersion(v1) = true")
	}
	if hasVersion(versions, "v3") {
		t.Error("expected hasVersion(v3) = false")
	}
	if hasVersion(versions, "") {
		t.Error("expected hasVersion(\"\") = false")
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("12345"), 0o644)
	_ = os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("123"), 0o644)

	size, err := dirSize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 8 {
		t.Errorf("dirSize() = %d, want 8", size)
	}
}
