package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Store struct {
	mu       sync.RWMutex
	path     string
	data     *Dashboard
	onChange func(*Dashboard)
}

func NewStore(dataDir string, onChange func(*Dashboard)) (*Store, error) {
	path := filepath.Join(dataDir, "dashboard.yaml")
	s := &Store{path: path, onChange: onChange}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		s.data = DefaultDashboard()
		return s.save()
	}
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	d := DefaultDashboard()
	if err := yaml.Unmarshal(raw, d); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	s.data = d
	return nil
}

func (s *Store) save() error {
	raw, err := yaml.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0600)
}

// Get returns a deep copy of the current dashboard so callers can safely
// hold the result after the read-lock is released.
func (s *Store) Get() *Dashboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw, _ := json.Marshal(s.data)
	var d Dashboard
	_ = json.Unmarshal(raw, &d)
	return &d
}

func (s *Store) UpdateTheme(t Theme) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Theme = t
	return s.save()
}

func (s *Store) UpsertWidget(w Widget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Widgets {
		if existing.ID == w.ID {
			s.data.Widgets[i] = w
			return s.save()
		}
	}
	s.data.Widgets = append(s.data.Widgets, w)
	return s.save()
}

func (s *Store) DeleteWidget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := s.data.Widgets[:0]
	for _, w := range s.data.Widgets {
		if w.ID != id {
			filtered = append(filtered, w)
		}
	}
	s.data.Widgets = filtered
	return s.save()
}

func (s *Store) UpdateLayouts(positions []struct {
	ID string `json:"id"`
	Position
}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	posMap := make(map[string]Position, len(positions))
	for _, p := range positions {
		posMap[p.ID] = p.Position
	}
	for i, w := range s.data.Widgets {
		if pos, ok := posMap[w.ID]; ok {
			s.data.Widgets[i].Position = pos
		}
	}
	return s.save()
}

// Reload is called by the file watcher when the YAML changes externally.
func (s *Store) Reload() error {
	s.mu.Lock()
	err := s.load()
	data := s.data
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if s.onChange != nil {
		s.onChange(data)
	}
	return nil
}

func (s *Store) Path() string { return s.path }
