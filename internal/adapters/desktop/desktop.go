// Package desktop is Markitect Desktop, the app over the command line that
// users know as Markitect: one service per user that serves the embedded UI
// and its JSON routes on a loopback address, and the window that shows it. It
// reaches Git and the commands that change a project through ports the
// composition supplies, and it parses no command output.
package desktop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Workspace is the working tree of a project's repository.
type Workspace interface {
	project.Repository
	Root() string
}

// Repos opens, creates and clones Git repositories.
type Repos interface {
	// Open opens the working tree of the repository that contains dir.
	Open(ctx context.Context, dir string) (Workspace, error)
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
	// OpenWindow opens a URL in a window; nil opens it with OpenWindow.
	OpenWindow func(url string) error
}

// record is the running service as its file names it, so that a second
// start opens a window on it instead of starting another.
type record struct {
	Address string `json:"address"`
	Token   string `json:"token"`
	PID     int    `json:"pid"`
}

const recordFile = "desktop.json"

// Start runs the desktop service until ctx ends, or, when a service of this
// user already runs, opens a window on it and returns.
func Start(ctx context.Context, cfg Config) error {
	if cfg.OpenWindow == nil {
		cfg.OpenWindow = OpenWindow
	}
	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return err
	}
	if r, ok := running(ctx, cfg.Dir); ok {
		return show(cfg, r, "Markitect is already running")
	}
	host, _, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		return fmt.Errorf("--listen %q: %w", cfg.Listen, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("--listen %q: Markitect serves only on a loopback address", cfg.Listen)
	}
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	token, err := newToken()
	if err != nil {
		ln.Close()
		return err
	}
	r := record{Address: ln.Addr().String(), Token: token, PID: os.Getpid()}
	if err := writeRecord(cfg.Dir, r); err != nil {
		ln.Close()
		return err
	}
	defer removeRecord(cfg.Dir, r)
	_, port, _ := net.SplitHostPort(r.Address)
	srv := &http.Server{
		Handler:           newServer(token, port, &Store{File: filepath.Join(cfg.Dir, projectsFile)}, cfg.Repos, cfg.Commands),
		ReadHeaderTimeout: 10 * time.Second,
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	if err := show(cfg, r, "Markitect "+buildinfo.Build()+" runs"); err != nil {
		fmt.Fprintln(cfg.Stdout, err)
	}
	select {
	case <-ctx.Done():
	case err := <-served:
		return err
	}
	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(stop)
}

// show prints where the service runs and opens a window on it.
func show(cfg Config, r record, what string) error {
	url := "http://" + r.Address + "/?t=" + r.Token
	fmt.Fprintf(cfg.Stdout, "%s at http://%s/\n", what, r.Address)
	if !cfg.Open {
		fmt.Fprintf(cfg.Stdout, "open %s\n", url)
		return nil
	}
	if err := cfg.OpenWindow(url); err != nil {
		return fmt.Errorf("no window opened (%v); open %s", err, url)
	}
	return nil
}

// running returns the record of a service that answers.
func running(ctx context.Context, dir string) (record, bool) {
	data, err := os.ReadFile(filepath.Join(dir, recordFile))
	if err != nil {
		return record{}, false
	}
	var r record
	if json.Unmarshal(data, &r) != nil || r.Address == "" || r.Token == "" {
		return record{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+r.Address+"/api/ping", nil)
	if err != nil {
		return record{}, false
	}
	req.AddCookie(&http.Cookie{Name: cookieName, Value: r.Token})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return record{}, false
	}
	resp.Body.Close()
	return r, resp.StatusCode == http.StatusOK
}

func writeRecord(dir string, r record) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(dir, recordFile), data)
}

// removeRecord removes the record unless a later service replaced it.
func removeRecord(dir string, r record) {
	file := filepath.Join(dir, recordFile)
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	var current record
	if json.Unmarshal(data, &current) == nil && current.Token == r.Token {
		os.Remove(file)
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
