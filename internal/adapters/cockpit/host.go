package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// OpenedProject is a project's Server with what closing the project needs:
// Close ends its model sessions and waits for a run or a publication it
// started, as a stopping cockpit does. It may be nil.
type OpenedProject struct {
	Server *Server
	Close  func() error
}

// ProjectOpener builds a project's Server from its entry, as the cockpit's
// flags would for the same repository, model, dependencies and runtime
// directory. It starts nothing; ctx ends the project's runs and sessions.
type ProjectOpener func(ctx context.Context, entry ProjectEntry) (OpenedProject, error)

// SetupInputs are the inputs of the function behind markitect setup for a
// project: the form's fields and the project's repository, model and
// dependencies in Project.
type SetupInputs struct {
	Project    ProjectEntry
	RuntimeDir string
	Image      string
	Credential string
	AgentModel string
	Effort     string
	Engine     string
	// Checks maps a check profile to its executable.
	Checks   map[string]string
	Store    string
	Deadline string
}

// SetupBound lists the identities setup observed and wrote.
type SetupBound struct {
	ImageID       string `json:"imageID"`
	AgentdVersion string `json:"agentdVersion"`
	CLIVersion    string `json:"cliVersion"`
	EngineVersion string `json:"engineVersion"`
}

// SetupFile is one file setup composed; Action is "written" or "unchanged".
type SetupFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Action string `json:"action"`
}

// SetupResult is what setup reported. Status is "passed", "failed" for a
// refusal or "not_completed"; Error is setup's own text, shown unchanged.
type SetupResult struct {
	Status        string      `json:"status"`
	Error         string      `json:"error,omitempty"`
	Bound         *SetupBound `json:"bound,omitempty"`
	AgentProfiles []string    `json:"agentProfiles"`
	CheckProfiles []string    `json:"checkProfiles"`
	Files         []SetupFile `json:"files"`
}

// SetupFunc calls the function behind markitect setup.
type SetupFunc func(ctx context.Context, in SetupInputs) SetupResult

// SetupDefaults are the form's starting values, as markitect setup defaults
// them.
type SetupDefaults struct {
	Engine string `json:"engine"`
	Effort string `json:"effort"`
}

// HostConfig composes the host. Without a Store the list is unavailable and
// Reason says why; the current project can still be served.
type HostConfig struct {
	Store  *ProjectStore
	Reason string
	Open   ProjectOpener
	Setup  SetupFunc
	// RuntimeBase is the directory under which the form suggests a runtime
	// directory per project.
	RuntimeBase string
	Defaults    SetupDefaults
}

// Host owns the cockpit's one HTTP handler and one swappable project. It
// serves the page and the project list, and delegates every other API route
// to the current project's Server, which stays single-project. Switching the
// project is refused while the current one is busy.
//
// A write to the current project is admitted and counted under the host's
// lock. A switch marks the project draining under the same lock, so no write
// is admitted after the busy state was read, waits for the writes already
// admitted and reads the busy state again before it swaps and closes.
type Host struct {
	cfg  HostConfig
	ctx  context.Context
	web  fs.FS
	addr string
	// wg joins the switches and setups under way; Serve waits for them.
	wg sync.WaitGroup

	mu        sync.Mutex
	idle      *sync.Cond // signalled when an admitted write ended
	cur       *hostProject
	switching bool
	setting   bool // a setup is under way
	closing   bool // Serve ended: no switch and no setup starts any more
	openError string
}

// hostProject is the current project. The fields below the handler belong to
// the host's lock.
type hostProject struct {
	entry   ProjectEntry
	opened  OpenedProject
	handler http.Handler

	inflight  int  // admitted writes that did not end
	draining  bool // a switch is under way: no write is admitted
	settingUp bool // a setup rewrites its runtime files: no run or session starts
}

func (p *hostProject) close() error {
	if p.opened.Close == nil {
		return nil
	}
	return p.opened.Close()
}

// NewHost composes a host whose projects end with ctx.
func NewHost(ctx context.Context, cfg HostConfig) (*Host, error) {
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		return nil, err
	}
	h := &Host{cfg: cfg, ctx: ctx, web: web}
	h.idle = sync.NewCond(&h.mu)
	return h, nil
}

// hostError is a refusal with its status.
type hostError struct {
	code int
	text string
}

func (e *hostError) Error() string { return e.text }

func refuseWith(code int, format string, args ...any) error {
	return &hostError{code, fmt.Sprintf(format, args...)}
}

