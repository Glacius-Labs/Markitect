package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// Consume the shipped examples through actual decoding, checked preparation
// and CLI routing. Only image observations and the final execution port are
// controlled; this test never acquires a provider, credential or container.
func TestExplicitAccountConfigurationExamplesReachApply(t *testing.T) {
	repo := productFixture(t)
	runtimeDir := testkit.TempDir(t)
	engine := filepath.Join(runtimeDir, "engine-not-executed")
	login := filepath.Join(runtimeDir, "account-login-not-read.json")
	state := filepath.Join(runtimeDir, "state-not-created")
	runtimePath := filepath.Join(runtimeDir, "runtime.yaml")
	configPath := filepath.Join(runtimeDir, "run.json")
	// Obtain the invocation's build assertion through the actual CLI port;
	// Bootstrap does not own or import the build-information implementation.
	var build string
	probe := cli.New(nil, nil, nil, nil, nil, func(_ context.Context, request cli.ApplyRequest) (cli.ApplyReport, error) {
		build = request.Build
		return cli.ApplyReport{Status: model.NotCompleted}, nil
	})
	if code := probe.Run(context.Background(), []string{"apply", "--repo", repo.Dir, "--runtime-profile", runtimePath, "--run-config", configPath}, io.Discard, io.Discard); code != 3 || build == "" {
		t.Fatalf("CLI build assertion unavailable: code=%d build=%q", code, build)
	}
	values := map[string]string{
		"ENGINE_EXECUTABLE": engine, "ACCOUNT_LOGIN_PATH": login, "STATE_DIRECTORY": state,
		"IMAGE_ID": bootstrapImageID, "AGENTD_BUILD": "development", "CODEX_SCHEMA_DIGEST": bootstrapSchema,
		"ACCOUNT_MODEL": "explicit-test-account-model", "PRODUCT_BUILD": build,
	}
	for _, file := range []struct{ source, target string }{
		{"runtime-profile.example.yaml", runtimePath}, {"run.example.json", configPath},
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", file.source))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for key, value := range values {
			text = strings.ReplaceAll(text, strconv.Quote(key), strconv.Quote(value))
			text = strings.ReplaceAll(text, ": "+key+"\n", ": "+strconv.Quote(value)+"\n")
		}
		if err := os.WriteFile(file.target, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	preparation := apply.NewPreparationService(apply.NewModelService(repository, repository))
	images := &configurationImages{}
	agents := agentMetadataRouter{codex: codexappserver.NewMetadataPreparer(func(config codexappserver.EngineConfig) (codexappserver.ImageReader, error) {
		if config.Executable != engine {
			t.Fatal("agent profile lost explicit engine")
		}
		return images, nil
	})}
	checker := checks.NewMetadataPreparer(func(config checks.EngineConfig) (checks.ImageReader, error) {
		if config.Executable != engine {
			t.Fatal("check profile lost explicit engine")
		}
		return images, nil
	})
	calls := 0
	command := cli.New(preparation, agents, checker, repository, nil, func(ctx context.Context, request cli.ApplyRequest) (cli.ApplyReport, error) {
		calls++
		if request.RunConfig != configPath || request.Selection.Repository != repo.Dir || request.Build != build {
			t.Fatalf("CLI changed explicit invocation: %+v", request)
		}
		cfg, err := loadRunConfig(ctx, request.RunConfig)
		if err != nil || cfg.ProductBuild != request.Build || cfg.MaxMessages <= 0 || cfg.MessageTimeoutSeconds <= 0 {
			t.Fatalf("example run configuration cannot invoke assignment Apply: %+v %v", cfg, err)
		}
		credential := cfg.Credentials["example-account"]
		if credential.Provider != "codex-app-server" || credential.Mode != "dedicated-runtime-login" || credential.Path != login {
			t.Fatalf("example lost normal account-login reference: %+v", credential)
		}
		prepared, err := preparation.CaptureSettings(ctx, apply.CaptureRequest{Selection: request.Selection}, model.Limits{}, agents, checker)
		attempt, ok := prepared.Prepared()
		if err != nil || !ok {
			t.Fatalf("example Runtime does not pass real preparation: status=%s err=%v", prepared.Status(), err)
		}
		settings := attempt.Settings()
		agentProfiles := 0
		for _, profile := range settings.ProfileObservations() {
			if profile.Input().Namespace() != "agentProfiles" {
				continue
			}
			agentProfiles++
			p, err := codexappserver.ProfileFromPrepared(profile)
			if err != nil || credentialMapped(cfg, p.CredentialRef, "codex-app-server") != nil || !profileEngine(profile, engine) || p.Session.Model != values["ACCOUNT_MODEL"] || p.ImageID != bootstrapImageID {
				t.Fatalf("example account/profile binding refused: %v", err)
			}
		}
		if agentProfiles != 1 {
			t.Fatalf("example did not resolve the actual Agent profile: %d", agentProfiles)
		}
		selected, complete := settings.Checks()
		if !complete || len(selected) != 1 {
			t.Fatalf("example did not resolve project checks: complete=%t count=%d", complete, len(selected))
		}
		for _, check := range selected {
			if _, err := executionCheck(check, cfg); err != nil {
				t.Fatal("example check cannot be constructed", err)
			}
		}
		return cli.ApplyReport{Status: model.NotCompleted}, errors.New("controlled test stops before runtime acquisition")
	})
	var out, stderr bytes.Buffer
	code := command.Run(context.Background(), []string{"apply", "--repo", repo.Dir, "--runtime-profile", runtimePath, "--run-config", configPath}, &out, &stderr)
	if code != 3 || calls != 1 || !strings.Contains(out.String(), "controlled test stops before runtime acquisition") {
		t.Fatalf("explicit examples did not reach execution boundary: code=%d calls=%d stdout=%s stderr=%s", code, calls, out.String(), stderr.String())
	}
	for _, path := range []string{engine, login, state} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("deterministic example test acquired runtime input %s: %v", path, err)
		}
	}
}

type configurationImages struct{ bootstrapImageReader }

func (*configurationImages) ObserveImageFileWithRequiredPaths(_ context.Context, image, file string, _ int64, _, _ []string) (container.ImageFileObservation, error) {
	if image != bootstrapImageID || file != "/usr/local/go/bin/go" {
		return container.ImageFileObservation{}, errors.New("unexpected example image/executable")
	}
	return container.ImageFileObservation{SHA256: sha256.Sum256([]byte("go")), Size: 2}, nil
}
