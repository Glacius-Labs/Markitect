package projectcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

// parseFields parses CLI arguments into the verb's JSON input fields, exactly
// as the CLI hands them to the shared handler.
func parseFields(t *testing.T, args ...string) (map[string]any, error) {
	t.Helper()
	v, ok := lookupVerb(args[0])
	if !ok {
		t.Fatalf("unknown verb %q", args[0])
	}
	inv, help, err := parseInvocation(v, args[1:])
	if err != nil {
		return nil, err
	}
	if help {
		t.Fatalf("%v parsed as a help request", args)
	}
	fields := map[string]any{}
	if err := json.Unmarshal(inv.raw, &fields); err != nil {
		t.Fatal(err)
	}
	return fields, nil
}

func TestParseVerbsOperandsAndClosedFlags(t *testing.T) {
	revision := strings.Repeat("a", 40)
	accepted := []struct {
		name string
		args []string
		want map[string]any
	}{
		{"check reads the working tree", []string{"check"}, map[string]any{}},
		{"context operand", []string{"context", "engineering/Manager/root"}, map[string]any{"manager": "engineering/Manager/root"}},
		{"flags around an operand", []string{"context", "--revision", revision, "engineering/Manager/root"}, map[string]any{"manager": "engineering/Manager/root", "revision": revision}},
		{"repeated manager", []string{"plan", "--goal", "Cancel order", "--manager", "sales/Manager/orders", "--manager", "sales/Manager/inventory"}, map[string]any{"goal": "Cancel order", "manager": []any{"sales/Manager/orders", "sales/Manager/inventory"}}},
		{"plan since baseline", []string{"plan", "--goal", "Cancel", "--since", revision}, map[string]any{"goal": "Cancel", "since": revision}},
		{"impact since", []string{"impact", "--since", revision, "--revision", "HEAD"}, map[string]any{"since": revision, "revision": "HEAD"}},
		{"run plan operand", []string{"run", "p1", "--execute"}, map[string]any{"plan": "p1", "execute": true}},
		{"resume run operand", []string{"resume", "r1", "--execute"}, map[string]any{"run": "r1", "execute": true}},
		{"repair run operand", []string{"repair", "--execute", "r1"}, map[string]any{"run": "r1", "execute": true}},
		{"status overview", []string{"status"}, map[string]any{}},
		{"status run", []string{"status", "r1"}, map[string]any{"run": "r1"}},
		{"verify run", []string{"verify", "r1", "--execute"}, map[string]any{"run": "r1", "execute": true}},
		{"verify revision", []string{"verify", "--revision", revision, "--execute", "--write"}, map[string]any{"revision": revision, "execute": true, "write": true}},
		{"apply preflight", []string{"apply", "--plan", "p1", "--run", "r1", "--candidate", "c1"}, map[string]any{"plan": "p1", "run": "r1", "candidate": "c1"}},
		{"apply write", []string{"apply", "--plan", "p1", "--run", "r1", "--candidate", "c1", "--branch", "feature/x", "--head", "abc", "--worktree", "sha256:tree", "--expect", "sha256:verify", "--write"},
			map[string]any{"plan": "p1", "run": "r1", "candidate": "c1", "branch": "feature/x", "head": "abc", "worktree": "sha256:tree", "expect": "sha256:verify", "write": true}},
		{"brief default action", []string{"brief", "--since", revision, "--revision", "HEAD", "--provenance", "ADR-1"}, map[string]any{"action": "create", "since": revision, "revision": "HEAD", "provenance": "ADR-1"}},
		{"brief list", []string{"brief", "list", "--manager", "m"}, map[string]any{"action": "list", "manager": "m"}},
		{"brief dismiss", []string{"brief", "--event", "e", "dismiss", "--manager", "m", "--expect", "d", "--write"}, map[string]any{"action": "dismiss", "event": "e", "manager": "m", "expect": "d", "write": true}},
		{"adopt stage", []string{"adopt", "status", "--session", "s"}, map[string]any{"action": "status", "session": "s"}},
		{"ready acknowledge", []string{"ready", "--exploration", "e", "--scope", "s", "--acknowledge", "--actor", "a", "--authority", "b", "--decision-ref", "c", "--acknowledged-at", "2026-10-09T12:00:00Z"},
			map[string]any{"exploration": "e", "scope": "s", "acknowledge": true, "actor": "a", "authority": "b", "decisionRef": "c", "acknowledgedAt": "2026-10-09T12:00:00Z"}},
		{"config native profile", []string{"config", "--provider", "codex", "--model", "gpt-6-luna", "--effort", "high", "--codex-profile", "explicit-profile", "--input-micros-per-million", "5", "--output-micros-per-million", "10", "--max-cost-micros", "1000"},
			map[string]any{"provider": "codex", "model": "gpt-6-luna", "effort": "high", "codexProfile": "explicit-profile", "inputMicrosPerMillion": float64(5), "outputMicrosPerMillion": float64(10), "maxCostMicros": float64(1000)}},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			fields, err := parseFields(t, tc.args...)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !reflect.DeepEqual(fields, tc.want) {
				t.Fatalf("fields = %#v, want %#v", fields, tc.want)
			}
		})
	}

	rejected := []struct {
		name string
		args []string
		fail string
	}{
		{"unknown flag", []string{"check", "--write"}, "flag provided but not defined: -write"},
		{"flag of another verb", []string{"impact", "--goal", "ship"}, "flag provided but not defined: -goal"},
		{"base is now since", []string{"impact", "--base", "abc", "--revision", "HEAD"}, "flag provided but not defined: -base"},
		{"positional", []string{"check", "extra"}, `unexpected argument "extra"`},
		{"missing revision", []string{"impact", "--since", "abc"}, "requires --revision"},
		{"missing manager operand", []string{"context"}, "requires the MANAGER operand"},
		{"missing plan operand", []string{"run", "--execute"}, "requires the PLAN operand"},
		{"no flag alias for an operand", []string{"run", "--plan", "p1", "--execute"}, "flag provided but not defined: -plan"},
		{"missing run operand", []string{"repair", "--execute"}, "requires the RUN operand"},
		{"removed execution mode flag", []string{"config", "--provider", "codex", "--model", "gpt-6-luna", "--execution-mode", "proposal-only"}, "flag provided but not defined: -execution-mode"},
		{"acknowledge-structure is now acknowledge", []string{"ready", "--exploration", "e", "--scope", "s", "--acknowledge-structure"}, "flag provided but not defined: -acknowledge-structure"},
		{"adopt needs a stage", []string{"adopt", "--session", "s"}, "requires a stage"},
		{"apply-adoption is now apply", []string{"adopt", "apply-adoption", "--session", "s"}, `unknown stage "apply-adoption"`},
		{"resume is now status", []string{"adopt", "resume", "--session", "s"}, `unknown stage "resume"`},
		{"brief unknown action", []string{"brief", "briefings"}, `unknown action "briefings"`},
		{"integer flag", []string{"config", "--max-cost-micros", "ten"}, "--max-cost-micros must be an integer"},
		{"empty repeated value", []string{"plan", "--goal", "g", "--manager", ""}, "value must not be empty"},
		{"single-valued flag twice", []string{"init", "--name", "a", "--name", "b"}, "may be given only once"},
		{"record path outside drafts", []string{"edit", "--input", "proposal.json"}, "Markitect records may be read or written only under .markitect/drafts/ or .markitect/runs/"},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseFields(t, tc.args...); err == nil || !strings.Contains(err.Error(), tc.fail) {
				t.Fatalf("parse error = %v, want containing %q", err, tc.fail)
			}
		})
	}
}

