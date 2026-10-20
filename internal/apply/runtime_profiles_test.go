package apply_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestRuntimeEnvelopeRejectsInvalidStructureBeforeEveryAdapter(t *testing.T) {
	for _, tc := range []struct{ name, document string }{
		{"empty", ""},
		{"syntax", "apiVersion: [unterminated"},
		{"extra document", testRuntimeDocument + "---\n{}\n"},
		{"unknown version", strings.Replace(testRuntimeDocument, "markitect.runtime/v1", "markitect.runtime/v2", 1)},
		{"unknown root", testRuntimeDocument + "unknown: true\n"},
		{"duplicate root", testRuntimeDocument + "agentProfiles: {}\n"},
		{"missing namespace", "apiVersion: markitect.runtime/v1\nagentProfiles: {}\n"},
		{"wrong namespace type", strings.Replace(testRuntimeDocument, "agentProfiles:", "agentProfiles: []\nunused:", 1)},
		{"wrong wrapper", strings.Replace(testRuntimeDocument, "codexAppServer:", "process:", 1)},
		{"wrapper extras", strings.Replace(testRuntimeDocument, "    codexAppServer:", "    unknown: true\n    codexAppServer:", 1)},
		{"inline type", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "[]", 1)},
		{"duplicate profile", strings.Replace(testRuntimeDocument, "checkProfiles:", "  library-agent: {codexAppServer: {profileVersion: 1}}\ncheckProfiles:", 1)},
		{"nested duplicate", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: {same: 1, same: 2}}", 1)},
		{"nested nonstring key", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: {1: value}}", 1)},
		{"nested sequence nonstring key", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: [{false: value}]}", 1)},
		{"null", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: null}", 1)},
		{"anchor", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: &anchor []}", 1)},
		{"alias", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: &anchor [], repeated: *anchor}", 1)},
		{"custom tag", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: !custom value}", 1)},
		{"standard tag", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: !!str value}", 1)},
		{"standard key tag", strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: {!!str key: value}}", 1)},
		{"empty name", strings.Replace(testRuntimeDocument, "library-agent:", "'':", 1)},
		{"blank name", strings.Replace(testRuntimeDocument, "library-agent:", "'  ':", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := runtimeSettingsInputs(t)
			addRuntimeDocument(t, &selection, &raw, tc.document)
			m := registryModel(t, selection, raw)
			calls := 0
			agent := runtimeAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
				calls++
				return testObservation(t, true), nil
			})
			check := runtimeCheckPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
				calls++
				return testObservation(t, false), nil
			})
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
			if err != nil || result.Status() != model.Failed || calls != 0 || len(result.Diagnostics()) == 0 {
				t.Fatalf("invalid envelope: %s calls=%d err=%v %+v", result.Status(), calls, err, result.Diagnostics())
			}
			if _, ok := result.Settings(); ok {
				t.Fatal("invalid envelope exposed settings")
			}
			if _, ok := result.Checks(); ok {
				t.Fatal("invalid envelope exposed check set")
			}
			for _, diagnostic := range result.Diagnostics() {
				if strings.HasPrefix(diagnostic.Code, "runtime.") && diagnostic.Source.ContentID != raw.RuntimeProfiles[0].File.ContentID() {
					t.Fatal("invalid document lost original content binding")
				}
			}
		})
	}
}

func TestRuntimeNamespacesAreExactSeparateAndDoNotFallBack(t *testing.T) {
	for _, tc := range []struct {
		name, runtimeProfile, execution string
		status                          model.Status
	}{
		{"same name", strings.Replace(testRuntimeDocument, "library-tests", "library-agent", 1), strings.Replace(defaultsAbsentExecution(t), "profile: library-tests", "profile: library-agent", 1), model.Passed},
		{"agent missing", strings.Replace(testRuntimeDocument, "library-agent:", "other-agent:", 1), defaultsAbsentExecution(t), model.Failed},
		{"check missing", strings.Replace(testRuntimeDocument, "library-tests:", "other-check:", 1), defaultsAbsentExecution(t), model.Failed},
		{"case is exact", strings.Replace(testRuntimeDocument, "library-agent:", "Library-agent:", 1), defaultsAbsentExecution(t), model.Failed},
		{"check namespace only", "apiVersion: markitect.runtime/v1\nagentProfiles: {}\ncheckProfiles:\n  library-agent: {process: {profileVersion: 1}}\n  library-tests: {process: {profileVersion: 1}}\n", defaultsAbsentExecution(t), model.Failed},
		{"agent namespace only", "apiVersion: markitect.runtime/v1\nagentProfiles:\n  library-agent: {codexAppServer: {profileVersion: 1}}\n  library-tests: {codexAppServer: {profileVersion: 1}}\ncheckProfiles: {}\n", defaultsAbsentExecution(t), model.Failed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := runtimeSettingsInputs(t)
			addRuntimeDocument(t, &selection, &raw, tc.runtimeProfile)
			rewriteFileCheckSource(t, &raw, &raw.Execution[0], tc.execution)
			m := registryModel(t, selection, raw)
			agent, check := runtimePorts(t)
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
			if err != nil || result.Status() != tc.status {
				t.Fatalf("namespace handling: %v %s %+v", err, result.Status(), result.Diagnostics())
			}
			if tc.status == model.Passed {
				profiles := result.ProfileObservations()
				a, _ := profiles[0].Identity()
				b, _ := profiles[1].Identity()
				if a == b || profiles[0].Input().Name() != profiles[1].Input().Name() {
					t.Fatal("namespace identity collapsed")
				}
			}
		})
	}
}

func TestRuntimeMissingAndMultipleDocumentsAndFiniteLimits(t *testing.T) {
	for _, tc := range []string{"missing", "multiple", "bytes", "depth"} {
		t.Run(tc, func(t *testing.T) {
			selection, raw := runtimeSettingsInputs(t)
			want := model.NotCompleted
			switch tc {
			case "missing":
				selection.RuntimeProfiles, raw.RuntimeProfiles = nil, nil
			case "multiple":
				selection.RuntimeProfiles = append(selection.RuntimeProfiles, selection.RuntimeProfiles[0])
				raw.RuntimeProfiles = append(raw.RuntimeProfiles, raw.RuntimeProfiles[0])
				want = model.Failed
			case "bytes":
				addRuntimeDocument(t, &selection, &raw, "#"+strings.Repeat("x", 4<<20))
			case "depth":
				addRuntimeDocument(t, &selection, &raw, strings.Replace(testRuntimeDocument, "{profileVersion: 1}", "{nested: "+strings.Repeat("[", 70)+"0"+strings.Repeat("]", 70)+"}", 1))
			}
			m := registryModel(t, selection, raw)
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), nil, nil)
			if err != nil || result.Status() != want || len(result.ProfileObservations()) != 0 {
				t.Fatalf("finite/refused document: %v %s %+v", err, result.Status(), result.Diagnostics())
			}
		})
	}
}
