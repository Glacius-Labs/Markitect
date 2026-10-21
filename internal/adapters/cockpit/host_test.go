package cockpit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/authoring"
)

// busyAuthoring is the proposal service of a fake project: its model session
// and its proposal states are steered by the test.
type busyAuthoring struct {
	fakeAuthoring
	mu       sync.Mutex
	session  string
	proposal string
}

func (b *busyAuthoring) set(session, proposal string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.session, b.proposal = session, proposal
}

func (b *busyAuthoring) Sessions() []authoring.Session {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session == "" {
		return nil
	}
	return []authoring.Session{{ID: "model-1", State: b.session}}
}

func (b *busyAuthoring) Proposals() []authoring.Proposal {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.proposal == "" {
		return nil
	}
	return []authoring.Proposal{{ID: "configuration-1", State: b.proposal}}
}

// hostFixture is a host over fake projects: opening one builds a real Server
// over a state directory with one run named after the project.
type hostFixture struct {
	t      *testing.T
	host   *Host
	h      http.Handler
	store  *ProjectStore
	config string
	engine string

	mu      sync.Mutex
	opened  []string
	closed  []string
	fail    map[string]error
	auth    map[string]*busyAuthoring
	setups  []SetupInputs
	setup   SetupResult
	closeCh chan struct{}
}

func (f *hostFixture) record(list *[]string, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	*list = append(*list, name)
}

func (f *hostFixture) names(list *[]string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), (*list)...)
}

func newHostFixture(t *testing.T) *hostFixture {
	t.Helper()
	f := &hostFixture{t: t, fail: map[string]error{}, auth: map[string]*busyAuthoring{}, setup: SetupResult{Status: "passed", Files: []SetupFile{{Path: "x", SHA256: "y", Action: "written"}}}}
	f.config = t.TempDir()
	f.engine = filepath.Join(f.config, "docker")
	f.store = NewProjectStore(filepath.Join(f.config, "markitect", "cockpit", "projects.json"))
	open := func(_ context.Context, entry ProjectEntry) (OpenedProject, error) {
		if err := f.fail[entry.Name]; err != nil {
			return OpenedProject{}, err
		}
		s, _ := library(t)
		state := s.writeRun("20261020T120000Z-"+strings.ToLower(entry.Name), applyReport("success", "applied"))
		auth := &busyAuthoring{}
		f.mu.Lock()
		f.auth[entry.Name] = auth
		f.mu.Unlock()
		srv, err := New(Config{States: []string{state}, Authoring: auth})
		if err != nil {
			return OpenedProject{}, err
		}
		f.record(&f.opened, entry.Name)
		return OpenedProject{Server: srv, Close: func() error { f.record(&f.closed, entry.Name); return nil }}, nil
	}
	host, err := NewHost(context.Background(), HostConfig{
		Store: f.store, Open: open, RuntimeBase: filepath.Join(f.config, "markitect", "runtimes"),
		Defaults: SetupDefaults{Engine: f.engine, Effort: "high"},
		Setup: func(_ context.Context, in SetupInputs) SetupResult {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.setups = append(f.setups, in)
			return f.setup
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	host.addr = local
	f.host, f.h = host, host.Handler()
	return f
}

// repo makes a repository directory with its model file.
func (f *hostFixture) repo(name string) string {
	f.t.Helper()
	dir := filepath.Join(f.t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(dir, ".markitect"), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".markitect", "model.yaml"), []byte("x"), 0o600); err != nil {
		f.t.Fatal(err)
	}
	return dir
}

// runtimeDir makes a runtime directory as setup leaves it.
func (f *hostFixture) runtimeDir() string {
	f.t.Helper()
	dir := f.t.TempDir()
	for _, n := range []string{"run.json", "runtime.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o600); err != nil {
			f.t.Fatal(err)
		}
	}
	return dir
}

func (f *hostFixture) post(header, target, body string) *httptest.ResponseRecorder {
	return post(f.h, local, "", header, target, body)
}

func (f *hostFixture) add(name, repo, runtime string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{"name": name, "repository": repo, "runtime": runtime})
	return f.post("project-add", "/api/projects", string(body))
}

func (f *hostFixture) open(name string) *httptest.ResponseRecorder {
	return f.post("project-open", "/api/projects/"+name+"/open", "")
}