func TestRepoDefaultsToTheCurrentDirectory(t *testing.T) {
	repo := copyProjectWorld(t)
	t.Chdir(repo)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	inv, _, err := parseInvocation(checkVerb, nil)
	if err != nil || inv.env.root != cwd || inv.env.sourceRoot != cwd {
		t.Fatalf("default root = %q source %q, want %q (err=%v)", inv.env.root, inv.env.sourceRoot, cwd, err)
	}
	other := t.TempDir()
	inv, _, err = parseInvocation(adoptVerb, []string{"status", "--repo", ".", "--source-repo", other, "--session", "s"})
	if err != nil || inv.env.root != cwd || inv.env.sourceRoot != other {
		t.Fatalf("adopt roots = %q source %q (err=%v)", inv.env.root, inv.env.sourceRoot, err)
	}
	inv, _, err = parseInvocation(mcpVerb, []string{"--repo", other, "--read-only"})
	if err != nil || inv.env.root != other || !inv.env.readOnly {
		t.Fatalf("mcp composition = %+v err=%v", inv.env, err)
	}
	if report := decodeOutput[checkReport](t, mustCLI(t, "check")); report.Status != "succeeded" || !report.Provisional || report.ProjectDigest == "" {
		t.Fatalf("check without --repo = %+v", report)
	}
}

