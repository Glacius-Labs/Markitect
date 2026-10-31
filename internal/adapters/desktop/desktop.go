// Package desktop is Markitect Desktop, the app over the command line that
// users know as Markitect: one service per user that serves the embedded UI
// and its JSON routes on a loopback address, and the window that shows it. It
// reaches Git and the commands that change a project through ports the
// composition supplies, and it parses no command output.
package desktop

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Workspace is the working tree of a project's repository.
type Workspace interface {
	project.Repository
	Root() string
	// Stamp is a cheap mark of one file, such as its size and modification
	// time, that changes when the file does.
	Stamp(file string) string
	// Write writes a file of the working tree, creating its folders; a
	// reader sees the old file or the new one, never a part.
	Write(file string, data []byte) error
	// Remove removes a file of the working tree.
	Remove(file string) error
}

// Commit is one commit of a project's repository.
type Commit interface {
	project.Repository
	ID() string
	// Changed lists the repository files that differ between the commit and
	// the working tree, outside Markitect's folder.
	Changed() ([]analysis.Change, error)
}

// Repos opens, creates and clones Git repositories.
type Repos interface {
	// Open opens the working tree of the repository that contains dir.
	Open(ctx context.Context, dir string) (Workspace, error)
	// Commit resolves a revision, such as HEAD, of the repository at root.
	Commit(ctx context.Context, root, rev string) (Commit, error)
	// LastLanded returns the newest commit reachable from tip that adds a
	// Run's Result page, the base apply starts from; ok is false when no Run
	// has landed.
	LastLanded(ctx context.Context, root, tip string) (id string, ok bool, err error)
	// Init makes the existing, empty folder dir a Git repository.
	Init(ctx context.Context, dir string) error
	// Clone clones url into dir, which does not exist yet.
	Clone(ctx context.Context, url, dir string) error
}

// Commands runs the markitect commands that change a project. Their output
// goes to out unchanged; ok says whether the command succeeded.
type Commands interface {
	Init(ctx context.Context, dir string, out io.Writer) (ok bool, err error)
}

// Config is what the desktop service needs.
type Config struct {
	Dir      string // the user's config folder for Markitect
	Listen   string // the loopback address to serve on; port 0 picks one
	Open     bool   // open a window once the service runs
	Repos    Repos
	Commands Commands
	Stdout   io.Writer
	// OpenWindow opens a URL in a window; nil opens it with OpenWindow. The
	// service calls it for every window, also for those a later start asks
	// for, so the windows are always the first start's kind.
	OpenWindow func(url string) error
	// ChooseFolder asks the user for a folder in the window's own dialog,
	// starting at start, and returns "" when the user cancels. Nil means the
	// window has no such dialog, and the user types the path.
	ChooseFolder func(ctx context.Context, start string) (string, error)
	// Chats starts the chat agents; nil means the app has no chat.
	Chats Chats
}

// Service is this user's running desktop service. It answers from the moment
// Listen returns it, so that a later start finds it alive.
type Service struct {
	cfg    Config
	record []byte
	srv    *server
	http   *http.Server
	done   chan error // the HTTP server's end
	closed sync.Once
}

// record is the running service as its file names it, so that a later start
// opens a window on it instead of starting another. The secret proves that a
// start belongs to the same user; it never leaves the file and the service.
type record struct {
	Address string `json:"address"`
	Secret  string `json:"secret"`
	PID     int    `json:"pid"`
}

const recordFile = "desktop.json"

// Start runs the desktop service until ctx ends, or, when a service of this
// user already runs, opens a window on it and returns.
func Start(ctx context.Context, cfg Config) error {
	s, err := Listen(ctx, cfg)
	if s == nil {
		return err
	}
	return s.Serve(ctx)
}

// Listen claims this user's desktop service. When one already runs, it asks
// that one for a window, or prints a window's link, and returns nil.
// Otherwise it starts a new service, publishes its record and returns it;
// Serve then opens the first window. A native shell starts its window loop
// in between.
func Listen(ctx context.Context, cfg Config) (*Service, error) {
	if cfg.OpenWindow == nil {
		cfg.OpenWindow = OpenWindow
	}
	host, _, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		return nil, fmt.Errorf("--listen %q: %w", cfg.Listen, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, fmt.Errorf("--listen %q: Markitect serves only on a loopback address", cfg.Listen)
	}
	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return nil, err
	}
	// Two starts at once both find no service; publishing the record is
	// exclusive, so the later one finds the earlier one's service instead.
	for range 3 {
		if r, data, found := readRecord(cfg.Dir); found {
			if alive(ctx, r) {
				return nil, openOn(ctx, cfg, r)
			}
			removeIfUnchanged(cfg.Dir, data)
		}
		if s, err := listen(cfg); !errors.Is(err, fs.ErrExist) {
			return s, err
		}
	}
	return nil, errors.New("another Markitect keeps starting at the same time; try again")
}

// listen listens, publishes the record and starts answering. It returns
// fs.ErrExist when another service published its record first.
func listen(cfg Config) (_ *Service, err error) {
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			ln.Close()
		}
	}()
	secret, err := newToken()
	if err != nil {
		return nil, err
	}
	r := record{Address: ln.Addr().String(), Secret: secret, PID: os.Getpid()}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	srv, err := newServer(secret, r.Address, &Store{File: filepath.Join(cfg.Dir, projectsFile)}, cfg.Repos, cfg.Commands)
	if err != nil {
		return nil, err
	}
	srv.chat.chats = cfg.Chats
	srv.openWindow, srv.chooseFolder = cfg.OpenWindow, cfg.ChooseFolder
	if err := publishRecord(cfg.Dir, data); err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, record: data, srv: srv, done: make(chan error, 1),
		http: &http.Server{Handler: srv, ReadHeaderTimeout: 10 * time.Second}}
	go func() { s.done <- s.http.Serve(ln) }()
	return s, nil
}

