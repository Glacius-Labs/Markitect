package claudecode

import (
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

// ProfileFromPrepared projects captured claudeCode Runtime settings after
// complete static preparation. It performs no acquisition, credential
// resolution or runtime call; static preparation grants no execution
// authority. The image is the one observed during preparation.
func ProfileFromPrepared(prepared runtimeconfig.PreparedProfile) (Profile, error) {
	refuse := func(reason string) (Profile, error) {
		return Profile{}, fmt.Errorf("%w: %s", ErrSession, reason)
	}
	input, observation := prepared.Input(), prepared.Observation()
	if _, ok := prepared.Identity(); !ok || observation.Status != model.Passed || input.Namespace() != "agentProfiles" || input.Section() != "claudeCode" || input.Binding() == (runtimeconfig.ProfileBinding{}) || observation.InputBinding != input.Binding() {
		return refuse("complete source-bound Agent profile preparation is required")
	}
	p := preparation{input: input, out: runtimeconfig.ProfileObservation{Status: model.Passed}}
	config, ok := p.decode(input.Inline())
	if !ok {
		return refuse("captured Claude Code configuration is invalid")
	}
	image := observation.AgentImage
	if observation.Contract != (runtimeconfig.ProfileContract{Name: metadataContract, Version: 1}) || image == nil || image.Provider != runtimeconfig.AgentImageClaudeCode || image.ImageID != config.imageID || image.Metadata.Path != runtimeMetadataPath || image.CLIVersion != supportedClaudeVersion || image.AgentdVersion != config.agentd || image.RecordedCodexSchemaDigest != "" || !slices.Equal(observation.Instructions, config.instructions) {
		return refuse("prepared Claude Code contract, image or instruction binding differs from captured configuration")
	}
	profile := Profile{ImageID: image.ImageID, CredentialRef: config.credentialRef, MaxWireBytes: config.maxWireBytes, Session: config.session}
	if err := validateProfile(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}
