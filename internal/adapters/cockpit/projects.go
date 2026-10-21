package cockpit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// ProjectsFormatVersion is the format of the per-user project list.
const ProjectsFormatVersion = 1

// DefaultModel is the model manifest of a repository, relative to it.
const DefaultModel = ".markitect/model.yaml"

// ProjectEntry is one project of the per-user list: a repository with the
// model and dependency models markitect check selects, and the runtime
// directory that markitect setup fills with runtime.yaml and run.json. A
// project without a runtime is observed but starts no run and no model
// session.
type ProjectEntry struct {
	Name         string   `json:"name"`
	Repository   string   `json:"repository"`
	Model        string   `json:"model"`
	Dependencies []string `json:"dependencies"`
	Runtime      string   `json:"runtime"`
}

// RunConfigPath is the run configuration setup wrote into the runtime
// directory; empty without a runtime.
func (p ProjectEntry) RunConfigPath() string {
	if p.Runtime == "" {
		return ""
	}
	return filepath.Join(p.Runtime, "run.json")
}

// RuntimeProfilePath is the Runtime profile setup wrote into the runtime
// directory; empty without a runtime.
func (p ProjectEntry) RuntimeProfilePath() string {
	if p.Runtime == "" {
		return ""
	}
	return filepath.Join(p.Runtime, "runtime.yaml")
}

// ProjectList is the content of the per-user project list. Last names the
// project the cockpit opened most recently.
type ProjectList struct {
	Projects []ProjectEntry
	Last     string
}

// Find returns the project of that name.
func (l ProjectList) Find(name string) (ProjectEntry, bool) {
	for _, p := range l.Projects {
		if p.Name == name {
			return p, true
		}
	}
	return ProjectEntry{}, false
}

// ProjectsFileError reports a project list that cannot be used. The cockpit
// never overwrites such a file: the owner fixes or removes it.
type ProjectsFileError struct {
	Path   string
	Reason string
}

func (e *ProjectsFileError) Error() string {
	return fmt.Sprintf("the project list %s cannot be used (%s); fix or remove it, the cockpit does not overwrite it", e.Path, e.Reason)
}

type projectsDocument struct {
	FormatVersion int            `json:"formatVersion"`
	Projects      []ProjectEntry `json:"projects"`
	Last          string         `json:"last"`
}

// ProjectStore is the per-user project list in one JSON file. Every change
// reads the file again and replaces it atomically, so another cockpit's
// change is not lost to a stale copy and a crash leaves the old file.
type ProjectStore struct {
	path string
	mu   sync.Mutex
}

// NewProjectStore selects the list's file, such as
// <user config dir>/markitect/cockpit/projects.json.
func NewProjectStore(path string) *ProjectStore { return &ProjectStore{path: path} }

// Path is the list's file.
func (s *ProjectStore) Path() string { return s.path }

// Load reads the list. A missing file is an empty list.
func (s *ProjectStore) Load() (ProjectList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Update reads the list, applies fn and replaces the file when fn succeeds.
func (s *ProjectStore) Update(fn func(*ProjectList) error) (ProjectList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return ProjectList{}, err
	}
	if err := fn(&list); err != nil {
		return list, err
	}
	return list, s.save(list)
}

// projectName admits the names the list and its routes use.
var projectName = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} ._-]{0,63}$`)

// ValidName reports whether name can name a project.
func ValidName(name string) bool { return projectName.MatchString(name) }

func (s *ProjectStore) bad(reason string) error {
	return &ProjectsFileError{Path: s.path, Reason: reason}
}

func (s *ProjectStore) load() (ProjectList, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return ProjectList{}, nil
	}
	if err != nil {
		return ProjectList{}, s.bad(err.Error())
	}
	var version struct {
		FormatVersion *int `json:"formatVersion"`
	}
	if err := json.Unmarshal(data, &version); err != nil {
		return ProjectList{}, s.bad("it is not JSON: " + err.Error())
	}
	if version.FormatVersion == nil {
		return ProjectList{}, s.bad("it has no formatVersion")
	}
	if *version.FormatVersion != ProjectsFormatVersion {
		return ProjectList{}, s.bad(fmt.Sprintf("format version %d is not the supported version %d", *version.FormatVersion, ProjectsFormatVersion))
	}
	var doc projectsDocument
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil || dec.More() {
		if err == nil {
			err = errors.New("trailing content")
		}
		return ProjectList{}, s.bad("it is malformed: " + err.Error())
	}
	seen := map[string]bool{}
	for i, p := range doc.Projects {
		switch {
		case !ValidName(p.Name):
			return ProjectList{}, s.bad(fmt.Sprintf("project %d has no valid name", i+1))
		case seen[p.Name]:
			return ProjectList{}, s.bad(fmt.Sprintf("the name %q is used twice", p.Name))
		case !filepath.IsAbs(p.Repository):
			return ProjectList{}, s.bad(fmt.Sprintf("the repository of %q is not an absolute path", p.Name))
		case p.Runtime != "" && !filepath.IsAbs(p.Runtime):
			return ProjectList{}, s.bad(fmt.Sprintf("the runtime of %q is not an absolute path", p.Name))
		}
		seen[p.Name] = true
		if doc.Projects[i].Model == "" {
			doc.Projects[i].Model = DefaultModel
		}
	}
	return ProjectList{Projects: doc.Projects, Last: doc.Last}, nil
}

// save writes the list to a temporary file next to it, then renames it over
// the list, so a reader sees the old or the new file, never a part.
func (s *ProjectStore) save(list ProjectList) error {
	doc := projectsDocument{FormatVersion: ProjectsFormatVersion, Projects: list.Projects, Last: list.Last}
	if doc.Projects == nil {
		doc.Projects = []ProjectEntry{}
	}
	for i := range doc.Projects {
		if doc.Projects[i].Dependencies == nil {
			doc.Projects[i].Dependencies = []string{}
		}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".projects-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	// CreateTemp creates the file with mode 0600, which the rename keeps.
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, s.path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// ValidateEntry checks what adding a project needs: absolute native
// repository, a model file in it, dependency models, and, with a runtime
// directory, the run configuration and Runtime profile setup wrote.
func ValidateEntry(p ProjectEntry) error {
	if !ValidName(p.Name) {
		return errors.New("the name needs 1 to 64 letters, digits, spaces, dots, underscores or hyphens, starting with a letter or digit")
	}
	if err := absoluteDirectory(p.Repository, "the repository"); err != nil {
		return err
	}
	model := p.Model
	if model == "" {
		model = DefaultModel
	}
	for _, m := range append([]string{model}, p.Dependencies...) {
		path := m
		if !filepath.IsAbs(m) {
			if !filepath.IsLocal(m) {
				return fmt.Errorf("the model path %q leaves the repository", m)
			}
			path = filepath.Join(p.Repository, m)
		}
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("the repository has no model file %s", path)
		}
	}
	if p.Runtime == "" {
		return nil
	}
	if err := absoluteDirectory(p.Runtime, "the runtime directory"); err != nil {
		return err
	}
	for _, f := range []string{"run.json", "runtime.yaml"} {
		if info, err := os.Stat(filepath.Join(p.Runtime, f)); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("the runtime directory has no %s; set the project up with markitect setup or the setup form", f)
		}
	}
	return nil
}

func absoluteDirectory(path, what string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("%s needs an absolute native path without redundant parts", what)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return fmt.Errorf("%s %s is not a directory", what, path)
	}
	return nil
}
