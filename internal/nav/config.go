package nav

import (
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

type ConfigEntry struct {
	Title string        `yaml:"title"`
	Path  string        `yaml:"path"`
	Items []ConfigEntry `yaml:"items"`
}

func ParseConfig(src []byte) ([]ConfigEntry, error) {
	var cfg []ConfigEntry
	if err := yaml.Unmarshal(src, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func FromConfig(cfg []ConfigEntry, pages []Page) []*Entry {
	byPath := make(map[string]Page, len(pages))
	for _, p := range pages {
		byPath[strings.Trim(p.URLPath, "/")] = p
	}
	return fromConfigEntries(cfg, byPath)
}

func fromConfigEntries(cfg []ConfigEntry, byPath map[string]Page) []*Entry {
	if len(cfg) == 0 {
		return nil
	}

	entries := make([]*Entry, 0, len(cfg))
	for _, c := range cfg {
		title := c.Title
		if title == "" {
			if p, ok := byPath[strings.Trim(c.Path, "/")]; ok {
				title = p.Title
			}
		}
		entries = append(entries, &Entry{
			Title:    title,
			Path:     c.Path,
			Children: fromConfigEntries(c.Items, byPath),
		})
	}
	return entries
}
