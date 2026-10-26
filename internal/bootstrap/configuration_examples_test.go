package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	runexec "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestExplicitAccountConfigurationExampleLoadsAndInvalidPreflightStartsNothing(t *testing.T) {
	data, err := os.ReadFile("../../examples/run.example.json")
	if err != nil {
		t.Fatal(err)
	}
	dir := testkit.TempDir(t)
	state := filepath.Join(dir, "state")
	replacements := map[string]string{"PRODUCT_BUILD": "source-test-build", "STATE_DIRECTORY": state, "ENGINE_EXECUTABLE": filepath.Join(dir, "never-run-engine"), "DEADLINE_UTC": "2026-10-23T23:00:00Z", "ACCOUNT_LOGIN_PATH": filepath.Join(dir, "never-read-account-login.json")}
	text := string(data)
	for token, value := range replacements {
		encoded, _ := json.Marshal(value)
		text = strings.ReplaceAll(text, `"`+token+`"`, string(encoded))
	}
	config := filepath.Join(dir, "run.json")
	if err = os.WriteFile(config, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadRunConfig(context.Background(), config)
	if err != nil || cfg.FormatVersion != 4 || cfg.Credentials["example-account"].Mode != "dedicated-runtime-login" {
		t.Fatal(cfg, err)
	}
	var out, stderr bytes.Buffer
	calls := 0
	code := run(context.Background(), []string{"apply", "--repo", dir, "--runtime-profile", filepath.Join(dir, "never-read-runtime.yaml"), "--run-config", config}, &out, &stderr, func(codexappserver.EngineConfig) (codexappserver.ImageReader, error) {
		calls++
		t.Fatal("invalid Apply preflight acquired runtime metadata")
		return nil, nil
	})
	var report map[string]any
	if err = json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if code != 3 || report["status"] != "not_completed" || report["outcome"] != "incomplete" || calls != 0 {
		t.Fatalf("code=%d report=%v stderr=%s", code, report, stderr.String())
	}
	if _, err = os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("invalid Apply preflight created state", err)
	}
}

// This reader supplies controlled metadata only; it never acquires an image.
type exampleCheckImages struct {
	t     *testing.T
	reads int
}

func (r *exampleCheckImages) ImageID(_ context.Context, image string) (string, error) {
	if image != bootstrapImageID {
		r.t.Fatal("example Check selected a different image", image)
	}
	return image, nil
}

func (r *exampleCheckImages) ObserveImageFileWithRequiredPaths(_ context.Context, image, path string, _ int64, _, _ []string) (container.ImageFileObservation, error) {
	r.reads++
	if image != bootstrapImageID || path != "/usr/local/go/bin/go" {
		r.t.Fatal("example Check changed its literal image/executable", image, path)
	}
	data := []byte("controlled example executable metadata")
	return container.ImageFileObservation{SHA256: sha256.Sum256(data), Size: int64(len(data))}, nil
}

