package projectcli

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
)

// adopt resolve takes only human choices. The closed input schema rejects
// caller-supplied binding fields and ambiguous JSON before any session is
// read; the Host derives every binding (see TestAdoptApplyAppliesModelAndRecordsTrustedReceipt).
func TestAdoptResolveChoicesAreClosedAndExplicit(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "--initial-branch=feature-resolve")
	valid := writeDraft(t, repo, ".markitect/drafts/choices.json", []byte(`{"resolve":{"iterationId":"root-pass","choices":{"actor":"user","authorityClaim":"Owner supplied this decision","decisionReference":"decision-1","questions":[],"scopes":[{"scopeId":"orders","status":"defer","reason":"Not in scope"}]}}}`))
	fields, err := parseFields(t, "adopt", "resolve", "--repo", repo, "--session", "s", "--input", valid)
	if err != nil {
		t.Fatal(err)
	}
	in, err := mcp.DecodeArguments[adoptInput](mustRaw(t, fields))
	if err != nil || in.Input == nil || in.Input.Resolve == nil {
		t.Fatalf("decode choices = %+v, err=%v", in.Input, err)
	}
	choices := in.Input.Resolve.Choices
	if choices.Actor != "user" || choices.Questions == nil || len(choices.Questions) != 0 || len(choices.Scopes) != 1 || in.Input.Resolve.IterationID != "root-pass" {
		t.Fatalf("decoded choices = %+v", in.Input.Resolve)
	}
	for name, choices := range map[string]string{
		"authenticated is host-derived": `{"actor":"user","authorityClaim":"x","decisionReference":"d","authenticated":false,"questions":[],"scopes":[]}`,
		"target basis is host-derived":  `{"actor":"user","authorityClaim":"x","decisionReference":"d","targetBasis":"caller-controlled","questions":[],"scopes":[]}`,
		"case variant field":            `{"actor":"user","Actor":"admin","authorityClaim":"x","decisionReference":"d","questions":[],"scopes":[]}`,
		"duplicate field":               `{"actor":"user","actor":"admin","authorityClaim":"x","decisionReference":"d","questions":[],"scopes":[]}`,
		"missing scopes":                `{"actor":"user","authorityClaim":"x","decisionReference":"d","questions":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := writeDraft(t, repo, ".markitect/drafts/bad-"+strings.ReplaceAll(name, " ", "-")+".json", []byte(`{"resolve":{"iterationId":"root-pass","choices":`+choices+`}}`))
			code, out, errout := runCLI(t, "adopt", "resolve", "--repo", repo, "--session", "missing-session", "--input", path)
			if code != 2 || out != "" || !strings.Contains(errout, "markitect adopt: ") || strings.Contains(errout, "missing-session") {
				t.Fatalf("accepted non-closed or incomplete choices: exit=%d stdout=%q stderr=%q", code, out, errout)
			}
		})
	}
}
