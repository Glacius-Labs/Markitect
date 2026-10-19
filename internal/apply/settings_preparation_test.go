package apply_test

import (
	"context"
	"crypto/sha256"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const testHostDocument = `apiVersion: markitect.host/v1
agentProfiles:
  library-agent:
    codexAppServer: {profileVersion: 1}
checkProfiles:
  library-tests:
    process: {profileVersion: 1}
`

// These controlled ports are test-only metadata evidence. They start no process
// and make no claim about the actual Codex/process validation contracts.
type hostAgentPort func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error)

func (p hostAgentPort) PrepareAgentProfile(c context.Context, i apply.ProfileInput) (apply.ProfileObservation, error) {
	o, err := p(c, i)
	o.InputBinding = i.Binding()
	return o, err
}

type hostCheckPort func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error)

func (p hostCheckPort) PrepareCheckProfile(c context.Context, i apply.ProfileInput) (apply.ProfileObservation, error) {
	o, err := p(c, i)
	o.InputBinding = i.Binding()
	return o, err
}

func testMetadata(t *testing.T, boundary, name, bytes string) apply.ProfileMetadataFile {
	t.Helper()
	return apply.ProfileMetadataFile{Location: apply.SourceLocation{Boundary: boundary, Path: path(t, name)}, ContentID: snapshot.ContentID(sha256.Sum256([]byte(bytes))), Size: int64(len(bytes)), Mode: snapshot.Regular}
}
func testObservation(t *testing.T, agent bool) apply.ProfileObservation {
	t.Helper()
	o := apply.ProfileObservation{Status: model.Passed, Contract: apply.ProfileContract{Name: "controlled-test-metadata", Version: 1}, Files: []apply.ProfileMetadataFile{testMetadata(t, "actual-executable", "program", "original executable metadata")}}
	if agent {
		o.Inventories = []apply.ProfileMetadataInventory{{Boundary: "actual-schemas", Files: []apply.ProfileMetadataFile{testMetadata(t, "actual-schemas", "schema.json", "original schema")}}}
	}
	return o
}
func hostPorts(t *testing.T) (hostAgentPort, hostCheckPort) {
	t.Helper()
	return func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
			return testObservation(t, true), nil
		}, func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
			return testObservation(t, false), nil
		}
}

func addHostDocument(t *testing.T, selection *apply.InputSelection, raw *apply.RawInputs, document string) {
	t.Helper()
	location := apply.SourceLocation{Boundary: "explicit-host-boundary", Path: path(t, "profiles/host.yaml")}
	file, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte(document))
	if err != nil {
		t.Fatal(err)
	}
	selection.HostProfiles = []apply.SourceLocation{location}
	raw.HostProfiles = []apply.SourceFile{{Location: location, File: file}}
}

func hostSettingsInputs(t *testing.T) (apply.InputSelection, apply.RawInputs) {
	t.Helper()
	selection, raw := fileCheckInputs(t)
	rewriteFileCheckSource(t, &raw, &raw.Execution[0], defaultsAbsentExecution(t))
	addHostDocument(t, &selection, &raw, testHostDocument)
	return selection, raw
}

func TestM1HostSettingsUsesActualFileCheckAndProductDefaultsParents(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	files := m.PrepareRegistry().CheckFiles()
	defaults := m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults())
	agent, check := hostPorts(t)
	result, err := files.PrepareSettings(context.Background(), defaults, agent, check)
	settings, ok := result.Settings()
	checks, checksOK := result.Checks()
	if err != nil || result.Status() != model.Passed || !ok || !checksOK || len(checks) != 1 || len(result.Diagnostics()) != 0 {
		t.Fatalf("M1 host settings: %v %s %+v; files=%+v defaults=%+v", err, result.Status(), result.Diagnostics(), files.Diagnostics(), defaults.Diagnostics())
	}
	root, ok := settings.Manager(".")
	if !ok || len(root.Execute.InstructionSources) != 1 || root.Execute.InstructionSources[0].File != defaults.ProductSource().File || root.Execute.Profile.Value.Name != "library-agent" || root.Execute.Profile.Value.Identity == "" {
		t.Fatalf("actual resolution/default provenance lost: %+v", root)
	}
	declaration := checks[0].Declaration()
	authored, _ := m.DecodeExecution().Document()
	if !reflect.DeepEqual(declaration, authored.Checks.Value[0]) || !reflect.DeepEqual(result.FileCheck(), files) || !reflect.DeepEqual(result.Defaults(), defaults) || !reflect.DeepEqual(result.FileCheck().RegistryPreparation().Model().Inputs().Raw, raw) {
		t.Fatal("original authored check, defaults or guarded parent changed")
	}
	if len(result.ProfileObservations()) != 2 || len(result.RequiredInstructions()) != 0 {
		t.Fatal("profile evidence missing")
	}
	for _, profile := range result.ProfileObservations() {
		input := profile.Input()
		if input.Document() != raw.HostProfiles[0] || input.Location() != raw.HostProfiles[0].Location || input.Inline() == nil || profile.Source().ContentID != raw.HostProfiles[0].File.ContentID() || profile.Source().Field != input.Field() {
			t.Fatal("actual host bytes/field binding lost")
		}
		if _, ok := profile.Identity(); !ok {
			t.Fatal("passed metadata identity missing")
		}
	}
	for _, revision := range []string{"", fixtureHead} {
		source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
			return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: revision}, nil
		}}
		preview, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		pf := preview.PrepareRegistry().CheckFiles()
		pd := preview.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults())
		prepared, err := pf.PrepareSettings(context.Background(), pd, agent, check)
		ps, available := prepared.Settings()
		if err != nil || prepared.Status() != model.Passed || !available || !reflect.DeepEqual(ps, settings) || !reflect.DeepEqual(prepared.FileCheck(), pf) || !reflect.DeepEqual(prepared.Defaults(), pd) || prepared.FileCheck().RegistryPreparation().Model().Inputs().Revision != revision {
			t.Fatalf("preview binding: %v %+v", err, prepared.Diagnostics())
		}
	}
}

func TestHostSettingsExposesTheAuthoredBudgetOnlyWithResolvedSettings(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	agent, check := hostPorts(t)
	result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	budget, ok := result.Budget()
	if err != nil || !ok || result.Status() != model.Passed {
		t.Fatalf("budget unavailable: %v %s %+v", err, result.Status(), result.Diagnostics())
	}
	if budget.ExecutionTime.Value != 25*time.Minute || budget.TurnTimeout.Value != 10*time.Minute || budget.AgentTurns.Value != 120 || budget.Rounds.Value != 4 || budget.Rounds.Source.Field != "/budget/rounds" || budget.Rounds.Source.File != ".markitect/execution.yaml" {
		t.Fatalf("budget or its provenance changed: %+v", budget)
	}
	bad := apply.ProductStrategyDefaults()
	bad.Bytes = nil
	refused, _ := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(bad), agent, check)
	if _, ok := refused.Budget(); ok {
		t.Fatal("a refused preparation exposed a budget")
	}
}