func TestMainUsageHelpAndInformationVerbs(t *testing.T) {
	code, out, errout := runCLI(t)
	if code != 2 || out != "" || !strings.Contains(errout, "Usage: markitect <verb>") {
		t.Fatalf("no arguments exit=%d stdout=%q stderr=%q", code, out, errout)
	}
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		code, out, _ := runCLI(t, args...)
		if code != 0 || !strings.HasPrefix(out, "Usage: markitect <verb>") {
			t.Fatalf("%v exit=%d output=%q", args, code, out)
		}
		for _, v := range verbTable() {
			if !strings.Contains(out, "\n"+v.group+"\n") || !strings.Contains(out, "  "+v.name+" ") || !strings.Contains(out, v.summary) {
				t.Fatalf("%v omitted verb %s in group %s:\n%s", args, v.name, v.group, out)
			}
		}
	}
	_, planHelp, _ := runCLI(t, "help", "plan")
	if !strings.Contains(planHelp, "Usage: markitect "+planVerb.synopsis) || !strings.Contains(planHelp, "--exploration ID") || !strings.Contains(planHelp, "--scope ID") || !strings.Contains(planHelp, "Effect: write. MCP tool: plan.") {
		t.Fatalf("help plan = %s", planHelp)
	}
	for _, args := range [][]string{{"plan", "--help"}, {"plan", "-h"}, {"plan", "--goal", "g", "--help"}} {
		if code, out, _ := runCLI(t, args...); code != 0 || out != planHelp {
			t.Fatalf("%v exit=%d output differs from help plan:\n%s", args, code, out)
		}
	}
	if _, out, _ := runCLI(t, "help", "mcp"); !strings.Contains(out, "MCP tool: no (CLI only)") || !strings.Contains(out, "--read-only") {
		t.Fatalf("help mcp = %s", out)
	}
	if _, out, _ := runCLI(t, "adopt", "--help"); !strings.Contains(out, "Stages:") || !strings.Contains(out, "  apply ") || !strings.Contains(out, "  status ") {
		t.Fatalf("adopt --help = %s", out)
	}
	if _, out, _ := runCLI(t, "help", "brief"); !strings.Contains(out, "Actions:") || !strings.Contains(out, "  dismiss ") {
		t.Fatalf("help brief = %s", out)
	}
	if code, out, _ := runCLI(t, "version"); code != 0 || !strings.HasPrefix(out, "Markitect test (") {
		t.Fatalf("version exit=%d output=%q", code, out)
	}
	if code, out, _ := runCLI(t, "licenses"); code != 0 || out == "" {
		t.Fatalf("licenses exit=%d", code)
	}
	for _, tc := range []struct {
		args []string
		fail string
	}{
		{[]string{"migrate"}, `markitect: unknown verb "migrate"`},
		{[]string{"project", "check"}, `markitect: unknown verb "project"`},
		{[]string{"project", "--help"}, `markitect: unknown verb "project"`},
		{[]string{"help", "migrate"}, `markitect help: unknown verb "migrate"`},
		{[]string{"help", "plan", "run"}, "markitect help: accepts at most one verb"},
		{[]string{"version", "extra"}, "markitect version: accepts no arguments"},
	} {
		if code, out, errout := runCLI(t, tc.args...); code != 2 || out != "" || !strings.Contains(errout, tc.fail) {
			t.Fatalf("%v exit=%d stdout=%q stderr=%q, want %q", tc.args, code, out, errout, tc.fail)
		}
	}
}