func (f *hostFixture) view() ProjectsView {
	f.t.Helper()
	var v ProjectsView
	if rec := get(f.t, f.h, local, "/api/projects", &v); rec.Code != http.StatusOK {
		f.t.Fatalf("list = %d %s", rec.Code, rec.Body)
	}
	return v
}

func (f *hostFixture) runIDs() []string {
	f.t.Helper()
	var runs []RunSummary
	if rec := get(f.t, f.h, local, "/api/runs", &runs); rec.Code != http.StatusOK {
		f.t.Fatalf("runs = %d %s", rec.Code, rec.Body)
	}
	var ids []string
	for _, r := range runs {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestHostServesThePageAndAnswers409WithoutAProject(t *testing.T) {
	f := newHostFixture(t)
	rec := get(t, f.h, local, "/", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Markitect cockpit") {
		t.Fatalf("page = %d", rec.Code)
	}
	if rec := get(t, f.h, local, "/js/projects.js", nil); rec.Code != http.StatusOK {
		t.Fatalf("the projects module = %d", rec.Code)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "style-src 'self'") {
		t.Fatalf("csp = %q", csp)
	}
	v := f.view()
	if !v.Available || v.Current != nil || len(v.Projects) != 0 || v.Setup.Engine != f.engine || v.RuntimeBase == "" {
		t.Fatalf("empty list = %+v", v)
	}
	for _, target := range []string{"/api/project", "/api/runs", "/api/run-control", "/api/settings", "/api/sessions"} {
		rec := get(t, f.h, local, target, nil)
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "No project is open") {
			t.Errorf("%s without a project = %d %s", target, rec.Code, rec.Body)
		}
	}
	if rec := get(t, f.h, "evil.example:7341", "/api/projects", nil); rec.Code != http.StatusMisdirectedRequest {
		t.Fatalf("a foreign host = %d", rec.Code)
	}
}

func TestHostWritesNeedTheCockpitsOwnRequest(t *testing.T) {
	f := newHostFixture(t)
	repo := f.repo("alpha")
	body, _ := json.Marshal(map[string]any{"repository": repo})
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"no header":      post(f.h, local, "", "", "/api/projects", string(body)),
		"another header": post(f.h, local, "", "project-open", "/api/projects", string(body)),
		"foreign origin": post(f.h, local, "http://evil.example", "project-add", "/api/projects", string(body)),
	} {
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s = %d", name, rec.Code)
		}
	}
	if len(f.view().Projects) != 0 {
		t.Fatal("a refused write reached the list")
	}
	if rec := post(f.h, local, "http://"+local, "project-add", "/api/projects", string(body)); rec.Code != http.StatusOK {
		t.Fatalf("own origin = %d %s", rec.Code, rec.Body)
	}
}

func TestHostAddsProjectsWithAValidatedEntryAndTheDirectoryNameByDefault(t *testing.T) {
	f := newHostFixture(t)
	repo := f.repo("alpha")
	rec := f.add("", repo, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("add = %d %s", rec.Code, rec.Body)
	}
	v := f.view()
	if len(v.Projects) != 1 || v.Projects[0].Name != "alpha" || v.Projects[0].SetUp || v.Projects[0].Model != DefaultModel || v.Projects[0].Repository != repo {
		t.Fatalf("list = %+v", v.Projects)
	}
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"a duplicate name":             f.add("alpha", f.repo("beta"), ""),
		"a relative repository":        f.add("rel", "alpha", ""),
		"a repository without a model": f.add("none", t.TempDir(), ""),
		"a runtime without files":      f.add("rt", f.repo("gamma"), t.TempDir()),
		"a relative runtime":           f.add("rt2", f.repo("delta"), "runtime"),
		"a bad name":                   f.add("../x", f.repo("eps"), ""),
	} {
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusConflict {
			t.Errorf("%s = %d %s", name, rec.Code, rec.Body)
		}
	}
	if rec := f.add("alpha", f.repo("beta"), ""); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate = %d", rec.Code)
	}
	if len(f.view().Projects) != 1 {
		t.Fatal("a refused entry reached the list")
	}
	rt := f.runtimeDir()
	if rec := f.add("setup", f.repo("zeta"), rt); rec.Code != http.StatusOK {
		t.Fatalf("add with runtime = %d %s", rec.Code, rec.Body)
	}
	if v := f.view(); !v.Projects[1].SetUp || v.Projects[1].Runtime != rt {
		t.Fatalf("runtime = %+v", v.Projects[1])
	}
}

