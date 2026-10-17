package apply_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Controlled static assertions exercise Apply's shape/binding boundary. These
// tests inspect no real image, run no provider and establish no CLI support.
func imageObservation(t *testing.T, provider apply.AgentImageProvider) apply.ProfileObservation {
	t.Helper()
	raw := "codex=codex-cli 0.162.0\nclaude=2.1.301 (Claude Code)\ncodexSchema=sha256:" + strings.Repeat("a", 64) + "\nagentd=development\n"
	metadata := testMetadata(t, "unused-host-coordinate", "runtime.env", raw)
	image := &apply.AgentImageObservation{
		Provider: provider, ImageID: "sha256:" + strings.Repeat("b", 64),
		Metadata:   apply.AgentImageMetadataFile{Path: "/opt/markitect/runtime.env", ContentID: metadata.ContentID, Size: metadata.Size},
		CLIVersion: "codex-cli 0.162.0", AgentdVersion: "development",
		RecordedCodexSchemaDigest: "sha256:" + strings.Repeat("a", 64),
	}
	if provider == apply.AgentImageClaudeCode {
		image.CLIVersion = "2.1.301 (Claude Code)"
		image.RecordedCodexSchemaDigest = ""
	}
	return apply.ProfileObservation{Status: model.Passed, Contract: apply.ProfileContract{Name: "controlled-image-validation", Version: 1}, AgentImage: image}
}

func prepareImageProfile(t *testing.T, observation apply.ProfileObservation, portErr error) (apply.CapturedSettingsPreparation, error) {
	t.Helper()
	selection, raw := hostSettingsInputs(t)
	if observation.AgentImage != nil && observation.AgentImage.Provider == apply.AgentImageClaudeCode {
		// A Claude image belongs to a claudeCode section.
		addHostDocument(t, &selection, &raw, strings.Replace(testHostDocument, "codexAppServer:", "claudeCode:", 1))
	}
	m := registryModel(t, selection, raw)
	_, checks := hostPorts(t)
	agent := rawHostAgentPort(func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		observation.InputBinding = input.Binding()
		return observation, portErr
	})
	return m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, checks)
}

func imageIdentity(t *testing.T, observation apply.ProfileObservation) string {
	t.Helper()
	prepared, err := prepareImageProfile(t, observation, nil)
	if err != nil || prepared.Status() != model.Passed {
		t.Fatalf("image static preparation: %v %s %+v", err, prepared.Status(), prepared.Diagnostics())
	}
	profile := prepared.ProfileObservations()[0]
	identity, ok := profile.Identity()
	if !ok || profile.Observation().AgentImage == nil || len(profile.Observation().Files) != 0 || len(profile.Observation().Inventories) != 0 {
		t.Fatal("image evidence required fabricated executable/inventory metadata")
	}
	return identity
}

func TestAgentImageStaticEvidenceAndEveryLiteralAffectIdentity(t *testing.T) {
	base := imageIdentity(t, imageObservation(t, apply.AgentImageCodexAppServer))
	for _, field := range []string{"provider", "contract", "contract version", "image ID", "metadata location", "metadata bytes", "metadata size", "CLI literal", "agentd literal", "recorded schema"} {
		t.Run(field, func(t *testing.T) {
			o := imageObservation(t, apply.AgentImageCodexAppServer)
			i := o.AgentImage
			switch field {
			case "provider":
				o = imageObservation(t, apply.AgentImageClaudeCode)
			case "contract":
				o.Contract.Name += "/another"
			case "contract version":
				o.Contract.Version++
			case "image ID":
				i.ImageID = "sha256:" + strings.Repeat("c", 64)
			case "metadata location":
				i.Metadata.Path = "/other/runtime.env"
			case "metadata bytes":
				i.Metadata.ContentID = testMetadata(t, "unused", "runtime.env", "other actual bytes").ContentID
			case "metadata size":
				i.Metadata.Size++
			case "CLI literal":
				i.CLIVersion = "adapter-supported-literal/42"
			case "agentd literal":
				i.AgentdVersion += " with recorded build"
			case "recorded schema":
				i.RecordedCodexSchemaDigest = "sha256:" + strings.Repeat("d", 64)
			}
			if got := imageIdentity(t, o); got == base {
				t.Fatalf("comparison identity omitted %s", field)
			}
		})
	}
	claude := imageObservation(t, apply.AgentImageClaudeCode)
	first := imageIdentity(t, claude)
	// The metadata file still binds Codex's recorded value for a Claude image,
	// even though its typed evidence makes no Codex schema assertion.
	claude.AgentImage.Metadata.ContentID = testMetadata(t, "unused", "runtime.env", "changed other recorded metadata").ContentID
	if imageIdentity(t, claude) == first {
		t.Fatal("whole metadata file binding was omitted for Claude")
	}
}