func fail(w http.ResponseWriter, err error) {
	var he *hostError
	if errors.As(err, &he) {
		http.Error(w, he.text, he.code)
		return
	}
	var pe *ProjectsFileError
	if errors.As(err, &pe) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// Adopt makes an already built project the current one without saving it, as
// the flags of markitect cockpit open one directly.
func (h *Host) Adopt(entry ProjectEntry, opened OpenedProject) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cur = h.newProject(entry, opened)
}

func (h *Host) newProject(entry ProjectEntry, opened OpenedProject) *hostProject {
	opened.Server.host = h.addr
	return &hostProject{entry: entry, opened: opened, handler: opened.Server.Handler()}
}

// OpenLast opens the project the owner used last when it is still in the
// list. When that fails the host stays at the list and reports why.
func (h *Host) OpenLast() {
	if h.cfg.Store == nil {
		return
	}
	list, err := h.cfg.Store.Load()
	if err != nil || list.Last == "" {
		return
	}
	entry, ok := list.Find(list.Last)
	if !ok {
		return
	}
	if _, err := h.swap(entry); err != nil {
		h.mu.Lock()
		h.openError = fmt.Sprintf("The project %s could not be opened: %v", entry.Name, err)
		h.mu.Unlock()
	}
}

// Serve listens on the loopback address and serves until ctx ends, then ends
// the current project and waits for a setup that is under way.
func (h *Host) Serve(ctx context.Context, listen string, ready func(url string)) error {
	err := serveLoopback(ctx, listen, func(addr string) http.Handler {
		h.addr = addr
		h.mu.Lock()
		if h.cur != nil {
			h.cur.opened.Server.host = addr
			h.cur.handler = h.cur.opened.Server.Handler()
		}
		h.mu.Unlock()
		return h.Handler()
	}, ready)
	// A switch or a setup under way ends first, so the project closed here is
	// the one that is current when they settled.
	h.mu.Lock()
	h.closing = true
	h.mu.Unlock()
	h.wg.Wait()
	h.mu.Lock()
	cur := h.cur
	h.cur = nil
	h.mu.Unlock()
	if cur != nil {
		err = errors.Join(err, cur.close())
	}
	return err
}

func (h *Host) admit(host string) bool { return admitLoopback(h.addr, host) }

// Handler is the cockpit's HTTP interface.
func (h *Host) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", h.handleList)
	mux.HandleFunc("POST /api/projects", h.handleAdd)
	mux.HandleFunc("POST /api/projects/keep", h.handleKeep)
	mux.HandleFunc("POST /api/projects/{name}/open", h.handleOpen)
	mux.HandleFunc("POST /api/projects/{name}/remove", h.handleRemove)
	mux.HandleFunc("POST /api/projects/{name}/setup", h.handleSetup)
	files := http.FileServerFS(h.web)
	// Every other API route belongs to the current project; the rest is the page.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"):
			h.delegate(w, r)
		case r.Method == http.MethodGet || r.Method == http.MethodHead:
			files.ServeHTTP(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return guardHandler(mux, h.admit)
}

// delegate gives the request to the current project's Server. A write is
// admitted only while the project is not draining for a switch, and a run or a
// model session starts only while it is not being set up; an admitted write is
// counted until it ends, which a switch waits for.
func (h *Host) delegate(w http.ResponseWriter, r *http.Request) {
	write := r.Method != http.MethodGet && r.Method != http.MethodHead
	h.mu.Lock()
	cur := h.cur
	switch {
	case cur == nil:
		h.mu.Unlock()
		http.Error(w, "No project is open. Open one from the project list.", http.StatusConflict)
		return
	case write && cur.draining:
		h.mu.Unlock()
		http.Error(w, "The cockpit is switching projects; try again in a moment.", http.StatusConflict)
		return
	case write && cur.settingUp && startsWork(r):
		h.mu.Unlock()
		http.Error(w, "The project is being set up; start after the setup ended.", http.StatusConflict)
		return
	}
	if write {
		cur.inflight++
	}
	h.mu.Unlock()
	if write {
		defer func() {
			h.mu.Lock()
			cur.inflight--
			h.idle.Broadcast()
			h.mu.Unlock()
		}()
	}
	cur.handler.ServeHTTP(w, r)
}

// startsWork is a request that starts a run or a model session or a turn of one.
func startsWork(r *http.Request) bool {
	p := r.URL.Path
	return p == "/api/run-control/start" || p == "/api/sessions" || strings.HasPrefix(p, "/api/sessions/") && strings.HasSuffix(p, "/prompt")
}

