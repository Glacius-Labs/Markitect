package agentd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

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
	s.home = &privateHome{claude: req.Claude != nil}
	if err := s.home.install(s.cfg.PrivateHome, privateFiles(req)); err != nil {
		_ = s.home.clear() // shutdown retries and reports any remaining cleanup error
		message := "private home initialization failed"
		if errors.Is(err, errClaudeConfigNotEmpty) {
			message += ": " + err.Error()
		}
		return s.refuse(request, message)
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
	return []privateFile{
		{".markitect", "system-prompt.md", []byte(req.Claude.SystemPrompt)},
		{".markitect", "append-system-prompt.md", []byte(req.Claude.AppendSystemPrompt)},
		{".claude", ".credentials.json", req.Claude.Credentials},
	}
}

// errClaudeConfigNotEmpty refuses a fresh home whose .claude already holds
// entries, for example copied up from an image's /home/agent/.claude.
var errClaudeConfigNotEmpty = errors.New("the private .claude directory is not empty, for example because the image ships /home/agent/.claude")

// claudeUserScope names what Claude Code loads from its user-scope
// configuration directory besides the login: settings (any settings*.json,
// which can carry hooks, environment and permissions), memory, and agents,
// skills, commands, output styles, plugins, rules, hooks and workflows.
var claudeUserScope = []string{"CLAUDE.md", "CLAUDE.local.md", "agents", "skills", "commands", "output-styles", "plugins", "rules", "hooks", "workflows"}

func claudeUserScopeEntry(name string) bool {
	return slices.Contains(claudeUserScope, name) || strings.HasPrefix(name, "settings") && strings.HasSuffix(name, ".json")
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
	claude  bool
	// instructions are the Claude instruction files with the bytes installed,
	// verified before every start.
	instructions []privateFile
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
	// An image's /home/agent/.claude reaches a fresh home volume through
	// copy-up; Claude Code would load it as user-scope configuration.
	if h.claude {
		dir, err := h.dir(".claude")
		if err != nil {
			return refuse()
		}
		if names, err := dir.names(); err != nil || len(names) != 0 {
			return errClaudeConfigNotEmpty
		}
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
		if h.claude && file.dir == ".markitect" {
			h.instructions = append(h.instructions, file)
		}
	}
	return nil
}

// verify runs before every start of a Claude home. The Manager's tools run as
// the provider's user and could rewrite the home between responses; each
// fresh provider process would then load the change. The instruction files
// must still hold exactly the installed bytes, and no user-scope Claude Code
// entry may have appeared. A Codex home has nothing to verify.
func (h *privateHome) verify() error {
	if !h.claude {
		return nil
	}
	for _, file := range h.instructions {
		dir := h.dirs[file.dir]
		if current, err := h.root.Lstat(file.dir); err != nil || !os.SameFile(dir.info, current) {
			return fmt.Errorf("%s was replaced", file.dir)
		}
		info, err := dir.root.Lstat(file.name)
		if err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(file.data)) {
			return fmt.Errorf("%s/%s is no longer the installed file", file.dir, file.name)
		}
		f, err := dir.root.Open(file.name)
		if err != nil {
			return fmt.Errorf("%s/%s is unreadable", file.dir, file.name)
		}
		data, err := io.ReadAll(io.LimitReader(f, int64(len(file.data))+1))
		closeErr := f.Close()
		if err != nil || closeErr != nil || !bytes.Equal(data, file.data) {
			return fmt.Errorf("%s/%s changed", file.dir, file.name)
		}
	}
	dir := h.dirs[".claude"]
	if current, err := h.root.Lstat(".claude"); err != nil || !os.SameFile(dir.info, current) {
		return errors.New(".claude was replaced")
	}
	names, err := dir.names()
	if err != nil {
		return errors.New(".claude is unreadable")
	}
	for _, name := range names {
		if claudeUserScopeEntry(name) {
			return fmt.Errorf(".claude/%s appeared; user-scope Claude Code settings, memory and extensions may not load", name)
		}
	}
	return nil
}

// names lists the directory's entries through its held handle.
func (d *privateDir) names() ([]string, error) {
	f, err := d.root.Open(".")
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	return names, errors.Join(err, f.Close())
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
