package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// claudeImages serves the shared image's runtime record with both CLIs.
type claudeImages struct{}

func (claudeImages) ImageID(context.Context, string) (string, error) { return bootstrapImageID, nil }
func (claudeImages) ReadImageFile(context.Context, string, string) ([]byte, error) {
	return []byte("codex=codex-cli 0.162.0\ncodexSchema=" + bootstrapSchema + "\nagentd=development\nclaude=2.1.308 (Claude Code)\n"), nil
}
func (claudeImages) ImageEnvironment(context.Context, string) ([]string, error) {
	return []string{"PATH=/usr/local/bin:/usr/bin", "HOME=/home/agent"}, nil
}
func (claudeImages) ImageDirectoryNames(context.Context, string, string, int64) ([]string, error) {
	return []string{"etc", "etc/hosts"}, nil
}

func claudeSection(engine string) string {
	return fmt.Sprintf(`    claudeCode:
      profileVersion: 1
      engineExecutable: %q
      imageID: %s
      agentdVersion: development
      instructions: []
      credentialRef: claude-account
      maxWireBytes: 4194304
      session: {version: 1, model: sonnet, effort: high, baseInstructions: '', developerInstructions: '', store: markitect-claude, cpus: 1, memoryBytes: 2147483648, pids: 192, tmpBytes: 268435456, network: ''}
`, engine, bootstrapImageID)
}

func preparedAgent(t *testing.T, section string, agents runtimeconfig.AgentProfilePreparer) runtimeconfig.PreparedProfile {
	t.Helper()
	path, err := snapshot.NewPath("runtime.yaml")
	if err != nil {
		t.Fatal(err)
	}
	entry, err := snapshot.NewEntry(path, snapshot.Regular, []byte("apiVersion: markitect.runtime/v1\nagentProfiles:\n  account:\n"+section+"checkProfiles: {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	// A port error stays an observation of the returned profile.
	result, _ := runtimeconfig.Prepare(context.Background(), project.RawInputs{RuntimeProfiles: []project.SourceFile{{Location: project.SourceLocation{Boundary: t.TempDir(), Path: path}, File: entry}}}, agents, nil)
	if len(result.Profiles) != 1 {
		t.Fatalf("preparation: %+v", result.Diagnostics)
	}
	return result.Profiles[0]
}

// fakeEngine names an existing file: binding an engine only checks its path.
func fakeEngine(t *testing.T) string {
	t.Helper()
	engine := filepath.Join(testkit.TempDir(t), "engine")
	writeFakeCredential(t, engine, []byte("not executed"))
	return engine
}

func testRouter() agentMetadata {
	return agentMetadata{
		"codexAppServer": codexappserver.NewMetadataPreparer(func(codexappserver.EngineConfig) (codexappserver.ImageReader, error) { return claudeImages{}, nil }),
		"claudeCode":     claudecode.NewMetadataPreparer(func(claudecode.EngineConfig) (claudecode.ImageReader, error) { return claudeImages{}, nil }),
	}
}

// Each Agent section reaches its own metadata preparer; an uncomposed section
// is incomplete, never routed to another adapter.
func TestAgentMetadataRoutesEachSectionToItsAdapter(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	claude := preparedAgent(t, claudeSection(engine), testRouter())
	if claude.Observation().Status != model.Passed || claude.Observation().AgentImage.Provider != runtimeconfig.AgentImageClaudeCode {
		t.Fatalf("claudeCode preparation %+v", claude.Observation())
	}
	partial := agentMetadata{"codexAppServer": testRouter()["codexAppServer"]}
	missing := preparedAgent(t, claudeSection(engine), partial)
	if missing.Observation().Status != model.NotCompleted {
		t.Fatalf("an uncomposed section was prepared: %+v", missing.Observation())
	}
}

func TestComposeAgentsSelectsClaudeWithItsSubscriptionLogin(t *testing.T) {
	engine := fakeEngine(t)
	selected := preparedAgent(t, claudeSection(engine), testRouter())
	binary, err := container.NewEngine(engine)
	if err != nil {
		t.Fatal(err)
	}
	login := filepath.Join(testkit.TempDir(t), "credentials.json")
	cfg := runConfig{Credentials: map[string]credentialSource{"claude-account": {Provider: "claude-code", Mode: claudecode.ModeCredentials, Path: login}}}
	native, err := composeAgents(binary, selected, cfg, []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}
	sessions, ok := native.sessions.(*claudecode.Agents)
	if !ok || native.transcripts != &sessions.TranscriptRoot || native.retries() != 0 {
		t.Fatal("composition lost the Claude Agents")
	}
	if native.spec.Image != bootstrapImageID || native.spec.Labels["markitect.store"] != "markitect-claude" || native.spec.Network != "none" || native.spec.User != "1000:1000" {
		t.Fatalf("workspace spec %+v", native.spec)
	}
	// An API key in the Apply environment is refused by name before any port.
	_, err = composeAgents(binary, selected, cfg, []string{"ANTHROPIC_API_KEY=private-key"})
	if err == nil || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") || strings.Contains(err.Error(), "private-key") {
		t.Fatalf("API key environment: %v", err)
	}
	// The selected reference must name a Claude subscription login, and the
	// refusal names the provider and mode it needs.
	for _, source := range []credentialSource{
		{Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: login},
		{Provider: "claude-code", Mode: "api-key", Path: login},
	} {
		wrong := runConfig{Credentials: map[string]credentialSource{"claude-account": source}}
		_, err := composeAgents(binary, selected, wrong, nil)
		if !errors.Is(err, errRuntimeCredential) || !strings.Contains(err.Error(), "provider claude-code and mode subscription-credentials") || strings.Contains(err.Error(), login) {
			t.Fatalf("%+v admitted or unexplained: %v", source, err)
		}
	}
}

// A mapping with an unknown provider or a mode its provider does not take is
// refused with the provider and modes it needs.
func TestCredentialMappingNamesTheExpectedProviderAndMode(t *testing.T) {
	path := filepath.Join(testkit.TempDir(t), "never-read")
	for source, want := range map[credentialSource]string{
		{Provider: "claude-code", Mode: "subscription-token", Path: path}:         "provider claude-code takes mode subscription-credentials",
		{Provider: "codex-app-server", Mode: "api-key", Path: path}:               "provider codex-app-server takes mode dedicated-runtime-login",
		{Provider: "anthropic-api", Mode: "subscription-credentials", Path: path}: "use codex-app-server or claude-code",
	} {
		_, err := newCredentialReader(map[string]credentialSource{"ref": source})
		if !errors.Is(err, errRuntimeCredential) || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), path) {
			t.Fatalf("%+v: %v", source, err)
		}
	}
}

