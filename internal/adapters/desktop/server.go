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
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
)

// ui is the built UI: desktop/ in the repository, built into ui/ and
// committed, so that go build needs no Node.
//
//go:embed all:ui
var ui embed.FS

const cookieName = "markitect"

// csp lets the page load only its own files. Inline style attributes stay
// blocked; styles set through the DOM, as React sets them, are not affected.
const csp = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// server serves the UI and its JSON routes to whoever holds the token: the
// window the service opened, after the token exchange.
type server struct {
	token    string
	port     string
	store    *Store
	repos    Repos
	commands Commands
	mux      *http.ServeMux
}

func newServer(token, port string, store *Store, repos Repos, commands Commands) *server {
	s := &server{token: token, port: port, store: store, repos: repos, commands: commands, mux: http.NewServeMux()}
	files, err := fs.Sub(ui, "ui")
	if err != nil {
		panic(err)
	}
	s.mux.HandleFunc("GET /api/ping", func(w http.ResponseWriter, r *http.Request) { reply(w, map[string]bool{"ok": true}) })
	s.mux.HandleFunc("GET /api/app", s.app)
	s.mux.HandleFunc("GET /api/projects", s.listProjects)
	s.mux.HandleFunc("POST /api/projects", s.addProject)
	s.mux.HandleFunc("GET /api/projects/{id}", s.getProject)
	s.mux.HandleFunc("DELETE /api/projects/{id}", s.removeProject)
	s.mux.HandleFunc("POST /api/projects/{id}/init", s.initProject)
	s.mux.Handle("GET /", http.FileServerFS(files))
	return s
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
	if r.Method == http.MethodGet && r.URL.Path == "/" && r.URL.Query().Has("t") {
		if !s.valid(r.URL.Query().Get("t")) {
			http.Error(w, "This link is no longer valid. Start Markitect again with markitect desktop.", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: s.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if c, err := r.Cookie(cookieName); err != nil || !s.valid(c.Value) {
		http.Error(w, "Open Markitect with markitect desktop.", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
			http.Error(w, "foreign origin", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "JSON only", http.StatusUnsupportedMediaType)
			return
		}
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		h.Set("Cache-Control", "no-store")
	}
	s.mux.ServeHTTP(w, r)
}

func (s *server) valid(token string) bool {
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) == 1
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
	home, _ := os.UserHomeDir()
	reply(w, map[string]string{"version": buildinfo.Build(), "home": home, "separator": string(filepath.Separator)})
}

func (s *server) listProjects(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.List()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	views := make([]projectView, 0, len(list))
	for _, p := range list {
		views = append(views, s.view(r.Context(), p))
	}
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
	if name == "" || name == "." || name == ".." || strings.TrimSpace(name) != name || strings.HasSuffix(name, ".") || strings.ContainsAny(name, `/\:*?"<>|`) {
		return "", fmt.Errorf("%q is no folder name", name)
	}
	target := filepath.Join(parent, name)
	if _, err := os.Lstat(target); err == nil {
		return "", fmt.Errorf("%s already exists", target)
	}
	return target, nil
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
