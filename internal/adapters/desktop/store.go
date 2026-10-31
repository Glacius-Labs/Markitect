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

// Store is the user's project list in one file of the config folder, with
// the folder that received the last new project.
type Store struct {
	File string
	mu   sync.Mutex
}

type storeFile struct {
	Projects []Project `json:"projects"`
	Parent   string    `json:"parent,omitempty"`
}

// List returns the projects by name.
func (s *Store) List() ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.read()
	return f.Projects, err
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
	f, err := s.read()
	if err != nil {
		return Project{}, err
	}
	p := Project{ID: projectID(root), Name: filepath.Base(root), Path: root}
	for _, q := range f.Projects {
		if q.ID == p.ID {
			return q, nil
		}
	}
	f.Projects = append(f.Projects, p)
	return p, s.write(f)
}

// Remove takes the project off the list; its folder stays as it is.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.read()
	if err != nil {
		return err
	}
	kept := f.Projects[:0]
	for _, p := range f.Projects {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	f.Projects = kept
	return s.write(f)
}

// Parent returns the folder that received the last new project.
func (s *Store) Parent() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.read()
	return f.Parent, err
}

// SetParent remembers the folder that received a new project.
func (s *Store) SetParent(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.read()
	if err != nil {
		return err
	}
	f.Parent = dir
	return s.write(f)
}

func (s *Store) read() (storeFile, error) {
	var f storeFile
	data, err := os.ReadFile(s.File)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return f, err
	default:
		if err := json.Unmarshal(data, &f); err != nil {
			return storeFile{}, errors.New(s.File + ": " + err.Error())
		}
	}
	if f.Projects == nil {
		f.Projects = []Project{}
	}
	sort.SliceStable(f.Projects, func(i, j int) bool {
		return strings.ToLower(f.Projects[i].Name) < strings.ToLower(f.Projects[j].Name)
	})
	return f, nil
}

func (s *Store) write(f storeFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(s.File, append(data, '\n'))
}

// projectID names a repository root stably. On Windows and macOS, whose file
// systems ignore case by default, paths compare without case.
func projectID(root string) string {
	key := filepath.Clean(root)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		key = strings.ToLower(key)
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:6])
}
