package host

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

// The unit fixture uses a supplied fixed identity; the executable example uses
// real Git commits and guarded writes. This test never mutates fixture files.
func reconciliationFixture(t *testing.T) (*CanonicalSource, *snapshot.Snapshot) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := LoadCanonicalSource(root, "", "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatal(err)
	}
	source.Snapshot.ID = strings.Repeat("a", 40)
	source.Snapshot.Provisional = false
	source.Model, source.Diagnostics = core.Compile(source.Model.Schemas, source.Model.Definitions, source.Snapshot.ID)
	if len(source.Diagnostics) != 0 {
		t.Fatal(source.Diagnostics)
	}
	observed := &snapshot.Snapshot{ID: "working-tree", Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	for name, data := range source.Snapshot.Files {
		observed.Files[name] = append([]byte(nil), data...)
		observed.Modes[name] = source.Snapshot.Modes[name]
	}
	return source, observed
}
func reconciliationRecords(t *testing.T, fixed *CanonicalSource, observed *snapshot.Snapshot) []records.ProjectionRecord {
	t.Helper()
	requests, err := projectionRequestIndex(fixed)
	if err != nil {
		t.Fatal(err)
	}
	result := []records.ProjectionRecord{}
	for _, request := range requests {
		path := "src/Commerce/Example.cs"
		if request.Projector.Target == "markdown" {
			path = "docs/represented/index.md"
		}
		bytes := []byte("owned fixture bytes")
		observed.Files[path] = bytes
		observed.Modes[path] = snapshot.RegularMode
		facts := []records.ArtifactFact{{Path: path, Digest: sha256Prefix(sha256Hex(bytes)), Mode: snapshot.RegularMode}}
		target, err := records.TargetSnapshotDigest(facts)
		if err != nil {
			t.Fatal(err)
		}
		scope, policies := []string{}, []string{}
		for _, d := range request.Definitions {
			scope = append(scope, d.Identity().Key())
		}
		for _, d := range request.Policies {
			policies = append(policies, d.Identity().Key())
		}
		record, err := records.NewProjectionRecord(records.ProjectionRecord{
			Revision: fixed.Snapshot.ID, ModelDigest: request.ModelDigest, PlanDigest: sha256Prefix(sha256Hex([]byte("plan"))), InputSnapshotDigest: sha256Prefix(observed.Digest()), RequestDigest: request.RequestDigest,
			Module: records.ModuleIdentity{Name: request.ModulePin.Name, Version: request.ModulePin.Version, Digest: request.ModulePin.Digest}, ProjectionID: request.Projection.Identity().Key(), Projector: records.ProjectorIdentity{ID: request.Projector.ID, Version: request.Projector.Version}, ScopeIDs: scope, PolicyIDs: policies,
			Artifacts: []records.Artifact{{Path: path, Digest: facts[0].Digest, Mode: facts[0].Mode, Change: records.ChangeCreated}}, TargetSnapshotDigest: target, State: records.StateMaterializedUnverified,
		})
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, record)
	}
	return result
}
func TestCanonicalReconciliationKeepsScopedWorkSeparateFromConservativeFreshness(t *testing.T) {
	fixed, observed := reconciliationFixture(t)
	active := reconciliationRecords(t, fixed, observed)
	initial, err := PlanCanonicalReconciliation(fixed, fixed, observed, active)
	if err != nil || len(initial.Work) != 0 || len(initial.NoApplicableWork) != 2 {
		t.Fatalf("no-work: %#v %v", initial, err)
	}
	// Change one .NET-only policy, preserving the independently scoped Markdown request.
	candidate := *fixed
	candidate.Model.Definitions = append([]core.Definition(nil), fixed.Model.Definitions...)
	changed := false
	for i, d := range candidate.Model.Definitions {
		if d.Kind == "ProjectionPolicy" {
			candidate.Model.Definitions[i].Purpose += " policy change"
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("fixture has no policy")
	}
	candidate.Model, candidate.Diagnostics = core.Compile(candidate.Model.Schemas, candidate.Model.Definitions, strings.Repeat("b", 40))
	// Impact consumes IR values; source bytes remain unchanged in this unit fixture.
	impact, err := AnalyzeCanonicalImpact(fixed, &candidate, active)
	if err != nil {
		t.Fatal(err)
	}
	if len(impact.ScopeAffectedProjections) != 1 || len(impact.ConservativeInvalidatedProjections) != 2 {
		t.Fatalf("scoped/conservative=%#v", impact)
	}
	if got := impact.ScopeAffectedProjections[0].RecordedArtifacts; !reflect.DeepEqual(got, []string{"src/Commerce/Example.cs"}) {
		t.Fatalf("sibling representation leaked into ownership: %#v", got)
	}
}
func TestCanonicalReconciliationReportsDriftUnknownsAndPartialRecords(t *testing.T) {
	fixed, observed := reconciliationFixture(t)
	active := reconciliationRecords(t, fixed, observed)
	observed.Files["src/Commerce/Example.cs"] = []byte("drift")
	plan, err := PlanCanonicalReconciliation(fixed, fixed, observed, active)
	if err != nil || len(plan.Work) != 1 || len(plan.NoApplicableWork) != 1 || !reflect.DeepEqual(plan.Work[0].Reasons, []string{"projection-drift"}) {
		t.Fatalf("drift=%#v %v", plan, err)
	}
	encoded, _ := json.Marshal(plan)
	again, err := PlanCanonicalReconciliation(fixed, fixed, observed, active)
	repeated, _ := json.Marshal(again)
	if err != nil || string(encoded) != string(repeated) {
		t.Fatal("nondeterministic plan")
	}
	observed.Files["src/Commerce/unclaimed.cs"] = []byte("unknown")
	plan, err = PlanCanonicalReconciliation(fixed, fixed, observed, active)
	if err != nil || plan.Status != "escalated" {
		t.Fatalf("unknown artifact silently accepted: %#v %v", plan, err)
	}
	delete(observed.Files, "src/Commerce/unclaimed.cs")
	for i, record := range active {
		if record.Module.Name == "markitect-markdown" {
			record.State = records.StatePartialFailure
			active[i], err = records.NewProjectionRecord(record)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err = PlanCanonicalReconciliation(fixed, fixed, observed, active)
	if err != nil || len(plan.Work) != 2 {
		t.Fatalf("partial record became no-work: %#v %v", plan, err)
	}
}
func TestCanonicalReconciliationRejectsDuplicateActiveOwnersAndChangedSource(t *testing.T) {
	fixed, observed := reconciliationFixture(t)
	active := reconciliationRecords(t, fixed, observed)
	if _, err := PlanCanonicalReconciliation(fixed, fixed, observed, append(active, active[0])); err == nil {
		t.Fatal("duplicate active owners accepted")
	}
	observed.Files["examples/canonical-projection/canonical.yaml"] = []byte("changed source")
	if _, err := PlanCanonicalReconciliation(fixed, fixed, observed, active); err == nil {
		t.Fatal("working canonical edits accepted against fixed intent")
	}
}
func TestCanonicalProjectionIdentityAndCheckerFreshness(t *testing.T) {
	if _, err := projectionIdentityFromKey(`["v1", "Projection","n","a"]`); err == nil {
		t.Fatal("non-normalized identity accepted")
	}
	fixed, _ := reconciliationFixture(t)
	check := fixed.Config.Checks
	if len(check) == 0 {
		check = append(check, canonicalFixtureCheck())
	}
	first := CanonicalCheckEvidenceDigest(check[0], fixed.Snapshot)
	changed := *fixed.Snapshot
	changed.Files = map[string][]byte{}
	for name, data := range fixed.Snapshot.Files {
		changed.Files[name] = data
	}
	changed.Files["checker.go"] = []byte("different checker")
	if first == CanonicalCheckEvidenceDigest(check[0], &changed) {
		t.Fatal("changed checker input did not stale check identity")
	}
}

func canonicalFixtureCheck() authoring.Check {
	return authoring.Check{Name: "fixture", Run: []string{"go", "run", "checker.go"}}
}

func TestCanonicalReconciliationReportsEvidenceRefreshWithoutInventingEdits(t *testing.T) {
	fixed, observed := reconciliationFixture(t)
	active := reconciliationRecords(t, fixed, observed)
	current := *fixed
	changedSnapshot := *fixed.Snapshot
	current.Snapshot = &changedSnapshot
	current.Snapshot.ID = strings.Repeat("b", 40)
	current.Model, current.Diagnostics = core.Compile(fixed.Model.Schemas, fixed.Model.Definitions, current.Snapshot.ID)
	plan, err := PlanCanonicalReconciliation(fixed, &current, observed, active)
	if err != nil || len(plan.Work) != 0 || len(plan.NoApplicableWork) != 2 || len(plan.EvidenceRefreshRequired) != 2 {
		t.Fatalf("revision-only refresh=%#v %v", plan, err)
	}
}
func TestCanonicalReconciliationEscalatesProtectedTargetBeforeExecution(t *testing.T) {
	fixed, observed := reconciliationFixture(t)
	for i, registered := range fixed.Activation.Projectors {
		if registered.Registration.Target == "dotnet" {
			fixed.Activation.Projectors[i].Registration.AllowedRoots = []string{"examples/canonical-projection"}
		}
	}
	for i, definition := range fixed.Model.Definitions {
		if definition.Kind == "Projection" && definition.Metadata.Name == "application-dotnet" {
			copied, _ := fixed.Model.Definition(definition.Identity())
			copied.Spec["target"] = map[string]any{"repository": ".", "path": "examples/canonical-projection"}
			fixed.Model.Definitions[i] = copied
		}
	}
	fixed.Model, fixed.Diagnostics = core.Compile(fixed.Model.Schemas, fixed.Model.Definitions, fixed.Snapshot.ID)
	plan, err := PlanCanonicalReconciliation(fixed, fixed, observed, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, escalation := range plan.Escalations {
		if escalation.Code == "projection.protected-target" {
			found = true
		}
	}
	if !found || plan.Status != "escalated" {
		t.Fatalf("protected source proposed for execution: %#v", plan)
	}
}
