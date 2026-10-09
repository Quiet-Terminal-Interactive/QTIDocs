package reconcile

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type fakeStore struct {
	entries []storage.Entry
	listErr error
}

func (f *fakeStore) Get(subdomain string) (storage.Entry, bool, error) {
	for _, e := range f.entries {
		if e.Subdomain == subdomain {
			return e, true, nil
		}
	}
	return storage.Entry{}, false, nil
}
func (f *fakeStore) Set(e storage.Entry) error { f.entries = append(f.entries, e); return nil }
func (f *fakeStore) List() ([]storage.Entry, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.entries, nil
}
func (f *fakeStore) Delete(subdomain string) error { return nil }

type fakeGitHub struct {
	mu   sync.Mutex
	shas map[string]string
	errs map[string]error
}

func newFakeGitHub() *fakeGitHub {
	return &fakeGitHub{shas: map[string]string{}, errs: map[string]error{}}
}

func (f *fakeGitHub) key(owner, repo, ref string) string { return owner + "/" + repo + "@" + ref }

func (f *fakeGitHub) set(owner, repo, ref, sha string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shas[f.key(owner, repo, ref)] = sha
}

func (f *fakeGitHub) setErr(owner, repo, ref string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[f.key(owner, repo, ref)] = err
}

func (f *fakeGitHub) ResolveRef(ctx context.Context, owner, repo, ref string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := f.key(owner, repo, ref)
	if err, ok := f.errs[k]; ok {
		return "", err
	}
	return f.shas[k], nil
}

