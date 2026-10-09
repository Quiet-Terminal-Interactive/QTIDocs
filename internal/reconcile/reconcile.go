package reconcile

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type GitHub interface {
	ResolveRef(ctx context.Context, owner, repo, ref string) (sha string, err error)
}

type HTTPGitHub struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

func (g HTTPGitHub) baseURL() string {
	if g.BaseURL != "" {
		return g.BaseURL
	}
	return "https://api.github.com"
}

func (g HTTPGitHub) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return http.DefaultClient
}

func (g HTTPGitHub) ResolveRef(ctx context.Context, owner, repo, ref string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/commits/%s", g.baseURL(), owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("reconcile: building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.sha")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := g.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("reconcile: resolving %s/%s@%s: %w", owner, repo, ref, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reconcile: resolving %s/%s@%s: unexpected status %s", owner, repo, ref, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reconcile: reading response for %s/%s@%s: %w", owner, repo, ref, err)
	}
	return strings.TrimSpace(string(body)), nil
}

type Reconciler struct {
	GitHub  GitHub
	Store   storage.Store
	Enqueue func(build.Job)
}

func (r Reconciler) Run(ctx context.Context) (int, error) {
	entries, err := r.Store.List()
	if err != nil {
		return 0, fmt.Errorf("reconcile: listing registered entries: %w", err)
	}

	enqueued := 0
	for _, e := range entries {
		if e.Repo == "" || e.Branch == "" || e.Path == "" {
			continue // not fully registered yet
		}

		owner, name, err := registry.ParseRepo(e.Repo)
		if err != nil {
			log.Printf("reconcile: %q: invalid repo %q: %v", e.Subdomain, e.Repo, err)
			continue
		}

		if len(e.Versions) > 0 {
			if r.reconcileVersioned(ctx, owner, name, e) {
				enqueued++
			}
			continue
		}

		sha, err := r.GitHub.ResolveRef(ctx, owner, name, e.Branch)
		if err != nil {
			log.Printf("reconcile: %q: resolving %s/%s@%s: %v", e.Subdomain, owner, name, e.Branch, err)
			continue
		}

		if e.OutputDir != "" && sha == e.DeployedSHA {
			continue // already up to date
		}

		r.Enqueue(build.Job{Subdomain: e.Subdomain, Title: e.Title, Owner: owner, Repo: name, Ref: sha, Path: e.Path})
		enqueued++
	}
	return enqueued, nil
}

func (r Reconciler) reconcileVersioned(ctx context.Context, owner, repo string, e storage.Entry) bool {
	resolved := make([]build.Version, len(e.Versions))
	mismatch := e.OutputDir == ""
	for i, v := range e.Versions {
		sha, err := r.GitHub.ResolveRef(ctx, owner, repo, v.Ref)
		if err != nil {
			log.Printf("reconcile: %q: resolving %s/%s@%s (version %q): %v", e.Subdomain, owner, repo, v.Ref, v.Name, err)
			return false
		}
		resolved[i] = build.Version{Name: v.Name, Ref: sha}
		if sha != e.DeployedVersions[v.Name] {
			mismatch = true
		}
	}
	if !mismatch {
		return false
	}

	r.Enqueue(build.Job{
		Subdomain: e.Subdomain, Title: e.Title, Owner: owner, Repo: repo, Path: e.Path,
		Versions: resolved, DefaultVersion: e.DefaultVersion,
	})
	return true
}