// busyReasonLocked says why the project must not be left: a setup rewrites its
// runtime files, or what its Server reports. The host's lock is held.
func busyReasonLocked(p *hostProject) string {
	if p.settingUp {
		return "a setup is writing its runtime files; wait for it"
	}
	return p.opened.Server.Busy()
}

// ---------- the list ----------

// ProjectItem is a project of the list as the page shows it.
type ProjectItem struct {
	ProjectEntry
	SetUp   bool `json:"setUp"`
	Current bool `json:"current"`
}

// CurrentProject is the project that is open. Saved says the list holds it.
type CurrentProject struct {
	ProjectEntry
	SetUp bool `json:"setUp"`
	Saved bool `json:"saved"`
}

// ProjectsView is the answer of the project routes.
type ProjectsView struct {
	Available   bool            `json:"available"`
	Reason      string          `json:"reason,omitempty"`
	Error       string          `json:"error,omitempty"`
	Path        string          `json:"path,omitempty"`
	Projects    []ProjectItem   `json:"projects"`
	Current     *CurrentProject `json:"current"`
	Last        string          `json:"last,omitempty"`
	OpenError   string          `json:"openError,omitempty"`
	Busy        string          `json:"busy,omitempty"`
	RuntimeBase string          `json:"runtimeBase,omitempty"`
	Setup       SetupDefaults   `json:"setup"`
	Warning     string          `json:"warning,omitempty"`
	// SetupResult and Opened answer a setup.
	SetupResult *SetupResult `json:"setupResult,omitempty"`
	Opened      *bool        `json:"opened,omitempty"`
}

func (h *Host) view() ProjectsView {
	v := ProjectsView{Projects: []ProjectItem{}, RuntimeBase: h.cfg.RuntimeBase, Setup: h.cfg.Defaults}
	h.mu.Lock()
	cur := h.cur
	v.OpenError = h.openError
	if cur != nil {
		v.Busy = busyReasonLocked(cur)
	}
	h.mu.Unlock()
	var list ProjectList
	if h.cfg.Store == nil {
		v.Reason = h.cfg.Reason
		if v.Reason == "" {
			v.Reason = "The cockpit has no project list."
		}
	} else {
		v.Available, v.Path = true, h.cfg.Store.Path()
		var err error
		if list, err = h.cfg.Store.Load(); err != nil {
			v.Error = err.Error()
		}
	}
	v.Last = list.Last
	for _, p := range list.Projects {
		v.Projects = append(v.Projects, ProjectItem{ProjectEntry: p, SetUp: p.Runtime != "", Current: cur != nil && cur.entry.Name == p.Name && cur.entry.Repository == p.Repository})
	}
	if cur != nil {
		saved, ok := list.Find(cur.entry.Name)
		v.Current = &CurrentProject{ProjectEntry: normalize(cur.entry), SetUp: cur.entry.Runtime != "", Saved: ok && saved.Repository == cur.entry.Repository}
	}
	return v
}

func normalize(p ProjectEntry) ProjectEntry {
	if p.Model == "" {
		p.Model = DefaultModel
	}
	if p.Dependencies == nil {
		p.Dependencies = []string{}
	}
	return p
}

func (h *Host) handleList(w http.ResponseWriter, r *http.Request) { writeJSON(w, h.view()) }

func (h *Host) own(w http.ResponseWriter, r *http.Request, purpose string) bool {
	return checkOwnRequest(w, r, purpose, h.admit)
}

func (h *Host) body(w http.ResponseWriter, r *http.Request, into any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "a project request is JSON", http.StatusForbidden)
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil || dec.More() {
		http.Error(w, "the request is not the JSON object this route reads", http.StatusBadRequest)
		return false
	}
	return true
}

func (h *Host) storeOrRefuse(w http.ResponseWriter) bool {
	if h.cfg.Store == nil {
		reason := h.cfg.Reason
		if reason == "" {
			reason = "The cockpit has no project list."
		}
		http.Error(w, reason, http.StatusNotImplemented)
		return false
	}
	return true
}

// ---------- add, remove, keep ----------

type addBody struct {
	Name         string   `json:"name"`
	Repository   string   `json:"repository"`
	Model        string   `json:"model"`
	Dependencies []string `json:"dependencies"`
	Runtime      string   `json:"runtime"`
}

