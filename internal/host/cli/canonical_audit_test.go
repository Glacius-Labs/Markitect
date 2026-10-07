package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalControllerAuditOptionBoundaries(t *testing.T) {
	const base = "0123456789abcdef0123456789abcdef01234567"
	const revision = "89abcdef0123456789abcdef0123456789abcdef"
	common := []string{
		"--action", "controller-audit",
		"--repo", ".",
		"--config", "canonical.yaml",
		"--runtime", "runtime.json",
		"--base", base,
		"--revision", revision,
	}
	parse := func(args ...string) (commandOptions, int) {
		t.Helper()
		options, code, done := parseOptions("canonical", append([]string{"canonical"}, args...), mustCommandFlags("canonical"), &bytes.Buffer{}, &bytes.Buffer{})
		if !done {
			return options, 0
		}
		return options, code
	}
	if options, code := parse(common...); code != 0 || options.action != "controller-audit" {
		t.Fatalf("valid audit options rejected: code=%d options=%#v", code, options)
	}
	for _, test := range []struct {
		name string
		args []string
	}{
		{"runtime required", removeArgPair(common, "--runtime")},
		{"base required", removeArgPair(common, "--base")},
		{"revision required", removeArgPair(common, "--revision")},
		{"config required", removeArgPair(common, "--config")},
		{"write forbidden", append(append([]string{}, common...), "--write")},
		{"plan forbidden", append(append([]string{}, common...), "--plan", "plan.json")},
		{"expect forbidden", append(append([]string{}, common...), "--expect", "sha256:reviewed")},
		{"report forbidden", append(append([]string{}, common...), "--report", "report.json")},
		{"evidence forbidden", append(append([]string{}, common...), "--evidence", "records.json")},
		{"selector forbidden", append(append([]string{}, common...), "--kind", "Projection")},
		{"explicit false write forbidden", append(append([]string{}, common...), "--write=false")},
		{"explicit empty plan forbidden", append(append([]string{}, common...), "--plan=")},
		{"explicit empty selector forbidden", append(append([]string{}, common...), "--namespace=")},
		{"short base forbidden", replaceArg(common, base, "abc123")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, code := parse(test.args...); code != 2 {
				t.Fatalf("invalid audit options returned code %d", code)
			}
		})
	}
}

func TestCanonicalControllerAuditDispatchPrecedesGenericSourceLoad(t *testing.T) {
	args := []string{
		"canonical", "--action", "controller-audit",
		"--repo", filepath.Join(t.TempDir(), "missing-repository"),
		"--config", "canonical.yaml",
		"--runtime", filepath.Join(t.TempDir(), "missing-runtime.json"),
		"--base", strings.Repeat("a", 40),
		"--revision", strings.Repeat("b", 40),
	}
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code != 2 {
		t.Fatalf("audit with missing runtime exit=%d stdout=%s stderr=%s", code, out.String(), errout.String())
	}
	if !strings.Contains(errout.String(), "read controller runtime configuration") {
		t.Fatalf("audit did not enter the controller path before loading its repository: stderr=%s", errout.String())
	}
}

func TestCanonicalControllerAuditStatusExitUsesIncompleteCode(t *testing.T) {
	if got := canonicalControllerAuditStatusExit("complete"); got != 0 {
		t.Fatalf("complete audit exit=%d, want 0", got)
	}
	if got := canonicalControllerAuditStatusExit("incomplete"); got != 2 {
		t.Fatalf("incomplete audit exit=%d, want 2", got)
	}
	if got := canonicalControllerAuditStatusExit("unexpected"); got == 0 {
		t.Fatalf("unknown audit status exit=%d, want a nonzero code", got)
	}
}

func removeArgPair(args []string, key string) []string {
	result := make([]string, 0, len(args)-2)
	for index := 0; index < len(args); index++ {
		if args[index] == key {
			index++
			continue
		}
		result = append(result, args[index])
	}
	return result
}
