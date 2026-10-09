package server

import "github.com/quiet-terminal-interactive/qtidocs/internal/storage"

type StorageResolver struct {
	Store storage.Store
}

func (r StorageResolver) Resolve(subdomain string) (string, bool) {
	e, ok, err := r.Store.Get(subdomain)
	if err != nil || !ok || e.OutputDir == "" {
		return "", false
	}
	return e.OutputDir, true
}

func (r StorageResolver) Versions(subdomain string) ([]string, bool) {
	e, ok, err := r.Store.Get(subdomain)
	if err != nil || !ok || len(e.Versions) == 0 {
		return nil, false
	}
	names := make([]string, len(e.Versions))
	for i, v := range e.Versions {
		names[i] = v.Name
	}
	return names, true
}

func (r StorageResolver) Title(subdomain string) (string, bool) {
	e, ok, err := r.Store.Get(subdomain)
	if err != nil || !ok || e.Title == "" {
		return "", false
	}
	return e.Title, true
}