func TestExactConfigurationExamplesPrepareAndReachControlledApplyBoundary(t *testing.T) {
	repo := productFixture(t)
	outside := testkit.TempDir(t)
	engine := filepath.Join(outside, "engine with spaces", "never-executed")
	login, state := filepath.Join(outside, "never-read-login.json"), filepath.Join(outside, "state")
	runtimePath, configPath := filepath.Join(outside, "runtime.yaml"), filepath.Join(outside, "run.json")
	deadline := time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)
	// The CLI owns the binary build identity; read its existing request boundary.
	var build string
	identity := cli.New(nil, nil, nil, nil, nil, func(_ context.Context, request cli.ApplyRequest) (cli.ApplyReport, error) {
		build = request.Build
		return cli.ApplyReport{Status: model.NotCompleted}, nil
	})
	if code := identity.Run(context.Background(), []string{"apply", "--repo", repo.Dir, "--runtime-profile", runtimePath, "--run-config", configPath}, io.Discard, io.Discard); code != 3 || build == "" {
		t.Fatal("CLI did not supply its actual binary build identity", code)
	}
	for name, replacements := range map[string]map[string]string{
		"runtime-profile.example.yaml": {"ENGINE_EXECUTABLE": engine, "IMAGE_ID": bootstrapImageID, "AGENTD_BUILD": "development", "CODEX_SCHEMA_DIGEST": bootstrapSchema, "ACCOUNT_MODEL": "explicit-account-model"},
		"run.example.json":             {"PRODUCT_BUILD": build, "STATE_DIRECTORY": state, "ENGINE_EXECUTABLE": engine, "DEADLINE_UTC": deadline, "ACCOUNT_LOGIN_PATH": login},
	} {
		data, err := os.ReadFile("../../examples/" + name)
		if err != nil {
			t.Fatal(err)
		}
		text, destination := string(data), runtimePath
		for token, value := range replacements {
			from, to := token, strconv.Quote(value) // JSON string quoting is valid YAML.
			if strings.HasSuffix(name, ".json") {
				encoded, _ := json.Marshal(value)
				from, to = strconv.Quote(token), string(encoded)
				destination = configPath
			}
			if !strings.Contains(text, from) {
				t.Fatal("example placeholder missing", name, token)
			}
			text = strings.ReplaceAll(text, from, to)
		}
		if err := os.WriteFile(destination, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	preparation := project.NewPreparationService(project.NewModelService(repository, repository))
	agentImages, checkImages := &bootstrapImageReader{}, &exampleCheckImages{t: t}
	agentFactories, checkFactories := 0, 0
	agents := codexappserver.NewMetadataPreparer(func(c codexappserver.EngineConfig) (codexappserver.ImageReader, error) {
		agentFactories++
		if c.Executable != engine {
			t.Fatal("Agent factory lost the example engine", c)
		}
		return agentImages, nil
	})
	checkMetadata := checks.NewMetadataPreparer(func(c checks.EngineConfig) (checks.ImageReader, error) {
		checkFactories++
		if c.Executable != engine {
			t.Fatal("Check factory lost the example engine", c)
		}
		return checkImages, nil
	})
	boundary := errors.New("controlled example composition complete; runtime not started")
	calls := 0
	command := cli.New(preparation, agents, checkMetadata, repository, nil, func(ctx context.Context, request cli.ApplyRequest) (cli.ApplyReport, error) {
		calls++
		if request.Build != build || request.RunConfig != configPath || request.Selection.Repository != repo.Dir || len(request.Selection.RuntimeProfiles) != 1 || request.Selection.RuntimeProfiles[0].Boundary != outside || request.Selection.RuntimeProfiles[0].Path.String() != "runtime.yaml" {
			t.Fatal("CLI changed the exact example invocation", request)
		}
		cfg, err := loadRunConfig(ctx, request.RunConfig)
		if err != nil || cfg.ProductBuild != request.Build || cfg.EngineExecutable != engine || cfg.Deadline != deadline || cfg.StateDirectory != state || cfg.MaxManagers != 32 || cfg.MaxMessages != 128 || cfg.ProviderIdleTimeoutSeconds != 600 || cfg.CleanupTimeoutSeconds != 45 || cfg.PersistenceTimeoutSeconds != 30 {
			t.Fatal("example invocation did not retain its explicit values", cfg, err)
		}
		prepared, metadata, status, err := captureApplyInputs(ctx, preparation, repository, request.Selection, agents, checkMetadata)
		if err != nil || status != model.Passed || metadata.Status != model.Passed || !prepared.Valid() || len(metadata.Profiles) != 2 {
			t.Fatal("exact example Runtime did not close", status, metadata, err)
		}
		for _, p := range metadata.Profiles {
			switch p.Input().Namespace() {
			case "agentProfiles":
				profile, err := codexappserver.ProfileFromPrepared(p)
				want := codexappserver.Profile{ImageID: bootstrapImageID, CredentialRef: "example-account", MaxWireBytes: 1048576, Session: codexappserver.SessionProfile{Version: 1, Model: "explicit-account-model", Effort: "high", Store: "markitect-example", CPUs: 1, MemoryBytes: 2147483648, Pids: 192, TmpBytes: 268435456}}
				if err != nil || p.Input().Name() != "markitect-agent" || !reflect.DeepEqual(profile, want) {
					t.Fatal("native v3 example projection differs", profile, err)
				}
				if len(cfg.Credentials) != 1 || cfg.Credentials[profile.CredentialRef] != (credentialSource{Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: login}) {
					t.Fatal("normal account reference differs between examples")
				}
				resolve, err := newCredentialResolver(cfg.Credentials)
				if err != nil {
					t.Fatal(err)
				}
				// Construct the native consumer from this exact prepared profile.
				// Never Open it or resolve the nonexistent login.
				if _, err := codexappserver.NewAgentsWithRuntime(func(context.Context, container.ContainerSpec) (codexappserver.RuntimeSession, error) {
					t.Fatal("static example test started a runtime")
					return nil, boundary
				}, p, resolve); err != nil {
					t.Fatal("native consumer refused the prepared example", err)
				}
			case "checkProfiles":
				profile, err := checks.ContainerProfileFromPrepared(p)
				want := checks.ContainerProfile{ImageID: bootstrapImageID, Executable: "/usr/local/go/bin/go", ArgumentPrefix: []string{}, Env: []string{"GOTOOLCHAIN=local", "GOCACHE=/home/agent/.cache/go-build", "GOTMPDIR=/home/agent"}, Limits: runexec.CheckLimits{MilliCPUs: 1000, MemoryBytes: 2147483648, Processes: 64}, MaxOutputBytes: 1048576}
				if err != nil || p.Input().Name() != "go" || !reflect.DeepEqual(profile, want) {
					t.Fatal("container Check example projection differs", profile, err)
				}
				if len(prepared.Checks()) == 0 {
					t.Fatal("fixture lost its declared go check")
				}
				for _, check := range prepared.Checks() {
					req, err := profile.Request(check, prepared.Attempt().Model().Inputs().Raw.Census)
					if err != nil || check.Profile.Value != "go" || req.ImageID != profile.ImageID || req.Executable != profile.Executable || req.Limits != profile.Limits || !reflect.DeepEqual(req.Args, check.Args.Value) || !reflect.DeepEqual(req.Env, profile.Env) || req.MaxOutputBytes != profile.MaxOutputBytes {
						t.Fatal("declared Check lost exact example settings", req, err)
					}
				}
			default:
				t.Fatal("unexpected prepared example profile", p.Input().Namespace())
			}
		}
		return cli.ApplyReport{Status: model.NotCompleted}, boundary
	})
	var out, stderr bytes.Buffer
	code := command.Run(context.Background(), []string{"apply", "--repo", repo.Dir, "--runtime-profile", runtimePath, "--run-config", configPath}, &out, &stderr)
	var report map[string]any
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || code != 3 || calls != 1 || report["error"] != boundary.Error() || report["status"] != "not_completed" || report["outcome"] != "incomplete" || agentFactories == 0 || checkFactories == 0 || len(agentImages.calls) != 3*agentFactories || checkImages.reads != checkFactories {
		t.Fatal("controlled CLI composition boundary not observed", code, report, stderr.String(), err)
	}
	for _, path := range []string{engine, login, state} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("static example composition acquired a runtime input", path, err)
		}
	}
}
