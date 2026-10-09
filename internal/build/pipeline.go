package build

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
	"github.com/quiet-terminal-interactive/qtidocs/internal/site"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
	"github.com/quiet-terminal-interactive/qtidocs/internal/theme"
)

type Version struct {
	Name string
	Ref  string
}

type Job struct {
	Subdomain      string
	Title          string
	Owner          string
	Repo           string
	Ref            string
	Path           string
	Versions       []Version
	DefaultVersion string
}

type Pipeline struct {
	Fetcher   Fetcher
	Store     storage.Store
	Limits    Limits
	SitesRoot string
}

func (p Pipeline) Run(ctx context.Context, job Job) error {
	ctx, cancel := context.WithTimeout(ctx, p.Limits.MaxBuildTime)
	defer cancel()

	if len(job.Versions) > 0 {
		return p.runVersioned(ctx, job)
	}

	rc, err := p.Fetcher.Fetch(ctx, job.Owner, job.Repo, job.Ref)
	if err != nil {
		return fmt.Errorf("build: fetch stage: %w", err)
	}
	defer func() { _ = rc.Close() }()

	extracted, err := Extract(rc, job.Path, "", p.Limits)
	if err != nil {
		return fmt.Errorf("build: fetch stage: %w", err)
	}
	defer func() { _ = os.RemoveAll(extracted.Dir) }()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("build: cancelled before render stage: %w", err)
	}

	outDir, err := p.newBuildDir(job.Subdomain)
	if err != nil {
		return err
	}

	if err := site.Build(extracted.Dir, outDir, job.Title); err != nil {
		_ = os.RemoveAll(outDir)
		return fmt.Errorf("build: render stage: %w", err)
	}

	if err := ctx.Err(); err != nil {
		_ = os.RemoveAll(outDir)
		return fmt.Errorf("build: build-time limit exceeded: %w", err)
	}

	outBytes, err := dirSize(outDir)
	if err != nil {
		_ = os.RemoveAll(outDir)
		return fmt.Errorf("build: %w", err)
	}
	if outBytes > p.Limits.MaxOutputBytes {
		_ = os.RemoveAll(outDir)
		return fmt.Errorf("build: rendered output is %d bytes, exceeding the %d-byte limit", outBytes, p.Limits.MaxOutputBytes)
	}

	return p.serve(job, outDir)
}

func (p Pipeline) serve(job Job, outDir string) error {
	return p.publish(job.Subdomain, outDir, func(e *storage.Entry) {
		e.DeployedSHA = job.Ref
	})
}

func (p Pipeline) serveVersioned(job Job, outDir string, deployed map[string]string) error {
	return p.publish(job.Subdomain, outDir, func(e *storage.Entry) {
		e.DeployedSHA = ""
		e.DeployedVersions = deployed
	})
}

func (p Pipeline) publish(subdomain, outDir string, mutate func(*storage.Entry)) error {
	existing, ok, err := p.Store.Get(subdomain)
	if err != nil {
		return fmt.Errorf("build: serve stage: reading routing entry: %w", err)
	}
	if !ok {
		return fmt.Errorf("build: serve stage: %q is not a registered subdomain", subdomain)
	}

	previous := existing.OutputDir
	existing.OutputDir = outDir
	mutate(&existing)
	if err := p.Store.Set(existing); err != nil {
		return fmt.Errorf("build: serve stage: %w", err)
	}

	if previous != "" && previous != outDir {
		_ = os.RemoveAll(previous)
	}
	return nil
}

func (p Pipeline) runVersioned(ctx context.Context, job Job) error {
	if err := validateVersionNames(job.Versions); err != nil {
		return err
	}

	outDir, err := p.newBuildDir(job.Subdomain)
	if err != nil {
		return err
	}

	deployed, err := p.buildEachVersion(ctx, job, outDir)
	if err != nil {
		_ = os.RemoveAll(outDir)
		return err
	}

	if err := p.finishVersionedOutput(ctx, job, outDir); err != nil {
		_ = os.RemoveAll(outDir)
		return err
	}

	return p.serveVersioned(job, outDir, deployed)
}

func (p Pipeline) newBuildDir(subdomain string) (string, error) {
	siteDir := filepath.Join(p.SitesRoot, subdomain)
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		return "", fmt.Errorf("build: %w", err)
	}
	outDir, err := os.MkdirTemp(siteDir, "build-*")
	if err != nil {
		return "", fmt.Errorf("build: %w", err)
	}
	return outDir, nil
}

func validateVersionNames(versions []Version) error {
	for _, v := range versions {
		if !registry.ValidVersionName(v.Name) {
			return fmt.Errorf("build: invalid version name %q", v.Name)
		}
	}
	return nil
}

func (p Pipeline) buildEachVersion(ctx context.Context, job Job, outDir string) (map[string]string, error) {
	names := make([]string, len(job.Versions))
	for i, v := range job.Versions {
		names[i] = v.Name
	}

	deployed := make(map[string]string, len(job.Versions))
	for _, v := range job.Versions {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("build: build-time limit exceeded before version %q: %w", v.Name, err)
		}

		rc, err := p.Fetcher.Fetch(ctx, job.Owner, job.Repo, v.Ref)
		if err != nil {
			return nil, fmt.Errorf("build: fetch stage: version %q: %w", v.Name, err)
		}
		extracted, err := Extract(rc, job.Path, "", p.Limits)
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("build: fetch stage: version %q: %w", v.Name, err)
		}

		err = site.BuildVersioned(extracted.Dir, filepath.Join(outDir, v.Name), job.Title, site.VersionInfo{
			Current: v.Name,
			Names:   names,
		})
		_ = os.RemoveAll(extracted.Dir)
		if err != nil {
			return nil, fmt.Errorf("build: render stage: version %q: %w", v.Name, err)
		}

		deployed[v.Name] = v.Ref
	}
	return deployed, nil
}

func (p Pipeline) finishVersionedOutput(ctx context.Context, job Job, outDir string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("build: build-time limit exceeded: %w", err)
	}

	outBytes, err := dirSize(outDir)
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}
	if outBytes > p.Limits.MaxOutputBytes {
		return fmt.Errorf("build: rendered output is %d bytes, exceeding the %d-byte limit", outBytes, p.Limits.MaxOutputBytes)
	}

	defaultVersion := job.DefaultVersion
	if !hasVersion(job.Versions, defaultVersion) {
		defaultVersion = job.Versions[0].Name
	}
	if err := writeVersionRedirect(outDir, job.Title, defaultVersion); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

func Teardown(store storage.Store, subdomain string) error {
	e, ok, err := store.Get(subdomain)
	if err != nil {
		return fmt.Errorf("build: teardown: reading entry for %q: %w", subdomain, err)
	}
	if !ok {
		return nil
	}
	if err := store.Delete(subdomain); err != nil {
		return fmt.Errorf("build: teardown: removing entry for %q: %w", subdomain, err)
	}
	if e.OutputDir != "" {
		_ = os.RemoveAll(e.OutputDir)
	}
	return nil
}

func hasVersion(versions []Version, name string) bool {
	if name == "" {
		return false
	}
	for _, v := range versions {
		if v.Name == name {
			return true
		}
	}
	return false
}

func writeVersionRedirect(outDir, siteTitle, defaultVersion string) error {
	f, err := os.Create(filepath.Join(outDir, "index.html"))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return theme.RenderVersionRedirect(f, siteTitle, "/"+defaultVersion+"/")
}

func dirSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}