func TestHostOpensAProjectDelegatesToItAndRemembersItAsLast(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	if rec := f.open("alpha"); rec.Code != http.StatusOK {
		t.Fatalf("open = %d %s", rec.Code, rec.Body)
	}
	if ids := f.runIDs(); len(ids) != 1 || ids[0] != "20261020T120000Z-alpha" {
		t.Fatalf("runs of alpha = %v", ids)
	}
	v := f.view()
	if v.Current == nil || v.Current.Name != "alpha" || !v.Current.Saved || v.Last != "alpha" || !v.Projects[0].Current || v.Projects[1].Current {
		t.Fatalf("view = %+v", v)
	}
	// A switch closes the old project and delegates to the new one.
	if rec := f.open("beta"); rec.Code != http.StatusOK {
		t.Fatalf("switch = %d %s", rec.Code, rec.Body)
	}
	if ids := f.runIDs(); len(ids) != 1 || ids[0] != "20261020T120000Z-beta" {
		t.Fatalf("runs of beta = %v", ids)
	}
	if got := f.names(&f.closed); len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("closed = %v", got)
	}
	if f.view().Last != "beta" {
		t.Fatal("the last project is not remembered")
	}
	if rec := f.open("nobody"); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown = %d", rec.Code)
	}
}

func TestHostRefusesToSwitchWhileTheCurrentProjectIsBusyAndAllowsItOtherwise(t *testing.T) {
	for _, tt := range []struct {
		name            string
		session, prop   string
		wantBusy, retry bool
	}{
		{"a working session", authoring.Working, "", true, false},
		{"a preparing session", authoring.Preparing, "", true, false},
		{"a closing session", authoring.Closing, "", true, false},
		{"a publishing proposal", "", authoring.Publishing, true, false},
		{"a waiting session", authoring.Waiting, "", false, false},
		{"a halted session", authoring.Halted, "", false, false},
		{"a proposal to decide", "", authoring.Proposed, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newHostFixture(t)
			f.add("alpha", f.repo("alpha"), "")
			f.add("beta", f.repo("beta"), "")
			f.open("alpha")
			f.auth["alpha"].set(tt.session, tt.prop)
			v := f.view()
			if (v.Busy != "") != tt.wantBusy {
				t.Fatalf("busy = %q", v.Busy)
			}
			rec := f.open("beta")
			if tt.wantBusy {
				if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "alpha") {
					t.Fatalf("switch = %d %s", rec.Code, rec.Body)
				}
				if f.view().Current.Name != "alpha" || len(f.names(&f.closed)) != 0 || len(f.names(&f.opened)) != 1 {
					t.Fatalf("a refused switch changed the project: closed %v opened %v", f.names(&f.closed), f.names(&f.opened))
				}
				// It switches once the work ended.
				f.auth["alpha"].set("", "")
				if rec := f.open("beta"); rec.Code != http.StatusOK {
					t.Fatalf("switch after the work = %d %s", rec.Code, rec.Body)
				}
				return
			}
			if rec.Code != http.StatusOK || f.view().Current.Name != "beta" {
				t.Fatalf("switch = %d %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHostRefusesToSwitchWhileACockpitStartedRunIsActive(t *testing.T) {
	f := newHostFixture(t)
	fake := newFakeRun()
	var runs *RunController
	f.host.cfg.Open = func(ctx context.Context, entry ProjectEntry) (OpenedProject, error) {
		state := t.TempDir()
		var cfg Config
		cfg.States = []string{state}
		if entry.Name == "alpha" {
			runs = NewRunController(ctx, RunControlConfig{StateDir: state, Head: func(context.Context) (string, error) { return confirmedHead, nil },
				Execute: fake.execute, Held: func() (string, error) { return "", nil }})
			cfg.Runs = runs
		}
		srv, err := New(cfg)
		return OpenedProject{Server: srv, Close: func() error {
			if runs != nil && entry.Name == "alpha" {
				runs.Wait()
			}
			return nil
		}}, err
	}
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	f.open("alpha")
	if rec := post(f.h, local, "", "start-run", "/api/run-control/start", startBody); rec.Code != http.StatusOK {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	<-fake.started
	if rec := f.open("beta"); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "run") {
		t.Fatalf("switch during a run = %d %s", rec.Code, rec.Body)
	}
	if rec := f.post("project-remove", "/api/projects/alpha/remove", ""); rec.Code != http.StatusConflict {
		t.Fatalf("remove during a run = %d", rec.Code)
	}
	close(fake.release)
	waitFor(t, "the run to end", func() bool { return runs.Status().State == RunEnded })
	if rec := f.open("beta"); rec.Code != http.StatusOK {
		t.Fatalf("switch after the run = %d %s", rec.Code, rec.Body)
	}
}

func TestHostKeepsTheCurrentProjectWhenTheNewOneCannotBeOpened(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	f.open("alpha")
	f.fail["beta"] = os.ErrNotExist
	rec := f.open("beta")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("open = %d %s", rec.Code, rec.Body)
	}
	if f.view().Current.Name != "alpha" || len(f.names(&f.closed)) != 0 {
		t.Fatal("a project that cannot open replaced the current one")
	}
	if ids := f.runIDs(); len(ids) != 1 || !strings.HasSuffix(ids[0], "alpha") {
		t.Fatalf("runs = %v", ids)
	}
}

func TestHostRemovesOnlyTheEntry(t *testing.T) {
	f := newHostFixture(t)
	repo, rt := f.repo("alpha"), f.runtimeDir()
	f.add("alpha", repo, rt)
	f.add("beta", f.repo("beta"), "")
	f.open("alpha")
	if rec := f.post("project-remove", "/api/projects/nobody/remove", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown = %d", rec.Code)
	}
	if rec := f.post("project-remove", "/api/projects/alpha/remove", ""); rec.Code != http.StatusOK {
		t.Fatalf("remove = %d %s", rec.Code, rec.Body)
	}
	v := f.view()
	if len(v.Projects) != 1 || v.Last != "" {
		t.Fatalf("list = %+v", v)
	}
	// The open project stays open and is no longer in the list.
	if v.Current == nil || v.Current.Name != "alpha" || v.Current.Saved {
		t.Fatalf("current = %+v", v.Current)
	}
	for _, p := range []string{repo, rt, filepath.Join(rt, "run.json")} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("remove touched %s: %v", p, err)
		}
	}
}

