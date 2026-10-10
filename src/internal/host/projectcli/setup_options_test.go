package projectcli

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
)

func TestSetupProcessExecutorFlagsAndCostModes(t *testing.T) {
	executor := filepath.Join(t.TempDir(), "executor.exe")
	parsed, _, err := parse([]string{"setup", "--repo", ".", "--provider", "process", "--model", "scripted", "--effort", "high",
		"--provider-executable", executor, "--provider-arg=--dir", "--provider-arg", "exchange", "--provider-version", "exchange/1",
		"--cost-mode", "unmetered", "--max-cost-micros", "1"}, io.Discard)
	if err != nil {
		t.Fatalf("parse process setup: %v", err)
	}
	options, err := setupOptions(parsed)
	if err != nil {
		t.Fatalf("map process setup: %v", err)
	}
	if options.Provider != projectsetup.ProviderProcess || !reflect.DeepEqual(options.ProviderArgs, []string{"--dir", "exchange"}) || options.ProviderVersion != "exchange/1" ||
		options.CostMode != projectrun.CostModeUnmetered || options.InputMicrosPerMillion != 0 || options.OutputMicrosPerMillion != 0 || options.MaxCostMicros != 1 {
		t.Fatalf("unexpected process setup options: %+v", options)
	}
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"unmetered with rates", []string{"--provider", "process", "--model", "m", "--cost-mode", "unmetered", "--input-micros-per-million", "1", "--max-cost-micros", "1"}, "omit the input/output rates"},
		{"metered without rates", []string{"--provider", "codex", "--model", "gpt-6-luna", "--max-cost-micros", "1"}, "requires --input-micros-per-million"},
		{"missing budget", []string{"--provider", "codex", "--model", "gpt-6-luna", "--input-micros-per-million", "1", "--output-micros-per-million", "1"}, "requires --max-cost-micros"},
		{"missing default profile", []string{"--max-cost-micros", "1"}, "requires --provider and --model"},
		{"input mixed with profile flags", []string{"--input", ".markitect/drafts/setup.json", "--model", "m"}, "do not combine it with profile flags"},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, _, err := parse(append([]string{"setup", "--repo", "."}, test.args...), io.Discard)
			if err == nil {
				_, err = setupOptions(parsed)
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("setup error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestSetupInputRecordCarriesRoleProfiles(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "setup-input")
	record := `{"provider":"codex","model":"gpt-6-luna","effort":"high","codexProfile":"","providerExecutable":"","inputMicrosPerMillion":1,"outputMicrosPerMillion":1,"maxCostMicros":10,` +
		`"roles":{"reviewer":{"model":"gpt-6-sol","effort":"medium"},"verifier":{"provider":"process","model":"scripted","providerExecutable":"C:/tools/executor.exe","costMode":"unmetered"}}}`
	path := filepath.Join(repo, ".markitect", "drafts", "setup.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	parsed, _, err := parse([]string{"setup", "--repo", repo, "--input", ".markitect/drafts/setup.json"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	options, err := setupOptions(parsed)
	if err != nil {
		t.Fatalf("read setup input: %v", err)
	}
	if options.Roles == nil || options.Roles.Reviewer == nil || options.Roles.Reviewer.Model != "gpt-6-sol" || options.Roles.Verifier == nil ||
		options.Roles.Verifier.Provider != projectsetup.ProviderProcess || options.Roles.Verifier.CostMode != projectrun.CostModeUnmetered || options.Roles.Manager != nil {
		t.Fatalf("role profiles were not decoded: %+v", options.Roles)
	}
	if err := os.WriteFile(path, []byte(`{"provider":"codex","unknown":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := setupOptions(parsed); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown setup input field was accepted: %v", err)
	}
}
