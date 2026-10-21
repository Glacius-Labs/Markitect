package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// projectHostFixture is the cockpit's host over a repository and a fake
// engine, with the per-user project list in a temporary configuration
// directory. Nothing starts a provider or a container.
type projectHostFixture struct {
	setup   setupFixture
	config  string
	service cockpitService
	host    http.Handler
	h       *cockpit.Host
}

// start composes the host for a product build; the list file persists.
func (f *projectHostFixture) start(t *testing.T, build string) {
	t.Helper()
	var err error
	if f.h, err = cockpit.NewHost(context.Background(), f.service.hostConfig(build)); err != nil {
		t.Fatal(err)
	}
	f.host = f.h.Handler()
}

func newProjectHostFixture(t *testing.T) *projectHostFixture {
	t.Helper()
	f := &projectHostFixture{setup: newSetupFixture(t), config: t.TempDir()}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	preparation := apply.NewPreparationService(apply.NewModelService(repository, repository))
	f.service = cockpitService{
		preparation: preparation, repository: repository,
		execute: func(context.Context, cli.ApplyRequest) (cli.ApplyReport, error) {
			return cli.ApplyReport{Status: model.NotCompleted}, errors.New("no run in this test")
		},
		userConfigDir: func() (string, error) { return f.config, nil },
		setup:         newSetup(preparation, f.setup.fake.engine).Setup,
	}
	f.start(t, "test-build")
	return f
}