func TestAgentImageRejectsAmbiguousAndMalformedSuccessfulEvidence(t *testing.T) {
	for _, kind := range []string{"empty image", "unknown provider", "mixed files", "mixed inventory", "image tag", "uppercase image digest", "relative metadata", "metadata traversal", "empty metadata digest", "empty metadata file", "oversize metadata", "blank CLI", "control in version", "invalid UTF8 literal", "oversize literal", "control in contract", "missing recorded schema", "Claude Codex schema"} {
		t.Run(kind, func(t *testing.T) {
			o := imageObservation(t, apply.AgentImageCodexAppServer)
			i := o.AgentImage
			want := model.Failed
			switch kind {
			case "empty image":
				*i = apply.AgentImageObservation{}
			case "unknown provider":
				i.Provider = "another-provider"
			case "mixed files":
				o.Files = testObservation(t, true).Files
			case "mixed inventory":
				o.Inventories = testObservation(t, true).Inventories
			case "image tag":
				i.ImageID = "runtime:latest"
			case "uppercase image digest":
				i.ImageID = "sha256:" + strings.Repeat("B", 64)
			case "relative metadata":
				i.Metadata.Path = "opt/markitect/runtime.env"
			case "metadata traversal":
				i.Metadata.Path = "/opt/../runtime.env"
			case "empty metadata digest":
				i.Metadata.ContentID = snapshot.ContentID{}
			case "empty metadata file":
				i.Metadata.Size = 0
			case "oversize metadata":
				i.Metadata.Size = (1 << 20) + 1
				want = model.NotCompleted
			case "blank CLI":
				i.CLIVersion = "  "
			case "control in version":
				i.AgentdVersion = "version\nother"
			case "invalid UTF8 literal":
				i.CLIVersion = string([]byte{0xff})
			case "oversize literal":
				i.CLIVersion = strings.Repeat("v", 4097)
			case "control in contract":
				o.Contract.Name += "\x00"
			case "missing recorded schema":
				i.RecordedCodexSchemaDigest = ""
			case "Claude Codex schema":
				i.Provider = apply.AgentImageClaudeCode
			}
			prepared, err := prepareImageProfile(t, o, nil)
			if err != nil || prepared.Status() != want {
				t.Fatalf("invalid image evidence: %s %v %+v", prepared.Status(), err, prepared.Diagnostics())
			}
			if _, ok := prepared.ProfileObservations()[0].Identity(); ok {
				t.Fatal("rejected observation acquired a profile identity")
			}
			if _, ok := prepared.Settings(); ok {
				t.Fatal("rejected image observation exposed complete settings")
			}
		})
	}
}

