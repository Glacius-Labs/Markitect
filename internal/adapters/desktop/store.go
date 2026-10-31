package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Project is one entry of the user's project list: a repository root.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

const projectsFile = "projects.json"

// Store is the user's project list in one file of the config folder.
type Store struct {
	File string
	mu   sync.Mutex
}

type storeFile struct {
	Projects []Project `json:"projects"`
}

// List returns the projects by name.
func (s *Store) List() ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

// Get returns the project with the ID.
func (s *Store) Get(id string) (Project, bool, error) {
	list, err := s.List()
	if err != nil {
		return Project{}, false, err
	}
	for _, p := range list {
		if p.ID == id {
			return p, true, nil
		}
	}
	return Project{}, false, nil
}

// Add adds the repository at root, or returns its entry if it is listed.
func (s *Store) Add(root string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.read()
	if err != nil {
		return Project{}, err
	}
	p := Project{ID: projectID(root), Name: filepath.Base(root), Path: root}
	for _, q := range list {
		if q.ID == p.ID {
			return q, nil
		}
	}
	return p, s.write(append(list, p))
}

// Remove takes the project off the list; its folder stays as it is.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.read()
	if err != nil {
		return err
	}
	kept := list[:0]
	for _, p := range list {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	return s.write(kept)
}

func (s *Store) read() ([]Project, error) {
	data, err := os.ReadFile(s.File)
	if errors.Is(err, fs.ErrNotExist) {
		return []Project{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f storeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, errors.New(s.File + ": " + err.Error())
	}
	if f.Projects == nil {
		f.Projects = []Project{}
	}
	sort.SliceStable(f.Projects, func(i, j int) bool {
		return strings.ToLower(f.Projects[i].Name) < strings.ToLower(f.Projects[j].Name)
	})
	return f.Projects, nil
}

func (s *Store) write(list []Project) error {
	data, err := json.MarshalIndent(storeFile{Projects: list}, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(s.File, append(data, '\n'))
}

// projectID names a repository root stably; Windows paths compare without
// case, as the file system does.
func projectID(root string) string {
	key := filepath.Clean(root)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:6])
}