func TestHostKeepsAProjectOpenedByFlags(t *testing.T) {
	f := newHostFixture(t)
	repo := f.repo("flagged")
	opened, err := f.host.cfg.Open(context.Background(), ProjectEntry{Name: "flagged"})
	if err != nil {
		t.Fatal(err)
	}
	f.host.Adopt(ProjectEntry{Name: "flagged", Repository: repo, Model: DefaultModel, Dependencies: []string{}}, opened)
	v := f.view()
	if v.Current == nil || v.Current.Saved || len(v.Projects) != 0 {
		t.Fatalf("view = %+v", v)
	}
	if rec := f.post("project-keep", "/api/projects/keep", `{}`); rec.Code != http.StatusOK {
		t.Fatalf("keep = %d %s", rec.Code, rec.Body)
	}
	v = f.view()
	if !v.Current.Saved || len(v.Projects) != 1 || v.Projects[0].Name != "flagged" || v.Last != "flagged" || !v.Projects[0].Current {
		t.Fatalf("kept = %+v", v)
	}
	if rec := f.post("project-keep", "/api/projects/keep", `{}`); rec.Code != http.StatusConflict {
		t.Fatalf("keeping twice = %d", rec.Code)
	}
	// Another name for a name that is taken.
	f.host.Adopt(ProjectEntry{Name: "flagged", Repository: f.repo("other"), Model: DefaultModel}, opened)
	if rec := f.post("project-keep", "/api/projects/keep", `{}`); rec.Code != http.StatusConflict {
		t.Fatalf("keeping a taken name = %d %s", rec.Code, rec.Body)
	}
	if rec := f.post("project-keep", "/api/projects/keep", `{"name":"second"}`); rec.Code != http.StatusOK {
		t.Fatalf("keeping under another name = %d %s", rec.Code, rec.Body)
	}
	if v := f.view(); len(v.Projects) != 2 || v.Current.Name != "second" || !v.Current.Saved {
		t.Fatalf("kept twice = %+v", v)
	}
}