func TestAgentImageCannotBecomeCheckEvidenceOrEchoAnotherRequest(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	agent, _ := hostPorts(t)
	check := hostCheckPort(func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		return imageObservation(t, apply.AgentImageCodexAppServer), nil
	})
	prepared, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	if err != nil || prepared.Status() != model.Failed {
		t.Fatal("Check accepted Agent image shape", err)
	}
	if _, ok := prepared.Checks(); ok {
		t.Fatal("Agent image claimed Check readiness")
	}
	first, _ := prepareImageProfile(t, imageObservation(t, apply.AgentImageCodexAppServer), nil)
	cached := first.ProfileObservations()[0].Observation()
	for _, missing := range []bool{false, true} {
		otherRaw, otherSelection := m.Inputs().Raw, m.Request().Selection
		addHostDocument(t, &otherSelection, &otherRaw, "# Different request\n"+testHostDocument)
		other := registryModel(t, otherSelection, otherRaw)
		o := cached
		if missing {
			o.InputBinding = apply.ProfileBinding{}
		}
		_, checks := hostPorts(t)
		port := rawHostAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) { return o, nil })
		got, err := other.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), other.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), port, checks)
		if err != nil || got.Status() != model.Failed {
			t.Fatal("image observation echoed a different or absent request", err)
		}
		if _, ok := got.ProfileObservations()[0].Identity(); ok {
			t.Fatal("bad request echo acquired an identity")
		}
	}
}

func TestAgentImageRetainsUnreadyPartialObservationAndOwnsCopies(t *testing.T) {
	failure := errors.New("image metadata read incomplete")
	partial := imageObservation(t, apply.AgentImageCodexAppServer)
	partial.AgentImage = &apply.AgentImageObservation{ImageID: partial.AgentImage.ImageID}
	prepared, err := prepareImageProfile(t, partial, failure)
	if !errors.Is(err, failure) || prepared.Status() != model.NotCompleted {
		t.Fatalf("partial image observation: %v %s", err, prepared.Status())
	}
	p := prepared.ProfileObservations()[0]
	if _, ok := p.Identity(); ok || !reflect.DeepEqual(p.Observation().AgentImage, partial.AgentImage) {
		t.Fatal("partial observation was lost or became ready")
	}
	observation := imageObservation(t, apply.AgentImageCodexAppServer)
	complete, err := prepareImageProfile(t, observation, nil)
	if err != nil || complete.Status() != model.Passed {
		t.Fatal(err)
	}
	before := complete.ProfileObservations()[0]
	identity, _ := before.Identity()
	retained := before.Observation()
	observation.AgentImage.CLIVersion = "mutated port observation"
	returned := before.Observation()
	returned.AgentImage.Metadata.Path = "/mutated/runtime.env"
	if got := before.Observation(); !reflect.DeepEqual(got, retained) {
		t.Fatal("image evidence aliases port or accessor pointers")
	}
	if got, _ := complete.ProfileObservations()[0].Identity(); got != identity {
		t.Fatal("immutable comparison identity changed through an alias")
	}
}

func TestAgentImageProfileBindsAndRestoresThroughExistingRUNCodec(t *testing.T) {
	selection, actual, host := settingsServiceInputs(t)
	selection = completeSettingsServiceSelection(t, selection, host)
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		return captured(preparationObservation(t, request.Selection, actual)), nil
	}}
	agent := rawHostAgentPort(func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		o := imageObservation(t, apply.AgentImageCodexAppServer)
		o.InputBinding = input.Binding()
		return o, nil
	})
	_, checks := hostPorts(t)
	prepared, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, checks)
	if err != nil {
		t.Fatal(err)
	}
	facts := runBindingHost(t)
	binding, err := prepared.BindRun(facts)
	if err != nil {
		t.Fatal("image identity did not flow through actual captured binding", err)
	}
	c, _ := run.NewCoordinator(facts.RepositoryIdentity)
	j := &controlJournal{}
	controlCommit(t, c, j, run.PrepareCommand("image-prepare", "assignment", "run", "", time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC), binding))
	restored, err := run.Restore(context.Background(), j, facts.RepositoryIdentity, controlOptions.RestoreLimits)
	if err != nil || restored.Current() != c.Current() || restored.Current().Basis != binding.Ref() {
		t.Fatal("new profile identity required a RUN wire change", err)
	}
}
