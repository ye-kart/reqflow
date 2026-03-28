package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

// Entry represents a single request/response history entry.
type Entry struct {
	ID        string              `json:"id"`
	Timestamp time.Time           `json:"timestamp"`
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Status    int                 `json:"status"`
	Duration  time.Duration       `json:"duration"`
	Request   domain.HTTPRequest  `json:"request"`
	Response  domain.HTTPResponse `json:"response"`
}

// Store manages history entries on disk as JSON files.
type Store struct {
	dir        string
	maxEntries int
}

// NewStore creates a new Store that persists entries in dir.
func NewStore(dir string, maxEntries int) *Store {
	return &Store{
		dir:        dir,
		maxEntries: maxEntries,
	}
}

// Add persists a history entry. If adding the entry would exceed maxEntries,
// the oldest entries are evicted.
func (s *Store) Add(entry Entry) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("creating history dir: %w", err)
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}

	path := filepath.Join(s.dir, entry.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing entry: %w", err)
	}

	return s.evict()
}

// List returns up to limit entries sorted by timestamp descending (most recent first).
func (s *Store) List(limit int) ([]Entry, error) {
	entries, err := s.readAll()
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}

	return entries, nil
}

// Get returns a single entry by ID.
func (s *Store) Get(id string) (Entry, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Entry{}, fmt.Errorf("history entry %q not found", id)
		}
		return Entry{}, fmt.Errorf("reading entry: %w", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return Entry{}, fmt.Errorf("parsing entry: %w", err)
	}

	return entry, nil
}

// Search returns entries whose URL contains the query string (case-insensitive).
func (s *Store) Search(query string) ([]Entry, error) {
	all, err := s.readAll()
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	var results []Entry
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.URL), lowerQuery) {
			results = append(results, e)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	return results, nil
}

// Clear removes all history entries from the store directory.
func (s *Store) Clear() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading history dir: %w", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(s.dir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// GenerateID creates a unique, timestamp-based ID for a history entry.
func GenerateID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%06d", now.Format("20060102-150405"), now.Nanosecond()/1000)
}

// readAll reads all JSON entries from the store directory.
func (s *Store) readAll() ([]Entry, error) {
	dirEntries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading history dir: %w", err)
	}

	var entries []Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.dir, de.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // skip unreadable files
		}

		var entry Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue // skip malformed files
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// evict removes the oldest entries if the total exceeds maxEntries.
func (s *Store) evict() error {
	entries, err := s.readAll()
	if err != nil {
		return err
	}

	if len(entries) <= s.maxEntries {
		return nil
	}

	// Sort oldest first.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	toRemove := len(entries) - s.maxEntries
	for i := 0; i < toRemove; i++ {
		path := filepath.Join(s.dir, entries[i].ID+".json")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("evicting %s: %w", entries[i].ID, err)
		}
	}

	return nil
}
