package analytics

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	rawDirName = "raw"
	aggDirName = "agg"
	rawExt     = ".jsonl"
)

type perSubdomainLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func (l *perSubdomainLocks) lock(subdomain string) func() {
	l.mu.Lock()
	m, ok := l.locks[subdomain]
	if !ok {
		m = &sync.Mutex{}
		if l.locks == nil {
			l.locks = map[string]*sync.Mutex{}
		}
		l.locks[subdomain] = m
	}
	l.mu.Unlock()

	m.Lock()
	return m.Unlock
}

type FileStore struct {
	dir   string
	locks perSubdomainLocks
}

func Open(dir string) *FileStore {
	return &FileStore{dir: dir}
}

func (s *FileStore) Record(subdomain string, e Event) error {
	defer s.locks.lock(subdomain)()

	dir := filepath.Join(s.dir, rawDirName, subdomain)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("analytics: creating %s: %w", dir, err)
	}

	day := e.Timestamp.UTC().Format(dateLayout)
	f, err := os.OpenFile(filepath.Join(dir, day+rawExt), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("analytics: opening raw log: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("analytics: encoding event: %w", err)
	}
	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("analytics: writing raw log: %w", err)
	}
	return nil
}

func (s *FileStore) Rollup(subdomain string, day time.Time) error {
	defer s.locks.lock(subdomain)()

	dateKey := day.UTC().Format(dateLayout)
	events, err := s.readRawDay(subdomain, dateKey)
	if err != nil {
		return err
	}

	aggMap, err := s.readAgg(subdomain)
	if err != nil {
		return err
	}
	if aggMap == nil {
		aggMap = map[string]DailyAggregate{}
	}
	aggMap[dateKey] = computeDaily(events)
	return s.writeAgg(subdomain, aggMap)
}

func (s *FileStore) Purge(subdomain string, before time.Time) error {
	defer s.locks.lock(subdomain)()

	dir := filepath.Join(s.dir, rawDirName, subdomain)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("analytics: listing %s: %w", dir, err)
	}

	cutoff := before.UTC().Format(dateLayout)
	for _, entry := range entries {
		name := entry.Name()
		date, ok := dateFromRawName(name)
		if !ok || date >= cutoff {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("analytics: removing %s: %w", name, err)
		}
	}
	return nil
}

func (s *FileStore) Aggregate(subdomain string, from *time.Time, to time.Time) (Summary, error) {
	defer s.locks.lock(subdomain)()

	aggMap, err := s.readAgg(subdomain)
	if err != nil {
		return Summary{}, err
	}
	rawDates, err := s.rawDates(subdomain)
	if err != nil {
		return Summary{}, err
	}

	dates := map[string]bool{}
	for d := range aggMap {
		dates[d] = true
	}
	for _, d := range rawDates {
		dates[d] = true
	}

	toKey := to.UTC().Format(dateLayout)
	var fromKey string
	if from != nil {
		fromKey = from.UTC().Format(dateLayout)
	}

	byPath := map[string]int{}
	byReferrer := map[string]int{}
	var views, uniques int

	for d := range dates {
		if d > toKey {
			continue
		}
		if from != nil && d < fromKey {
			continue
		}
		day, ok := aggMap[d]
		if !ok {
			events, err := s.readRawDay(subdomain, d)
			if err != nil {
				return Summary{}, err
			}
			day = computeDaily(events)
		}
		merge(byPath, byReferrer, &views, &uniques, day)
	}

	return Summary{
		Views:     views,
		Uniques:   uniques,
		TopPages:  rankPages(byPath),
		Referrers: rankReferrers(byReferrer),
	}, nil
}

func (s *FileStore) Subdomains() ([]string, error) {
	dir := filepath.Join(s.dir, rawDirName)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("analytics: listing %s: %w", dir, err)
	}

	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func (s *FileStore) readRawDay(subdomain, dateKey string) ([]Event, error) {
	path := filepath.Join(s.dir, rawDirName, subdomain, dateKey+rawExt)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("analytics: reading %s: %w", path, err)
	}

	var events []Event
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("analytics: decoding %s: %w", path, err)
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *FileStore) rawDates(subdomain string) ([]string, error) {
	dir := filepath.Join(s.dir, rawDirName, subdomain)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("analytics: listing %s: %w", dir, err)
	}

	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if date, ok := dateFromRawName(e.Name()); ok {
			out = append(out, date)
		}
	}
	return out, nil
}

func dateFromRawName(name string) (string, bool) {
	if !strings.HasSuffix(name, rawExt) {
		return "", false
	}
	return strings.TrimSuffix(name, rawExt), true
}

func (s *FileStore) readAgg(subdomain string) (map[string]DailyAggregate, error) {
	path := filepath.Join(s.dir, aggDirName, subdomain+".json")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("analytics: reading %s: %w", path, err)
	}

	var agg map[string]DailyAggregate
	if err := json.Unmarshal(data, &agg); err != nil {
		return nil, fmt.Errorf("analytics: decoding %s: %w", path, err)
	}
	return agg, nil
}

func (s *FileStore) writeAgg(subdomain string, agg map[string]DailyAggregate) error {
	dir := filepath.Join(s.dir, aggDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("analytics: creating %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(agg, "", "  ")
	if err != nil {
		return fmt.Errorf("analytics: encoding aggregate: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".agg-*.tmp")
	if err != nil {
		return fmt.Errorf("analytics: creating temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("analytics: writing %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("analytics: closing %s: %w", tmpPath, err)
	}

	path := filepath.Join(dir, subdomain+".json")
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("analytics: renaming %s to %s: %w", tmpPath, path, err)
	}
	return nil
}