func TestHostSetupCallsTheSetupFunctionAndStoresTheRuntime(t *testing.T) {
	f := newHostFixture(t)
	repo := f.repo("alpha")
	f.add("alpha", repo, "")
	base := t.TempDir()
	rt, cred := filepath.Join(base, "rt"), filepath.Join(base, "token")
	form := func(extra string) string {
		body, _ := json.Marshal(map[string]any{"runtimeDir": rt, "image": "ghcr.io/x/workspace:1", "credential": cred, "agentModel": "model-x",
			"effort": "", "engine": "", "checks": []map[string]string{{"profile": "go", "executable": "/usr/bin/go"}, {"profile": "", "executable": ""}}, "store": "lib", "deadline": "2026-10-30T10:00:00Z"})
		_ = extra
		return string(body)
	}
	// The setup writes the files the runtime check needs, as the real one does.
	f.host.cfg.Setup = func(_ context.Context, in SetupInputs) SetupResult {
		f.mu.Lock()
		f.setups = append(f.setups, in)
		f.mu.Unlock()
		if err := os.MkdirAll(in.RuntimeDir, 0o700); err != nil {
			t.Error(err)
		}
		for _, n := range []string{"run.json", "runtime.yaml"} {
			_ = os.WriteFile(filepath.Join(in.RuntimeDir, n), []byte("x"), 0o600)
		}
		return f.setup
	}
	rec := f.post("project-setup", "/api/projects/alpha/setup", form(""))
	if rec.Code != http.StatusOK {
		t.Fatalf("setup = %d %s", rec.Code, rec.Body)
	}
	if len(f.setups) != 1 {
		t.Fatalf("setups = %d", len(f.setups))
	}
	in := f.setups[0]
	if in.Project.Repository != repo || in.Project.Model != DefaultModel || in.RuntimeDir != rt || in.Image != "ghcr.io/x/workspace:1" || in.Credential != cred ||
		in.AgentModel != "model-x" || in.Effort != "high" || in.Engine != f.engine || in.Checks["go"] != "/usr/bin/go" || len(in.Checks) != 1 || in.Store != "lib" || in.Deadline != "2026-10-30T10:00:00Z" {
		t.Fatalf("inputs = %+v", in)
	}
	var v ProjectsView
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.SetupResult == nil || v.SetupResult.Status != "passed" || len(v.SetupResult.Files) != 1 || v.Opened == nil || !*v.Opened {
		t.Fatalf("result = %+v", v)
	}
	if v.Current == nil || v.Current.Name != "alpha" || !v.Current.SetUp || v.Current.Runtime != rt || v.Last != "alpha" {
		t.Fatalf("the project did not reopen with its runtime: %+v", v.Current)
	}
	list, _ := f.store.Load()
	if p, _ := list.Find("alpha"); p.Runtime != rt {
		t.Fatalf("stored = %+v", p)
	}
}

func TestHostSetupShowsRefusalsVerbatimAndStoresNothing(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	base := t.TempDir()
	body, _ := json.Marshal(map[string]any{"runtimeDir": filepath.Join(base, "rt"), "image": "img", "credential": filepath.Join(base, "token"), "agentModel": "m"})
	f.setup = SetupResult{Status: "failed", Error: "an Apply holds the state directory of the Runtime directory; wait for it to finish"}
	rec := f.post("project-setup", "/api/projects/alpha/setup", string(body))
	var v ProjectsView
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &v) != nil || v.SetupResult == nil || v.SetupResult.Status != "failed" || v.SetupResult.Error != f.setup.Error {
		t.Fatalf("refusal = %d %s", rec.Code, rec.Body)
	}
	if v.Opened != nil || v.Current != nil || v.Projects[0].SetUp {
		t.Fatalf("a refused setup changed the project: %+v", v)
	}
	list, _ := f.store.Load()
	if p, _ := list.Find("alpha"); p.Runtime != "" {
		t.Fatalf("stored = %+v", p)
	}
}

func TestHostSetupChecksTheFormAsMarkitectSetupChecksItsFlags(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	abs := t.TempDir()
	form := func(mutate func(map[string]any)) string {
		m := map[string]any{"runtimeDir": filepath.Join(abs, "rt"), "image": "img", "credential": filepath.Join(abs, "token"), "agentModel": "m"}
		mutate(m)
		b, _ := json.Marshal(m)
		return string(b)
	}
	for name, body := range map[string]string{
		"no image":              form(func(m map[string]any) { m["image"] = "" }),
		"no model":              form(func(m map[string]any) { m["agentModel"] = "" }),
		"a relative runtime":    form(func(m map[string]any) { m["runtimeDir"] = "rt" }),
		"a relative credential": form(func(m map[string]any) { m["credential"] = "token" }),
		"a relative engine":     form(func(m map[string]any) { m["engine"] = "docker" }),
		"a half check":          form(func(m map[string]any) { m["checks"] = []map[string]string{{"profile": "go"}} }),
		"a duplicate check": form(func(m map[string]any) {
			m["checks"] = []map[string]string{{"profile": "go", "executable": "/a"}, {"profile": "go", "executable": "/b"}}
		}),
		"an unknown field": form(func(m map[string]any) { m["ownerRequest"] = "do it" }),
	} {
		if rec := f.post("project-setup", "/api/projects/alpha/setup", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s = %d %s", name, rec.Code, rec.Body)
		}
	}
	if rec := f.post("project-setup", "/api/projects/nobody/setup", form(func(map[string]any) {})); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown project = %d", rec.Code)
	}
	if len(f.setups) != 0 {
		t.Fatal("a refused form reached setup")
	}
}