// The effect rules reject an invocation before the verb reads or writes
// project state, with exit code 2 and the verb named on stderr.
func TestEffectRulesRejectBeforeAnyOperation(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "--initial-branch=feature-effects")
	editDraft := writeDraft(t, repo, ".markitect/drafts/edit.json", projectwork.Mutation{APIVersion: projectwork.APIVersion, BaseDigest: "sha256:base", Actor: projectwork.HumanActor, Goal: "g", Files: []projectwork.FileChange{}})
	exploreDraft := writeDraft(t, repo, ".markitect/drafts/explore.json", projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "x", Status: projectexplore.StatusActive, Request: "r",
		Scopes: []projectexplore.Scope{}, Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	})
	runDraft := writeDraft(t, repo, ".markitect/drafts/run.json", []byte(`{"run":{"iterationId":"i","phase":"propose","agentManagerId":"m"}}`))
	// Revisions are resolved before a verb runs, so these cases need a real commit.
	runGitWithEnv(t, repo, testCommitEnv, "commit", "--allow-empty", "-m", "base")
	revision := gitOutput(t, repo, "rev-parse", "HEAD")
	profile := []string{"--provider", "codex", "--model", "gpt-6-luna", "--input-micros-per-million", "5", "--output-micros-per-million", "10", "--max-cost-micros", "1000"}
	applyIDs := []string{"--plan", "p1", "--run", "r1", "--candidate", "c1"}
	ready := []string{"--exploration", "e", "--scope", "s"}
	cases := []struct {
		name string
		args []string
		fail string
	}{
		{"run needs execute", []string{"run", "p1"}, "run starts agents or configured checks and requires --execute; inspect first with `markitect status`"},
		{"resume needs execute", []string{"resume", "r1"}, "requires --execute"},
		{"repair needs execute", []string{"repair", "r1"}, "requires --execute; inspect first with `markitect status`"},
		{"verify run needs execute", []string{"verify", "r1"}, "requires --execute"},
		{"verify revision needs execute", []string{"verify", "--revision", revision}, "requires --execute"},
		{"verify run is not persisted with write", []string{"verify", "r1", "--execute", "--write"}, "--write applies only to --revision"},
		{"verify one target", []string{"verify", "r1", "--revision", revision, "--execute"}, "requires exactly one of RUN or --revision"},
		{"deliver needs execute", []string{"deliver", "--exploration", "e", "--scope", "s", "--write"}, "requires --execute; inspect first with `markitect ready`"},
		{"deliver needs write", []string{"deliver", "--exploration", "e", "--scope", "s", "--execute"}, "requires --write as well as --execute"},
		{"init write needs expect", []string{"init", "--name", "x", "--write"}, "--write requires --expect"},
		{"expect needs write", []string{"init", "--name", "x", "--expect", "d"}, "--expect is valid only together with --write or --execute"},
		{"docs write needs expect", []string{"docs", "--write"}, "--write requires --expect"},
		{"docs write uses the working tree", []string{"docs", "--revision", revision, "--expect", "d", "--write"}, "--write uses the working tree; omit --revision"},
		{"edit write needs expect", []string{"edit", "--input", editDraft, "--write"}, "--write requires --expect"},
		{"config write needs expect", append(append([]string{"config"}, profile...), "--write"), "--write requires --expect"},
		{"plan write needs expect", []string{"plan", "--goal", "g", "--write"}, "--write requires --expect"},
		{"plan expect needs write", []string{"plan", "--goal", "g", "--expect", "d"}, "--expect is valid only together with --write or --execute"},
		{"apply write needs verification digest", append(append([]string{"apply"}, applyIDs...), "--branch", "feature/x", "--head", "a", "--worktree", "b", "--write"), "--write requires --expect"},
		{"apply write needs preflight", append(append([]string{"apply"}, applyIDs...), "--expect", "verify", "--write"), "--write requires --branch, --head and --worktree from the preflight"},
		{"explore one source", []string{"explore", "--exploration", "x", "--input", exploreDraft}, "use either --exploration or --input"},
		{"ready actor needs acknowledge", append(append([]string{"ready"}, ready...), "--actor", "a"), "need --acknowledge"},
		{"ready acknowledge needs authority", append(append([]string{"ready"}, ready...), "--acknowledge", "--actor", "a"), "--acknowledge requires --actor, --authority and --decision-ref"},
		{"ready acknowledge needs time", append(append([]string{"ready"}, ready...), "--acknowledge", "--actor", "a", "--authority", "b", "--decision-ref", "c", "--acknowledged-at", "yesterday"), "explicit RFC 3339 timestamp"},
		{"onboard provider", []string{"onboard", "--provider", "gemini"}, "--provider must be codex, claude or both"},
		{"adopt execute only for run", []string{"adopt", "status", "--session", "s", "--execute"}, "--execute applies only to adopt run --write"},
		{"adopt run preview refuses execute", []string{"adopt", "run", "--session", "s", "--input", runDraft, "--execute"}, "--execute applies only to adopt run --write"},
		{"adopt run needs its record", []string{"adopt", "run", "--session", "s"}, `adopt run requires --input with a "run" record`},
		{"adopt run write needs execute", []string{"adopt", "run", "--session", "s", "--input", runDraft, "--expect", "d", "--write"}, "adopt starts agents or configured checks and requires --execute; inspect first with `markitect adopt status`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errout := runCLI(t, append(append([]string{}, tc.args...), "--repo", repo)...)
			if code != 2 || out != "" || !strings.Contains(errout, "markitect "+tc.args[0]+": ") || !strings.Contains(errout, tc.fail) {
				t.Fatalf("exit=%d stdout=%q stderr=%q, want 2 and %q", code, out, errout, tc.fail)
			}
		})
	}
	entries, err := os.ReadDir(repo)
	if err != nil || len(entries) != 2 {
		t.Fatalf("rejected invocations wrote repository files: %v %v", entries, err)
	}
	if entries, err := os.ReadDir(filepath.Join(repo, ".markitect")); err != nil || len(entries) != 1 || entries[0].Name() != "drafts" {
		t.Fatalf("rejected invocations wrote control-plane state: %v %v", entries, err)
	}
}

