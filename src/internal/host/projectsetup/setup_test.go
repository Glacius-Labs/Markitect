package projectsetup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestBuildRuntimeMapsAllManagersAndPinsTools(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: "root-manager", Namespace: ""},
		{ID: "orders-manager", Namespace: "commerce.orders", Parent: "root-manager"},
		{ID: "inventory-manager", Namespace: "commerce.inventory", Parent: "root-manager"},
	}}}
	options := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000}
	config, err := BuildRuntime(project, options, Discovery{Provider: "codex", ProviderBinary: provider})
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	if len(config.Agents) != 3 {
		t.Fatalf("agent mappings = %d, want every active manager", len(config.Agents))
	}
	if config.Review == nil || config.Review.MaxRounds != DefaultReviewMaxRounds || config.Review.MaxManagerRounds != DefaultReviewMaxManagerRounds {
		t.Fatalf("review defaults = %#v, want maxRounds=%d and maxManagerRounds=%d", config.Review, DefaultReviewMaxRounds, DefaultReviewMaxManagerRounds)
	}
	if len(config.Review.Agents) != len(config.Agents) {
		t.Fatalf("reviewer mappings = %d, want one for every Manager", len(config.Review.Agents))
	}
	for _, id := range []string{"root-manager", "orders-manager", "inventory-manager"} {
		agent, ok := config.Agents[id]
		if !ok {
			t.Fatalf("missing runtime mapping for %s", id)
		}
		if agent.Command != provider.Path || len(agent.Args) != 0 || agent.Transport != projectrun.TransportCodexAppServer || agent.Model != "gpt-6-luna" || agent.ProviderVersion != provider.Version {
			t.Fatalf("unexpected mapping for %s: %#v", id, agent)
		}
		if agent.WorkspaceMode != "git" || agent.AppServer == nil || agent.AppServer.ReasoningEffort != "high" || agent.AppServer.PermissionProfile != "" || !agent.AppServer.Helpers.Enabled {
			t.Fatalf("default setup must select the typed native Codex App Server worker: %#v", agent)
		}
		if agent.ModelOptions != nil || agent.AppServer.Helpers.MaxStartRequests != DefaultMaxHelperStarts || agent.AppServer.Helpers.MaxDepth != 1 || agent.AppServer.MaxEventBytes != DefaultMaxEventBytes {
			t.Fatalf("native settings must be explicit, bounded, and leave account permissions inherited: %#v", agent.AppServer)
		}
		if len(agent.RuntimeFiles) != 3 {
			t.Fatalf("runtime file pins for %s = %d, want Codex and both instructions", id, len(agent.RuntimeFiles))
		}
		reviewer, ok := config.Review.Agents[id]
		if !ok {
			t.Fatalf("missing reviewer mapping for %s", id)
		}
		if reviewer.WorkspaceMode != "git" || reviewer.Transport != projectrun.TransportCodexAppServer || len(reviewer.InstructionPaths) != len(agent.InstructionPaths) || len(reviewer.RuntimeFiles) != len(agent.RuntimeFiles) || reviewer.Command != provider.Path {
			t.Fatalf("reviewer config for %s must use a fresh native read-only workspace with pinned instructions: %#v", id, reviewer)
		}
		if reviewer.ModelOptions != nil || reviewer.AppServer == nil || reviewer.AppServer.PermissionProfile != "" || reviewer.Model != agent.Model || reviewer.ProviderVersion != agent.ProviderVersion {
			t.Fatalf("reviewer model profile for %s differs from worker profile", id)
		}
		if containsName(reviewer.Environment, "CODEX_HOME") {
			t.Fatal("runtime must use the existing OS-default Codex profile, not override CODEX_HOME")
		}
		if runtime.GOOS == "windows" {
			for _, name := range []string{"USERPROFILE", "APPDATA", "LOCALAPPDATA"} {
				if !containsName(reviewer.Environment, name) {
					t.Fatalf("native reviewer allowlist omits OS-default profile variable %s: %#v", name, reviewer.Environment)
				}
			}
		} else if !containsName(reviewer.Environment, "HOME") {
			t.Fatalf("native reviewer allowlist omits OS-default HOME: %#v", reviewer.Environment)
		}
	}
	if config.Mode != "controlled-local" || config.RequireIsolation || config.Limits.MaxRetries != 1 || config.Limits.MaxParallel != 2 || config.Limits.MaxStarts != DefaultMaxStarts || config.Limits.MaxDuration != projectrun.Duration(DefaultMaxRunTime) || config.Limits.MaxCostMicros != 5000 {
		t.Fatalf("unsafe or unexpected limits: %#v", config)
	}
	if config.Limits.MaxDuration <= 0 || config.Limits.MaxCostMicros <= 0 {
		t.Fatalf("runtime budget and deadline must remain finite and positive: %#v", config.Limits)
	}
	if err := projectrun.ValidateRuntime(config); err != nil {
		t.Fatalf("generated runtime is invalid: %v", err)
	}
}