// Serve opens the first window, or prints its link, and serves until ctx
// ends; then it closes the service.
func (s *Service) Serve(ctx context.Context) error {
	defer s.Close()
	fmt.Fprintf(s.cfg.Stdout, "Markitect %s runs at http://%s/\n", buildinfo.Build(), s.srv.address)
	if !s.cfg.Open {
		printLink(s.cfg, s.srv.address, s.srv.launch())
	} else if err := s.Window(); err != nil {
		fmt.Fprintln(s.cfg.Stdout, err)
	}
	select {
	case <-ctx.Done():
		return nil
	case err := <-s.done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil // closed by its shell
		}
		return err
	}
}

// Window opens another window on the service, as the tray of a native shell
// asks for one.
func (s *Service) Window() error { return s.srv.window() }

// Close stops the service and removes its record. Serve closes the service
// when it ends; a shell that never serves closes it itself.
func (s *Service) Close() {
	s.closed.Do(func() {
		removeIfUnchanged(s.cfg.Dir, s.record)
		s.srv.chat.close()
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if s.http.Shutdown(stop) != nil {
			// A connection a client opened ahead and never used counts as busy
			// for its first seconds; after the grace period, close what is left.
			_ = s.http.Close()
		}
	})
}

// laterStartWait bounds how long a later start waits for the running service
// to open its window or hand out a link.
var laterStartWait = 15 * time.Second

// openOn asks the running service for a window, or, without one, prints the
// link of a fresh launch code. The service opens the window itself, so a
// native shell's windows stay its own.
func openOn(ctx context.Context, cfg Config, r record) error {
	path := "/api/launch"
	if cfg.Open {
		path = "/api/window"
	}
	ctx, cancel := context.WithTimeout(ctx, laterStartWait)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+r.Address+path, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.Secret)
	allowForeground(r.PID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("the running Markitect at %s did not answer: %w", r.Address, err)
	}
	defer resp.Body.Close()
	var v struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&v)
	switch {
	case resp.StatusCode != http.StatusOK && v.Error != "":
		return fmt.Errorf("the running Markitect at %s: %s", r.Address, v.Error)
	case resp.StatusCode == http.StatusUnauthorized:
		// It proved the record's secret, yet does not know the route: a
		// service of an earlier build.
		return fmt.Errorf("the running Markitect at %s is an earlier version; quit it and start Markitect again", r.Address)
	case resp.StatusCode != http.StatusOK || !cfg.Open && v.Code == "":
		return fmt.Errorf("the running Markitect at %s refused a window (%s)", r.Address, resp.Status)
	}
	fmt.Fprintf(cfg.Stdout, "Markitect is already running at http://%s/\n", r.Address)
	if !cfg.Open {
		printLink(cfg, r.Address, v.Code)
	}
	return nil
}

// printLink prints a window's link with its launch code.
func printLink(cfg Config, address, code string) {
	fmt.Fprintf(cfg.Stdout, "open http://%s/?t=%s within %s; the link works once\n", address, code, launchLife)
}

// alive asks the recorded service to prove that it holds the record's
// secret, without sending the secret: a program that took over the port
// after a crash cannot answer.
func alive(ctx context.Context, r record) bool {
	if r.Address == "" || r.Secret == "" {
		return false
	}
	challenge, err := newToken()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+r.Address+"/api/hello?c="+challenge, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var v struct {
		Proof string `json:"proof"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&v) != nil {
		return false
	}
	return hmac.Equal([]byte(v.Proof), []byte(proof(r.Secret, challenge)))
}

// proof is what a service that holds secret answers to challenge.
func proof(secret, challenge string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(challenge))
	return hex.EncodeToString(mac.Sum(nil))
}

// readRecord returns the record with its bytes. A record that cannot be read,
// damaged or from an older build, has no address, so it counts as stale.
func readRecord(dir string) (record, []byte, bool) {
	data, err := os.ReadFile(filepath.Join(dir, recordFile))
	if err != nil {
		return record{}, nil, false
	}
	var r record
	_ = json.Unmarshal(data, &r)
	return r, data, true
}

// publishRecord writes the record only if none exists: through a temporary
// file and a link, so that a reader never sees half of it. Where the file
// system has no hard links, it creates the record exclusively instead.
func publishRecord(dir string, data []byte) error {
	file := filepath.Join(dir, recordFile)
	tmp, err := os.CreateTemp(dir, recordFile+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmp.Name(), file); err == nil || errors.Is(err, fs.ErrExist) {
		return err
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(file)
		return err
	}
	return f.Close()
}

// removeIfUnchanged removes the record if it still holds data, so that a
// record another service published meanwhile stays. On Windows a reader's
// open handle refuses the removal for a moment, so it tries a few times.
func removeIfUnchanged(dir string, data []byte) {
	file := filepath.Join(dir, recordFile)
	for range 5 {
		_, current, found := readRecord(dir)
		if !found || !bytes.Equal(current, data) {
			return
		}
		if err := os.Remove(file); err == nil || errors.Is(err, fs.ErrNotExist) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// writeFile replaces file through a temporary file in the same folder, so a
// reader never sees half of it.
func writeFile(file string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(file), filepath.Base(file)+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), file); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("no random token: " + err.Error())
	}
	return hex.EncodeToString(b), nil
}