func (h *Host) handleAdd(w http.ResponseWriter, r *http.Request) {
	if !h.storeOrRefuse(w) || !h.own(w, r, "project-add") {
		return
	}
	var b addBody
	if !h.body(w, r, &b) {
		return
	}
	entry := ProjectEntry{Name: strings.TrimSpace(b.Name), Repository: b.Repository, Model: b.Model, Dependencies: b.Dependencies, Runtime: b.Runtime}
	if filepath.IsAbs(entry.Repository) {
		entry.Repository = filepath.Clean(entry.Repository)
	}
	if entry.Runtime != "" && filepath.IsAbs(entry.Runtime) {
		entry.Runtime = filepath.Clean(entry.Runtime)
	}
	if entry.Name == "" && entry.Repository != "" {
		entry.Name = filepath.Base(entry.Repository)
	}
	if entry.Model == "" {
		entry.Model = DefaultModel
	}
	if entry.Dependencies == nil {
		entry.Dependencies = []string{}
	}
	if err := ValidateEntry(entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.add(entry); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, h.view())
}

func (h *Host) add(entry ProjectEntry) error {
	_, err := h.cfg.Store.Update(func(l *ProjectList) error {
		if _, ok := l.Find(entry.Name); ok {
			return refuseWith(http.StatusConflict, "a project named %q is already in the list; choose another name", entry.Name)
		}
		l.Projects = append(l.Projects, entry)
		return nil
	})
	return err
}

func (h *Host) handleRemove(w http.ResponseWriter, r *http.Request) {
	if !h.storeOrRefuse(w) || !h.own(w, r, "project-remove") {
		return
	}
	name := r.PathValue("name")
	h.mu.Lock()
	reason := ""
	if h.cur != nil && h.cur.entry.Name == name {
		reason = busyReasonLocked(h.cur)
	}
	h.mu.Unlock()
	if reason != "" {
		http.Error(w, "The project is open and busy: "+reason, http.StatusConflict)
		return
	}
	// Only the entry goes; the repository and the runtime directory stay.
	_, err := h.cfg.Store.Update(func(l *ProjectList) error {
		for i, p := range l.Projects {
			if p.Name == name {
				l.Projects = append(l.Projects[:i:i], l.Projects[i+1:]...)
				if l.Last == name {
					l.Last = ""
				}
				return nil
			}
		}
		return refuseWith(http.StatusNotFound, "no project %q in the list", name)
	})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, h.view())
}

