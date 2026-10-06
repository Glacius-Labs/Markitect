package agentrules

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

const testAPI = "guidance.example.org/v1"

func fixture() ([]core.Schema, []core.Definition, []core.DefinitionIdentity) {
	schema := core.Schema{
		APIVersion: testAPI,
		Purpose:    "A project-defined schema for bounded engineering guidance.",
		Kinds: map[string]core.Kind{
			"Rule": {
				Purpose: "A project-defined desired constraint.",
				Properties: map[string]core.Property{
					"statement": {Purpose: "The exact project-owned rule statement.", Type: core.TypeString, MinCount: 1, MaxCount: 1},
					"related":   {Purpose: "References one related project fact.", Type: core.TypeReference, MinCount: 0, MaxCount: 1, Target: &core.KindIdentity{APIVersion: testAPI, Kind: "Signal"}},
				},
			},
			"Signal": {
				Purpose: "An unfamiliar project-specific concept retained without interpretation.",
				Properties: map[string]core.Property{
					"payload": {Purpose: "Opaque project meaning carried as a string.", Type: core.TypeString, MinCount: 1, MaxCount: 1},
				},
			},
		},
	}
	rule := core.Definition{APIVersion: testAPI, Kind: "Rule", Metadata: core.Metadata{Namespace: "ops", Name: "canonical-rule"}, Purpose: "Preserve the reviewed deployment boundary.", Spec: map[string]any{"statement": "Deploy only from an approved candidate.", "related": map[string]any{"namespace": "ops", "name": "unknown-signal"}}, Source: core.Source{Path: "intent/rule.yaml", Digest: "sha256:rule-source", Line: 4}}
	signal := core.Definition{APIVersion: testAPI, Kind: "Signal", Metadata: core.Metadata{Namespace: "ops", Name: "unknown-signal"}, Purpose: "Retain this custom fact as supplied.", Spec: map[string]any{"payload": "custom value"}, Source: core.Source{Path: "intent/signal.yaml", Digest: "sha256:signal-source", Line: 8}}
	return []core.Schema{schema}, []core.Definition{rule, signal}, []core.DefinitionIdentity{rule.Identity(), signal.Identity()}
}

