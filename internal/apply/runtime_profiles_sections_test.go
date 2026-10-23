package apply_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const sectionRuntimeDocument = `apiVersion: markitect.runtime/v1
agentProfiles:
  library-agent:
    claudeCode: {profileVersion: 1}
checkProfiles:
  library-tests:
    container: {profileVersion: 1}
`

func imageID(c byte) string { return "sha256:" + strings.Repeat(string(c), 64) }

// A profile selects exactly one adapter section, and its image evidence must
// match that section: a Claude image for claudeCode, a check image for a
// container check. Neither is bound to Runtime files.
func TestProfilesSelectOneAdapterSectionAndBindItsImage(t *testing.T) {
	agentImage := func(provider apply.AgentImageProvider) *apply.AgentImageObservation {
		return &apply.AgentImageObservation{Provider: provider, ImageID: imageID('a'), Metadata: apply.AgentImageMetadataFile{Path: "/opt/markitect/agent.json", ContentID: snapshot.ContentID{1}, Size: 10}, CLIVersion: "2.1.0", AgentdVersion: "1.0.0"}
	}
	checkImage := &apply.CheckImageObservation{ImageID: imageID('c'), Executable: apply.AgentImageMetadataFile{Path: "/usr/local/go/bin/go", ContentID: snapshot.ContentID{2}, Size: 1 << 20}}
	for _, tc := range []struct {
		name         string
		document     string
		agent        *apply.AgentImageObservation
		check        *apply.CheckImageObservation
		runtimeFiles bool
		want         model.Status
	}{
		{"claude and container images", sectionRuntimeDocument, agentImage(apply.AgentImageClaudeCode), checkImage, false, model.Passed},
		{"codex image under claudeCode", sectionRuntimeDocument, agentImage(apply.AgentImageCodexAppServer), checkImage, false, model.Failed},
		{"container check without its image", sectionRuntimeDocument, agentImage(apply.AgentImageClaudeCode), nil, true, model.Failed},
		{"two sections", strings.Replace(sectionRuntimeDocument, "claudeCode: {profileVersion: 1}", "claudeCode: {profileVersion: 1}\n    codexAppServer: {profileVersion: 1}", 1), agentImage(apply.AgentImageClaudeCode), checkImage, false, model.Failed},
		{"unknown section", strings.Replace(sectionRuntimeDocument, "container:", "kubernetes:", 1), agentImage(apply.AgentImageClaudeCode), checkImage, false, model.Failed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := runtimeSettingsInputs(t)
			addRuntimeDocument(t, &selection, &raw, tc.document)
			sections := map[string]string{}
			agents := runtimeAgentPort(func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
				sections[input.Name()] = input.Section()
				return apply.ProfileObservation{InputBinding: input.Binding(), Status: model.Passed, Contract: apply.ProfileContract{Name: "claude-code-image", Version: 1}, AgentImage: tc.agent}, nil
			})
			checks := runtimeCheckPort(func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
				sections[input.Name()] = input.Section()
				o := apply.ProfileObservation{InputBinding: input.Binding(), Status: model.Passed, Contract: apply.ProfileContract{Name: "container-check", Version: 1}, CheckImage: tc.check}
				if tc.runtimeFiles {
					o.Files = testObservation(t, false).Files
				}
				return o, nil
			})
			m := registryModel(t, selection, raw)
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agents, checks)
			if err != nil || result.Status() != tc.want {
				t.Fatalf("status %s, want %s: %v %+v", result.Status(), tc.want, err, result.Diagnostics())
			}
			if tc.want == model.Passed && (sections["library-agent"] != "claudeCode" || sections["library-tests"] != "container") {
				t.Fatal("the ports did not see the selected sections", sections)
			}
			if tc.want == model.Passed {
				// Neither a returned observation nor the preparer's own pointer
				// can change the prepared binding.
				selections, _ := result.Checks()
				before := selections[0].Profile()
				identity, _ := before.Identity()
				returned := before.Observation()
				returned.CheckImage.ImageID = imageID('d')
				held := *tc.check
				defer func() { *tc.check = held }()
				tc.check.ImageID = imageID('e')
				after := selections[0].Profile()
				if id, _ := after.Identity(); id != identity || after.Observation().CheckImage.ImageID != imageID('c') {
					t.Fatal("prepared check image evidence changed", after.Observation().CheckImage)
				}
			}
		})
	}
}
