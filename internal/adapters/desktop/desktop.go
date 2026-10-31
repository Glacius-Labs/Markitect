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
	if cfg.OpenWindow == nil {
		cfg.OpenWindow = OpenWindow
	}
	host, _, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		return fmt.Errorf("--listen %q: %w", cfg.Listen, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("--listen %q: Markitect serves only on a loopback address", cfg.Listen)
	}
	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return err
	}
	// Two starts at once both find no service; publishing the record is
	// exclusive, so the later one finds the earlier one's service instead.
	for range 3 {
		if r, data, found := readRecord(cfg.Dir); found {
			if alive(ctx, r) {
				return openOn(ctx, cfg, r)
			}
			removeIfUnchanged(cfg.Dir, data)
		}
		if err := serve(ctx, cfg); !errors.Is(err, fs.ErrExist) {
			return err
		}
	}
	return errors.New("another Markitect keeps starting at the same time; try again")
}

// serve listens, publishes the record and serves until ctx ends. It returns
// fs.ErrExist when another service published its record first.
func serve(ctx context.Context, cfg Config) error {
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	defer ln.Close()
	secret, err := newToken()
	if err != nil {
		return err
	}
	r := record{Address: ln.Addr().String(), Secret: secret, PID: os.Getpid()}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err := publishRecord(cfg.Dir, data); err != nil {
		return err
	}
	defer removeIfUnchanged(cfg.Dir, data)
	_, port, _ := net.SplitHostPort(r.Address)
	s, err := newServer(secret, port, &Store{File: filepath.Join(cfg.Dir, projectsFile)}, cfg.Repos, cfg.Commands)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	fmt.Fprintf(cfg.Stdout, "Markitect %s runs at http://%s/\n", buildinfo.Build(), r.Address)
	if err := show(cfg, r.Address, s.launch()); err != nil {
		fmt.Fprintln(cfg.Stdout, err)
	}
	select {
	case <-ctx.Done():
	case err := <-done:
		return err
	}
	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if srv.Shutdown(stop) != nil {
		// A connection a client opened ahead and never used counts as busy
		// for its first seconds; after the grace period, close what is left.
		return srv.Close()
	}
	return nil
}

// openOn opens a window on the running service with a fresh launch code.
func openOn(ctx context.Context, cfg Config, r record) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+r.Address+"/api/launch", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.Secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var v struct {
		Code string `json:"code"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&v) != nil || v.Code == "" {
		return fmt.Errorf("the running Markitect at %s refused a window (%s)", r.Address, resp.Status)
	}
	fmt.Fprintf(cfg.Stdout, "Markitect is already running at http://%s/\n", r.Address)
	return show(cfg, r.Address, v.Code)
}

// show opens a window with a launch code, or prints its link.
func show(cfg Config, address, code string) error {
	url := "http://" + address + "/?t=" + code
	if !cfg.Open {
		fmt.Fprintf(cfg.Stdout, "open %s within %s; the link works once\n", url, launchLife)
		return nil
	}
	if err := cfg.OpenWindow(url); err != nil {
		return fmt.Errorf("no window opened (%v); open %s within %s", err, url, launchLife)
	}
	return nil
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
