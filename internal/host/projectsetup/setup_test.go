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

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestBuildRuntimeMapsAllManagersAndPinsTools(t *testing.T) {
	root := t.TempDir()
	python := executableTool(t)
	adapter := testTool(t, root, "runner.py", false)
	provider := testTool(t, root, "codex.exe", true)
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: "root-manager", Namespace: ""},
		{ID: "orders-manager", Namespace: "commerce.orders", Parent: "root-manager"},
		{ID: "inventory-manager", Namespace: "commerce.inventory", Parent: "root-manager"},
	}}}
	options := Options{Provider: "codex", Model: "example-model", Effort: "high", InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000}
	config, err := BuildRuntime(project, options, Discovery{Provider: "codex", ProviderBinary: provider, Python: python, Adapter: adapter})
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
		if agent.Command != python.Path || agent.Model != "example-model" || agent.ProviderVersion != provider.Version {
			t.Fatalf("unexpected mapping for %s: %#v", id, agent)
		}
		if agent.WorkspaceMode != "" || strings.Contains(strings.Join(agent.Args, " "), "native-work") {
			t.Fatalf("omitted execution mode must preserve proposal-only config: %#v", agent)
		}
		if agent.ModelOptions.(map[string]string)["model_reasoning_effort"] != "high" {
			t.Fatalf("Codex effort option missing for %s: %#v", id, agent.ModelOptions)
		}
		if len(agent.RuntimeFiles) != 3 {
			t.Fatalf("runtime file pins for %s = %d, want Python, adapter, provider", id, len(agent.RuntimeFiles))
		}
		reviewer, ok := config.Review.Agents[id]
		if !ok {
			t.Fatalf("missing reviewer mapping for %s", id)
		}
		if !reflect.DeepEqual(agent, reviewer) {
			t.Fatalf("reviewer config for %s does not match selected worker config: worker=%#v reviewer=%#v", id, agent, reviewer)
		}
		if len(reviewer.RuntimeFiles) != len(agent.RuntimeFiles) || !reflect.DeepEqual(reviewer.RuntimeFiles, agent.RuntimeFiles) {
			t.Fatalf("reviewer source pins for %s do not match worker pins: worker=%#v reviewer=%#v", id, agent.RuntimeFiles, reviewer.RuntimeFiles)
		}
		for _, file := range agent.Environment {
			if strings.EqualFold(file, "HOME") || strings.EqualFold(file, "CODEX_HOME") || strings.EqualFold(file, "USERPROFILE") || strings.EqualFold(file, "APPDATA") {
				t.Fatalf("runtime allowlist includes private account variable %q", file)
			}
		}
	}
	if config.Mode != "controlled-local" || config.RequireIsolation || config.Limits.MaxRetries != 1 || config.Limits.MaxParallel != 1 || config.Limits.MaxCostMicros != 5000 {
		t.Fatalf("unsafe or unexpected limits: %#v", config)
	}
	if config.Limits.MaxDuration <= 0 || config.Limits.MaxCostMicros <= 0 {
		t.Fatalf("runtime budget and deadline must remain finite and positive: %#v", config.Limits)
	}
	if err := projectrun.ValidateRuntime(config); err != nil {
		t.Fatalf("generated runtime is invalid: %v", err)
	}
}

