package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const setupSentinel = "SENTINEL-TOKEN-0123456789-do-not-copy"

// setupFake is the controlled engine: it records what setup asks and answers
// with fixed image metadata. Nothing here starts a container.
type setupFake struct {
	paths        []string
	refs         []string
	version      string
	versionCalls int
}

const setupMetadata = "codex=codex-cli 0.162.0\nclaude=2.1.301 (Claude Code)\ncodexSchema=" + bootstrapSchema + "\nagentd=agentd-setup-test\n"

func (f *setupFake) ImageID(_ context.Context, ref string) (string, error) {
	f.refs = append(f.refs, ref)
	return bootstrapImageID, nil
}
func (f *setupFake) ReadImageFile(_ context.Context, id, path string) ([]byte, error) {
	if id != bootstrapImageID || path != "/opt/markitect/runtime.env" {
		return nil, errors.New("unexpected image file read")
	}
	return []byte(setupMetadata), nil
}
func (f *setupFake) ObserveImageFileWithRequiredPaths(_ context.Context, id, file string, _ int64, _, _ []string) (container.ImageFileObservation, error) {
	if id != bootstrapImageID || file != "/usr/local/go/bin/go" {
		return container.ImageFileObservation{}, errors.New("unexpected executable observation")
	}
	return container.ImageFileObservation{SHA256: sha256.Sum256([]byte("go")), Size: 2}, nil
}
func (f *setupFake) ServerVersion(context.Context) (string, error) {
	f.versionCalls++
	return f.version, nil
}

// engine is the engine factory that setup receives.
func (f *setupFake) engine(path string) (setupEngine, error) {
	f.paths = append(f.paths, path)
	return f, nil
}

type setupFixture struct {
	fake   *setupFake
	repo   *testkit.Repo
	host   string
	token  string
	engine string
	args   []string
}

func newSetupFixture(t *testing.T) setupFixture {
	t.Helper()
	repo := productFixture(t)
	root := testkit.TempDir(t)
	f := setupFixture{fake: &setupFake{version: "27.3.1"}, repo: repo, host: filepath.Join(root, "host"), token: filepath.Join(root, "token"), engine: filepath.Join(root, "docker")}
	if err := os.WriteFile(f.token, []byte(setupSentinel), 0o600); err != nil {
		t.Fatal(err)
	}
	f.args = []string{"setup", "--repo", repo.Dir, "--host", f.host, "--image", "markitect-runtime:setup-test", "--credential", f.token, "--engine", f.engine, "--agent-model", "claude-haiku-5-5", "--check", "go=/usr/local/go/bin/go"}
	return f
}

func (f setupFixture) run(t *testing.T, extra ...string) (code int, stdout, stderr string) {
	t.Helper()
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	preparation := apply.NewPreparationService(apply.NewModelService(repository, repository))
	command := cli.New(preparation, nil, nil, repository, nil, nil, nil).WithSetup(newSetup(preparation, f.fake.engine).Setup)
	var out, errout bytes.Buffer
	code = command.Run(context.Background(), append(slices.Clone(f.args), extra...), &out, &errout)
	return code, out.String(), errout.String()
}

type setupReportValue struct {
	Status string
	Bound  struct{ ImageID, AgentdVersion, CliVersion, EngineVersion string }
	Files  []struct{ Path, SHA256, Action string }
	// The profile names and the next command.
	AgentProfiles, CheckProfiles []string
	ApplyCommand, Error          string
}

func decodeSetupReport(t *testing.T, out string) setupReportValue {
	t.Helper()
	var report setupReportValue
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return report
}

func readTree(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || filepath.Base(path) == lockFileName {
			return err
		}
		data, err := os.ReadFile(path)
		files[path] = data
		return err
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return files
}

