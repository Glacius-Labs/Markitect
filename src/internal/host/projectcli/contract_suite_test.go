package projectcli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

// These lists are deliberately independent of verbTable and schema generation.
// A surface change must make an explicit contract case addition or removal.
var contractCLI = []string{
	"init", "config", "onboard", "doctor", "mcp", "schema", "check", "model", "context", "impact", "docs", "edit",
	"explore", "ready", "brief", "plan", "run", "resume", "repair", "status", "verify", "apply", "deliver", "adopt", "version", "licenses", "help",
	"packet-export", "packet-list", "packet-submit",
}
var contractMCP = []string{
	"init", "config", "onboard", "doctor", "schema", "check", "model", "context", "impact", "docs", "edit",
	"explore", "ready", "brief", "plan", "run", "resume", "repair", "status", "verify", "apply", "deliver", "adopt",
	"packet-export", "packet-list", "packet-submit",
}
var contractReadOnlyMCP = []string{
	"init", "config", "onboard", "doctor", "schema", "check", "model", "context", "impact", "docs", "edit",
	"explore", "ready", "brief", "plan", "status", "apply", "adopt", "packet-list",
}

func interfaceContractCases(t *testing.T) map[string]contractCase {
	t.Helper()
	cases := map[string]contractCase{}
	for _, group := range []map[string]contractCase{inspectionContractCases(), adoptionContractCases(), runtimeContractCases(), externalContractCases(), informationContractCases()} {
		for name, entry := range group {
			if _, duplicate := cases[name]; duplicate {
				t.Fatalf("duplicate contract case %s", name)
			}
			cases[name] = entry
		}
	}
	return cases
}

func TestContractCompleteness(t *testing.T) {
	actualCLI := []string{}
	for _, verb := range verbTable() {
		actualCLI = append(actualCLI, verb.name)
	}
	assertContractNames(t, "CLI", contractCLI, actualCLI)
	server, err := newMCPServer(env{root: testkit.TempDir(t), ops: projectapp.NewOperations()})
	if err != nil {
		t.Fatal(err)
	}
	_, tools := mcpSession(t, server)
	actualMCP := []string{}
	for _, tool := range tools {
		actualMCP = append(actualMCP, tool.Name)
		if strings.TrimSpace(tool.Description) == "" {
			t.Fatalf("%s has no published help", tool.Name)
		}
	}
	assertContractNames(t, "MCP tools/list", contractMCP, actualMCP)
	readOnly, err := newMCPServer(env{root: testkit.TempDir(t), ops: projectapp.NewOperations(), readOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	readOnlyNames := []string{}
	for _, tool := range readOnly.Tools() {
		readOnlyNames = append(readOnlyNames, tool.Name)
	}
	assertContractNames(t, "read-only MCP", contractReadOnlyMCP, readOnlyNames)
	cases := interfaceContractCases(t)
	assertContractNames(t, "behavioral cases", contractCLI, sortedKeys(cases))
	for name, entry := range cases {
		if entry.happy == nil || entry.noWrite == nil {
			t.Errorf("%s needs independent happy and no-write callbacks", name)
		}
	}
}

func assertContractNames(t *testing.T, surface string, expected, actual []string) {
	t.Helper()
	want, got := slices.Clone(expected), slices.Clone(actual)
	slices.Sort(want)
	slices.Sort(got)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s missing/stale/duplicate contracts: expected=%v actual=%v", surface, want, got)
	}
}

func TestInterfaceContractHelpAndBadArguments(t *testing.T) {
	root := testkit.TempDir(t)
	server, err := newMCPServer(env{root: root, ops: projectapp.NewOperations()})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range contractCLI {
		t.Run(name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			if code := Main([]string{"help", name}, &out, &stderr, "contract"); code != 0 || !strings.Contains(out.String(), name) || out.Len() < 30 {
				t.Fatalf("help code=%d stdout=%s stderr=%s", code, &out, &stderr)
			}
			out.Reset()
			stderr.Reset()
			before := contractInventory(t, root)
			if code := Main([]string{name, "--contract-unknown-flag"}, &out, &stderr, "contract"); code != 2 || stderr.Len() == 0 {
				t.Fatalf("bad CLI flag code=%d stdout=%s stderr=%s", code, &out, &stderr)
			}
			if slices.Contains(contractMCP, name) {
				_, err := server.Call(context.Background(), name, mustRaw(t, contractUnknownArguments(name)))
				if err == nil || !strings.Contains(err.Error(), "contractUnknownField is not a known field") {
					t.Fatalf("unknown MCP field was not rejected by the closed schema: %v", err)
				}
			}
			if !reflect.DeepEqual(before, contractInventory(t, root)) {
				t.Fatal("invalid arguments wrote project files")
			}
		})
	}
}

