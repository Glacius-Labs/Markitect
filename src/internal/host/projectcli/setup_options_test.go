package projectcli

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
)

// configOptions maps `config` arguments to setup options as the handler does,
// without previewing or writing anything.
func configOptions(t *testing.T, args ...string) (projectsetup.Options, error) {
	t.Helper()
	fields, err := parseFields(t, append([]string{"config"}, args...)...)
	if err != nil {
		return projectsetup.Options{}, err
	}
	in, err := mcp.DecodeArguments[configInput](mustRaw(t, fields))
	if err != nil {
		return projectsetup.Options{}, err
	}
	return in.options(false)
}

func TestConfigProcessExecutorFlagsAndCostModes(t *testing.T) {
	executor := filepath.Join(t.TempDir(), "executor.exe")
	options, err := configOptions(t, "--provider", "process", "--model", "scripted", "--effort", "high",
		"--provider-executable", executor, "--provider-arg=--dir", "--provider-arg", "exchange", "--provider-version", "exchange/1",
		"--cost-mode", "unmetered", "--max-cost-micros", "1")
	if err != nil {
		t.Fatalf("map process config: %v", err)
	}
	if options.Provider != projectsetup.ProviderProcess || options.ProviderExecutable != executor || !reflect.DeepEqual(options.ProviderArgs, []string{"--dir", "exchange"}) || options.ProviderVersion != "exchange/1" ||
		options.CostMode != projectrun.CostModeUnmetered || options.InputMicrosPerMillion != 0 || options.OutputMicrosPerMillion != 0 || options.MaxCostMicros != 1 {
		t.Fatalf("unexpected process config options: %+v", options)
	}
	repo := t.TempDir()
	runGit(t, repo, "init", "--initial-branch=config-flags")
	input := writeDraft(t, repo, ".markitect/drafts/setup.json", []byte(`{"provider":"codex","model":"m","effort":"","codexProfile":"","providerExecutable":"","inputMicrosPerMillion":1,"outputMicrosPerMillion":1,"maxCostMicros":10}`))
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"unmetered with rates", []string{"--provider", "process", "--model", "m", "--cost-mode", "unmetered", "--input-micros-per-million", "1", "--max-cost-micros", "1"}, "omit the input and output rates"},
		{"metered without rates", []string{"--provider", "codex", "--model", "gpt-6-luna", "--max-cost-micros", "1"}, "metered agents require --input-micros-per-million and --output-micros-per-million"},
		{"missing budget", []string{"--provider", "codex", "--model", "gpt-6-luna", "--input-micros-per-million", "1", "--output-micros-per-million", "1"}, "requires --max-cost-micros"},
		{"missing default profile", []string{"--max-cost-micros", "1"}, "requires --provider and --model"},
		{"input mixed with profile flags", []string{"--repo", repo, "--input", input, "--model", "m"}, "do not combine it with profile flags"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := configOptions(t, test.args...); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("config error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestConfigInputRecordCarriesRoleProfiles(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "setup-input")
	record := `{"provider":"codex","model":"gpt-6-luna","effort":"high","codexProfile":"","providerExecutable":"","inputMicrosPerMillion":1,"outputMicrosPerMillion":1,"maxCostMicros":10,` +
		`"roles":{"reviewer":{"model":"gpt-6-sol","effort":"medium"},"verifier":{"provider":"process","model":"scripted","providerExecutable":"C:/tools/executor.exe","costMode":"unmetered"}}}`
	path := writeDraft(t, repo, ".markitect/drafts/setup.json", []byte(record))
	options, err := configOptions(t, "--repo", repo, "--input", path)
	if err != nil {
		t.Fatalf("read config input: %v", err)
	}
	if options.Roles == nil || options.Roles.Reviewer == nil || options.Roles.Reviewer.Model != "gpt-6-sol" || options.Roles.Verifier == nil ||
		options.Roles.Verifier.Provider != projectsetup.ProviderProcess || options.Roles.Verifier.CostMode != projectrun.CostModeUnmetered || options.Roles.Manager != nil {
		t.Fatalf("role profiles were not decoded: %+v", options.Roles)
	}
	in := configInput{Input: &options}
	if _, err := in.options(true); err == nil || !strings.Contains(err.Error(), "read-only MCP server does not accept provider executables") {
		t.Fatalf("read-only config accepted a role provider executable: %v", err)
	}
	writeDraft(t, repo, path, []byte(`{"provider":"codex","model":"m","effort":"","codexProfile":"","providerExecutable":"","inputMicrosPerMillion":1,"outputMicrosPerMillion":1,"maxCostMicros":10,"unknown":true}`))
	if _, err := configOptions(t, "--repo", repo, "--input", path); err == nil || !strings.Contains(err.Error(), "arguments do not match closed tool schema: input") {
		t.Fatalf("unknown config input field was accepted: %v", err)
	}
	writeDraft(t, repo, path, []byte(`{"provider":`))
	if _, err := configOptions(t, "--repo", repo, "--input", path); err == nil || !strings.Contains(err.Error(), "is not valid JSON") {
		t.Fatalf("invalid config input JSON was accepted: %v", err)
	}
}
