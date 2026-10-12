package projectbriefing

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
)

func TestValidateBundleRejectsMalformedDefinitionIdentities(t *testing.T) {
	before, after := revisionFixtures()
	bundle, err := Build(before, after, Provenance{DecisionReference: "ADR-17", Actor: "owner", Authority: "accepted decision"})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*core.DefinitionIdentity)
	}{
		{"missing API version", func(i *core.DefinitionIdentity) { i.APIVersion = "" }},
		{"invalid API version", func(i *core.DefinitionIdentity) { i.APIVersion = "invalid" }},
		{"missing kind", func(i *core.DefinitionIdentity) { i.Kind = "" }},
		{"invalid kind", func(i *core.DefinitionIdentity) { i.Kind = "bad kind" }},
		{"missing name", func(i *core.DefinitionIdentity) { i.Name = "" }},
		{"invalid name", func(i *core.DefinitionIdentity) { i.Name = "bad/name" }},
		{"invalid namespace", func(i *core.DefinitionIdentity) { i.Namespace = "bad/namespace" }},
		{"whitespace namespace", func(i *core.DefinitionIdentity) { i.Namespace = " " }},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := cloneBundleForTest(t, bundle)
			event := &invalid.Events[0]
			test.mutate(&event.DefinitionID)
			for _, value := range []*core.Definition{event.Before, event.After} {
				value.APIVersion, value.Kind = event.DefinitionID.APIVersion, event.DefinitionID.Kind
				value.Metadata = core.Metadata{Namespace: event.DefinitionID.Namespace, Name: event.DefinitionID.Name}
			}
			// Seal the outer bundle so identity validation is the failing contract.
			invalid.Digest = FullBundleDigest(invalid)
			if err := ValidateBundle(invalid); !errors.Is(err, ErrInvalidBundle) || !strings.Contains(err.Error(), "invalid definition identity") {
				t.Fatalf("malformed identity passed or failed an unrelated binding: %v", err)
			}
		})
	}
}

func TestValidateBundleRejectsInventedEventsAndReferences(t *testing.T) {
	before, after := revisionFixtures()
	bundle, err := Build(before, after, Provenance{DecisionReference: "ADR-17", Actor: "owner", Authority: "accepted decision"})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*Bundle)
	}{
		{"event ID", func(b *Bundle) { b.Events[0].ID = "invented-event" }},
		{"event digest", func(b *Bundle) { b.Events[0].Digest = strings.Repeat("f", 64) }},
		{"provenance", func(b *Bundle) { b.Events[0].Provenance.Actor = "different actor" }},
		{"missing decision", func(b *Bundle) { b.Provenance.DecisionReference = "" }},
		{"unknown change", func(b *Bundle) { b.Events[0].Change = "invented" }},
		{"payload identity", func(b *Bundle) { b.Events[0].After.Metadata.Name = "unrelated" }},
		{"duplicate event", func(b *Bundle) { b.Events = append(b.Events, b.Events[0]) }},
		{"global event ref", func(b *Bundle) { b.Global.EventIDs = []string{"unknown-event"} }},
		{"manager event ref", func(b *Bundle) { b.Managers[0].EventIDs = []string{"unknown-event"} }},
		{"duplicate affected ref", func(b *Bundle) {
			b.Events[0].AffectedManagers = append(b.Events[0].AffectedManagers, b.Events[0].AffectedManagers[0])
		}},
		{"missing model digest", func(b *Bundle) { b.ModelDigest = "" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := cloneBundleForTest(t, bundle)
			test.mutate(&invalid)
			invalid.Digest = FullBundleDigest(invalid)
			if err := ValidateBundle(invalid); !errors.Is(err, ErrInvalidBundle) {
				t.Fatalf("invalid bundle accepted: %v", err)
			}
		})
	}
}

