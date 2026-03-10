package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Store persists plan versions to disk under ~/.pluto/history/{session_id}/.
type Store struct {
	baseDir string
}

// New creates a Store rooted at ~/.pluto/history/.
func New() *Store {
	home, _ := os.UserHomeDir()
	return &Store{baseDir: filepath.Join(home, ".pluto", "history")}
}

func (s *Store) sessionDir(sessionID string) string {
	return filepath.Join(s.baseDir, sessionID)
}

// LoadLatest returns the content of the most recently saved plan for the session,
// or ("", nil) if none exists.
func (s *Store) LoadLatest(sessionID string) (string, error) {
	dir := s.sessionDir(sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", nil
	}

	sort.Strings(names)
	latest := names[len(names)-1]
	data, err := os.ReadFile(filepath.Join(dir, latest))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SavePlan persists planText as the next numbered file for the session.
func (s *Store) SavePlan(sessionID, planText string) error {
	dir := s.sessionDir(sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	entries, _ := os.ReadDir(dir)
	n := 1
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}

	filename := fmt.Sprintf("%04d.txt", n)
	return os.WriteFile(filepath.Join(dir, filename), []byte(planText), 0644)
}
