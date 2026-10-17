package apply_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestHostEnvelopeRejectsInvalidStructureBeforeEveryAdapter(t *testing.T) {
	for _, tc := range []struct{ name, document string }{
		{"empty", ""},
		{"syntax", "apiVersion: [unterminated"},
		{"extra document", testHostDocument + "---\n{}\n"},
		{"unknown version", strings.Replace(testHostDocument, "markitect.host/v1", "markitect.host/v2", 1)},
		{"unknown root", testHostDocument + "unknown: true\n"},
		{"duplicate root", testHostDocument + "agentProfiles: {}\n"},
		{"missing namespace", "apiVersion: markitect.host/v1\nagentProfiles: {}\n"},
		{"wrong namespace type", strings.Replace(testHostDocument, "agentProfiles:", "agentProfiles: []\nunused:", 1)},
		{"wrong wrapper", strings.Replace(testHostDocument, "codexAppServer:", "process:", 1)},
		{"wrapper extras", strings.Replace(testHostDocument, "    codexAppServer:", "    unknown: true\n    codexAppServer:", 1)},
		{"inline type", strings.Replace(testHostDocument, "{profileVersion: 1}", "[]", 1)},
		{"duplicate profile", strings.Replace(testHostDocument, "checkProfiles:", "  library-agent: {codexAppServer: {profileVersion: 1}}\ncheckProfiles:", 1)},
		{"nested duplicate", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: {same: 1, same: 2}}", 1)},
		{"nested nonstring key", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: {1: value}}", 1)},
		{"nested sequence nonstring key", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: [{false: value}]}", 1)},
		{"null", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: null}", 1)},
		{"anchor", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: &anchor []}", 1)},
		{"alias", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: &anchor [], repeated: *anchor}", 1)},
		{"custom tag", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: !custom value}", 1)},
		{"standard tag", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: !!str value}", 1)},
		{"standard key tag", strings.Replace(testHostDocument, "{profileVersion: 1}", "{profileVersion: 1, nested: {!!str key: value}}", 1)},
		{"empty name", strings.Replace(testHostDocument, "library-agent:", "'':", 1)},
		{"blank name", strings.Replace(testHostDocument, "library-agent:", "'  ':", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := hostSettingsInputs(t)
			addHostDocument(t, &selection, &raw, tc.document)
			m := registryModel(t, selection, raw)
			calls := 0
			agent := hostAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
				calls++
				return testObservation(t, true), nil
			})
			check := hostCheckPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
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
				if strings.HasPrefix(diagnostic.Code, "host.") && diagnostic.Source.ContentID != raw.HostProfiles[0].File.ContentID() {
					t.Fatal("invalid document lost original content binding")
				}
			}
		})
	}
}

func TestHostNamespacesAreExactSeparateAndDoNotFallBack(t *testing.T) {
	for _, tc := range []struct {
		name, host, execution string
		status                model.Status
	}{
		{"same name", strings.Replace(testHostDocument, "library-tests", "library-agent", 1), strings.Replace(defaultsAbsentExecution(t), "profile: library-tests", "profile: library-agent", 1), model.Passed},
		{"agent missing", strings.Replace(testHostDocument, "library-agent:", "other-agent:", 1), defaultsAbsentExecution(t), model.Failed},
		{"check missing", strings.Replace(testHostDocument, "library-tests:", "other-check:", 1), defaultsAbsentExecution(t), model.Failed},
		{"case is exact", strings.Replace(testHostDocument, "library-agent:", "Library-agent:", 1), defaultsAbsentExecution(t), model.Failed},
		{"check namespace only", "apiVersion: markitect.host/v1\nagentProfiles: {}\ncheckProfiles:\n  library-agent: {process: {profileVersion: 1}}\n  library-tests: {process: {profileVersion: 1}}\n", defaultsAbsentExecution(t), model.Failed},
		{"agent namespace only", "apiVersion: markitect.host/v1\nagentProfiles:\n  library-agent: {codexAppServer: {profileVersion: 1}}\n  library-tests: {codexAppServer: {profileVersion: 1}}\ncheckProfiles: {}\n", defaultsAbsentExecution(t), model.Failed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := hostSettingsInputs(t)
			addHostDocument(t, &selection, &raw, tc.host)
			rewriteFileCheckSource(t, &raw, &raw.Execution[0], tc.execution)
			m := registryModel(t, selection, raw)
			agent, check := hostPorts(t)
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

func TestHostMissingAndMultipleDocumentsAndFiniteLimits(t *testing.T) {
	for _, tc := range []string{"missing", "multiple", "bytes", "depth"} {
		t.Run(tc, func(t *testing.T) {
			selection, raw := hostSettingsInputs(t)
			want := model.NotCompleted
			switch tc {
			case "missing":
				selection.HostProfiles, raw.HostProfiles = nil, nil
			case "multiple":
				selection.HostProfiles = append(selection.HostProfiles, selection.HostProfiles[0])
				raw.HostProfiles = append(raw.HostProfiles, raw.HostProfiles[0])
				want = model.Failed
			case "bytes":
				addHostDocument(t, &selection, &raw, "#"+strings.Repeat("x", 4<<20))
			case "depth":
				addHostDocument(t, &selection, &raw, strings.Replace(testHostDocument, "{profileVersion: 1}", "{nested: "+strings.Repeat("[", 70)+"0"+strings.Repeat("]", 70)+"}", 1))
			}
			m := registryModel(t, selection, raw)
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), nil, nil)
			if err != nil || result.Status() != want || len(result.ProfileObservations()) != 0 {
				t.Fatalf("finite/refused document: %v %s %+v", err, result.Status(), result.Diagnostics())
			}
		})
	}
}