func TestBuildDerivesGlobalAndNamedDefinitionOperations(t *testing.T) {
	for _, namespace := range []string{"", "example"} {
		for _, change := range []string{"added", "modified", "removed"} {
			t.Run(namespace+"/"+change, func(t *testing.T) {
				before, after := identityRevisionValues(namespace, change)
				bundle, err := Build(before, after, Provenance{DecisionReference: "ADR-17", Actor: "owner", Authority: "accepted decision"})
				if err != nil || len(bundle.Events) != 1 {
					t.Fatalf("Build returned %d events: %v", len(bundle.Events), err)
				}
				event := bundle.Events[0]
				if event.Change != change || event.DefinitionID.Namespace != namespace {
					t.Fatalf("changed definition identity or operation lost: %+v", event)
				}
				if (change == "added") != (event.Before == nil) || (change == "removed") != (event.After == nil) {
					t.Fatalf("event sides do not match %q operation: before=%v after=%v", change, event.Before, event.After)
				}
			})
		}
	}
}

func TestBuildProducesStructuredBeforeAfterAndManagerNeighborContract(t *testing.T) {
	before, after, rootID, childID, changedID, artifactID := managerNeighborRevisionFixtures()
	bundle, err := Build(before, after, Provenance{DecisionReference: "ADR-17", Actor: "project-owner", Authority: "accepted model decision"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(bundle.Events))
	}
	event := bundle.Events[0]
	if event.Change != "modified" || event.Before == nil || event.After == nil {
		t.Fatalf("event change payload incomplete: %#v", event)
	}
	if event.Before.Spec["description"] != "old rule" || event.After.Spec["description"] != "new rule" {
		t.Fatalf("declared before/after content lost: %#v", event)
	}
	if len(event.AffectedArtifacts) != 1 || event.AffectedArtifacts[0] != artifactID {
		t.Fatalf("affected artifacts = %v", event.AffectedArtifacts)
	}
	if !contains(event.AffectedManagers, childID) || !contains(event.AffectedManagers, rootID) {
		t.Fatalf("affected managers = %v", event.AffectedManagers)
	}
	var rootBriefing *Briefing
	for i := range bundle.Managers {
		if bundle.Managers[i].ManagerID == rootID {
			rootBriefing = &bundle.Managers[i]
		}
	}
	if rootBriefing == nil || len(rootBriefing.Contracts) != 1 || rootBriefing.Contracts[0].ID != changedID {
		t.Fatalf("parent briefing did not retain the relevant public neighbor contract: %#v", rootBriefing)
	}
	if bundle.Global.ID == "" || bundle.Global.ModelDigest != after.Model.Digest {
		t.Fatalf("global briefing is not bound to accepted model: %#v", bundle.Global)
	}
}

func cloneBundleForTest(t *testing.T, bundle Bundle) Bundle {
	t.Helper()
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var copy Bundle
	if err := json.Unmarshal(data, &copy); err != nil {
		t.Fatal(err)
	}
	return copy
}

func identityRevisionValues(namespace, change string) (RevisionModel, RevisionModel) {
	managerIdentity := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Namespace: namespace, Name: "project"}
	managerID := managerIdentity.Key()
	manager := projectmodel.Manager{ID: managerID, Name: "project", Namespace: namespace, Owns: []string{"."}}
	managerDefinition := core.Definition{APIVersion: managerIdentity.APIVersion, Kind: managerIdentity.Kind, Metadata: core.Metadata{Namespace: namespace, Name: managerIdentity.Name}}
	definitionIdentity := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: namespace, Name: "notification"}
	statementDefinition := func(description string) core.Definition {
		return core.Definition{APIVersion: definitionIdentity.APIVersion, Kind: definitionIdentity.Kind, Metadata: core.Metadata{Namespace: namespace, Name: definitionIdentity.Name}, Spec: map[string]any{"category": "rule", "description": description, "public": true}}
	}
	statement := func(description string) projectmodel.Statement {
		return projectmodel.Statement{ID: definitionIdentity.Key(), Name: definitionIdentity.Name, Namespace: namespace, Owner: managerID, Category: "rule", Description: description, Public: true}
	}
	oldDefinition, newDefinition := statementDefinition("before"), statementDefinition("after")
	oldStatement, newStatement := statement("before"), statement("after")
	beforeDefinitions, afterDefinitions := []core.Definition{managerDefinition, oldDefinition}, []core.Definition{managerDefinition, newDefinition}
	beforeStatements, afterStatements := []projectmodel.Statement{oldStatement}, []projectmodel.Statement{newStatement}
	if change == "added" {
		beforeDefinitions, beforeStatements = []core.Definition{managerDefinition}, []projectmodel.Statement{}
	} else if change == "removed" {
		afterDefinitions, afterStatements = []core.Definition{managerDefinition}, []projectmodel.Statement{}
	}
	before := RevisionModel{Revision: strings.Repeat("a", 40), Model: core.Model{Digest: "model-before", Definitions: beforeDefinitions}, Report: projectmodel.Report{Digest: "report-before", Managers: []projectmodel.Manager{manager}, Statements: beforeStatements}}
	after := RevisionModel{Revision: strings.Repeat("b", 40), Model: core.Model{Digest: "model-after", Definitions: afterDefinitions}, Report: projectmodel.Report{Digest: "report-after", Managers: []projectmodel.Manager{manager}, Statements: afterStatements}}
	return before, after
}

