package registry

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	yaml "go.yaml.in/yaml/v3"

	"github.com/quiet-terminal-interactive/qtidocs/internal/mailer"
)

var subdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

var versionNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func ValidVersionName(name string) bool {
	return versionNamePattern.MatchString(name)
}

type Version struct {
	Name string `yaml:"name"`
	Ref  string `yaml:"ref"`
}

type Entry struct {
	Subdomain      string    `yaml:"subdomain"`
	Title          string    `yaml:"title,omitempty"`
	Repo           string    `yaml:"repo"`
	Branch         string    `yaml:"branch"`
	Path           string    `yaml:"path"`
	Contact        string    `yaml:"contact"`
	Maintainers    []string  `yaml:"maintainers"`
	Versions       []Version `yaml:"versions,omitempty"`
	DefaultVersion string    `yaml:"default_version,omitempty"`
}

func ParseEntry(src []byte) (Entry, error) {
	var e Entry
	if err := yaml.Unmarshal(src, &e); err != nil {
		return Entry{}, fmt.Errorf("registry: %w", err)
	}
	return e, nil
}

func (e Entry) Filename() string {
	return e.Subdomain + ".yaml"
}

func (e Entry) Validate() error {
	var errs []error

	if !subdomainPattern.MatchString(e.Subdomain) {
		errs = append(errs, fmt.Errorf("subdomain %q must match %s", e.Subdomain, subdomainPattern.String()))
	}
	if _, _, err := ParseRepo(e.Repo); err != nil {
		errs = append(errs, err)
	}
	if e.Branch == "" {
		errs = append(errs, errors.New("branch is required"))
	}
	if e.Path == "" {
		errs = append(errs, errors.New("path is required"))
	}
	if e.Contact == "" {
		errs = append(errs, errors.New("contact is required (the email address your deploy secret is sent to)"))
	} else if _, err := mailer.ParseAddress(e.Contact); err != nil {
		errs = append(errs, fmt.Errorf("contact: %w", err))
	}
	if len(e.Maintainers) == 0 {
		errs = append(errs, errors.New("maintainers must list at least one GitHub username"))
	}
	for _, m := range e.Maintainers {
		if strings.TrimSpace(m) == "" {
			errs = append(errs, errors.New("maintainers entries must not be blank"))
			break
		}
	}

	seen := make(map[string]bool, len(e.Versions))
	for _, v := range e.Versions {
		if v.Name == "" || v.Ref == "" {
			errs = append(errs, errors.New("versions entries require both name and ref"))
			continue
		}
		if !ValidVersionName(v.Name) {
			errs = append(errs, fmt.Errorf("version name %q must match %s", v.Name, versionNamePattern.String()))
			continue
		}
		if seen[v.Name] {
			errs = append(errs, fmt.Errorf("duplicate version name %q", v.Name))
		}
		seen[v.Name] = true
	}
	if e.DefaultVersion != "" && !seen[e.DefaultVersion] {
		errs = append(errs, fmt.Errorf("default_version %q is not listed in versions", e.DefaultVersion))
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("registry: invalid entry: %w", errors.Join(errs...))
}

func ParseRepo(repo string) (owner, name string, err error) {
	s := strings.TrimPrefix(repo, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimPrefix(s, "github.com/")
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("registry: repo %q must look like github.com/owner/name", repo)
	}
	return parts[0], parts[1], nil
}