func inputFor(provider Provider) Input {
	schemas, definitions, ids := fixture()
	filename := "AGENTS.md"
	if provider == Claude {
		filename = "CLAUDE.md"
	}
	prefix := `C:\repo\project`
	return Input{
		Provider:      provider,
		Schemas:       schemas,
		Definitions:   definitions,
		Policies:      []ProjectionPolicy{{ID: "ops-guidance", Provider: provider, TargetPath: prefix + `\` + filename, DefinitionIDs: ids, Guidance: "Keep the selected project statements intact."}},
		TargetPrefix:  prefix,
		AllowedRoots:  []string{`C:\repo`},
		RequestDigest: "sha256:request-1",
		Observed:      []TargetObservation{{Path: prefix + `\` + filename, Owner: Owner, Writable: true}},
	}
}

func TestProviderAdaptersRenderIndependentlyAndRepeatAfterDelete(t *testing.T) {
	for _, provider := range []Provider{Codex, Claude} {
		t.Run(string(provider), func(t *testing.T) {
			input := inputFor(provider)
			first := Propose(input)
			second := Propose(input)
			if first.Decision != DecisionWork || len(first.Files) != 1 {
				t.Fatalf("first proposal = %#v", first)
			}
			if second.Decision != DecisionWork || len(second.Files) != 1 {
				t.Fatalf("proposal after target deletion = %#v", second)
			}
			if !bytes.Equal(first.Files[0].Content, second.Files[0].Content) || first.Files[0].Digest != second.Files[0].Digest {
				t.Fatal("rerender after deletion changed deterministic output")
			}
			if provider == Codex && (!strings.Contains(string(first.Files[0].Content), "# AGENTS.md") || strings.Contains(string(first.Files[0].Content), "# CLAUDE.md")) {
				t.Fatal("Codex adapter emitted another provider's framing")
			}
			if provider == Claude && (!strings.Contains(string(first.Files[0].Content), "# CLAUDE.md") || strings.Contains(string(first.Files[0].Content), "# AGENTS.md")) {
				t.Fatal("Claude adapter emitted another provider's framing")
			}
		})
	}
}

func TestCanonicalRuleChangeAffectsBothProviders(t *testing.T) {
	for _, provider := range []Provider{Codex, Claude} {
		t.Run(string(provider), func(t *testing.T) {
			before := inputFor(provider)
			before.Observed[0].Exists = false
			oldResult := Propose(before)
			after := inputFor(provider)
			after.Definitions[0].Purpose = "Preserve the reviewed release boundary."
			after.CanonicalAffected = []core.DefinitionIdentity{after.Definitions[0].Identity()}
			newResult := Propose(after)
			if oldResult.Decision != DecisionWork || newResult.Decision != DecisionWork || len(oldResult.Files) != 1 || len(newResult.Files) != 1 {
				t.Fatalf("old=%#v new=%#v", oldResult, newResult)
			}
			if bytes.Equal(oldResult.Files[0].Content, newResult.Files[0].Content) {
				t.Fatal("canonical Rule change did not affect provider output")
			}
			if !strings.Contains(string(newResult.Files[0].Content), "Preserve the reviewed release boundary.") {
				t.Fatal("updated canonical Rule purpose is absent")
			}
		})
	}
}

func TestUnknownSchemaAndKindPurposesArePreserved(t *testing.T) {
	result := Propose(inputFor(Codex))
	if result.Decision != DecisionWork || len(result.Files) != 1 {
		t.Fatalf("proposal = %#v", result)
	}
	content := string(result.Files[0].Content)
	for _, fact := range []string{"A project-defined schema for bounded engineering guidance.", "An unfamiliar project-specific concept retained without interpretation.", "Opaque project meaning carried as a string.", "intent/signal.yaml", "custom value", "related", "unknown-signal", "from", "to", "intent/rule.yaml", "line\": 4"} {
		if !strings.Contains(content, fact) {
			t.Errorf("output does not preserve supplied fact %q", fact)
		}
	}
}

func TestMissingOrAmbiguousGuidanceEscalates(t *testing.T) {
	missing := inputFor(Codex)
	missing.Policies = nil
	if got := Propose(missing); got.Decision != DecisionEscalate {
		t.Fatalf("missing guidance decision = %s", got.Decision)
	}
	ambiguous := inputFor(Codex)
	ambiguous.Policies = append(ambiguous.Policies, ambiguous.Policies[0])
	if got := Propose(ambiguous); got.Decision != DecisionEscalate {
		t.Fatalf("ambiguous guidance decision = %s", got.Decision)
	}
}

func TestTargetSafetyOwnershipAndReadOnlyEscalation(t *testing.T) {
	unsafe := inputFor(Codex)
	unsafe.Policies[0].TargetPath = `C:\outside\AGENTS.md`
	if got := Propose(unsafe); got.Decision != DecisionEscalate {
		t.Fatalf("unsafe target decision = %s", got.Decision)
	}
	unknownOwner := inputFor(Codex)
	unknownOwner.Observed[0].Owner = "unknown"
	if got := Propose(unknownOwner); got.Decision != DecisionEscalate {
		t.Fatalf("unknown owner decision = %s", got.Decision)
	}
	readOnly := inputFor(Codex)
	readOnly.ReadOnly = true
	if got := Propose(readOnly); got.Decision != DecisionEscalate || len(got.Files) != 0 {
		t.Fatalf("read-only decision = %#v", got)
	}
}

func TestNoOpRequiresExactObservedDigest(t *testing.T) {
	input := inputFor(Claude)
	candidate := Propose(input)
	if len(candidate.Files) != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}
	input.Observed[0].Exists = true
	input.Observed[0].Digest = candidate.Files[0].Digest
	input.Observed[0].Writable = false
	if got := Propose(input); got.Decision != DecisionNoOp {
		t.Fatalf("same bytes should be a no-op: %#v", got)
	}
}

func TestProposeDoesNotReadOrWriteTarget(t *testing.T) {
	input := inputFor(Codex)
	input.TargetPrefix = t.TempDir()
	input.AllowedRoots = []string{input.TargetPrefix}
	input.Policies[0].TargetPath = filepathJoin(input.TargetPrefix, "AGENTS.md")
	input.Observed[0].Path = input.Policies[0].TargetPath
	input.Observed[0].Owner = Owner
	before := sha256.Sum256([]byte("target absent"))
	if got := Propose(input); got.Decision != DecisionWork || len(got.Files) != 1 {
		t.Fatalf("proposal = %#v", got)
	}
	if _, err := os.Stat(input.Policies[0].TargetPath); !os.IsNotExist(err) {
		t.Fatalf("Propose touched target path: %v (guard %s)", err, hex.EncodeToString(before[:]))
	}
}

func TestUncoveredAffectedDefinitionEscalates(t *testing.T) {
	input := inputFor(Codex)
	input.Policies[0].DefinitionIDs = input.Policies[0].DefinitionIDs[:1]
	input.CanonicalAffected = []core.DefinitionIdentity{input.Definitions[1].Identity()}
	if got := Propose(input); got.Decision != DecisionEscalate {
		t.Fatalf("uncovered affected identity decision = %s", got.Decision)
	}
}

func filepathJoin(prefix, name string) string {
	return prefix + string(filepath.Separator) + name
}
