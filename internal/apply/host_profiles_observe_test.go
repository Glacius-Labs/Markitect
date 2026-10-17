package apply_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestHostNilPortsAndIncompleteMetadataNeverInventProfiles(t *testing.T) {
	for _, tc := range []string{"nil", "typed nil", "zero observation", "missing executable", "missing schemas", "unknown status", "port error", "failed port with error", "cancelled", "duplicate file", "invalid mode", "inventory boundary", "duplicate inventory", "negative size", "zero digest", "required instruction", "unused failed profile"} {
		t.Run(tc, func(t *testing.T) {
			selection, raw := hostSettingsInputs(t)
			if tc == "unused failed profile" {
				addHostDocument(t, &selection, &raw, strings.Replace(testHostDocument, "checkProfiles:", "  unused: {codexAppServer: {profileVersion: 1}}\ncheckProfiles:", 1))
			}
			m := registryModel(t, selection, raw)
			agent, check := hostPorts(t)
			observation := testObservation(t, true)
			want := model.Failed
			var portErr error
			switch tc {
			case "nil", "typed nil":
				agent = nil
				want = model.NotCompleted
			case "zero observation":
				observation = apply.ProfileObservation{}
				want = model.NotCompleted
			case "missing executable":
				observation.Files = nil
			case "missing schemas":
				observation.Inventories = nil
			case "unknown status":
				observation.Status = "unknown"
			case "port error":
				portErr = errors.New("finite metadata read failed")
				want = model.NotCompleted
			case "failed port with error":
				observation.Status = model.Failed
				portErr = errors.New("known invalid plus unavailable")
			case "cancelled":
				want = model.NotCompleted
			case "duplicate file":
				observation.Files = append(observation.Files, observation.Files[0])
			case "invalid mode":
				observation.Files[0].Mode = snapshot.Mode(99)
			case "inventory boundary":
				observation.Inventories[0].Files[0].Location.Boundary = "different"
			case "duplicate inventory":
				observation.Inventories = append(observation.Inventories, observation.Inventories[0])
			case "negative size":
				observation.Files[0].Size = -1
			case "zero digest":
				observation.Files[0].ContentID = snapshot.ContentID{}
			case "required instruction":
				observation.RequiredInstructions = []apply.SourceLocation{{Boundary: "needed", Path: path(t, "instruction.md")}}
				want = model.NotCompleted
			case "unused failed profile":
				agent = func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
					o := testObservation(t, true)
					if input.Name() == "unused" {
						o.Status = model.Failed
					}
					return o, nil
				}
			}
			if tc != "nil" && tc != "typed nil" && tc != "unused failed profile" {
				agent = func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
					return observation, portErr
				}
			}
			ctx := context.Background()
			if tc == "cancelled" {
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			var agentPort apply.AgentProfilePreparer = agent
			if tc == "nil" {
				agentPort = nil
			}
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(ctx, m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agentPort, check)
			if result.Status() != want || portErr != nil && !errors.Is(err, portErr) || tc == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatalf("unusable evidence passed: %v %s %+v", err, result.Status(), result.Diagnostics())
			}
			if _, ok := result.Settings(); ok {
				t.Fatal("incomplete/invalid profile fabricated settings")
			}
			for _, p := range result.ProfileObservations() {
				if p.Input().Namespace() == "agentProfiles" && (tc != "unused failed profile" || p.Input().Name() == "unused") {
					if _, ok := p.Identity(); ok {
						t.Fatal("incomplete profile fabricated identity")
					}
				}
			}
			if len(result.ProfileObservations()) < 2 {
				t.Fatal("partial profile observations lost")
			}
		})
	}
}