func mustPath(t *testing.T, s string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSetupPreparesAHostThatApplyAndCheckAccept(t *testing.T) {
	f := newSetupFixture(t)
	fake := f.fake

	code, out, errout := f.run(t)
	if code != 0 {
		t.Fatalf("exit %d\nstdout=%s\nstderr=%s", code, out, errout)
	}
	if strings.Contains(out, setupSentinel) || strings.Contains(errout, setupSentinel) {
		t.Fatal("the credential bytes reached the output")
	}
	if fake.versionCalls != 1 || len(fake.paths) != 1 || fake.paths[0] != f.engine || len(fake.refs) == 0 || fake.refs[0] != "markitect-runtime:setup-test" {
		t.Fatalf("engine or image reference not used as given: %v %v", fake.paths, fake.refs)
	}
	report := decodeSetupReport(t, out)
	hostPath, configPath := filepath.Join(f.host, "host.yaml"), filepath.Join(f.host, "run.json")
	if report.Status != "passed" || report.Bound.ImageID != bootstrapImageID || report.Bound.AgentdVersion != "agentd-setup-test" || report.Bound.CliVersion != "2.1.301 (Claude Code)" || report.Bound.EngineVersion != "27.3.1" {
		t.Fatalf("bound identities: %+v", report)
	}
	if len(report.Files) != 2 || report.Files[0].Path != hostPath || report.Files[1].Path != configPath {
		t.Fatalf("files: %+v", report.Files)
	}
	want := "markitect apply --repo " + f.repo.Dir + " --host-profile " + hostPath + " --run-config " + configPath
	if strings.NewReplacer("'", "", `"`, "").Replace(report.ApplyCommand) != want {
		t.Fatalf("apply command %q want %q", report.ApplyCommand, want)
	}
	written := readTree(t, f.host)
	for _, file := range report.Files {
		sum := sha256.Sum256(written[file.Path])
		if file.Action != "written" || file.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatalf("file report %+v", file)
		}
	}
	for path, data := range written {
		if bytes.Contains(data, []byte(setupSentinel)) {
			t.Fatalf("%s contains the credential bytes", path)
		}
	}
	if runtime.GOOS != "windows" {
		for _, path := range []string{f.host, hostPath, configPath} {
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm()&0o077 != 0 {
				t.Fatalf("%s is not private: %v %v", path, info, err)
			}
		}
	}

	// Apply's decoder loads the run configuration; the optional keys are omitted.
	cfg, err := loadRunConfig(context.Background(), configPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(written[configPath], &raw); err != nil {
		t.Fatal(err)
	}
	if _, found := raw["ownerDecisions"]; found {
		t.Fatal("ownerDecisions written without a decision")
	}
	if _, found := raw["deadline"]; found {
		t.Fatal("deadline written without one")
	}
	source := cfg.Credentials[setupCredentialRef]
	store := cfg.WorkspaceHelper.Labels["markitect.store"]
	if cfg.EngineExecutable != f.engine || cfg.StateDirectory != filepath.Join(f.host, "state") || cfg.WorkspaceHelper.Image != bootstrapImageID || source.Provider != "claude-code" || source.Mode != "dedicated-runtime-token" || source.Path != f.token || store == "" || cfg.RepositoryIdentity != store || cfg.ProductBuild == "" || cfg.OwnerRequest != defaultOwnerRequest {
		t.Fatalf("run configuration: %+v", cfg)
	}

	// check --metadata --host-profile passes and every Manager resolves the
	// profile with the requested model.
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	preparation := apply.NewPreparationService(apply.NewModelService(repository, repository))
	agents := agentMetadataRouter{claude: claudecode.NewMetadataPreparer(func(claudecode.EngineConfig) (claudecode.ImageReader, error) { return fake, nil })}
	checker := checks.NewMetadataPreparer(func(checks.EngineConfig) (checks.ImageReader, error) { return fake, nil })
	var checkOut, checkErr bytes.Buffer
	command := cli.New(preparation, agents, checker, repository, nil, nil, nil)
	if code := command.Run(context.Background(), []string{"check", "--repo", f.repo.Dir, "--metadata", "--host-profile", hostPath}, &checkOut, &checkErr); code != 0 {
		t.Fatalf("check --metadata exit %d\n%s\n%s", code, checkOut.String(), checkErr.String())
	}
	selection := apply.InputSelection{Repository: f.repo.Dir, Local: apply.ModelSelection{Boundary: ".", ManifestPath: mustPath(t, ".markitect/model.yaml")}}
	selection.HostProfiles = []apply.SourceLocation{{Boundary: f.host, Path: mustPath(t, "host.yaml")}}
	result, err := preparation.PreviewSettings(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{}, agents, checker)
	attempt, ok := result.Prepared()
	if err != nil || !ok {
		t.Fatalf("%v %v", err, result.Status())
	}
	resolved, ok := attempt.Settings().Settings()
	if !ok || len(resolved.Managers()) == 0 {
		t.Fatal("no Manager settings resolved")
	}
	for _, manager := range resolved.Managers() {
		for _, name := range []string{manager.Execute.Profile.Value.Name, manager.Verify.Profile.Value.Name} {
			if name != "markitect-agent" {
				t.Fatalf("manager %s resolves profile %q", manager.Package, name)
			}
		}
	}
	projected := 0
	for _, profile := range attempt.Settings().ProfileObservations() {
		if profile.Input().Namespace() != "agentProfiles" {
			continue
		}
		p, err := claudecode.ProfileFromPrepared(profile)
		if err != nil || p.Session.Model != "claude-haiku-5-5" || p.Session.Effort != "high" || p.CredentialRef != setupCredentialRef || p.Session.SystemPrompt != setupSystemPrompt {
			t.Fatalf("agent profile: %v %+v", err, p)
		}
		projected++
	}
	if projected != 1 {
		t.Fatalf("expected one Agent profile, got %d", projected)
	}

	// A second run changes nothing.
	code, out, errout = f.run(t)
	if code != 0 {
		t.Fatalf("second run exit %d\n%s\n%s", code, out, errout)
	}
	again := decodeSetupReport(t, out)
	if len(again.Files) != 2 || again.Files[0].Action != "unchanged" || again.Files[1].Action != "unchanged" || again.Files[0].SHA256 != report.Files[0].SHA256 || again.Files[1].SHA256 != report.Files[1].SHA256 {
		t.Fatalf("second run: %+v", again.Files)
	}
	for path, data := range readTree(t, f.host) {
		if !bytes.Equal(data, written[path]) {
			t.Fatalf("%s changed on an identical run", path)
		}
	}
}

func TestRunComposesSetup(t *testing.T) {
	// Without the composition the verb would be unavailable (exit 3); a
	// composed setup refuses the incomplete invocation as usage (exit 2).
	var out, errout bytes.Buffer
	if code := Run(context.Background(), []string{"setup"}, &out, &errout); code != 2 {
		t.Fatalf("Run(setup)=%d want=2\nstdout=%s\nstderr=%s", code, out.String(), errout.String())
	}
}

func TestSetupRefusalsLeaveExistingFilesUntouched(t *testing.T) {
	f := newSetupFixture(t)
	if code, out, errout := f.run(t); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errout)
	}
	before := readTree(t, f.host)
	unchanged := func(name string) {
		t.Helper()
		after := readTree(t, f.host)
		if len(after) != len(before) {
			t.Fatalf("%s: files appeared or vanished", name)
		}
		for path, data := range before {
			if !bytes.Equal(after[path], data) {
				t.Fatalf("%s: %s changed", name, path)
			}
		}
	}
	// Every case would change a file if it were accepted.
	t.Run("missing check", func(t *testing.T) {
		g := f
		g.args = withoutArgs(f.args, "--check", "go=/usr/local/go/bin/go")
		code, out, _ := g.run(t, "--agent-model", "claude-sonnet-5-5")
		if code != 1 || !strings.Contains(out, "--check go=EXECUTABLE") {
			t.Fatalf("exit %d: %s", code, out)
		}
		unchanged("missing check")
	})
	t.Run("unused check", func(t *testing.T) {
		code, out, _ := f.run(t, "--check", "unused=/bin/true", "--agent-model", "claude-sonnet-5-5")
		if code != 1 || !strings.Contains(out, `profile \"unused\"`) {
			t.Fatalf("exit %d: %s", code, out)
		}
		unchanged("unused check")
	})
	t.Run("relative check executable", func(t *testing.T) {
		g := f
		g.args = replaceArg(f.args, "go=/usr/local/go/bin/go", "go=bin/go")
		code, out, _ := g.run(t, "--agent-model", "claude-sonnet-5-5")
		if code != 1 || !strings.Contains(out, "error") {
			t.Fatalf("exit %d: %s", code, out)
		}
		unchanged("relative check executable")
	})
	t.Run("invalid effort", func(t *testing.T) {
		code, out, _ := f.run(t, "--effort", "max")
		if code != 1 || !strings.Contains(out, "error") {
			t.Fatalf("exit %d: %s", code, out)
		}
		unchanged("invalid effort")
	})
	t.Run("held state lock", func(t *testing.T) {
		held, err := lockState(filepath.Join(f.host, "state"))
		if err != nil {
			t.Fatal(err)
		}
		defer held.Release()
		code, out, _ := f.run(t, "--agent-model", "claude-sonnet-5-5")
		if code != 3 || !strings.Contains(out, "Apply holds") {
			t.Fatalf("exit %d: %s", code, out)
		}
		unchanged("held state lock")
	})
	t.Run("directory inside the repository", func(t *testing.T) {
		g := f
		inside := filepath.Join(f.repo.Dir, "host")
		g.args = replaceArg(f.args, f.host, inside)
		code, out, _ := g.run(t)
		if code != 1 || !strings.Contains(out, "overlap") {
			t.Fatalf("exit %d: %s", code, out)
		}
		if _, err := os.Stat(inside); !os.IsNotExist(err) {
			t.Fatal("a refused setup created a directory inside the repository")
		}
	})
	t.Run("directory containing the repository", func(t *testing.T) {
		g := f
		g.args = replaceArg(f.args, f.host, filepath.Dir(f.repo.Dir))
		if code, out, _ := g.run(t); code != 1 || !strings.Contains(out, "overlap") {
			t.Fatalf("exit %d: %s", code, out)
		}
	})
	t.Run("credential inside the repository", func(t *testing.T) {
		g := f
		inside := filepath.Join(f.repo.Dir, "token")
		if err := os.WriteFile(inside, []byte(setupSentinel), 0o600); err != nil {
			t.Fatal(err)
		}
		g.args = replaceArg(f.args, f.token, inside)
		if code, out, _ := g.run(t); code != 1 || !strings.Contains(out, "overlap") {
			t.Fatalf("exit %d: %s", code, out)
		}
	})
	t.Run("credential that others can read", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows has no owner-only mode bits")
		}
		open := filepath.Join(filepath.Dir(f.token), "open-token")
		if err := os.WriteFile(open, []byte(setupSentinel), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(open, 0o644); err != nil {
			t.Fatal(err)
		}
		g := f
		g.args = replaceArg(f.args, f.token, open)
		code, out, _ := g.run(t)
		if code != 1 || !strings.Contains(out, "0600") || strings.Contains(out, setupSentinel) {
			t.Fatalf("exit %d: %s", code, out)
		}
	})
	t.Run("credential that is missing or a directory", func(t *testing.T) {
		for _, path := range []string{filepath.Join(filepath.Dir(f.token), "absent"), filepath.Dir(f.token)} {
			g := f
			g.args = replaceArg(f.args, f.token, path)
			if code, out, _ := g.run(t); code != 1 {
				t.Fatalf("%s: exit %d: %s", path, code, out)
			}
		}
	})
	unchanged("end")
}

