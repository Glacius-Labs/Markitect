package desktop

import (
	"bytes"
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
)

// ui is the built UI: desktop/ in the repository, built into ui/ and
// committed, so that go build needs no Node.
//
//go:embed all:ui
var ui embed.FS

// csp lets the page load only its own files. Inline style attributes stay
// blocked; styles set through the DOM, as React sets them, are not affected.
const csp = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// launchLife is how long a launch code in a window's link stays valid.
const launchLife = 2 * time.Minute

// server serves the UI and its JSON routes to the windows it launched. A
// window's link carries a launch code that works once and only briefly; it
// buys the session cookie. The record's secret only proves a later start to
// the service and never reaches a browser.
type server struct {
	secret   string
	session  string
	cookie   string // the session cookie's name; a browser keeps cookies per host, not per port
	address  string // the address the service listens on, as a window's link names it
	port     string
	store    *Store
	repos    Repos
	commands Commands
	models   *models
	mux      *http.ServeMux
	now      func() time.Time

	// openWindow and chooseFolder are the window's, as Config describes
	// them; chooseFolder may be nil.
	openWindow   func(url string) error
	chooseFolder func(ctx context.Context, start string) (string, error)
	dialog       sync.Mutex // held while a folder dialog is open

	mu       sync.Mutex
	launches map[string]time.Time // launch code → when it expires
}