func TestWriteIsAnExplicitOperationFlag(t *testing.T) {
	if _, err := parseFields(t, "check", "--write"); err == nil {
		t.Fatal("check accepted --write")
	}
	if fields, err := parseFields(t, "init", "--name", "shop", "--expect", "sha256:init", "--write"); err != nil || fields["write"] != true || fields["expect"] != "sha256:init" {
		t.Fatalf("explicit init write was rejected: %#v, %v", fields, err)
	}
	if fields, err := parseFields(t, "init", "--name", "shop"); err != nil || fields["write"] != nil {
		t.Fatalf("init without --write = %#v, %v", fields, err)
	}
	repo := t.TempDir()
	runGit(t, repo, "init", "--initial-branch=feature-write")
	input := writeDraft(t, repo, ".markitect/drafts/apply.json", []byte(`{"apply":{"iterationId":"root-pass","expectedPlanDigest":"sha256:plan"}}`))
	fields, err := parseFields(t, "adopt", "apply", "--repo", repo, "--session", "s", "--input", input, "--expect", "sha256:session", "--write")
	if err != nil || fields["write"] != true || fields["expect"] != "sha256:session" || fields["action"] != "apply" {
		t.Fatalf("exact adoption apply was rejected: %#v, %v", fields, err)
	}
	in, err := mcp.DecodeArguments[adoptInput](mustRaw(t, fields))
	if err != nil || in.Input == nil || in.Input.Apply == nil || in.Input.Apply.ExpectedPlanDigest != "sha256:plan" {
		t.Fatalf("adopt apply input did not decode under its stage key: %+v %v", in.Input, err)
	}
}

func mustRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestConfigNativeProfileFlags(t *testing.T) {
	fields, err := parseFields(t, "config", "--provider", "codex",
		"--model", "gpt-6-luna", "--effort", "high", "--codex-profile", "explicit-profile",
		"--windows-sandbox-backend", "mxc",
		"--input-micros-per-million", "5", "--output-micros-per-million", "10", "--max-cost-micros", "1000")
	if err != nil {
		t.Fatalf("parse config native options: %v", err)
	}
	in, err := mcp.DecodeArguments[configInput](mustRaw(t, fields))
	if err != nil {
		t.Fatalf("decode config input: %v", err)
	}
	setup, err := in.options(false)
	if err != nil {
		t.Fatalf("map CLI config options: %v", err)
	}
	if setup.CodexProfile != "explicit-profile" || setup.WindowsSandboxBackend != "mxc" || setup.InputMicrosPerMillion != 5 || setup.OutputMicrosPerMillion != 10 || setup.MaxCostMicros != 1000 {
		t.Fatalf("mapped config options = %+v", setup)
	}
	if fields, err = parseFields(t, "config", "--windows-sandbox-backend", "other", "--provider", "codex"); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if _, err := mcp.DecodeArguments[configInput](mustRaw(t, fields)); err == nil {
		t.Fatal("config accepted an unknown Windows sandbox backend")
	}
}

func TestConfigRatesUseBoundedCallerSuppliedEstimate(t *testing.T) {
	rates := func(input, output, maxCost int64) configInput {
		return configInput{Provider: "codex", Model: "gpt-6-luna", InputMicrosPerMillion: &input, OutputMicrosPerMillion: &output, MaxCostMicros: &maxCost}
	}
	options, err := rates(0, 125000, 2500000).options(false)
	if err != nil || options.InputMicrosPerMillion != 0 || options.OutputMicrosPerMillion != 125000 || options.MaxCostMicros != 2500000 {
		t.Fatalf("parsed rates = %+v, err=%v", options, err)
	}
	for _, values := range [][3]int64{{0, 0, 10}, {-1, 2, 10}, {1, 1, 0}, {1000000000001, 0, 10}, {1, 1, 1000000000001}} {
		if _, err := rates(values[0], values[1], values[2]).options(false); err == nil {
			t.Errorf("accepted out-of-contract caller values %v", values)
		}
	}
}