func (h *Host) handleKeep(w http.ResponseWriter, r *http.Request) {
	if !h.storeOrRefuse(w) || !h.own(w, r, "project-keep") {
		return
	}
	var b struct {
		Name string `json:"name"`
	}
	if !h.body(w, r, &b) {
		return
	}
	h.mu.Lock()
	cur := h.cur
	h.mu.Unlock()
	if cur == nil {
		http.Error(w, "No project is open, so there is nothing to keep.", http.StatusConflict)
		return
	}
	entry := normalize(cur.entry)
	if entry.Repository == "" {
		http.Error(w, "This cockpit observes state directories only; a project needs a repository to be kept.", http.StatusConflict)
		return
	}
	if name := strings.TrimSpace(b.Name); name != "" {
		entry.Name = name
	}
	if err := ValidateEntry(entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err := h.cfg.Store.Update(func(l *ProjectList) error {
		if existing, ok := l.Find(entry.Name); ok {
			if existing.Repository == entry.Repository && cur.entry.Name == entry.Name {
				return refuseWith(http.StatusConflict, "the project %q is already in the list", entry.Name)
			}
			return refuseWith(http.StatusConflict, "a project named %q is already in the list; choose another name", entry.Name)
		}
		l.Projects = append(l.Projects, entry)
		l.Last = entry.Name
		return nil
	})
	if err != nil {
		fail(w, err)
		return
	}
	h.mu.Lock()
	if h.cur == cur {
		cur.entry.Name = entry.Name
	}
	h.mu.Unlock()
	writeJSON(w, h.view())
}

// ---------- open ----------

func (h *Host) handleOpen(w http.ResponseWriter, r *http.Request) {
	if !h.storeOrRefuse(w) || !h.own(w, r, "project-open") {
		return
	}
	list, err := h.cfg.Store.Load()
	if err != nil {
		fail(w, err)
		return
	}
	entry, ok := list.Find(r.PathValue("name"))
	if !ok {
		http.Error(w, "no project "+r.PathValue("name")+" in the list", http.StatusNotFound)
		return
	}
	warning, err := h.swap(entry)
	if err != nil {
		fail(w, err)
		return
	}
	v := h.view()
	v.Warning = warning
	if _, err := h.cfg.Store.Update(func(l *ProjectList) error { l.Last = entry.Name; return nil }); err != nil {
		v.Warning = strings.TrimSpace(v.Warning + " The project is open, but the list could not remember it: " + err.Error())
	}
	writeJSON(w, v)
}

// swap opens the project and makes it the current one, refusing while the
// current project is busy. The new project is built first, so a project that
// cannot be built leaves the current one as it is; the old one is closed
// after the swap, as a stopping cockpit closes it; what closing reported
// comes back as the warning.
func (h *Host) swap(entry ProjectEntry) (warning string, err error) {
	if h.cfg.Open == nil {
		return "", refuseWith(http.StatusNotImplemented, "this cockpit cannot open projects")
	}
	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		return "", refuseWith(http.StatusServiceUnavailable, "the cockpit is ending")
	}
	if h.switching {
		h.mu.Unlock()
		return "", refuseWith(http.StatusConflict, "another project is being opened; wait for it")
	}
	if err := h.busyLocked(); err != nil {
		h.mu.Unlock()
		return "", err
	}
	h.switching = true
	h.wg.Add(1) // admitted under the lock Serve takes before it waits
	h.mu.Unlock()
	defer h.wg.Done()
	defer func() {
		h.mu.Lock()
		h.switching = false
		h.mu.Unlock()
	}()
	opened, err := h.cfg.Open(h.ctx, normalize(entry))
	if err != nil {
		return "", refuseWith(http.StatusUnprocessableEntity, "%v", err)
	}
	h.mu.Lock()
	old := h.cur
	if old != nil {
		// No write is admitted from here on. The busy state is read, the
		// writes admitted before are waited for, and it is read again: a
		// write that started work meanwhile shows in it, and the project
		// stays open and takes writes again.
		old.draining = true
		undo := func(err error) (string, error) {
			old.draining = false
			h.mu.Unlock()
			if opened.Close != nil {
				_ = opened.Close()
			}
			return "", err
		}
		if err := h.busyLocked(); err != nil {
			return undo(err)
		}
		for old.inflight > 0 {
			h.idle.Wait()
		}
		if err := h.busyLocked(); err != nil {
			return undo(err)
		}
	}
	h.cur = h.newProject(normalize(entry), opened)
	h.openError = ""
	h.mu.Unlock()
	if old != nil {
		if err := old.close(); err != nil {
			return "The previous project reported while closing: " + err.Error(), nil
		}
	}
	return "", nil
}

// busyLocked refuses while the current project is busy. The lock is held.
func (h *Host) busyLocked() error {
	if h.cur == nil {
		return nil
	}
	if reason := busyReasonLocked(h.cur); reason != "" {
		return refuseWith(http.StatusConflict, "The project %s is busy and stays open: %s", h.cur.entry.Name, reason)
	}
	return nil
}

// ---------- setup ----------

type setupBody struct {
	RuntimeDir string `json:"runtimeDir"`
	Image      string `json:"image"`
	Credential string `json:"credential"`
	AgentModel string `json:"agentModel"`
	Effort     string `json:"effort"`
	Engine     string `json:"engine"`
	Checks     []struct {
		Profile    string `json:"profile"`
		Executable string `json:"executable"`
	} `json:"checks"`
	Store    string `json:"store"`
	Deadline string `json:"deadline"`
}

// setupTimeout bounds one setup, which observes the engine and the image.
const setupTimeout = 10 * time.Minute