func TestSetupRefusesAnEngineWithoutVolumeSubpaths(t *testing.T) {
	f := newSetupFixture(t)
	f.fake.version = "25.0.5"
	code, out, _ := f.run(t)
	if code != 3 || !strings.Contains(out, "require Docker 26") {
		t.Fatalf("exit %d: %s", code, out)
	}
	if _, err := os.Stat(f.host); !os.IsNotExist(err) {
		t.Fatal("a refused setup created the Host directory")
	}
}

func TestSetupWritesOwnerDecisionsAndDeadlineOnlyWhenGiven(t *testing.T) {
	f := newSetupFixture(t)
	code, out, errout := f.run(t, "--request", "Apply the model.", "--decision", "Use euro.", "--decision", "No tabs.", "--deadline", "2030-01-02T03:04:05Z", "--store", "setup-store")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errout)
	}
	cfg, err := loadRunConfig(context.Background(), filepath.Join(f.host, "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OwnerRequest != "Apply the model." || len(cfg.OwnerDecisions) != 2 || cfg.OwnerDecisions[1] != "No tabs." || cfg.Deadline != "2030-01-02T03:04:05Z" || cfg.RepositoryIdentity != "setup-store" || cfg.WorkspaceHelper.Labels["markitect.store"] != "setup-store" {
		t.Fatalf("%+v", cfg)
	}
	// A deadline that is not UTC is refused before any file changes.
	before := readTree(t, f.host)
	if code, out, _ := f.run(t, "--deadline", "2030-01-02T03:04:05+02:00"); code != 1 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for path, data := range before {
		now, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(now, data) {
			t.Fatalf("%s changed: %v", path, err)
		}
	}
}

