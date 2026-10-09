package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Version struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}

type Entry struct {
	Subdomain        string            `json:"subdomain"`
	Title            string            `json:"title,omitempty"`
	Repo             string            `json:"repo"`
	Branch           string            `json:"branch"`
	Path             string            `json:"path"`
	Contact          string            `json:"contact,omitempty"`
	OutputDir        string            `json:"output_dir,omitempty"`
	WebhookSecret    string            `json:"webhook_secret,omitempty"`
	DeployedSHA      string            `json:"deployed_sha,omitempty"`
	Versions         []Version         `json:"versions,omitempty"`
	DefaultVersion   string            `json:"default_version,omitempty"`
	DeployedVersions map[string]string `json:"deployed_versions,omitempty"`
}

func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("storage: generating secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type Store interface {
	Get(subdomain string) (Entry, bool, error)
	Set(e Entry) error
	List() ([]Entry, error)
	Delete(subdomain string) error
}

type FileStore struct {
	path string
	mu   sync.Mutex
}

func Open(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Get(subdomain string) (Entry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.read()
	if err != nil {
		return Entry{}, false, err
	}
	e, ok := entries[subdomain]
	return e, ok, nil
}

func (s *FileStore) Set(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.read()
	if err != nil {
		return err
	}
	if entries == nil {
		entries = make(map[string]Entry)
	}
	entries[e.Subdomain] = e
	return s.write(entries)
}

func (s *FileStore) Delete(subdomain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.read()
	if err != nil {
		return err
	}
	if _, ok := entries[subdomain]; !ok {
		return nil
	}
	delete(entries, subdomain)
	return s.write(entries)
}

func (s *FileStore) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.read()
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, e)
	}
	return out, nil
}

func (s *FileStore) read() (map[string]Entry, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return map[string]Entry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: reading %s: %w", s.path, err)
	}
	var entries map[string]Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("storage: decoding %s: %w", s.path, err)
	}
	return entries, nil
}

func (s *FileStore) write(entries map[string]Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: encoding routing table: %w", err)
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".routing-*.tmp")
	if err != nil {
		return fmt.Errorf("storage: creating temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("storage: writing %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("storage: closing %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("storage: renaming %s to %s: %w", tmpPath, s.path, err)
	}
	return nil
}