func (h *Host) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !h.storeOrRefuse(w) || !h.own(w, r, "project-setup") {
		return
	}
	if h.cfg.Setup == nil {
		http.Error(w, "this cockpit cannot set projects up", http.StatusNotImplemented)
		return
	}
	var b setupBody
	if !h.body(w, r, &b) {
		return
	}
	list, err := h.cfg.Store.Load()
	if err != nil {
		fail(w, err)
		return
	}
	entry, ok := list.Find(r.PathValue("name"))
	if !ok {
		http.Error(w, "no project "+r.PathValue("name")+" in the list", http.StatusNotFound)
		return
	}
	in, err := h.setupInputs(entry, b)
	if err != nil {
		fail(w, err)
		return
	}
	// A project that is open and busy keeps its runtime files as they are.
	// While the setup of the open project runs it is busy itself: no run and
	// no session starts, and no switch happens. Writes admitted before are
	// waited for and the busy state is read again, so a start that slipped in
	// is seen.
	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		http.Error(w, "the cockpit is ending", http.StatusServiceUnavailable)
		return
	}
	if h.setting {
		h.mu.Unlock()
		http.Error(w, "another setup is under way; wait for it", http.StatusConflict)
		return
	}
	var mine *hostProject
	if h.cur != nil && h.cur.entry.Name == entry.Name {
		mine = h.cur
		if reason := busyReasonLocked(mine); reason != "" {
			h.mu.Unlock()
			http.Error(w, "The project is open and busy: "+reason, http.StatusConflict)
			return
		}
	}
	h.setting = true
	h.wg.Add(1) // admitted under the lock Serve takes before it waits
	defer func() {
		h.mu.Lock()
		h.setting = false
		if mine != nil {
			mine.settingUp = false
		}
		h.mu.Unlock()
		h.wg.Done()
	}()
	if mine != nil {
		mine.settingUp = true
		for mine.inflight > 0 {
			h.idle.Wait()
		}
		if reason := mine.opened.Server.Busy(); reason != "" {
			h.mu.Unlock()
			http.Error(w, "The project is open and busy: "+reason, http.StatusConflict)
			return
		}
	}
	h.mu.Unlock()
	// Setup writes two files; it runs to its end even when the page goes away.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), setupTimeout)
	defer cancel()
	result := h.cfg.Setup(ctx, in)
	h.mu.Lock()
	if mine != nil {
		mine.settingUp = false // the files are written; the project may be reopened
	}
	h.mu.Unlock()
	if result.AgentProfiles == nil {
		result.AgentProfiles = []string{}
	}
	if result.CheckProfiles == nil {
		result.CheckProfiles = []string{}
	}
	if result.Files == nil {
		result.Files = []SetupFile{}
	}
	if result.Status != "passed" {
		v := h.view()
		v.SetupResult = &result
		writeJSON(w, v)
		return
	}
	entry.Runtime = in.RuntimeDir
	if _, err := h.cfg.Store.Update(func(l *ProjectList) error {
		for i, p := range l.Projects {
			if p.Name == entry.Name {
				l.Projects[i].Runtime = entry.Runtime
				return nil
			}
		}
		return refuseWith(http.StatusNotFound, "the project %q left the list during the setup", entry.Name)
	}); err != nil {
		fail(w, fmt.Errorf("setup wrote the runtime files, but the list could not remember them: %w", err))
		return
	}
	opened := true
	warning, err := h.swap(entry)
	if err == nil {
		_, _ = h.cfg.Store.Update(func(l *ProjectList) error { l.Last = entry.Name; return nil })
	}
	v := h.view()
	if err != nil {
		opened = false
		v.Warning = "The runtime is set up; the project is not open with it yet: " + err.Error()
	} else {
		v.Warning = warning
	}
	v.SetupResult, v.Opened = &result, &opened
	writeJSON(w, v)
}

// setupInputs checks the form as markitect setup checks its flags.
func (h *Host) setupInputs(entry ProjectEntry, b setupBody) (SetupInputs, error) {
	bad := func(format string, args ...any) error { return refuseWith(http.StatusBadRequest, format, args...) }
	if b.RuntimeDir == "" || b.Image == "" || b.Credential == "" || b.AgentModel == "" {
		return SetupInputs{}, bad("setup requires a runtime directory, a workspace image, a credential file and an Agent model")
	}
	engine, effort := b.Engine, b.Effort
	if engine == "" {
		engine = h.cfg.Defaults.Engine
	}
	if effort == "" {
		effort = h.cfg.Defaults.Effort
	}
	for _, path := range []string{b.RuntimeDir, b.Credential, engine} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return SetupInputs{}, bad("the runtime directory, the credential file and the container engine need explicit absolute native paths")
		}
	}
	checks := map[string]string{}
	for _, c := range b.Checks {
		if c.Profile == "" && c.Executable == "" {
			continue // an empty row
		}
		if _, duplicate := checks[c.Profile]; c.Profile == "" || c.Executable == "" || duplicate {
			return SetupInputs{}, bad("a check needs one profile and one executable, and each profile once")
		}
		checks[c.Profile] = c.Executable
	}
	return SetupInputs{
		Project: normalize(entry), RuntimeDir: b.RuntimeDir, Image: b.Image, Credential: b.Credential, AgentModel: b.AgentModel,
		Effort: effort, Engine: engine, Checks: checks, Store: b.Store, Deadline: b.Deadline,
	}, nil
}