func TestHostComparisonIdentityBindsActualSourcesAndCompleteMetadataSets(t *testing.T) {
	identity := func(change string) (string, string) {
		selection, raw := hostSettingsInputs(t)
		observation := testObservation(t, true)
		instruction := apply.SourceLocation{Boundary: "instruction-boundary", Path: path(t, "instruction.md")}
		bytes := "instruction bytes"
		if change == "instruction bytes" {
			bytes += " changed"
		}
		entry, err := snapshot.NewEntry(instruction.Path, snapshot.Regular, []byte(bytes))
		if err != nil {
			t.Fatal(err)
		}
		raw.Instructions = []apply.SourceFile{{Location: instruction, File: entry}}
		selection.Instructions = []apply.SourceLocation{instruction}
		observation.Instructions = []apply.SourceLocation{instruction}
		switch change {
		case "host bytes":
			addHostDocument(t, &selection, &raw, "# exact original source change\n"+testHostDocument)
		case "host boundary":
			selection.HostProfiles[0].Boundary += "other"
			raw.HostProfiles[0].Location = selection.HostProfiles[0]
		case "profile field":
			addHostDocument(t, &selection, &raw, strings.Replace(testHostDocument, "library-agent:", "library~/agent:", 1))
			rewriteFileCheckSource(t, &raw, &raw.Execution[0], strings.ReplaceAll(defaultsAbsentExecution(t), "library-agent", "library~/agent"))
		case "contract":
			observation.Contract.Name += "/different"
		case "contract version":
			observation.Contract.Version++
		case "executable bytes":
			observation.Files[0] = testMetadata(t, "actual-executable", "program", "changed executable")
		case "executable location":
			observation.Files[0].Location.Boundary += "other"
		case "mode":
			observation.Files[0].Mode = snapshot.Executable
		case "size":
			observation.Files[0].Size++
		case "schema bytes":
			observation.Inventories[0].Files[0] = testMetadata(t, "actual-schemas", "schema.json", "changed schema")
		case "schema added":
			observation.Inventories[0].Files = append(observation.Inventories[0].Files, testMetadata(t, "actual-schemas", "extra.json", "extra"))
		case "schema deleted":
			observation.Inventories[0].Files = []apply.ProfileMetadataFile{testMetadata(t, "actual-schemas", "remaining.json", "remaining")}
		case "schema root":
			observation.Inventories[0].Boundary += "other"
			observation.Inventories[0].Files[0].Location.Boundary = observation.Inventories[0].Boundary
		case "instruction selection":
			observation.Instructions = nil
		}
		m := registryModel(t, selection, raw)
		agent, check := hostPorts(t)
		agent = func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) { return observation, nil }
		result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
		if err != nil || result.Status() != model.Passed {
			t.Fatalf("identity fixture: %v %+v", err, result.Diagnostics())
		}
		p := result.ProfileObservations()[0]
		id, ok := p.Identity()
		if !ok {
			t.Fatal("missing comparison identity")
		}
		if change == "profile field" && p.Source().Field != "/agentProfiles/library~0~1agent/codexAppServer" {
			t.Fatal("field pointer not escaped")
		}
		return id, p.Source().File
	}
	baseline, locator := identity("")
	for _, change := range []string{"host bytes", "host boundary", "profile field", "contract", "contract version", "executable bytes", "executable location", "mode", "size", "schema bytes", "schema added", "schema deleted", "schema root", "instruction bytes", "instruction selection"} {
		got, source := identity(change)
		if got == baseline {
			t.Fatalf("comparison identity omitted %s", change)
		}
		if change == "host boundary" && source == locator {
			t.Fatal("different external boundary collided in SettingSource.File")
		}
	}
}

func TestHostPortInputsAndRetainedResultsOwnEveryMutableCopy(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	agent, check := hostPorts(t)
	observation := testObservation(t, true)
	observation.Diagnostics = []apply.HostProfileDiagnostic{{Code: "test.static_only", Position: &model.Position{Line: 1, Column: 2}, Message: "controlled metadata"}}
	agent = func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		first := input.Inline()
		first.Content[1].Value = "changed"
		if input.Inline().Content[1].Value != "1" {
			t.Fatal("Inline accessor aliases request")
		}
		return observation, nil
	}
	result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	if err != nil || result.Status() != model.Passed {
		t.Fatalf("copy fixture failed: %v %+v", err, result.Diagnostics())
	}
	before := result.ProfileObservations()
	beforeDiagnostics := result.Diagnostics()
	observation.Files[0].Size++
	observation.Inventories[0].Files[0].Size++
	observation.Diagnostics[0].Position.Line++
	returned := result.ProfileObservations()
	o := returned[0].Observation()
	o.Files[0].Location.Boundary = "changed"
	o.Inventories[0].Files[0].Size++
	o.Diagnostics[0].Position.Line++
	returned[0].Input().Inline().Content[1].Value = "changed"
	checks, _ := result.Checks()
	d := checks[0].Declaration()
	d.Args.Value[0] = "changed"
	if d.RequiredBy != nil {
		d.RequiredBy.Value.ID = "changed"
	}
	diagnostics := result.Diagnostics()
	diagnostics[0].Position.Line++
	if !reflect.DeepEqual(before, result.ProfileObservations()) || !reflect.DeepEqual(beforeDiagnostics, result.Diagnostics()) {
		t.Fatal("retained evidence aliases caller/port/result data")
	}
	fresh, _ := result.Checks()
	if reflect.DeepEqual(d, fresh[0].Declaration()) {
		t.Fatal("check declaration aliases returned arguments/requiredBy")
	}
}