func TestCredentialViewsResolveOnlyTheirProvider(t *testing.T) {
	dir := testkit.TempDir(t)
	codexPath, claudePath := filepath.Join(dir, "codex.json"), filepath.Join(dir, "claude.json")
	writeFakeCredential(t, codexPath, []byte("codex-login"))
	writeFakeCredential(t, claudePath, []byte("claude-login"))
	sources := map[string]credentialSource{
		"codex":  {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: codexPath},
		"claude": {Provider: "claude-code", Mode: claudecode.ModeCredentials, Path: claudePath},
	}
	codex, err := newCredentialResolver(sources)
	if err != nil {
		t.Fatal(err)
	}
	claude, err := newClaudeCredentialResolver(sources)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := codex(context.Background(), "codex"); err != nil || string(got.RuntimeLogin) != "codex-login" {
		t.Fatal(got, err)
	}
	if _, err := codex(context.Background(), "claude"); !errors.Is(err, errRuntimeCredential) {
		t.Fatal("the Codex view resolved a Claude login", err)
	}
	if got, err := claude(context.Background(), "claude"); err != nil || got.Mode != claudecode.ModeCredentials || string(got.Login) != "claude-login" {
		t.Fatal(got, err)
	}
	if _, err := claude(context.Background(), "codex"); !errors.Is(err, errRuntimeCredential) {
		t.Fatal("the Claude view resolved a Codex login", err)
	}
	for _, mode := range []string{"api-key", "dedicated-runtime-login", "environment", "subscription-token"} {
		if _, err := newClaudeCredentialResolver(map[string]credentialSource{"x": {Provider: "claude-code", Mode: mode, Path: claudePath}}); !errors.Is(err, errRuntimeCredential) {
			t.Fatalf("Claude mode %q admitted: %v", mode, err)
		}
	}
}
