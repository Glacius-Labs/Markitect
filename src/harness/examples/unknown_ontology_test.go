package examples

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

const unknownOntologyConfigPath = "examples/unknown-ontology/canonical.yaml"

func TestUnknownOntologyLoadsExactPackagesAndPreparesBoundedScope(t *testing.T) {
	root, revision := publicExampleFixture(t, "examples/unknown-ontology")
	fixed, err := host.LoadCanonicalSource(root, revision, unknownOntologyConfigPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if fixed.Snapshot.Provisional || fixed.Snapshot.ID != revision {
		t.Fatalf("canonical source is not fixed to HEAD %q: snapshot=%q provisional=%t", revision, fixed.Snapshot.ID, fixed.Snapshot.Provisional)
	}
	if len(fixed.Diagnostics) != 0 {
		t.Fatalf("unknown-ontology source has structural diagnostics: %#v", fixed.Diagnostics)
	}
	if len(fixed.Pins) != 3 || len(fixed.Activation.Projectors) != 1 {
		t.Fatalf("want three exact Module pins and one Projection binding; pins=%d bindings=%d", len(fixed.Pins), len(fixed.Activation.Projectors))
	}

	var check authoring.Check
	for _, configured := range fixed.Config.Checks {
		if configured.Name == "unknown-ontology-behavior" {
			check = configured
			break
		}
	}
	if check.Name == "" {
		t.Fatal("missing explicit unknown-ontology-behavior check")
	}
	observed, err := source.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	observed = cloneSnapshot(observed)
	observed.ID = ""
	observed.Provisional = true
	identity := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "unknown-ontology", Name: "mission-dotnet"}
	prepared, err := host.PrepareCanonicalProjection(fixed, observed, identity, "unknown-ontology-shape/1", host.Hash([]byte("unknown-ontology-shape")), nil, check)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSuffix(prepared.Request.TargetPath, "/"); got != "src/UnknownOntology" {
		t.Fatalf("target root %q is not the declared root", got)
	}
	if prepared.Request.Projector.ID != "dotnet-source" || prepared.Request.Projector.Target != "dotnet" {
		t.Fatalf("unexpected generic .NET capability binding: %#v", prepared.Request.Projector)
	}
	if len(prepared.Request.Projector.RequiredChecks) != 1 || prepared.Request.Projector.RequiredChecks[0] != check.Name {
		t.Fatalf("required check scope = %#v", prepared.Request.Projector.RequiredChecks)
	}
	if len(prepared.Request.Definitions) != 3 {
		t.Fatalf("selected %d Definitions, want the Mission, EffectAxis and Capability", len(prepared.Request.Definitions))
	}
	want := map[string]bool{"Mission/mission-planning/daily-reflection": true, "EffectAxis/mission-planning/focus-duration": true, "Capability/mission-planning/guided-focus": true}
	for _, definition := range prepared.Request.Definitions {
		identity := definition.Identity()
		delete(want, identity.Kind+"/"+identity.Namespace+"/"+identity.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing selected project-owned Kinds: %#v", want)
	}
	if len(prepared.Request.Policies) != 3 {
		t.Fatalf("selected %d ProjectionPolicies, want one for each source Kind", len(prepared.Request.Policies))
	}
	policyNames := map[string]bool{"mission-dotnet": true, "effect-axis-dotnet": true, "capability-dotnet": true}
	for _, policy := range prepared.Request.Policies {
		delete(policyNames, policy.Identity().Name)
	}
	if len(policyNames) != 0 {
		t.Fatalf("missing per-Kind policies: %#v", policyNames)
	}
	if prepared.Plan != nil || !hasUnknownOntologyEscalation(prepared.Escalations, "projection.candidate-required") {
		t.Fatalf("no-candidate Prepare must remain incomplete, plan=%t escalations=%#v", prepared.Plan != nil, prepared.Escalations)
	}
}

func hasUnknownOntologyEscalation(escalations []host.CanonicalProjectionEscalation, code string) bool {
	for _, escalation := range escalations {
		if escalation.Code == code {
			return true
		}
	}
	return false
}
