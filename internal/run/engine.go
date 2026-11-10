// Package run is the record of what was verified (DEC-099): for each Definition,
// .markitect/review.sum holds the inputs at which a reviewer last verified it,
// modelled on Go's test cache. status says what is to verify, mark records a
// verdict at the state the reviewer read, and the hook keeps the record and the
// spec out of the agents' hands. Nothing here starts or drives an agent, steers
// the work or uses Git (DEC-096, DEC-097).
//
// The record is described in internal/run/README.md.
package run

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Engine reads a project folder and its record.
type Engine struct {
	// Folder reads the project folder, and Cache is the user's cache folder for
	// markitect, where the lock of a project's record lives.
	Folder Folder
	Cache  string
}

// folder is the project folder as one command finds it.
type folder struct {
	root  string
	files project.Repository
	set   map[string]bool
	cur   *project.Project
	// hashes holds the hashes read so far, by file.
	hashes map[string]string
}

// readFolder reads and loads the project folder that holds dir.
func (e *Engine) readFolder(dir string) (*folder, error) {
	root, err := e.Folder.Root(dir)
	if err != nil {
		return nil, err
	}
	files, err := e.Folder.Open(root)
	if err != nil {
		return nil, err
	}
	cur, err := project.Load(files)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, f := range cur.Files {
		set[f] = true
	}
	return &folder{root: root, files: files, set: set, cur: cur, hashes: map[string]string{}}, nil
}

// hash is the hash of a file of the folder as an input: "" when the walk does not
// find it.
func (f *folder) hash(file string) (string, error) {
	if h, ok := f.hashes[file]; ok {
		return h, nil
	}
	if !f.set[file] {
		return "", nil
	}
	data, err := f.files.Read(file)
	if err != nil {
		return "", err
	}
	h := Hash(data)
	f.hashes[file] = h
	return h, nil
}

// lockDir is where the locks of the records live, outside the project.
func (e *Engine) lockDir() (string, error) {
	if e.Cache == "" {
		return "", errors.New("no cache folder for the lock of the record")
	}
	return filepath.Join(e.Cache, "locks"), nil
}

// readRecordFile reads a project's record; nil without one.
func readRecordFile(root string) ([]byte, error) {
	file := filepath.Join(root, filepath.FromSlash(project.ReviewFile))
	var data []byte
	var err error
	for i := 0; i < 20; i++ {
		if data, err = os.ReadFile(file); err == nil || errors.Is(err, os.ErrNotExist) {
			break
		}
		// A rename beside a read is refused for a moment on Windows.
		time.Sleep(50 * time.Millisecond)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}