func TestBuildRuntimeDefaultsCodexToNativeLunaHighAndBindsInstructions(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{
		Root:   root,
		Config: projectwork.Config{},
		Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}, {ID: "orders", Parent: "root"}}},
	}
	found := Discovery{Provider: "codex", ProviderBinary: provider}
	options := Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high",
		InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000,
	}
	config, err := BuildRuntime(project, options, found)
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	if len(config.Agents) != 2 || config.Review == nil {
		t.Fatalf("native setup did not map both roles for each Manager: %#v", config)
	}
	for _, id := range []string{"root", "orders"} {
		agent := config.Agents[id]
		if agent.WorkspaceMode != "git" {
			t.Fatalf("workspace mode for %s = %q", id, agent.WorkspaceMode)
		}
		wantPaths := []string{".agents/skills/markitect-implement/SKILL.md", "AGENTS.md"}
		if !reflect.DeepEqual(agent.InstructionPaths, wantPaths) {
			t.Fatalf("instruction paths for %s = %#v, want %#v", id, agent.InstructionPaths, wantPaths)
		}
		if agent.Transport != projectrun.TransportCodexAppServer || len(agent.Args) != 0 || agent.Command != provider.Path || agent.ModelOptions != nil {
			t.Fatalf("manager must use the selected native executable directly with no wrapper arguments/options: %#v", agent)
		}
		if len(agent.RuntimeFiles) != 3 {
			t.Fatalf("runtime pins for %s = %d, want Codex and two instructions: %#v", id, len(agent.RuntimeFiles), agent.RuntimeFiles)
		}
		if !hasRuntimePin(agent.RuntimeFiles, found.ProviderBinary) {
			t.Fatalf("native runtime does not pin selected Codex executable %s: %#v", provider.Path, agent.RuntimeFiles)
		}
		for _, path := range agent.InstructionPaths {
			absolute := filepath.Join(root, filepath.FromSlash(path))
			foundPin := false
			for _, pin := range agent.RuntimeFiles {
				if pin.Path == absolute && pin.Mode == "0644" && strings.HasPrefix(pin.Digest, "sha256:") {
					foundPin = true
				}
			}
			if !foundPin {
				t.Fatalf("instruction %s lacks its absolute mode/digest runtime pin: %#v", path, agent.RuntimeFiles)
			}
		}
		if agent.AppServer == nil || agent.AppServer.ReasoningEffort != "high" || agent.AppServer.PermissionProfile != "" {
			t.Fatalf("native model effort/inherited permissions = %#v", agent.AppServer)
		}
		reviewer := config.Review.Agents[id]
		if reviewer.WorkspaceMode != "git" || len(reviewer.RuntimeFiles) != 3 || !reflect.DeepEqual(reviewer.InstructionPaths, agent.InstructionPaths) {
			t.Fatalf("reviewer config should use its own read-only Git workspace with the same instruction pins: %#v", reviewer)
		}
	}
}

