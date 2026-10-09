package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/quiet-terminal-interactive/qtidocs/internal/build"
	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type versionFlags []build.Version

func (v *versionFlags) String() string { return "" }

func (v *versionFlags) Set(s string) error {
	name, ref, ok := strings.Cut(s, "=")
	if !ok || name == "" || ref == "" {
		return fmt.Errorf("must look like <name>=<ref>, got %q", s)
	}
	*v = append(*v, build.Version{Name: name, Ref: ref})
	return nil
}

func main() {
	owner := flag.String("owner", "", "repo owner")
	repo := flag.String("repo", "", "repo name (without owner)")
	ref := flag.String("ref", "", "branch, tag, or commit SHA to build (ignored if -version is given)")
	path := flag.String("path", "", "qtidocs/ path within the repo")
	subdomain := flag.String("subdomain", "", "registered subdomain to build/serve (must already exist in the routing table)")
	routing := flag.String("routing", "", "path to the routing table file (see internal/storage)")
	sitesRoot := flag.String("sites-root", "", "base directory rendered site output is written under")
	githubToken := flag.String("github-token", os.Getenv("QTIDOCS_GITHUB_TOKEN"), "GitHub token for the tarball fetch (optional; defaults to $QTIDOCS_GITHUB_TOKEN)")
	defaultVersion := flag.String("default-version", "", "which -version name is served at the output's root path (only meaningful with -version; defaults to the first -version given)")
	var versions versionFlags
	flag.Var(&versions, "version", "a tracked <name>=<ref> pair for a versioned build; repeatable. Building at least one version switches this run to a versioned build, ignoring -ref")
	flag.Parse()

	required := []struct{ name, val string }{
		{"owner", *owner}, {"repo", *repo}, {"path", *path},
		{"subdomain", *subdomain}, {"routing", *routing}, {"sites-root", *sitesRoot},
	}
	for _, r := range required {
		if r.val == "" {
			fmt.Fprintf(os.Stderr, "builder: -%s is required\n", r.name)
			flag.Usage()
			os.Exit(2)
		}
	}
	if len(versions) == 0 && *ref == "" {
		fmt.Fprintln(os.Stderr, "builder: -ref is required unless -version is given")
		flag.Usage()
		os.Exit(2)
	}

	store := storage.Open(*routing)
	entry, _, err := store.Get(*subdomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "builder: reading routing entry for %q: %v\n", *subdomain, err)
		os.Exit(1)
	}

	pipeline := build.Pipeline{
		Fetcher:   build.GitHubFetcher{Token: *githubToken},
		Store:     store,
		SitesRoot: *sitesRoot,
		Limits:    build.DefaultLimits(),
	}

	job := build.Job{Subdomain: *subdomain, Title: entry.Title, Owner: *owner, Repo: *repo, Ref: *ref, Path: *path}
	if len(versions) > 0 {
		job.Versions = versions
		job.DefaultVersion = *defaultVersion
	}
	if err := pipeline.Run(context.Background(), job); err != nil {
		fmt.Fprintf(os.Stderr, "builder: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("builder: built and served %q\n", *subdomain)
}
