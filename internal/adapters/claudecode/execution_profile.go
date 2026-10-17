package claudecode

import (
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// SessionProfileFromPrepared projects captured Claude v1 Host settings after complete
// static preparation. It performs no acquisition, credential resolution or
// runtime call. The caller binds the returned values and observed image ID into
// its admitted SessionPlan; static preparation grants no execution authority.
func SessionProfileFromPrepared(prepared apply.PreparedProfile, stepReadOnly bool) (profile run.Payload, credentialRef string, maxWireBytes int64, err error) {
	refuse := func(reason string) (run.Payload, string, int64, error) {
		return run.Payload{}, "", 0, fmt.Errorf("%w: %s", ErrSession, reason)
	}
	input, observation := prepared.Input(), prepared.Observation()
	if _, ok := prepared.Identity(); !ok || observation.Status != model.Passed || input.Namespace() != "agentProfiles" || input.Section() != "claudeCode" || input.Binding() == (apply.ProfileBinding{}) || observation.InputBinding != input.Binding() {
		return refuse("complete source-bound Agent profile preparation is required")
	}
	p := preparation{input: input, out: apply.ProfileObservation{Status: model.Passed}}
	config, ok := p.decode(input.Inline())
	if !ok {
		return refuse("captured Claude configuration is invalid")
	}
	image := observation.AgentImage
	if observation.Contract != (apply.ProfileContract{Name: "markitect.claude-code.image-metadata", Version: 1}) || image == nil || image.Provider != apply.AgentImageClaudeCode || image.ImageID != config.imageID || image.Metadata.Path != runtimeMetadataPath || image.CLIVersion != supportedClaudeVersion || image.AgentdVersion != config.agentd || image.RecordedCodexSchemaDigest != "" || !slices.Equal(observation.Instructions, config.instructions) {
		return refuse("prepared Claude contract, image or instruction binding differs from captured configuration")
	}
	if stepReadOnly {
		tools := []string{}
		for _, tool := range config.session.BuiltinTools {
			if tool == "Read" {
				tools = append(tools, tool)
			}
		}
		config.session.BuiltinTools = tools
	}
	profile, err = NewSessionProfile(config.session)
	if err != nil {
		return run.Payload{}, "", 0, err
	}
	return profile, config.credentialRef, config.maxWireBytes, nil
}