func (f *projectHostFixture) call(t *testing.T, method, target, purpose, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if purpose != "" {
		req.Header.Set("X-Markitect-Cockpit", purpose)
	}
	rec := httptest.NewRecorder()
	f.host.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func (f *projectHostFixture) view(t *testing.T) cockpit.ProjectsView {
	t.Helper()
	code, body := f.call(t, http.MethodGet, "/api/projects", "", "")
	var v cockpit.ProjectsView
	if code != http.StatusOK || json.Unmarshal([]byte(body), &v) != nil {
		t.Fatalf("list = %d %s", code, body)
	}
	return v
}

func (f *projectHostFixture) runControl(t *testing.T) (available bool, reason string) {
	t.Helper()
	code, body := f.call(t, http.MethodGet, "/api/run-control", "", "")
	var rc struct {
		Available bool
		Reason    string
	}
	if code != http.StatusOK || json.Unmarshal([]byte(body), &rc) != nil {
		t.Fatalf("run-control = %d %s", code, body)
	}
	return rc.Available, rc.Reason
}

func (f *projectHostFixture) form(t *testing.T, extra map[string]any) string {
	t.Helper()
	m := map[string]any{
		"runtimeDir": f.setup.runtimeDir, "image": "markitect-runtime:setup-test", "credential": f.setup.token, "engine": f.setup.engine,
		"agentModel": "claude-haiku-5-5", "effort": "high", "checks": []map[string]string{{"profile": "go", "executable": "/usr/local/go/bin/go"}},
	}
	for k, v := range extra {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCockpitHostKeepsTheProjectListUnderTheUserConfigurationDirectory(t *testing.T) {
	c := cockpitService{userConfigDir: func() (string, error) { return filepath.Join("base", "config"), nil }}
	cfg := c.hostConfig("b")
	if cfg.Store == nil || cfg.Store.Path() != filepath.Join("base", "config", "markitect", "cockpit", "projects.json") || cfg.RuntimeBase != filepath.Join("base", "config", "markitect", "runtimes") {
		t.Fatalf("store %v, runtimes %q", cfg.Store, cfg.RuntimeBase)
	}
	if cfg.Defaults.Engine != cli.DefaultSetupEngine || cfg.Defaults.Effort != cli.DefaultSetupEffort {
		t.Fatalf("defaults = %+v", cfg.Defaults)
	}
	c.userConfigDir = func() (string, error) { return "", errors.New("no home") }
	if cfg := c.hostConfig("b"); cfg.Store != nil || !strings.Contains(cfg.Reason, "no home") {
		t.Fatalf("without a configuration directory = %+v", cfg)
	}
}

func TestCockpitProjectWithoutRuntimeIsObservedButStartsNoRun(t *testing.T) {
	f := newProjectHostFixture(t)
	body, _ := json.Marshal(map[string]any{"name": "library", "repository": f.setup.repo.Dir})
	if code, out := f.call(t, http.MethodPost, "/api/projects", "project-add", string(body)); code != http.StatusOK {
		t.Fatalf("add = %d %s", code, out)
	}
	if code, out := f.call(t, http.MethodPost, "/api/projects/library/open", "project-open", ""); code != http.StatusOK {
		t.Fatalf("open = %d %s", code, out)
	}
	v := f.view(t)
	if v.Current == nil || v.Current.Name != "library" || v.Current.SetUp || v.Last != "library" {
		t.Fatalf("view = %+v", v)
	}
	if available, reason := f.runControl(t); available || reason != projectsUnsetReason {
		t.Fatalf("run control = %v %q", available, reason)
	}
	var sessions struct{ Available bool }
	code, out := f.call(t, http.MethodGet, "/api/sessions", "", "")
	if json.Unmarshal([]byte(out), &sessions); code != http.StatusOK || sessions.Available {
		t.Fatalf("sessions = %d %s", code, out)
	}
	if code, out := f.call(t, http.MethodGet, "/api/settings", "", ""); code != http.StatusOK {
		t.Fatalf("settings of the observed project = %d %s", code, out)
	}
}

func TestCockpitGuidedSetupWritesWhatMarkitectSetupWritesAndEnablesRuns(t *testing.T) {
	f := newProjectHostFixture(t)
	// markitect setup writes the files first.
	if code, out, errout := f.setup.run(t); code != 0 {
		t.Fatalf("setup exit %d\n%s\n%s", code, out, errout)
	}
	before := readTree(t, f.setup.runtimeDir)
	// The cockpit binds the product build it was started with; markitect setup
	// bound this build's identity into the run configuration.
	written, err := loadRunConfig(context.Background(), filepath.Join(f.setup.runtimeDir, "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.start(t, written.ProductBuild)
	body, _ := json.Marshal(map[string]any{"name": "library", "repository": f.setup.repo.Dir})
	f.call(t, http.MethodPost, "/api/projects", "project-add", string(body))
	f.call(t, http.MethodPost, "/api/projects/library/open", "project-open", "")

	code, out := f.call(t, http.MethodPost, "/api/projects/library/setup", "project-setup", f.form(t, nil))
	var v cockpit.ProjectsView
	if code != http.StatusOK || json.Unmarshal([]byte(out), &v) != nil || v.SetupResult == nil || v.SetupResult.Status != "passed" {
		t.Fatalf("setup = %d %s", code, out)
	}
	// The same inputs write the same bytes: the form calls the same function.
	if len(v.SetupResult.Files) != 2 {
		t.Fatalf("files = %+v", v.SetupResult.Files)
	}
	for _, file := range v.SetupResult.Files {
		if file.Action != "unchanged" {
			t.Errorf("%s was %s, not identical to what markitect setup wrote", file.Path, file.Action)
		}
	}
	after := readTree(t, f.setup.runtimeDir)
	if len(after) != len(before) {
		t.Fatalf("the form wrote other files: %d, was %d", len(after), len(before))
	}
	for path, data := range before {
		if string(after[path]) != string(data) {
			t.Errorf("%s changed", path)
		}
	}
	if v.Opened == nil || !*v.Opened || v.Current == nil || !v.Current.SetUp || v.Current.Runtime != f.setup.runtimeDir {
		t.Fatalf("the project did not reopen with its runtime: %+v", v)
	}
	if available, reason := f.runControl(t); !available {
		t.Fatalf("runs are not enabled after the setup: %q", reason)
	}
	var sessions struct{ Available bool }
	_, sessionsBody := f.call(t, http.MethodGet, "/api/sessions", "", "")
	if json.Unmarshal([]byte(sessionsBody), &sessions); !sessions.Available {
		t.Fatal("model sessions are not available after the setup")
	}
	list, err := os.ReadFile(filepath.Join(f.config, "markitect", "cockpit", "projects.json"))
	if err != nil || !strings.Contains(string(list), "runtime") || !strings.Contains(string(list), "library") {
		t.Fatalf("list = %s %v", list, err)
	}
	if strings.Contains(string(list), setupSentinel) || strings.Contains(out, setupSentinel) {
		t.Fatal("a credential reached the list or an answer")
	}
}

func TestCockpitGuidedSetupFirstSetupAndRefusalsAreSetupsOwn(t *testing.T) {
	f := newProjectHostFixture(t)
	body, _ := json.Marshal(map[string]any{"name": "library", "repository": f.setup.repo.Dir})
	f.call(t, http.MethodPost, "/api/projects", "project-add", string(body))

	// A check profile the project selects is missing: setup refuses in its own words.
	code, out := f.call(t, http.MethodPost, "/api/projects/library/setup", "project-setup", f.form(t, map[string]any{"checks": []map[string]string{}}))
	var v cockpit.ProjectsView
	if code != http.StatusOK || json.Unmarshal([]byte(out), &v) != nil || v.SetupResult == nil {
		t.Fatalf("refusal = %d %s", code, out)
	}
	if v.SetupResult.Status != "failed" || !strings.Contains(v.SetupResult.Error, `the project selects the check profile "go"; pass --check go=EXECUTABLE`) {
		t.Fatalf("result = %+v", v.SetupResult)
	}
	if _, err := os.Stat(f.setup.runtimeDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused setup left %s: %v", f.setup.runtimeDir, err)
	}
	if v.Projects[0].SetUp {
		t.Fatal("a refused setup stored a runtime")
	}

	// An Apply holds the state directory: setup refuses where it refuses.
	if code, out := f.call(t, http.MethodPost, "/api/projects/library/setup", "project-setup", f.form(t, nil)); code != http.StatusOK {
		t.Fatalf("setup = %d %s", code, out)
	}
	lock, err := lockState(filepath.Join(f.setup.runtimeDir, setupStateDir))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	code, out = f.call(t, http.MethodPost, "/api/projects/library/setup", "project-setup", f.form(t, map[string]any{"agentModel": "another-model"}))
	v = cockpit.ProjectsView{}
	if code != http.StatusOK || json.Unmarshal([]byte(out), &v) != nil || v.SetupResult == nil || v.SetupResult.Status == "passed" || !strings.Contains(v.SetupResult.Error, "an Apply holds the state directory") {
		t.Fatalf("setup while an Apply holds the state = %d %s", code, out)
	}
}

func TestCockpitGuidedSetupCallsTheSetupFunctionWithTheProjectAndNoOwnerInputs(t *testing.T) {
	var got cli.SetupRequest
	c := cockpitService{setup: func(_ context.Context, r cli.SetupRequest) (cli.SetupReport, error) {
		got = r
		return cli.SetupReport{Status: model.Failed}, errors.New("refused: for a reason")
	}}
	result := c.guidedSetup(context.Background(), cockpit.SetupInputs{
		Project:    cockpit.ProjectEntry{Name: "p", Repository: filepath.Join(t.TempDir(), "repo"), Model: "models/m.yaml", Dependencies: []string{"deps/d.yaml"}},
		RuntimeDir: "/rt", Image: "img", Credential: "/tok", AgentModel: "m", Effort: "low", Engine: "/eng", Checks: map[string]string{"go": "/go"}, Store: "lib", Deadline: "2026-11-01T18:00:00Z",
	}, "test-build")
	if result.Status != "failed" || result.Error != "refused: for a reason" {
		t.Fatalf("result = %+v", result)
	}
	if got.Selection.Local.ManifestPath.String() != "models/m.yaml" || len(got.Selection.Dependencies) != 1 || got.RuntimeDir != "/rt" || got.Image != "img" || got.Credential != "/tok" ||
		got.AgentModel != "m" || got.Effort != "low" || got.Engine != "/eng" || got.Checks["go"] != "/go" || got.Store != "lib" || got.Deadline != "2026-11-01T18:00:00Z" {
		t.Fatalf("request = %+v", got)
	}
	if got.OwnerRequest != "" || len(got.OwnerDecisions) != 0 || got.ProductBuild != "test-build" {
		t.Fatalf("the form sent owner inputs or no build: %+v", got)
	}
}

func TestDirectProjectNamesTheProjectTheFlagsOpened(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "library")
	selection, err := cli.RepositorySelection(repo, ".markitect/model.yaml", []string{".markitect/dependencies/core/model.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "rt")
	entry := directProject(cli.CockpitRequest{Selection: selection, States: []string{"s"}, RunConfig: filepath.Join(dir, "run.json"), RuntimeProfile: filepath.Join(dir, "runtime.yaml")})
	if entry.Name != "library" || entry.Repository != repo || entry.Model != ".markitect/model.yaml" || len(entry.Dependencies) != 1 || entry.Dependencies[0] != filepath.FromSlash(".markitect/dependencies/core/model.yaml") && entry.Dependencies[0] != ".markitect/dependencies/core/model.yaml" || entry.Runtime != dir {
		t.Fatalf("entry = %+v", entry)
	}
	// Files that are not the pair setup writes give no runtime to keep.
	other := directProject(cli.CockpitRequest{Selection: selection, RunConfig: filepath.Join(dir, "other.json"), RuntimeProfile: filepath.Join(dir, "runtime.yaml")})
	if other.Runtime != "" {
		t.Fatalf("runtime = %q", other.Runtime)
	}
	if directProject(cli.CockpitRequest{Selection: selection}).Runtime != "" {
		t.Fatal("a runtime without a run configuration")
	}
}