func TestReconciler_Run_EnqueuesNeverBuiltEntry(t *testing.T) {
	gh := newFakeGitHub()
	gh.set("acme", "widgets", "main", "sha123")
	store := &fakeStore{entries: []storage.Entry{
		{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 || len(enqueued) != 1 {
		t.Fatalf("n = %d, enqueued = %v, want 1", n, enqueued)
	}
	if enqueued[0].Ref != "sha123" || enqueued[0].Owner != "acme" || enqueued[0].Repo != "widgets" {
		t.Errorf("enqueued[0] = %+v", enqueued[0])
	}
}

func TestReconciler_Run_SkipsUpToDateEntry(t *testing.T) {
	gh := newFakeGitHub()
	gh.set("acme", "widgets", "main", "sha123")
	store := &fakeStore{entries: []storage.Entry{
		{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", OutputDir: "/out", DeployedSHA: "sha123"},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 || len(enqueued) != 0 {
		t.Fatalf("n = %d, enqueued = %v, want 0", n, enqueued)
	}
}

func TestReconciler_Run_EnqueuesOnMismatch(t *testing.T) {
	gh := newFakeGitHub()
	gh.set("acme", "widgets", "main", "newsha")
	store := &fakeStore{entries: []storage.Entry{
		{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", OutputDir: "/out", DeployedSHA: "oldsha"},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, _ := r.Run(context.Background())
	if n != 1 || enqueued[0].Ref != "newsha" {
		t.Fatalf("n = %d, enqueued = %v", n, enqueued)
	}
}

func TestReconciler_Run_SkipsIncompleteRegistration(t *testing.T) {
	gh := newFakeGitHub()
	store := &fakeStore{entries: []storage.Entry{
		{Subdomain: "acme", Repo: "", Branch: "main", Path: "qtidocs"},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("n = %d, want 0 for incomplete registration", n)
	}
}

func TestReconciler_Run_SkipsInvalidRepo(t *testing.T) {
	gh := newFakeGitHub()
	store := &fakeStore{entries: []storage.Entry{
		{Subdomain: "acme", Repo: "not-a-valid-repo-format!!", Branch: "main", Path: "qtidocs"},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("n = %d, want 0 for invalid repo", n)
	}
}

func TestReconciler_Run_SkipsOnResolveError(t *testing.T) {
	gh := newFakeGitHub()
	gh.setErr("acme", "widgets", "main", errors.New("rate limited"))
	store := &fakeStore{entries: []storage.Entry{
		{Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs"},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("n = %d, want 0 when resolve fails", n)
	}
}

func TestReconciler_Run_ListError(t *testing.T) {
	store := &fakeStore{listErr: errors.New("disk error")}
	r := Reconciler{GitHub: newFakeGitHub(), Store: store, Enqueue: func(build.Job) {}}

	_, err := r.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReconciler_Run_MultipleEntriesOneFailureDoesntStopOthers(t *testing.T) {
	gh := newFakeGitHub()
	gh.setErr("acme", "broken", "main", errors.New("fail"))
	gh.set("acme", "good", "main", "sha456")
	store := &fakeStore{entries: []storage.Entry{
		{Subdomain: "broken", Repo: "acme/broken", Branch: "main", Path: "qtidocs"},
		{Subdomain: "good", Repo: "acme/good", Branch: "main", Path: "qtidocs"},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 || len(enqueued) != 1 || enqueued[0].Subdomain != "good" {
		t.Errorf("n = %d, enqueued = %v, want only good to succeed", n, enqueued)
	}
}

func TestReconciler_Run_VersionedNeverBuilt(t *testing.T) {
	gh := newFakeGitHub()
	gh.set("acme", "widgets", "v1-branch", "sha-v1")
	gh.set("acme", "widgets", "v2-branch", "sha-v2")
	store := &fakeStore{entries: []storage.Entry{
		{
			Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs",
			Versions: []storage.Version{{Name: "v1", Ref: "v1-branch"}, {Name: "v2", Ref: "v2-branch"}},
		},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, _ := r.Run(context.Background())
	if n != 1 || len(enqueued) != 1 {
		t.Fatalf("n = %d, enqueued = %v", n, enqueued)
	}
	if len(enqueued[0].Versions) != 2 {
		t.Fatalf("Versions = %+v, want 2", enqueued[0].Versions)
	}
}

func TestReconciler_Run_VersionedUpToDate(t *testing.T) {
	gh := newFakeGitHub()
	gh.set("acme", "widgets", "v1-branch", "sha-v1")
	store := &fakeStore{entries: []storage.Entry{
		{
			Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", OutputDir: "/out",
			Versions:         []storage.Version{{Name: "v1", Ref: "v1-branch"}},
			DeployedVersions: map[string]string{"v1": "sha-v1"},
		},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, _ := r.Run(context.Background())
	if n != 0 {
		t.Errorf("n = %d, want 0 (versioned entry up to date)", n)
	}
}

func TestReconciler_Run_VersionedMismatchOnOneVersion(t *testing.T) {
	gh := newFakeGitHub()
	gh.set("acme", "widgets", "v1-branch", "sha-v1-new")
	store := &fakeStore{entries: []storage.Entry{
		{
			Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs", OutputDir: "/out",
			Versions:         []storage.Version{{Name: "v1", Ref: "v1-branch"}},
			DeployedVersions: map[string]string{"v1": "sha-v1-old"},
		},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, _ := r.Run(context.Background())
	if n != 1 {
		t.Errorf("n = %d, want 1 (mismatch on v1)", n)
	}
}

func TestReconciler_Run_VersionedResolveErrorSkipsEntry(t *testing.T) {
	gh := newFakeGitHub()
	gh.setErr("acme", "widgets", "v1-branch", errors.New("fail"))
	store := &fakeStore{entries: []storage.Entry{
		{
			Subdomain: "acme", Repo: "acme/widgets", Branch: "main", Path: "qtidocs",
			Versions: []storage.Version{{Name: "v1", Ref: "v1-branch"}},
		},
	}}

	var enqueued []build.Job
	r := Reconciler{GitHub: gh, Store: store, Enqueue: func(j build.Job) { enqueued = append(enqueued, j) }}

	n, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("n = %d, want 0 when a version fails to resolve", n)
	}
}