func TestBuildRuntimeDefaultNativeRequiresExistingCodexInstructions(t *testing.T) {
	root := t.TempDir()
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "codex", ProviderBinary: provider}
	_, err := BuildRuntime(project, Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high",
		InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaxCostMicros: 2,
	}, found)
	if err == nil || !strings.Contains(err.Error(), "existing generated Codex project instructions") {
		t.Fatalf("default native setup without instructions error = %v", err)
	}
}

func writeNativeInstructions(t *testing.T, root string) {
	t.Helper()
	for path, content := range map[string]string{
		"AGENTS.md": "# Project agent instructions\n",
		".agents/skills/markitect-implement/SKILL.md": "# Generated implementation skill\n",
		"CLAUDE.md": "# Claude guidance is not selected for Codex\n",
		".claude/skills/markitect-implement/SKILL.md": "# Claude skill is not selected for Codex\n",
	} {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func hasRuntimePin(files []agentexec.RuntimeFile, tool Tool) bool {
	for _, file := range files {
		if file.Path == tool.Path && file.Mode == tool.Mode && file.Digest == tool.Digest {
			return true
		}
	}
	return false
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if strings.EqualFold(name, want) {
			return true
		}
	}
	return false
}

func TestDiscoverProviderUsesCurrentWindowsCodexVendorPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows npm vendor layout")
	}
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	want := filepath.Join(appData, "npm", "node_modules", "@openai", "codex", "node_modules", "@openai", "codex-win32-x64", "vendor", "x86_64-pc-windows-msvc", "bin", "codex.exe")
	if err := os.MkdirAll(filepath.Dir(want), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("codex fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := discoverProvider("codex"); got != want {
		t.Fatalf("discovered Codex path = %q, want %q", got, want)
	}
}

func TestBuildRuntimeRejectsUnsupportedNativeProfiles(t *testing.T) {
	root := t.TempDir()
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "codex", ProviderBinary: provider}
	base := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", CodexProfile: "luna-high", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaxCostMicros: 2}
	cases := []struct {
		name   string
		change func(*Options, *Discovery)
	}{
		{"unsupported profile", func(o *Options, _ *Discovery) { o.CodexProfile = "other" }},
		{"different model", func(o *Options, _ *Discovery) { o.Model = "gpt-6-sol" }},
		{"different effort", func(o *Options, _ *Discovery) { o.Effort = "medium" }},
		{"different CLI version", func(_ *Options, d *Discovery) { d.ProviderBinary.Version = "0.161.0" }},
		{"default must not substitute model", func(o *Options, _ *Discovery) { o.CodexProfile = ""; o.Model = "gpt-6-sol" }},
		{"unsupported provider version", func(_ *Options, d *Discovery) { d.ProviderBinary.Version = "codex-cli 0.163.0" }},
		{"Claude native mode", func(o *Options, d *Discovery) { o.Provider = "claude"; d.Provider = "claude" }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			options := base
			discovery := found
			test.change(&options, &discovery)
			if _, err := BuildRuntime(project, options, discovery); err == nil {
				t.Fatal("unsupported native setup was accepted")
			}
		})
	}
}

func TestBuildRuntimeRejectsNonCodexSetup(t *testing.T) {
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "claude", ProviderBinary: Tool{Path: filepath.Join(t.TempDir(), "provider"), Version: "claude 2.1.295", Digest: "sha256:" + strings.Repeat("a", 64), Mode: "0644"}}
	if _, err := BuildRuntime(project, Options{Provider: "custom", Model: "m", InputMicrosPerMillion: 1, MaxCostMicros: 1}, found); err == nil {
		t.Fatal("unsupported provider was accepted")
	}
	if _, err := BuildRuntime(project, Options{Provider: "claude"}, found); err == nil || !strings.Contains(err.Error(), "native Codex App Server only") {
		t.Fatalf("Claude setup error = %v", err)
	}
}

