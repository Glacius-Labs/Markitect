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
	if err := s.home.install(s.cfg.PrivateHome, req.CodexAuth); err != nil {
		_ = s.home.clear() // shutdown retries and reports any remaining cleanup error
		return s.refuse(request, "private home initialization failed")
	}
	s.homeReady = true
	return s.sendControl(frame.ControlMessage{Type: frame.TypeInitialized, Request: request, Initialized: &frame.Initialized{}})
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
	root        *os.Root
	dir         *os.Root
	dirInfo     os.FileInfo
	authCreated bool
}

func (h *privateHome) install(home string, data []byte) error {
	refuse := func() error { return errors.New("private home initialization failed") }
	if !filepath.IsAbs(home) || len(data) == 0 || len(data) > frame.MaxPrivateAuthBytes {
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
	if err := h.root.Mkdir(".codex", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return refuse()
	}
	info, err = h.root.Lstat(".codex")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
		return refuse()
	}
	h.dir, err = h.root.OpenRoot(".codex")
	if err != nil {
		return refuse()
	}
	h.dirInfo, err = h.dir.Stat(".")
	if err != nil || !os.SameFile(info, h.dirInfo) {
		return refuse()
	}
	// Exclusive creation refuses existing files, links and devices. No login
	// is overwritten and there is no shell or caller-supplied permission mode.
	f, err := h.dir.OpenFile("auth.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return refuse()
	}
	h.authCreated = true
	n, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	current, statErr := h.root.Lstat(".codex")
	if n != len(data) || writeErr != nil || syncErr != nil || closeErr != nil || statErr != nil || !os.SameFile(h.dirInfo, current) {
		return refuse()
	}
	return nil
}

func (h *privateHome) clear() error {
	if !h.authCreated {
		return nil
	}
	// Unlink via the held directory even if the provider replaced the auth
	// file with a symlink. Never follow that link or remove another directory.
	err := h.dir.Remove("auth.json")
	current, statErr := h.root.Lstat(".codex")
	if err != nil && !errors.Is(err, os.ErrNotExist) || statErr != nil || !os.SameFile(h.dirInfo, current) {
		return errors.New("private home cleanup failed")
	}
	h.authCreated = false
	return nil
}

func (h *privateHome) close() error {
	var err error
	if h.dir != nil {
		err = h.dir.Close()
	}
	if h.root != nil {
		err = errors.Join(err, h.root.Close())
	}
	if err != nil {
		return errors.New("private home handles did not close")
	}
	return nil
}