func managerNeighborRevisionFixtures() (RevisionModel, RevisionModel, string, string, string, string) {
	rootID := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Namespace: "demo", Name: "root"}.Key()
	childID := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Namespace: "demo", Name: "orders"}.Key()
	changedID := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "demo", Name: "order-rule"}.Key()
	neighborID := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "demo", Name: "integration-contract"}.Key()
	artifactID := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Artifact", Namespace: "demo", Name: "orders-api"}.Key()
	rootManager := projectmodel.Manager{ID: rootID, Name: "root", Namespace: "demo", Owns: []string{"."}}
	childManager := projectmodel.Manager{ID: childID, Name: "orders", Namespace: "demo", Parent: rootID, Owns: []string{"orders/"}}
	changedStatement := func(description string) projectmodel.Statement {
		return projectmodel.Statement{ID: changedID, Name: "order-rule", Namespace: "demo", Owner: childID, Category: "rule", Description: description, Public: true, Uses: []string{}, Requires: []string{}}
	}
	neighbor := projectmodel.Statement{ID: neighborID, Name: "integration-contract", Namespace: "demo", Owner: rootID, Category: "architecture", Description: "uses order rule", Public: true, Uses: []string{changedID}, Requires: []string{}}
	artifact := projectmodel.Artifact{ID: artifactID, Name: "orders-api", Owner: childID, Role: "implementation", Realizes: []string{changedID}, Paths: []string{"orders/api.go"}, Checks: []string{}, Required: true}
	definition := func(description string) core.Definition {
		return core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Statement", Metadata: core.Metadata{Name: "order-rule", Namespace: "demo"}, Purpose: "Rule", Spec: map[string]any{"category": "rule", "description": description, "public": true, "uses": []any{}, "requires": []any{}}, Source: core.Source{Path: ".markitect/model/demo.yaml", Digest: description}}
	}
	rootDefinition := core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Manager", Metadata: core.Metadata{Name: "root", Namespace: "demo"}, Purpose: "Root", Spec: map[string]any{"owns": []any{"."}}, Source: core.Source{Path: ".markitect/model/demo.yaml"}}
	childDefinition := core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Manager", Metadata: core.Metadata{Name: "orders", Namespace: "demo"}, Purpose: "Orders", Spec: map[string]any{"parent": rootID, "owns": []any{"orders/"}}, Source: core.Source{Path: ".markitect/model/demo.yaml"}}
	managers := []projectmodel.Manager{rootManager, childManager}
	beforeStatements := []projectmodel.Statement{changedStatement("old rule"), neighbor}
	afterStatements := []projectmodel.Statement{changedStatement("new rule"), neighbor}
	artifacts := []projectmodel.Artifact{artifact}
	before := RevisionModel{Revision: strings.Repeat("a", 40), Model: core.Model{Digest: "model-a", Definitions: []core.Definition{rootDefinition, childDefinition, definition("old rule")}}, Report: projectmodel.Report{Digest: "report-a", Managers: managers, Statements: beforeStatements, Artifacts: artifacts, Checks: []projectmodel.Check{}}}
	after := RevisionModel{Revision: strings.Repeat("b", 40), Model: core.Model{Digest: "model-b", Definitions: []core.Definition{rootDefinition, childDefinition, definition("new rule")}}, Report: projectmodel.Report{Digest: "report-b", Managers: managers, Statements: afterStatements, Artifacts: artifacts, Checks: []projectmodel.Check{}}}
	return before, after, rootID, childID, changedID, artifactID
}