// Supply required envelope fields explicitly so rejection cannot pass merely
// because the request was missing an unrelated required field.
func contractUnknownArguments(name string) map[string]any {
	arguments := map[string]any{"contractUnknownField": true}
	fields := map[string]map[string]any{
		"init": {"name": "contract"}, "onboard": {"provider": "codex"}, "doctor": {"provider": "process"},
		"context": {"manager": "project-owner"}, "impact": {"since": "HEAD", "revision": "HEAD"},
		"edit":  {"input": projectwork.Mutation{APIVersion: projectwork.APIVersion, BaseDigest: "fixture", Actor: "user", Goal: "contract", Files: []projectwork.FileChange{}}},
		"ready": {"exploration": "contract", "scope": "contract"}, "plan": {"goal": "contract"},
		"run": {"plan": "contract"}, "resume": {"run": "contract"}, "repair": {"run": "contract"},
		"packet-export": {"run": "contract"}, "packet-list": {"run": "contract"},
		"packet-submit": {"run": "contract", "packet": "packet-contract", "expect": "sha256:contract", "response": map[string]any{"contract": true}, "execute": true},
		"apply":         {"plan": "contract", "run": "contract", "candidate": "contract"},
		"deliver":       {"exploration": "contract", "scope": "contract"}, "adopt": {"action": "status"},
	}
	for key, value := range fields[name] {
		arguments[key] = value
	}
	return arguments
}

func TestInterfaceContracts(t *testing.T) {
	cases := interfaceContractCases(t)
	for _, name := range contractCLI {
		t.Run(name, func(t *testing.T) {
			entry, exists := cases[name]
			if !exists || entry.happy == nil || entry.noWrite == nil {
				t.Fatal("missing happy/no-write contract")
			}
			surfaces := []string{"cli"}
			if slices.Contains(contractMCP, name) {
				surfaces = append(surfaces, "mcp")
			}
			for _, surface := range surfaces {
				t.Run(surface, func(t *testing.T) {
					t.Run("happy", func(t *testing.T) { entry.happy(t, surface) })
					t.Run("no-write", func(t *testing.T) { entry.noWrite(t, surface) })
				})
			}
		})
	}
}

func informationContractCases() map[string]contractCase {
	cases := map[string]contractCase{}
	for _, name := range []string{"version", "licenses", "help"} {
		invoke := func(t *testing.T, _ string) {
			root := testkit.TempDir(t)
			before := contractInventory(t, root)
			output := contractCLIProcess(t, root, "", name)
			needle := map[string]string{"version": "Markitect contract", "licenses": "Copyright", "help": "deliver"}[name]
			if !strings.Contains(string(output), needle) {
				t.Fatalf("%s missing %q: %s", name, needle, output)
			}
			if !reflect.DeepEqual(before, contractInventory(t, root)) {
				t.Fatal("information command wrote its working directory")
			}
		}
		cases[name] = contractCase{happy: invoke, noWrite: invoke}
	}
	invokeMCP := func(t *testing.T, _ string) { contractMCPProcess(t) }
	cases["mcp"] = contractCase{happy: invokeMCP, noWrite: invokeMCP}
	return cases
}

// A child process exercises Main's actual mcp stdio dispatch without changing
// global stdin or environment in the test process.
func TestContractMCPProcess(t *testing.T) {
	for index, arg := range os.Args {
		if arg == "contract-mcp-child" {
			os.Exit(Main(os.Args[index+1:], os.Stdout, os.Stderr, "contract"))
		}
	}
}

func contractMCPProcess(t *testing.T) {
	t.Helper()
	for _, readOnly := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "read-only"}[readOnly], func(t *testing.T) { contractMCPMode(t, readOnly) })
	}
}

func contractMCPMode(t *testing.T, readOnly bool) {
	t.Helper()
	root := testkit.TempDir(t)
	before := contractInventory(t, root)
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` + mcp.ProtocolVersion + `","capabilities":{},"clientInfo":{"name":"contract","version":"1"}}}` + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"
	args, expected := []string{"mcp", "--repo", root}, contractMCP
	if readOnly {
		args, expected = append(args, "--read-only"), contractReadOnlyMCP
	}
	output := contractCLIProcess(t, root, input, args...)
	names := []string{}
	initialized := false
	for _, line := range bytes.Split(bytes.TrimSpace(output), []byte("\n")) {
		var response struct {
			ID     int `json:"id"`
			Result struct {
				ProtocolVersion string     `json:"protocolVersion"`
				Tools           []mcp.Tool `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(line, &response); err != nil {
			t.Fatal(err)
		}
		if response.ID == 1 {
			initialized = response.Result.ProtocolVersion == mcp.ProtocolVersion
		}
		if response.ID == 2 {
			for _, tool := range response.Result.Tools {
				names = append(names, tool.Name)
			}
		}
	}
	if !initialized {
		t.Fatalf("missing initialize protocol response: %s", output)
	}
	assertContractNames(t, "actual CLI mcp stdio", expected, names)
	if !reflect.DeepEqual(before, contractInventory(t, root)) {
		t.Fatal("mcp stdio wrote project files")
	}
}

func contractCLIProcess(t *testing.T, root, input string, args ...string) []byte {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, append([]string{"-test.run=^TestContractMCPProcess$", "--", "contract-mcp-child"}, args...)...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("CLI %v: %v stderr=%s", args, err, &stderr)
	}
	return output
}