func newServer(secret, address string, store *Store, repos Repos, commands Commands) (*server, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	session, err := newToken()
	if err != nil {
		return nil, err
	}
	s := &server{secret: secret, session: session, cookie: "markitect_" + port, address: address, port: port, store: store, repos: repos, commands: commands,
		models: newModels(), mux: http.NewServeMux(), now: time.Now, launches: map[string]time.Time{}}
	files, err := fs.Sub(ui, "ui")
	if err != nil {
		return nil, err
	}
	s.mux.HandleFunc("GET /api/app", s.app)
	s.mux.HandleFunc("POST /api/folder", s.folderDialog)
	s.mux.HandleFunc("GET /api/projects", s.listProjects)
	s.mux.HandleFunc("POST /api/projects", s.addProject)
	s.mux.HandleFunc("GET /api/projects/{id}", s.getProject)
	s.mux.HandleFunc("DELETE /api/projects/{id}", s.removeProject)
	s.mux.HandleFunc("POST /api/projects/{id}/init", s.initProject)
	s.mux.HandleFunc("GET /api/projects/{id}/model", s.modelRoute)
	s.mux.HandleFunc("PUT /api/projects/{id}/definitions/{path...}", s.saveDefinition)
	s.mux.Handle("GET /", http.FileServerFS(files))
	return s, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Security-Policy", csp)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	// A page of another site that resolves its own name to 127.0.0.1 sends
	// its own name as the host; only the loopback names pass.
	if r.Host != "127.0.0.1:"+s.port && r.Host != "localhost:"+s.port && r.Host != "[::1]:"+s.port {
		http.Error(w, "unknown host", http.StatusMisdirectedRequest)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		h.Set("Cache-Control", "no-store")
	}
	// A write that names an origin must name the service's own, and a POST
	// or PUT must be JSON, which a form of another page cannot send.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
			http.Error(w, "foreign origin", http.StatusForbidden)
			return
		}
		if (r.Method == http.MethodPost || r.Method == http.MethodPut) && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "JSON only", http.StatusUnsupportedMediaType)
			return
		}
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/" && r.URL.Query().Has("t"):
		if !s.redeem(r.URL.Query().Get("t")) {
			http.Error(w, "This link is no longer valid. Start Markitect again with markitect desktop.", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: s.cookie, Value: s.session, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/api/hello":
		// A later start asks for proof that this service holds the record's
		// secret; the answer does not reveal it.
		reply(w, map[string]string{"proof": proof(s.secret, r.URL.Query().Get("c"))})
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/window":
		// A later start, which holds the record's secret, has the service
		// open its window.
		if s.laterStart(w, r) {
			if err := s.window(); err != nil {
				fail(w, http.StatusInternalServerError, err)
			} else {
				reply(w, map[string]bool{"opened": true})
			}
		}
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/launch":
		// A later start without a window gets a window's link.
		if s.laterStart(w, r) {
			reply(w, map[string]string{"code": s.launch()})
		}
		return
	}
	if c, err := r.Cookie(s.cookie); err != nil || !equal(c.Value, s.session) {
		http.Error(w, "Open Markitect with markitect desktop.", http.StatusUnauthorized)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// launch makes a launch code for a window's link.
func (s *server) launch() string {
	code, err := newToken()
	if err != nil {
		panic(err) // crypto/rand does not fail on supported platforms
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for c, until := range s.launches {
		if now.After(until) {
			delete(s.launches, c)
		}
	}
	s.launches[code] = now.Add(launchLife)
	return code
}

// laterStart reports whether the request carries the record's secret, and
// refuses it otherwise.
func (s *server) laterStart(w http.ResponseWriter, r *http.Request) bool {
	if !equal(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), s.secret) {
		fail(w, http.StatusUnauthorized, errors.New("not this user's Markitect"))
		return false
	}
	return true
}

// window opens a window with a fresh launch code. When none opens, the error
// carries the link, so that whoever asked can open it.
func (s *server) window() error {
	url := "http://" + s.address + "/?t=" + s.launch()
	if err := s.openWindow(url); err != nil {
		return fmt.Errorf("no window opened (%v); open %s within %s", err, url, launchLife)
	}
	return nil
}

// redeem uses a launch code up; it works once and only before it expires.
func (s *server) redeem(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.launches[code]
	delete(s.launches, code)
	return ok && !s.now().After(until)
}

func equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// projectView is a project with its current status.
type projectView struct {
	Project
	Status Status `json:"status"`
}

func (s *server) view(ctx context.Context, p Project) projectView {
	return projectView{Project: p, Status: readStatus(ctx, s.repos, p)}
}

func (s *server) app(w http.ResponseWriter, r *http.Request) {
	parent, err := s.store.Parent()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if parent == "" {
		parent, _ = os.UserHomeDir()
	}
	reply(w, map[string]any{"version": buildinfo.Build(), "parent": parent, "folderDialog": s.chooseFolder != nil})
}

// folderDialog asks the user for a folder in the window's own dialog, one
// dialog at a time. The answer is the folder, or "" when the user cancelled.
func (s *server) folderDialog(w http.ResponseWriter, r *http.Request) {
	if s.chooseFolder == nil {
		fail(w, http.StatusNotImplemented, errors.New("this window has no folder dialog; type the folder's path"))
		return
	}
	var req struct {
		Start string `json:"start"` // the folder to start in, if it is a full path
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if !filepath.IsAbs(req.Start) {
		req.Start = ""
	}
	if !s.dialog.TryLock() {
		fail(w, http.StatusConflict, errors.New("a folder dialog is already open"))
		return
	}
	defer s.dialog.Unlock()
	path, err := s.chooseFolder(r.Context(), req.Start)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	reply(w, map[string]string{"path": path})
}

// statusReaders bounds how many projects are read at once.
const statusReaders = 4

func (s *server) listProjects(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.List()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	views := make([]projectView, len(list))
	slots := make(chan struct{}, statusReaders)
	var wg sync.WaitGroup
	for i, p := range list {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; wg.Done() }()
			views[i] = s.view(r.Context(), p)
		}()
	}
	wg.Wait()
	reply(w, map[string]any{"projects": views})
}

func (s *server) getProject(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if ok {
		reply(w, map[string]any{"project": s.view(r.Context(), p)})
	}
}

func (s *server) removeProject(w http.ResponseWriter, r *http.Request) {
	if p, ok := s.find(w, r); ok {
		if err := s.store.Remove(p.ID); err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) initProject(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if !ok {
		return
	}
	var out bytes.Buffer
	done, err := s.commands.Init(r.Context(), p.Path, &out)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	reply(w, map[string]any{"project": s.view(r.Context(), p), "ok": done, "output": out.String()})
}

func (s *server) find(w http.ResponseWriter, r *http.Request) (Project, bool) {
	p, ok, err := s.store.Get(r.PathValue("id"))
	switch {
	case err != nil:
		fail(w, http.StatusInternalServerError, err)
	case !ok:
		fail(w, http.StatusNotFound, errors.New("no such project"))
	}
	return p, err == nil && ok
}

// addRequest brings a project into the list in one of three ways.
type addRequest struct {
	Action string `json:"action"` // open, create or clone
	Path   string `json:"path"`   // open: a folder inside the repository
	Parent string `json:"parent"` // create and clone: the folder that receives it
	Name   string `json:"name"`   // create: the new folder; clone: optional
	URL    string `json:"url"`    // clone: the repository to clone
}

func (s *server) addProject(w http.ResponseWriter, r *http.Request) {
	var req addRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()
	var root string
	var out bytes.Buffer
	done := true
	switch req.Action {
	case "open":
		if !filepath.IsAbs(req.Path) {
			fail(w, http.StatusBadRequest, errors.New("give the folder's full path"))
			return
		}
		ws, err := s.repos.Open(ctx, req.Path)
		if err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		root = ws.Root()
	case "create", "clone":
		name := req.Name
		if req.Action == "clone" {
			if req.URL = strings.TrimSpace(req.URL); req.URL == "" || strings.HasPrefix(req.URL, "-") {
				fail(w, http.StatusBadRequest, errors.New("give the repository's URL"))
				return
			}
			if name == "" {
				name = nameFromURL(req.URL)
			}
		}
		target, err := newFolder(req.Parent, name)
		if err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		if req.Action == "clone" {
			if err := s.repos.Clone(ctx, req.URL, target); err != nil {
				fail(w, http.StatusBadRequest, err)
				return
			}
		} else {
			if err := os.Mkdir(target, 0o755); err != nil {
				fail(w, http.StatusBadRequest, err)
				return
			}
			if err := s.repos.Init(ctx, target); err != nil {
				fail(w, http.StatusInternalServerError, err)
				return
			}
			if done, err = s.commands.Init(ctx, target, &out); err != nil {
				fail(w, http.StatusInternalServerError, err)
				return
			}
		}
		root = target
	default:
		fail(w, http.StatusBadRequest, fmt.Errorf("unknown action %q: open, create or clone", req.Action))
		return
	}
	if req.Action != "open" {
		// The next new project most likely goes beside this one; a lost
		// suggestion costs nothing.
		_ = s.store.SetParent(req.Parent)
	}
	p, err := s.store.Add(root)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	reply(w, map[string]any{"project": s.view(ctx, p), "ok": done, "output": out.String()})
}

// newFolder checks that a new project folder can be made: the parent exists,
// the name is one plain folder name, and nothing has that name yet.
func newFolder(parent, name string) (string, error) {
	if !filepath.IsAbs(parent) {
		return "", errors.New("give the parent folder's full path")
	}
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s is no folder", parent)
	}
	if !plainName(name) {
		return "", fmt.Errorf("%q is no plain folder name: no separator, reserved character, leading dot, trailing dot or space, or device name such as CON", name)
	}
	target := filepath.Join(parent, name)
	if _, err := os.Lstat(target); err == nil {
		return "", fmt.Errorf("%s already exists", target)
	}
	return target, nil
}

// plainName reports whether name is a folder name that means the same on
// Windows, Linux and macOS: no separator or character Windows refuses, no
// control character, no leading dot, no trailing dot or space, and none of
// Windows' device names, which name a device even with an extension.
func plainName(name string) bool {
	if name == "" || strings.HasPrefix(name, ".") || strings.TrimSpace(name) != name || strings.HasSuffix(name, ".") || strings.ContainsAny(name, `/\:*?"<>|`) {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	stem, _, _ := strings.Cut(strings.ToUpper(name), ".")
	switch stem {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}

// nameFromURL is the folder git clone would choose: the URL's last part
// without .git.
func nameFromURL(url string) string {
	url = strings.TrimRight(url, "/")
	if i := strings.LastIndexAny(url, "/:\\"); i >= 0 {
		url = url[i+1:]
	}
	return strings.TrimSuffix(url, ".git")
}

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