func TestEncodedRunConfigRoundTripsThroughTheDecoder(t *testing.T) {
	root := testkit.TempDir(t)
	token, state := filepath.Join(root, "token"), filepath.Join(root, "state")
	document := setupRunConfig(cli.SetupRequest{Credential: token, Engine: filepath.Join(root, "docker"), ProductBuild: "0.0.1", OwnerRequest: "Do it.", OwnerDecisions: []string{"A", "B <&> C"}, Deadline: "2031-05-06T07:08:09Z"}, bootstrapImageID, "store-1", state)
	data, err := encodeRunConfig(document)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := encodeRunConfig(document)
	if !bytes.Equal(data, again) || !bytes.HasSuffix(data, []byte("\n")) || bytes.Contains(data, []byte{'\\', 'u', '0', '0', '3', 'c'}) {
		t.Fatalf("encoding is not deterministic or escapes HTML:\n%s", data)
	}
	cfg, err := decodeRunConfig(data)
	if err != nil || cfg.OwnerDecisions[1] != "B <&> C" || cfg.deadline().IsZero() || cfg.Credentials[setupCredentialRef].Path != token || cfg.Harvest.MaxEntries != 50000 {
		t.Fatalf("%v %+v", err, cfg)
	}
	// An invalid document never encodes.
	document.OwnerRequest = " "
	if _, err := encodeRunConfig(document); err == nil {
		t.Fatal("an invalid run configuration was encoded")
	}
}

