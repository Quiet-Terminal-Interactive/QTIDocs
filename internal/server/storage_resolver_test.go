package server

import (
	"errors"
	"testing"

	"github.com/quiet-terminal-interactive/qtidocs/internal/storage"
)

type fakeRoutingStore struct {
	entries map[string]storage.Entry
	getErr  error
}

func (f *fakeRoutingStore) Get(subdomain string) (storage.Entry, bool, error) {
	if f.getErr != nil {
		return storage.Entry{}, false, f.getErr
	}
	e, ok := f.entries[subdomain]
	return e, ok, nil
}
func (f *fakeRoutingStore) Set(e storage.Entry) error      { return nil }
func (f *fakeRoutingStore) List() ([]storage.Entry, error) { return nil, nil }
func (f *fakeRoutingStore) Delete(subdomain string) error  { return nil }

func TestStorageResolver_Resolve_Deployed(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{
		"acme": {Subdomain: "acme", OutputDir: "/out/acme"},
	}}
	r := StorageResolver{Store: store}
	dir, ok := r.Resolve("acme")
	if !ok || dir != "/out/acme" {
		t.Errorf("Resolve(acme) = (%q, %v)", dir, ok)
	}
}

func TestStorageResolver_Resolve_NotYetBuilt(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{
		"acme": {Subdomain: "acme", OutputDir: ""},
	}}
	r := StorageResolver{Store: store}
	_, ok := r.Resolve("acme")
	if ok {
		t.Error("expected Resolve = false when OutputDir is empty")
	}
}

func TestStorageResolver_Resolve_Unregistered(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{}}
	r := StorageResolver{Store: store}
	_, ok := r.Resolve("ghost")
	if ok {
		t.Error("expected Resolve = false for unregistered subdomain")
	}
}

func TestStorageResolver_Resolve_StoreError(t *testing.T) {
	store := &fakeRoutingStore{getErr: errors.New("disk error")}
	r := StorageResolver{Store: store}
	_, ok := r.Resolve("acme")
	if ok {
		t.Error("expected Resolve = false on store error")
	}
}

func TestStorageResolver_Versions_Versioned(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{
		"acme": {Subdomain: "acme", Versions: []storage.Version{{Name: "v1"}, {Name: "v2"}}},
	}}
	r := StorageResolver{Store: store}
	names, ok := r.Versions("acme")
	if !ok || len(names) != 2 || names[0] != "v1" || names[1] != "v2" {
		t.Errorf("Versions(acme) = (%v, %v)", names, ok)
	}
}

func TestStorageResolver_Versions_Unversioned(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{
		"acme": {Subdomain: "acme"},
	}}
	r := StorageResolver{Store: store}
	_, ok := r.Versions("acme")
	if ok {
		t.Error("expected Versions = false for an unversioned entry")
	}
}

func TestStorageResolver_Versions_Unregistered(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{}}
	r := StorageResolver{Store: store}
	_, ok := r.Versions("ghost")
	if ok {
		t.Error("expected Versions = false for unregistered subdomain")
	}
}

func TestStorageResolver_Title_Set(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{
		"acme": {Subdomain: "acme", Title: "Acme Docs"},
	}}
	r := StorageResolver{Store: store}
	title, ok := r.Title("acme")
	if !ok || title != "Acme Docs" {
		t.Errorf("Title(acme) = (%q, %v)", title, ok)
	}
}

func TestStorageResolver_Title_Unset(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{
		"acme": {Subdomain: "acme"},
	}}
	r := StorageResolver{Store: store}
	_, ok := r.Title("acme")
	if ok {
		t.Error("expected Title = false when no title is registered")
	}
}

func TestStorageResolver_Title_Unregistered(t *testing.T) {
	store := &fakeRoutingStore{entries: map[string]storage.Entry{}}
	r := StorageResolver{Store: store}
	_, ok := r.Title("ghost")
	if ok {
		t.Error("expected Title = false for unregistered subdomain")
	}
}
