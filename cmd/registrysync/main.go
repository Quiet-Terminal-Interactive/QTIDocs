package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
)

func main() {
	sitesDir := flag.String("sites-dir", "sites", "registry repo directory holding one <subdomain>.yaml per site")
	base := flag.String("base", "", "git ref/SHA to diff against to find changed site files (e.g. the push event's \"before\" SHA; required)")
	platformURL := flag.String("platform-url", "", "URL of the platform's registration endpoint (required)")
	secret := flag.String("secret", os.Getenv("QTIDOCS_PLATFORM_SECRET"), "shared secret authenticating against the platform (defaults to $QTIDOCS_PLATFORM_SECRET)")
	flag.Parse()

	if *base == "" {
		fmt.Fprintln(os.Stderr, "registrysync: -base is required")
		flag.Usage()
		os.Exit(2)
	}
	if *platformURL == "" {
		fmt.Fprintln(os.Stderr, "registrysync: -platform-url is required")
		flag.Usage()
		os.Exit(2)
	}
	if *secret == "" {
		fmt.Fprintln(os.Stderr, "registrysync: -secret (or $QTIDOCS_PLATFORM_SECRET) is required")
		flag.Usage()
		os.Exit(2)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	if err := run(*sitesDir, *base, *platformURL, *secret, client); err != nil {
		fmt.Fprintf(os.Stderr, "registrysync: FAIL\n%v\n", err)
		os.Exit(1)
	}
	fmt.Println("registrysync: PASS")
}

func run(sitesDir, base, platformURL, secret string, client *http.Client) error {
	current, err := registry.LoadAll(sitesDir)
	if err != nil {
		return err
	}

	changes, err := registry.ChangedFiles(base, sitesDir)
	if err != nil {
		return err
	}

	var errs []error
	for _, c := range changes {
		subdomain, ok := registry.SiteEntryFile(c, sitesDir)
		if !ok {
			continue
		}

		switch c.Status {
		case 'A', 'M':
			entry, ok := current[subdomain]
			if !ok {
				errs = append(errs, fmt.Errorf("%s: changed but not found on disk at HEAD", c.Path))
				continue
			}
			if err := register(client, platformURL, secret, entry); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", c.Path, err))
			}

		case 'D':
			fmt.Fprintf(os.Stderr, "registrysync: %s: deleted; unregistration happens via the nightly cleanup cron, not here\n", c.Path)

		default:
			errs = append(errs, registry.ErrUnsupportedChange(c.Path, c.Status))
		}
	}

	return errors.Join(errs...)
}

func register(client *http.Client, platformURL, secret string, entry registry.Entry) error {
	versions := make([]struct {
		Name string `json:"name"`
		Ref  string `json:"ref"`
	}, len(entry.Versions))
	for i, v := range entry.Versions {
		versions[i] = struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
		}{v.Name, v.Ref}
	}

	body, err := json.Marshal(struct {
		Subdomain string `json:"subdomain"`
		Title     string `json:"title"`
		Repo      string `json:"repo"`
		Branch    string `json:"branch"`
		Path      string `json:"path"`
		Contact   string `json:"contact"`
		Versions  []struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
		} `json:"versions,omitempty"`
		DefaultVersion string `json:"default_version,omitempty"`
	}{entry.Subdomain, entry.Title, entry.Repo, entry.Branch, entry.Path, entry.Contact, versions, entry.DefaultVersion})
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, platformURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling platform: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("platform returned %s", resp.Status)
	}

	var result struct {
		SecretEmailed bool `json:"secret_emailed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding platform response: %w", err)
	}

	fmt.Printf("registrysync: registered %s\n", entry.Subdomain)
	if result.SecretEmailed {
		fmt.Printf("registrysync: %s: deploy secret emailed to the entry's contact address\n", entry.Subdomain)
	}
	return nil
}
