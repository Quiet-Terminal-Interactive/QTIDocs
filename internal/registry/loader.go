package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadAll(dir string) (map[string]Entry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("registry: reading %s: %w", dir, err)
	}

	entries := make(map[string]Entry, len(files))
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, f.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("registry: reading %s: %w", path, err)
		}
		e, err := ParseEntry(src)
		if err != nil {
			return nil, fmt.Errorf("registry: %s: %w", path, err)
		}
		wantName := strings.TrimSuffix(f.Name(), ".yaml")
		if e.Subdomain != wantName {
			return nil, fmt.Errorf("registry: %s: subdomain field %q does not match filename", path, e.Subdomain)
		}
		entries[e.Subdomain] = e
	}
	return entries, nil
}