func TestBuildRuntimeNativeCodexBindsHelperAndExistingCodexInstructions(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"AGENTS.md": "# Project agent instructions\n",
		".agents/skills/markitect-model-first/SKILL.md": "# Model-first project skill\n",
		"CLAUDE.md": "# Claude guidance is not selected for Codex\n",
		".claude/skills/markitect-model-first/SKILL.md": "# Claude skill is not selected for Codex\n",
	} {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "0.162.0"
	project := &projectwork.Project{
		Root:   root,
		Config: projectwork.Config{},
		Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}, {ID: "orders", Parent: "root"}}},
	}
	found := Discovery{
		Provider: "codex", ProviderBinary: provider, Python: executableTool(t),
		Adapter: testTool(t, root, "runner.py", false), NativeWork: testTool(t, root, "native_work.py", false),
	}
	options := Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high", ExecutionMode: "native-work", CodexProfile: "luna-high",
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
		if agent.WorkspaceMode != "scoped" {
			t.Fatalf("workspace mode for %s = %q", id, agent.WorkspaceMode)
		}
		wantPaths := []string{".agents/skills/markitect-model-first/SKILL.md", "AGENTS.md"}
		if !reflect.DeepEqual(agent.InstructionPaths, wantPaths) {
			t.Fatalf("instruction paths for %s = %#v, want %#v", id, agent.InstructionPaths, wantPaths)
		}
		joinedArgs := strings.Join(agent.Args, " ")
		for _, arg := range []string{"--execution-mode native-work", "--codex-profile luna-high", "--native-helper-limit 0"} {
			if !strings.Contains(joinedArgs, arg) {
				t.Fatalf("native adapter args for %s lack %q: %s", id, arg, joinedArgs)
			}
		}
		if len(agent.RuntimeFiles) != 6 {
			t.Fatalf("runtime pins for %s = %d, want Python, Codex, both adapter files, and two instructions: %#v", id, len(agent.RuntimeFiles), agent.RuntimeFiles)
		}
		for _, tool := range []Tool{found.Python, found.ProviderBinary, found.Adapter, found.NativeWork} {
			if !hasRuntimePin(agent.RuntimeFiles, tool) {
				t.Fatalf("native runtime does not pin selected tool %s: %#v", tool.Path, agent.RuntimeFiles)
			}
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
		if got := agent.ModelOptions.(map[string]string)["model_reasoning_effort"]; got != "high" {
			t.Fatalf("native model effort = %q", got)
		}
		for _, name := range []string{"CODEX_HOME"} {
			for _, allowed := range agent.Environment {
				if strings.EqualFold(allowed, name) {
					t.Fatalf("native environment allowlist includes nonstandard override %q", allowed)
				}
			}
		}
		if runtime.GOOS == "windows" {
			for _, name := range []string{"USERPROFILE", "APPDATA", "LOCALAPPDATA"} {
				if !containsName(agent.Environment, name) {
					t.Fatalf("native environment allowlist omits Windows home variable %s: %#v", name, agent.Environment)
				}
			}
		} else if !containsName(agent.Environment, "HOME") {
			t.Fatalf("native environment allowlist omits HOME: %#v", agent.Environment)
		}
		reviewer := config.Review.Agents[id]
		if reviewer.WorkspaceMode != "" || len(reviewer.RuntimeFiles) != 3 || strings.Contains(strings.Join(reviewer.Args, " "), "native-work") {
			t.Fatalf("reviewer config should retain proposal-only setup: %#v", reviewer)
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
	provider.Version = "0.162.0"
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "codex", ProviderBinary: provider, Python: executableTool(t), Adapter: testTool(t, root, "runner.py", false), NativeWork: testTool(t, root, "native_work.py", false)}
	base := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", ExecutionMode: "native-work", CodexProfile: "luna-high", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaxCostMicros: 2}
	cases := []struct {
		name   string
		change func(*Options, *Discovery)
	}{
		{"unsupported profile", func(o *Options, _ *Discovery) { o.CodexProfile = "other" }},
		{"different model", func(o *Options, _ *Discovery) { o.Model = "gpt-6-sol" }},
		{"different effort", func(o *Options, _ *Discovery) { o.Effort = "medium" }},
		{"different CLI version", func(_ *Options, d *Discovery) { d.ProviderBinary.Version = "0.161.0" }},
		{"missing helper source", func(_ *Options, d *Discovery) { d.NativeWork = Tool{} }},
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

func TestBuildRuntimeRejectsUnsupportedProviderAndMissingPrices(t *testing.T) {
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "claude", ProviderBinary: Tool{Path: filepath.Join(t.TempDir(), "provider"), Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Mode: "0644"}, Python: Tool{Path: filepath.Join(t.TempDir(), "python"), Version: "3.13.0", Digest: "sha256:" + strings.Repeat("b", 64), Mode: "0644"}, Adapter: Tool{Path: filepath.Join(t.TempDir(), "runner.py"), Digest: "sha256:" + strings.Repeat("c", 64), Mode: "0644"}}
	if _, err := BuildRuntime(project, Options{Provider: "custom", Model: "m", InputMicrosPerMillion: 1, MaxCostMicros: 1}, found); err == nil {
		t.Fatal("unsupported provider was accepted")
	}
	if _, err := BuildRuntime(project, Options{Provider: "claude", Model: "m", MaxCostMicros: 1}, found); err == nil {
		t.Fatal("missing token price rates were accepted")
	}
}

func TestBuildRuntimeClaudeUsesOnlyItsDeclaredEffortOption(t *testing.T) {
	root := t.TempDir()
	found := Discovery{Provider: "claude", ProviderBinary: testTool(t, root, "claude.exe", true), Python: executableTool(t), Adapter: testTool(t, root, "claude-runner.py", false)}
	found.ProviderBinary.Version = "2.1.0"
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	config, err := BuildRuntime(project, Options{Provider: "claude", Model: "model", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 2, MaxCostMicros: 10}, found)
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	agent := config.Agents["root"]
	if got := agent.ModelOptions.(map[string]string); len(got) != 1 || got["effort"] != "high" {
		t.Fatalf("Claude options = %#v", got)
	}
}

func TestDiscoverRejectsCommandShimAndParsesOnlyVersion(t *testing.T) {
	if !isCommandShim(`C:\tools\codex.cmd`) && runtime.GOOS == "windows" {
		t.Fatal("Windows command shim was accepted")
	}
	if match := versionPattern.FindStringSubmatch("codex-cli 0.130.0\n"); len(match) != 2 || match[1] != "0.130.0" {
		t.Fatalf("version parse = %#v", match)
	}
	if _, err := Discover(Options{Provider: "codex"}); err == nil || !strings.Contains(err.Error(), "--tool-root") {
		t.Fatalf("missing source root error = %v", err)
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
	if strings.Contains(name, "python") {
		version = "3.13.0"
	} else if strings.Contains(name, "codex") {
		version = "0.130.0"
	}
	return Tool{Path: resolved, Version: version, Digest: "sha256:" + hex.EncodeToString(sum[:]), Mode: actualMode}
}

func modeOctal(mode os.FileMode) string { return fmt.Sprintf("%04o", mode) }

func executableTool(t *testing.T) Tool {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mode := "0644"
	if runtime.GOOS != "windows" {
		mode = modeOctal(info.Mode().Perm())
	}
	sum := sha256.Sum256(content)
	return Tool{Path: path, Version: "3.13.0", Digest: "sha256:" + hex.EncodeToString(sum[:]), Mode: mode}
}