func withoutArgs(args []string, drop ...string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		if args[i] == drop[0] && i+1 < len(args) && args[i+1] == drop[1] {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func replaceArg(args []string, from, to string) []string {
	out := slices.Clone(args)
	for i := range out {
		if out[i] == from {
			out[i] = to
		}
	}
	return out
}

func TestSetupSelectsEveryProfileOfTheDefaultsAndPackageOverrides(t *testing.T) {
	f := newSetupFixture(t)
	execution, err := os.ReadFile(filepath.Join(f.repo.Dir, ".markitect", "execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(execution), "\r\n", "\n")
	if !strings.Contains(text, "packages: {}") {
		t.Fatal("fixture execution declarations changed")
	}
	f.repo.Write(".markitect/execution.yaml", strings.Replace(text, "packages: {}", "packages:\n  product/cli:\n    profiles: {verify: second-agent}", 1))
	f.repo.Commit("select a second Agent profile for one Package")
	code, out, errout := f.run(t)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errout)
	}
	report := decodeSetupReport(t, out)
	if !slices.Equal(report.AgentProfiles, []string{"markitect-agent", "second-agent"}) || !slices.Equal(report.CheckProfiles, []string{"go"}) {
		t.Fatalf("%+v", report)
	}
	host, err := os.ReadFile(filepath.Join(f.host, "host.yaml"))
	if err != nil || strings.Count(string(host), "claudeCode:") != 2 || strings.Count(string(host), "container:") != 1 {
		t.Fatalf("%v\n%s", err, host)
	}
}

func TestSetupRewritesOnlyTheFileWhoseBytesChange(t *testing.T) {
	f := newSetupFixture(t)
	code, out, errout := f.run(t)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errout)
	}
	first := decodeSetupReport(t, out)
	hostPath, configPath := filepath.Join(f.host, "host.yaml"), filepath.Join(f.host, "run.json")
	oldHost, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatal(err)
	}
	oldConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	// The model appears in host.yaml only, so only that file may change.
	code, out, errout = f.run(t, "--agent-model", "claude-sonnet-5-5")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errout)
	}
	second := decodeSetupReport(t, out)
	newHost, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(newHost)
	if len(second.Files) != 2 || second.Files[0].Action != "written" || second.Files[0].SHA256 != hex.EncodeToString(sum[:]) || second.Files[0].SHA256 == first.Files[0].SHA256 || !bytes.Contains(newHost, []byte("model: claude-sonnet-5-5")) || bytes.Equal(newHost, oldHost) {
		t.Fatalf("host.yaml was not rewritten with the new model: %+v", second.Files)
	}
	newConfig, err := os.ReadFile(configPath)
	if err != nil || second.Files[1].Action != "unchanged" || second.Files[1].SHA256 != first.Files[1].SHA256 || !bytes.Equal(newConfig, oldConfig) {
		t.Fatalf("run.json changed without a change of its inputs: %v %+v", err, second.Files)
	}
}