func TestHostSetupIsRefusedWhileTheOpenProjectIsBusy(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.open("alpha")
	f.auth["alpha"].set(authoring.Working, "")
	abs := t.TempDir()
	body, _ := json.Marshal(map[string]any{"runtimeDir": filepath.Join(abs, "rt"), "image": "img", "credential": filepath.Join(abs, "token"), "agentModel": "m"})
	if rec := f.post("project-setup", "/api/projects/alpha/setup", string(body)); rec.Code != http.StatusConflict {
		t.Fatalf("setup = %d %s", rec.Code, rec.Body)
	}
	if len(f.setups) != 0 {
		t.Fatal("setup ran for a busy project")
	}
}

func TestHostOpensTheLastProjectAndReportsWhenItCannotBeOpened(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	f.open("beta")
	// A new host over the same list opens the last project.
	f2, err := NewHost(context.Background(), f.host.cfg)
	if err != nil {
		t.Fatal(err)
	}
	f2.OpenLast()
	var v ProjectsView
	rec := httptest.NewRecorder()
	f2.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil || v.Current == nil || v.Current.Name != "beta" {
		t.Fatalf("last = %+v, %v", v.Current, err)
	}
	// When it cannot be opened, the list shows why.
	f.fail["beta"] = os.ErrNotExist
	f3, _ := NewHost(context.Background(), f.host.cfg)
	f3.OpenLast()
	rec = httptest.NewRecorder()
	f3.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	v = ProjectsView{}
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil || v.Current != nil || !strings.Contains(v.OpenError, "beta") {
		t.Fatalf("failed last = %+v, %v", v, err)
	}
}

