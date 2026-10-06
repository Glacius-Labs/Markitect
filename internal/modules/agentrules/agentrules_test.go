package agentrules

import (
	"bytes"
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
	prefix := "repo/project"
	return Input{
		Provider:      provider,
		Schemas:       schemas,
		Definitions:   definitions,
		Policies:      []ProjectionPolicy{{ID: "ops-guidance", Provider: provider, DefinitionIDs: ids, Guidance: "Keep the selected project statements intact."}},
		TargetPrefix:  prefix,
		AllowedRoots:  []string{"repo"},
		RequestDigest: "sha256:request-1",
		Observed:      []TargetObservation{{Path: prefix + "/" + providerFilename(provider), Owner: Owner, Writable: true}},
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
			if strings.Contains(first.Files[0].Path, `\`) || filepath.IsAbs(first.Files[0].Path) {
				t.Fatalf("candidate path is not portable and repository relative: %q", first.Files[0].Path)
			}
		})
	}
}

func TestCanonicalRuleChangeAffectsBothProviders(t *testing.T) {
	for _, provider := range []Provider{Codex, Claude} {
		t.Run(string(provider), func(t *testing.T) {
			before := inputFor(provider)
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

func TestSelectedDefinitionMayReferenceReadOnlyRelatedContext(t *testing.T) {
	input := inputFor(Codex)
	input.Definitions = input.Definitions[:1]
	input.RelatedDefinitions = []core.Definition{inputFor(Codex).Definitions[1]}
	input.Policies[0].DefinitionIDs = []core.DefinitionIdentity{input.Definitions[0].Identity()}
	input.CanonicalAffected = []core.DefinitionIdentity{input.Definitions[0].Identity()}
	file, err := Render(input)
	if err != nil {
		t.Fatalf("Render with related context: %v", err)
	}
	content := string(file.Content)
	for _, fact := range []string{"canonical-rule", "unknown-signal", "intent/rule.yaml", "related", "\"property\": \"related\"", "\"line\": 4"} {
		if !strings.Contains(content, fact) {
			t.Errorf("projected definition or resolved reference lacks %q", fact)
		}
	}
	for _, omitted := range []string{"Retain this custom fact as supplied.", "custom value", "intent/signal.yaml"} {
		if strings.Contains(content, omitted) {
			t.Errorf("read-only related Definition was projected: %q", omitted)
		}
	}
	input.Policies[0].DefinitionIDs = []core.DefinitionIdentity{input.RelatedDefinitions[0].Identity()}
	if _, err := Render(input); err == nil {
		t.Fatal("related Definition unexpectedly became projectable")
	}
}

func TestRenderNeedsNoTargetObservationOrWritableTarget(t *testing.T) {
	input := inputFor(Codex)
	input.Observed = nil
	input.ReadOnly = true
	file, err := Render(input)
	if err != nil {
		t.Fatalf("pure desired render should not require target evidence: %v", err)
	}
	if file.Path != "repo/project/AGENTS.md" {
		t.Fatalf("rendered path = %q", file.Path)
	}
	if got := Propose(input); got.Decision != DecisionEscalate {
		t.Fatalf("proposal without observed ownership = %#v", got)
	}
	uniquePrefix := "agentrules-no-io-" + filepath.Base(t.TempDir())
	input.TargetPrefix = uniquePrefix
	input.AllowedRoots = []string{uniquePrefix}
	input.Policies[0].TargetPath = ""
	file, err = Render(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(file.Path); !os.IsNotExist(err) {
		t.Fatalf("Render wrote desired target: %v", err)
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

func TestRepositoryPathSafetyAndAliases(t *testing.T) {
	explicit := inputFor(Codex)
	explicit.Policies[0].TargetPath = "repo/project/AGENTS.md"
	if file, err := Render(explicit); err != nil || file.Path != "repo/project/AGENTS.md" {
		t.Fatalf("matching explicit provider target = %#v, %v", file, err)
	}
	for _, prefix := range []string{"../project", "repo//project", `C:\repo\project`, "repo/./project"} {
		input := inputFor(Codex)
		input.TargetPrefix = prefix
		if _, err := Render(input); err == nil {
			t.Errorf("unsafe or aliased target prefix %q was accepted", prefix)
		}
	}
	traversalTarget := inputFor(Codex)
	traversalTarget.Policies[0].TargetPath = "repo/project/../AGENTS.md"
	if _, err := Render(traversalTarget); err == nil {
		t.Fatal("traversal alias in explicit target path was accepted")
	}
	mismatchedTarget := inputFor(Codex)
	mismatchedTarget.Policies[0].TargetPath = "repo/project/notes.md"
	if _, err := Render(mismatchedTarget); err == nil {
		t.Fatal("policy-selected filename was accepted")
	}
	rootCollision := inputFor(Codex)
	rootCollision.AllowedRoots = []string{"repo", "REPO"}
	if _, err := Render(rootCollision); err == nil {
		t.Fatal("case-colliding allowed roots were accepted")
	}
}

func TestCaseCollisionAndDuplicateTargetObservationEscalate(t *testing.T) {
	caseCollision := inputFor(Codex)
	caseCollision.Observed = append(caseCollision.Observed, TargetObservation{Path: "repo/project/agents.md", Owner: Owner, Writable: true})
	if got := Propose(caseCollision); got.Decision != DecisionEscalate {
		t.Fatalf("case-colliding inventory decision = %s (%v)", got.Decision, got.Reasons)
	}
	aliasCollision := inputFor(Codex)
	aliasCollision.Observed = append(aliasCollision.Observed, TargetObservation{Path: `repo\project\AGENTS.md`, Owner: Owner, Writable: true})
	if got := Propose(aliasCollision); got.Decision != DecisionEscalate {
		t.Fatalf("separator-alias inventory decision = %s (%v)", got.Decision, got.Reasons)
	}
	duplicate := inputFor(Codex)
	duplicate.Observed = append(duplicate.Observed, duplicate.Observed[0])
	if got := Propose(duplicate); got.Decision != DecisionEscalate {
		t.Fatalf("duplicate inventory decision = %s (%v)", got.Decision, got.Reasons)
	}
}

func TestTargetOwnershipReadOnlyAndNoOp(t *testing.T) {
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
	input := inputFor(Claude)
	candidate, err := Render(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Observed[0].Exists = true
	input.Observed[0].Digest = candidate.Digest
	input.Observed[0].Writable = false
	if got := Propose(input); got.Decision != DecisionNoOp {
		t.Fatalf("same bytes should be a no-op: %#v", got)
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
