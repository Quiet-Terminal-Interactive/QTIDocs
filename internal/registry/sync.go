package registry

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Change struct {
	Status byte
	Path   string
}

func SiteEntryFile(c Change, dir string) (subdomain string, ok bool) {
	if filepath.Dir(c.Path) != filepath.Clean(dir) || !strings.HasSuffix(c.Path, ".yaml") {
		return "", false
	}
	return strings.TrimSuffix(filepath.Base(c.Path), ".yaml"), true
}

func ErrUnsupportedChange(path string, status byte) error {
	return fmt.Errorf("%s: unsupported change status %q; renames/copies aren't supported, split into a separate removal PR and registration PR", path, string(status))
}

func ChangedFiles(base, dir string) ([]Change, error) {
	out, err := exec.Command("git", "diff", "--name-status", base, "--", dir).Output()
	if err != nil {
		return nil, fmt.Errorf("registry: git diff against %s: %w", base, err)
	}

	var changes []Change
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		changes = append(changes, Change{Status: fields[0][0], Path: fields[len(fields)-1]})
	}
	return changes, nil
}

func EntryAtRef(ref, path string) (Entry, error) {
	out, err := exec.Command("git", "show", ref+":"+path).Output()
	if err != nil {
		return Entry{}, fmt.Errorf("registry: reading %s at %s: %w", path, ref, err)
	}
	return ParseEntry(out)
}