func TestHostShowsAnUnusableListAndRefusesToChangeIt(t *testing.T) {
	f := newHostFixture(t)
	path := f.store.Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"formatVersion":9}`), 0o600); err != nil {
		t.Fatal(err)
	}
	v := f.view()
	if !strings.Contains(v.Error, "format version 9") || len(v.Projects) != 0 {
		t.Fatalf("view = %+v", v)
	}
	if rec := f.add("alpha", f.repo("alpha"), ""); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "does not overwrite") {
		t.Fatalf("add = %d %s", rec.Code, rec.Body)
	}
	if got, _ := os.ReadFile(path); string(got) != `{"formatVersion":9}` {
		t.Fatalf("the unusable list was overwritten: %s", got)
	}
}

func TestHostWithoutAListStillServesTheCurrentProject(t *testing.T) {
	host, err := NewHost(context.Background(), HostConfig{Reason: "no config directory"})
	if err != nil {
		t.Fatal(err)
	}
	h := host.Handler()
	var v ProjectsView
	if rec := get(t, h, "", "/api/projects", &v); rec.Code != http.StatusOK || v.Available || v.Reason != "no config directory" {
		t.Fatalf("view = %+v", v)
	}
	if rec := post(h, "", "", "project-add", "/api/projects", `{"repository":"/x"}`); rec.Code != http.StatusNotImplemented {
		t.Fatalf("add = %d", rec.Code)
	}
}

func TestHostServeClosesTheCurrentProjectWhenItEnds(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.open("alpha")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- f.host.Serve(ctx, "127.0.0.1:0", func(url string) { ready <- url }) }()
	url := <-ready
	resp, err := http.Get(url + "api/projects")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("served = %v %v", resp, err)
	}
	resp.Body.Close()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not end")
	}
	if got := f.names(&f.closed); len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("closed = %v", got)
	}
}

func TestServerWithoutStateDirectoriesNeedsARepository(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("a server without state directories and repository")
	}
	srv, err := New(Config{Project: func(context.Context) (ProjectInfo, error) { return ProjectInfo{}, nil }})
	if err != nil {
		t.Fatal(err)
	}
	var runs []RunSummary
	if rec := get(t, srv.Handler(), "", "/api/runs", &runs); rec.Code != http.StatusOK || len(runs) != 0 {
		t.Fatalf("runs = %d %v", rec.Code, runs)
	}
	var rc runControlView
	srvReason, _ := New(Config{Project: func(context.Context) (ProjectInfo, error) { return ProjectInfo{}, nil }, RunsReason: "set it up"})
	if get(t, srvReason.Handler(), "", "/api/run-control", &rc); rc.Available || rc.Reason != "set it up" {
		t.Fatalf("run control = %+v", rc)
	}
}

func TestHostSwitchWaitsForAdmittedWritesAndRefusesNewOnes(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	f.open("alpha")
	entered, release := make(chan struct{}), make(chan struct{})
	f.auth["alpha"].accept = func(id string) (authoring.Proposal, error) {
		close(entered)
		<-release
		return authoring.Proposal{ID: id, State: authoring.Accepted}, nil
	}
	accepted := make(chan *httptest.ResponseRecorder, 1)
	go func() { accepted <- f.post("accept", "/api/proposals/configuration-1/accept", "") }()
	<-entered
	opened := make(chan *httptest.ResponseRecorder, 1)
	go func() { opened <- f.open("beta") }()
	// The switch drains: a new write is refused in its own words, reads go on.
	waitFor(t, "the project to drain", func() bool {
		rec := f.post("discard", "/api/proposals/configuration-1/discard", "")
		return rec.Code == http.StatusConflict && strings.Contains(rec.Body.String(), "switching projects")
	})
	if rec := get(t, f.h, local, "/api/proposals", nil); rec.Code != http.StatusOK {
		t.Fatalf("a read while draining = %d", rec.Code)
	}
	select {
	case rec := <-opened:
		t.Fatalf("the switch ended before the admitted write: %d", rec.Code)
	case <-time.After(150 * time.Millisecond):
	}
	if len(f.names(&f.closed)) != 0 {
		t.Fatal("the project closed under an admitted write")
	}
	close(release)
	if rec := <-accepted; rec.Code != http.StatusOK {
		t.Fatalf("the admitted write = %d %s", rec.Code, rec.Body)
	}
	if rec := <-opened; rec.Code != http.StatusOK {
		t.Fatalf("switch = %d %s", rec.Code, rec.Body)
	}
	if got := f.names(&f.closed); len(got) != 1 || got[0] != "alpha" || f.view().Current.Name != "beta" {
		t.Fatalf("closed %v, current %+v", got, f.view().Current)
	}
}

func TestHostKeepsTheProjectWhenAnAdmittedWriteStartedWork(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	f.open("alpha")
	entered, release := make(chan struct{}), make(chan struct{})
	f.auth["alpha"].accept = func(id string) (authoring.Proposal, error) {
		close(entered)
		<-release
		f.auth["alpha"].set("", authoring.Publishing) // the write started a publication
		return authoring.Proposal{ID: id, State: authoring.Publishing}, nil
	}
	go f.post("accept", "/api/proposals/configuration-1/accept", "")
	<-entered
	opened := make(chan *httptest.ResponseRecorder, 1)
	go func() { opened <- f.open("beta") }()
	waitFor(t, "the project to drain", func() bool {
		return f.post("discard", "/api/proposals/configuration-1/discard", "").Code == http.StatusConflict
	})
	close(release)
	if rec := <-opened; rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "busy") {
		t.Fatalf("switch = %d %s", rec.Code, rec.Body)
	}
	f.auth["alpha"].set("", "")
	// The project that was built for the refused switch is closed unused.
	if f.view().Current.Name != "alpha" || slices.Contains(f.names(&f.closed), "alpha") {
		t.Fatal("a refused switch changed the project")
	}
	if rec := f.post("discard", "/api/proposals/configuration-1/discard", ""); rec.Code != http.StatusOK {
		t.Fatalf("the project takes writes again = %d %s", rec.Code, rec.Body)
	}
}

func TestHostServeWaitsForASwitchUnderWayAndClosesWhatIsCurrentThen(t *testing.T) {
	f := newHostFixture(t)
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	f.open("alpha")
	gate := make(chan struct{})
	build := f.host.cfg.Open
	f.host.cfg.Open = func(ctx context.Context, entry ProjectEntry) (OpenedProject, error) {
		if entry.Name == "beta" {
			<-gate
		}
		return build(ctx, entry)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- f.host.Serve(ctx, "127.0.0.1:0", func(url string) { ready <- url }) }()
	address := strings.TrimSuffix(strings.TrimPrefix(<-ready, "http://"), "/")
	opened := make(chan *httptest.ResponseRecorder, 1)
	go func() { opened <- post(f.h, address, "", "project-open", "/api/projects/beta/open", "") }()
	waitFor(t, "the switch to start", func() bool {
		f.host.mu.Lock()
		defer f.host.mu.Unlock()
		return f.host.switching
	})
	cancel()
	select {
	case <-done:
		t.Fatal("Serve ended while a switch was under way")
	case <-time.After(300 * time.Millisecond):
	}
	close(gate)
	if rec := <-opened; rec.Code != http.StatusOK {
		t.Fatalf("switch = %d %s", rec.Code, rec.Body)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not end")
	}
	got := f.names(&f.closed)
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("closed = %v: the old project closes in the switch, the new one when Serve ends", got)
	}
	// Nothing starts after the end.
	if _, err := f.host.swap(ProjectEntry{Name: "alpha"}); err == nil {
		t.Fatal("a switch started after Serve ended")
	}
}

func TestHostSetupOfTheOpenProjectBlocksStartsAndSwitches(t *testing.T) {
	f := newHostFixture(t)
	fake := newFakeRun()
	f.host.cfg.Open = func(ctx context.Context, entry ProjectEntry) (OpenedProject, error) {
		state := t.TempDir()
		runs := NewRunController(ctx, RunControlConfig{StateDir: state, Head: func(context.Context) (string, error) { return confirmedHead, nil },
			Execute: fake.execute, Held: func() (string, error) { return "", nil }})
		srv, err := New(Config{States: []string{state}, Runs: runs})
		return OpenedProject{Server: srv, Close: func() error { runs.Wait(); return nil }}, err
	}
	f.add("alpha", f.repo("alpha"), "")
	f.add("beta", f.repo("beta"), "")
	f.open("alpha")
	entered, release := make(chan struct{}), make(chan struct{})
	f.host.cfg.Setup = func(context.Context, SetupInputs) SetupResult {
		close(entered)
		<-release
		return SetupResult{Status: "failed", Error: "refused"}
	}
	abs := t.TempDir()
	body, _ := json.Marshal(map[string]any{"runtimeDir": filepath.Join(abs, "rt"), "image": "img", "credential": filepath.Join(abs, "token"), "agentModel": "m", "engine": filepath.Join(abs, "docker")})
	setup := make(chan *httptest.ResponseRecorder, 1)
	go func() { setup <- f.post("project-setup", "/api/projects/alpha/setup", string(body)) }()
	<-entered
	if rec := post(f.h, local, "", "start-run", "/api/run-control/start", startBody); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "being set up") {
		t.Fatalf("a start during the setup = %d %s", rec.Code, rec.Body)
	}
	if rec := f.post("start", "/api/sessions", `{"prompt":"x"}`); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "being set up") {
		t.Fatalf("a session during the setup = %d %s", rec.Code, rec.Body)
	}
	if rec := get(t, f.h, local, "/api/run-control", nil); rec.Code != http.StatusOK {
		t.Fatalf("a read during the setup = %d", rec.Code)
	}
	if v := f.view(); !strings.Contains(v.Busy, "setup") {
		t.Fatalf("busy = %q", v.Busy)
	}
	if rec := f.open("beta"); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "setup") {
		t.Fatalf("switch during the setup = %d %s", rec.Code, rec.Body)
	}
	if rec := f.post("project-remove", "/api/projects/alpha/remove", ""); rec.Code != http.StatusConflict {
		t.Fatalf("remove during the setup = %d", rec.Code)
	}
	close(release)
	if rec := <-setup; rec.Code != http.StatusOK {
		t.Fatalf("setup = %d %s", rec.Code, rec.Body)
	}
	if v := f.view(); v.Busy != "" {
		t.Fatalf("busy after the setup = %q", v.Busy)
	}
	if rec := post(f.h, local, "", "start-run", "/api/run-control/start", startBody); rec.Code != http.StatusOK {
		t.Fatalf("a start after the setup = %d %s", rec.Code, rec.Body)
	}
	<-fake.started
	close(fake.release)
}