func TestDiscoverRejectsCommandShimAndParsesOnlyVersion(t *testing.T) {
	if !isCommandShim(`C:\tools\codex.cmd`) && runtime.GOOS == "windows" {
		t.Fatal("Windows command shim was accepted")
	}
	if match := versionPattern.FindStringSubmatch("codex-cli 0.130.0\n"); len(match) != 2 || match[1] != "0.130.0" {
		t.Fatalf("version parse = %#v", match)
	}
	if _, err := Discover(Options{Provider: "claude"}); err == nil || !strings.Contains(err.Error(), "Codex App Server only") {
		t.Fatalf("unsupported provider discovery error = %v", err)
	}
}

func TestDiscoverInstalledCodexDoesNotRequireMarkitectCheckoutOrLegacyHelpers(t *testing.T) {
	installedDir := t.TempDir()
	providerName := "codex"
	if runtime.GOOS == "windows" {
		providerName += ".exe"
	}
	provider := testTool(t, installedDir, providerName, true)
	versionCalls := 0
	var probedPath string
	discovery, err := discoverWithVersion(Options{
		Provider: "codex", ProviderExecutable: provider.Path,
	}, func(path string) (string, error) {
		versionCalls++
		probedPath = path
		return "0.162.0", nil
	})
	if err != nil {
		t.Fatalf("installed binary discovery without a source checkout: %v", err)
	}
	if versionCalls != 1 || discovery.ProviderBinary.Path != probedPath || discovery.ProviderBinary.Version != "codex-cli 0.162.0" {
		t.Fatalf("installed native binary was not pinned by path and version: calls=%d discovery=%+v", versionCalls, discovery)
	}
	if discovery.Authentication != "not-verified" || !strings.Contains(discovery.AuthenticationNote, "does not read credentials") {
		t.Fatalf("discovery must leave existing sign-in untouched and unverified: %+v", discovery)
	}
	entries, err := os.ReadDir(installedDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != providerName {
		t.Fatalf("installed runtime discovery depended on source helpers: entries=%v", entries)
	}
}

func TestInspectFileSupportsLargeProviderWithoutBufferingAsset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current-provider.exe")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	const size = (256 << 20) + 1
	if err := file.Truncate(size); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	want := sha256.New()
	zeros := make([]byte, 32<<10)
	for remaining := int64(size); remaining > 0; {
		chunk := int64(len(zeros))
		if remaining < chunk {
			chunk = remaining
		}
		want.Write(zeros[:chunk])
		remaining -= chunk
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	tool, err := inspectFile(path)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("inspect current-size provider: %v", err)
	}
	if tool.Digest != "sha256:"+hex.EncodeToString(want.Sum(nil)) {
		t.Fatalf("streamed provider digest = %s", tool.Digest)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 16<<20 {
		t.Fatalf("fingerprinting buffered a large asset: allocated %d bytes", allocated)
	}
}

func TestInspectFileRetainsFiniteAssetSizeLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized-provider.exe")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(agentexec.MaxRuntimeAssetBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()
	if _, err := inspectFile(path); err == nil || !strings.Contains(err.Error(), "512 MiB") {
		t.Fatalf("oversized provider was not rejected: %v", err)
	}
}

func testTool(t *testing.T, root, name string, executable bool) Tool {
	t.Helper()
	path := filepath.Join(root, name)
	content := []byte("fingerprint fixture " + name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0600)
	if executable && runtime.GOOS != "windows" {
		mode = 0700
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		t.Fatal(err)
	}
	actualMode := "0644"
	if runtime.GOOS != "windows" {
		actualMode = modeOctal(info.Mode().Perm())
	}
	sum := sha256.Sum256(content)
	version := ""
	if strings.Contains(name, "codex") {
		version = "0.130.0"
	}
	return Tool{Path: resolved, Version: version, Digest: "sha256:" + hex.EncodeToString(sum[:]), Mode: actualMode}
}

func modeOctal(mode os.FileMode) string { return fmt.Sprintf("%04o", mode) }
