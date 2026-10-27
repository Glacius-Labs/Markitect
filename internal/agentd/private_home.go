package agentd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

func (s *server) refuse(request uint64, message string) error {
	return s.sendControl(frame.ControlMessage{Type: frame.TypeFailure, Request: request, Failure: &frame.Failure{Message: message}})
}

func (s *server) initialize(request uint64, req frame.Initialize) error {
	// Control requests are handled serially. Initialization is one-shot even
	// after a failed write; no provider can observe partial setup or retry it.
	if s.homeAttempted || s.providerAttempted {
		return s.refuse(request, "private home initialization is no longer available")
	}
	s.homeAttempted = true
	s.home = &privateHome{}
	if err := s.home.install(s.cfg.PrivateHome, privateFiles(req)); err != nil {
		_ = s.home.clear() // shutdown retries and reports any remaining cleanup error
		return s.refuse(request, "private home initialization failed")
	}
	s.homeReady = true
	return s.sendControl(frame.ControlMessage{Type: frame.TypeInitialized, Request: request, Initialized: &frame.Initialized{}})
}

// privateFile is one allowlisted file of a provider home: a fixed directory
// directly below the home and a fixed file name in it. No caller chooses a path.
type privateFile struct {
	dir, name string
	data      []byte
}

// privateFiles maps one validated initialization to its fixed files.
func privateFiles(req frame.Initialize) []privateFile {
	if req.Claude == nil {
		return []privateFile{{".codex", "auth.json", req.CodexAuth}}
	}
	files := []privateFile{
		{".markitect", "system-prompt.md", []byte(req.Claude.SystemPrompt)},
		{".markitect", "append-system-prompt.md", []byte(req.Claude.AppendSystemPrompt)},
	}
	if len(req.Claude.Credentials) != 0 {
		files = append(files, privateFile{".claude", ".credentials.json", req.Claude.Credentials})
	}
	return files
}

func (s *server) clearPrivateHome(request uint64) error {
	if p := s.current(); p != nil {
		select {
		case <-p.reported:
		default:
			return s.refuse(request, "private home cleanup requires provider exit")
		}
	}
	s.homeReady = false
	if s.home != nil {
		if err := s.home.clear(); err != nil {
			return s.refuse(request, "private home cleanup failed")
		}
	}
	return s.sendControl(frame.ControlMessage{Type: frame.TypePrivateHomeCleared, Request: request, PrivateHomeCleared: &frame.PrivateHomeCleared{}})
}

// privateHome holds anchored directory handles, not caller-selected paths.
// Initialization runs before any provider; the private home is never shared
// with another Manager or mounted from a mutable host login directory.
type privateHome struct {
	root    *os.Root
	dirs    map[string]*privateDir
	created []privateFile // installed files, removed again by clear
}

type privateDir struct {
	root *os.Root
	info os.FileInfo
}

func (h *privateHome) install(home string, files []privateFile) error {
	refuse := func() error { return errors.New("private home initialization failed") }
	if !filepath.IsAbs(home) || len(files) == 0 {
		return refuse()
	}
	info, err := os.Lstat(home)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return refuse()
	}
	h.root, err = os.OpenRoot(home)
	if err != nil {
		return refuse()
	}
	opened, err := h.root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return refuse()
	}
	for _, file := range files {
		if len(file.data) == 0 || len(file.data) > frame.MaxPayload {
			return refuse()
		}
		dir, err := h.dir(file.dir)
		if err != nil {
			return refuse()
		}
		// Exclusive creation refuses existing files, links and devices. No login
		// is overwritten and there is no shell or caller-supplied permission mode.
		f, err := dir.root.OpenFile(file.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return refuse()
		}
		h.created = append(h.created, privateFile{dir: file.dir, name: file.name})
		n, writeErr := f.Write(file.data)
		syncErr := f.Sync()
		closeErr := f.Close()
		current, statErr := h.root.Lstat(file.dir)
		if n != len(file.data) || writeErr != nil || syncErr != nil || closeErr != nil || statErr != nil || !os.SameFile(dir.info, current) {
			return refuse()
		}
	}
	return nil
}

// dir creates or opens one fixed private directory directly below the home
// and holds it, so later writes and removals never follow a replaced path.
func (h *privateHome) dir(name string) (*privateDir, error) {
	if d := h.dirs[name]; d != nil {
		return d, nil
	}
	refuse := errors.New("private home directory unavailable")
	if err := h.root.Mkdir(name, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, refuse
	}
	info, err := h.root.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
		return nil, refuse
	}
	root, err := h.root.OpenRoot(name)
	if err != nil {
		return nil, refuse
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = root.Close()
		return nil, refuse
	}
	if h.dirs == nil {
		h.dirs = map[string]*privateDir{}
	}
	d := &privateDir{root: root, info: opened}
	h.dirs[name] = d
	return d, nil
}

func (h *privateHome) clear() error {
	var remaining []privateFile
	for _, file := range h.created {
		// Unlink via the held directory even if the provider replaced the file
		// with a symlink. Never follow that link or remove another directory.
		dir := h.dirs[file.dir]
		err := dir.root.Remove(file.name)
		current, statErr := h.root.Lstat(file.dir)
		if err != nil && !errors.Is(err, os.ErrNotExist) || statErr != nil || !os.SameFile(dir.info, current) {
			remaining = append(remaining, file)
		}
	}
	h.created = remaining
	if len(remaining) != 0 {
		return errors.New("private home cleanup failed")
	}
	return nil
}

func (h *privateHome) close() error {
	var err error
	for _, d := range h.dirs {
		err = errors.Join(err, d.root.Close())
	}
	if h.root != nil {
		err = errors.Join(err, h.root.Close())
	}
	if err != nil {
		return errors.New("private home handles did not close")
	}
	return nil
}
