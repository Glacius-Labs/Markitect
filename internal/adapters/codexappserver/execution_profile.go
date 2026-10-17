package codexappserver

import (
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// SessionProfileFromPrepared projects captured v3 Host settings after complete
// static preparation. It performs no acquisition, credential resolution or
// runtime call. The caller binds the returned values and observed image ID into
// its admitted SessionPlan; static preparation grants no execution authority.
func SessionProfileFromPrepared(prepared apply.PreparedProfile) (profile run.Payload, credentialRef string, maxWireBytes int64, err error) {
	refuse := func(reason string) (run.Payload, string, int64, error) {
		return run.Payload{}, "", 0, fmt.Errorf("%w: %s", ErrSession, reason)
	}
	input, observation := prepared.Input(), prepared.Observation()
	if _, ok := prepared.Identity(); !ok || observation.Status != model.Passed || input.Namespace() != "agentProfiles" || input.Binding() == (apply.ProfileBinding{}) || observation.InputBinding != input.Binding() {
		return refuse("complete source-bound Agent profile preparation is required")
	}
	p := preparation{input: input, out: apply.ProfileObservation{Status: model.Passed}}
	config, ok := p.decode(input.Inline())
	if !ok {
		return refuse("captured Codex configuration is invalid")
	}
	if config.version != 3 {
		return refuse("profileVersion 2 is metadata-only; explicit profileVersion 3 execution settings are required")
	}
	image := observation.AgentImage
	if observation.Contract != (apply.ProfileContract{Name: "markitect.codex-app-server.image-metadata", Version: 3}) || image == nil || image.Provider != apply.AgentImageCodexAppServer || image.ImageID != config.imageID || image.Metadata.Path != runtimeMetadataPath || image.CLIVersion != supportedCodexVersion || image.AgentdVersion != config.agentd || image.RecordedCodexSchemaDigest != config.schema || !slices.Equal(observation.Instructions, config.instructions) {
		return refuse("prepared Codex contract, image or instruction binding differs from captured configuration")
	}
	profile, err = NewSessionProfile(config.session)
	if err != nil {
		return run.Payload{}, "", 0, err
	}
	return profile, config.credentialRef, config.maxWireBytes, nil
}
