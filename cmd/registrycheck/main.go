package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
)

func main() {
	sitesDir := flag.String("sites-dir", "sites", "registry repo directory holding one <subdomain>.yaml per site")
	reservedPath := flag.String("reserved", "reserved.yaml", "path to the reserved-subdomain config file")
	base := flag.String("base", "origin/main", "git ref to diff against to find changed site files")
	author := flag.String("author", "", "PR author's GitHub login (required)")
	qtiOrg := flag.String("qti-org", "quiet-terminal-interactive", "GitHub org exempted from the write-access check")
	githubToken := flag.String("github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (defaults to $GITHUB_TOKEN)")
	githubAPI := flag.String("github-api", "", "override the GitHub API base URL (for tests)")
	dryRun := flag.Bool("dry-run", false, "skip GitHub API calls; for offline linting only, never a real CI gate")
	flag.Parse()

	if *author == "" {
		fmt.Fprintln(os.Stderr, "registrycheck: -author is required")
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*sitesDir, *reservedPath, *base, *author, *qtiOrg, *githubToken, *githubAPI, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "registrycheck: FAIL\n%v\n", err)
		os.Exit(1)
	}
	fmt.Println("registrycheck: PASS")
}

func run(sitesDir, reservedPath, base, author, qtiOrg, githubToken, githubAPI string, dryRun bool) error {
	reservedSrc, err := os.ReadFile(reservedPath)
	if err != nil {
		return fmt.Errorf("reading reserved-word config: %w", err)
	}
	reserved, err := registry.ParseReserved(reservedSrc)
	if err != nil {
		return err
	}

	current, err := registry.LoadAll(sitesDir)
	if err != nil {
		return err
	}

	changes, err := registry.ChangedFiles(base, sitesDir)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return errors.New("no changed files under " + sitesDir + " found against " + base)
	}

	var gh registry.GitHub = registry.HTTPGitHub{Token: githubToken, BaseURL: githubAPI}
	if dryRun {
		gh = alwaysAllow{}
	}

	ctx := context.Background()
	var errs []error
	for _, c := range changes {
		subdomain, ok := registry.SiteEntryFile(c, sitesDir)
		if !ok {
			continue // nested file or non-entry file changed alongside
		}

		switch c.Status {
		case 'A':
			entry, ok := current[subdomain]
			if !ok {
				errs = append(errs, fmt.Errorf("%s: added but not found on disk at HEAD", c.Path))
				continue
			}
			others := make(map[string]registry.Entry, len(current)-1)
			for k, v := range current {
				if k != subdomain {
					others[k] = v
				}
			}
			if err := registry.CheckRegistration(ctx, gh, registry.RegistrationInput{
				Entry:    entry,
				Author:   author,
				Existing: others,
				Reserved: reserved,
				QTIOrg:   qtiOrg,
			}); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", c.Path, err))
			}

		case 'M':
			oldEntry, err := registry.EntryAtRef(base, c.Path)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", c.Path, err))
				continue
			}
			newEntry, ok := current[subdomain]
			if !ok {
				errs = append(errs, fmt.Errorf("%s: modified but not found on disk at HEAD", c.Path))
				continue
			}
			if err := registry.CheckEdit(registry.EditInput{
				Existing: oldEntry,
				New:      newEntry,
				Author:   author,
			}); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", c.Path, err))
			}

		case 'D':
			oldEntry, err := registry.EntryAtRef(base, c.Path)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", c.Path, err))
				continue
			}
			if err := registry.CheckUnregister(registry.UnregisterInput{
				Existing: oldEntry,
				Author:   author,
			}); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", c.Path, err))
				continue
			}
			fmt.Fprintf(os.Stderr, "registrycheck: %s: deletion authorized; unregistration cleanup happens via the nightly cron, not this check\n", c.Path)

		default:
			errs = append(errs, registry.ErrUnsupportedChange(c.Path, c.Status))
		}
	}

	return errors.Join(errs...)
}

type alwaysAllow struct{}

func (alwaysAllow) HasWriteAccess(context.Context, string, string, string) (bool, error) {
	return true, nil
}
func (alwaysAllow) OrgMember(context.Context, string, string) (bool, error) { return false, nil }
func (alwaysAllow) PathExists(context.Context, string, string, string, string) (bool, error) {
	return true, nil
}
