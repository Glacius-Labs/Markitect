package claudecode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

var ErrSession = errors.New("Claude turn refused or incomplete")

// Profile is one closed Agent profile of the Runtime profile: the immutable image, the exact
// credential reference, the wire limit and the explicit provider settings. No
// field is inferred from the Orchestrator environment.
type Profile struct {
	ImageID       string
	CredentialRef string
	MaxWireBytes  int64
	Session       SessionProfile
}

// SessionProfile is the closed provider and container configuration, separate
// from the static image-metadata profile.
type SessionProfile struct {
	Version      int      `json:"version" yaml:"version"`
	Model        string   `json:"model" yaml:"model"`
	Effort       string   `json:"effort" yaml:"effort"`
	SystemPrompt string   `json:"systemPrompt" yaml:"systemPrompt"`
	BuiltinTools []string `json:"builtinTools" yaml:"builtinTools"`
	Store        string   `json:"store" yaml:"store"`
	CPUs         float64  `json:"cpus" yaml:"cpus"`
	MemoryBytes  int64    `json:"memoryBytes" yaml:"memoryBytes"`
	Pids         int      `json:"pids" yaml:"pids"`
	TmpBytes     int64    `json:"tmpBytes" yaml:"tmpBytes"`
	Network      string   `json:"network" yaml:"network"`
}

var (
	volumeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	imageName  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	// builtinTools is the closed set a profile may offer, in the order Claude Code
	// names them.
	builtinTools = []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}
)

func (p Profile) validate() error {
	if !imageName.MatchString(p.ImageID) || p.CredentialRef == "" || p.MaxWireBytes <= 0 || p.MaxWireBytes > maxLineBytes {
		return ErrSession
	}
	return validateSessionProfile(p.Session)
}

func validateSessionProfile(profile SessionProfile) error {
	if profile.Version != 1 || !utf8.ValidString(profile.Model) || strings.TrimSpace(profile.Model) == "" ||
		!slices.Contains([]string{"low", "medium", "high"}, profile.Effort) ||
		!utf8.ValidString(profile.SystemPrompt) || strings.TrimSpace(profile.SystemPrompt) == "" || profile.BuiltinTools == nil ||
		!volumeName.MatchString(profile.Store) || profile.CPUs <= 0 || math.IsNaN(profile.CPUs) || math.IsInf(profile.CPUs, 0) ||
		profile.MemoryBytes <= 0 || profile.Pids <= 0 || profile.TmpBytes <= 0 ||
		profile.Network == "host" || profile.Network != "" && !volumeName.MatchString(profile.Network) {
		return ErrSession
	}
	seen := map[string]bool{}
	for _, name := range profile.BuiltinTools {
		if !slices.Contains(builtinTools, name) || seen[name] {
			return ErrSession
		}
		seen[name] = true
	}
	return nil
}

// ProfileFromPrepared projects captured Claude v1 Runtime settings after complete
// static preparation. It performs no acquisition, credential resolution or
// runtime call; static preparation grants no execution authority.
func ProfileFromPrepared(prepared apply.PreparedProfile) (Profile, error) {
	refuse := func(reason string) (Profile, error) {
		return Profile{}, fmt.Errorf("%w: %s", ErrSession, reason)
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
	profile := Profile{ImageID: config.imageID, CredentialRef: config.credentialRef, MaxWireBytes: config.maxWireBytes, Session: config.session}
	if err := profile.validate(); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

// RuntimeCredential is resolved by the Orchestrator from exactly CredentialRef.
// Token must be a long-lived dedicated runtime Claude token, never desktop auth.
type RuntimeCredential struct{ Token string }
type CredentialResolver func(context.Context, string) (RuntimeCredential, error)

// RuntimeSession is the existing container Session surface used by this adapter.
type RuntimeSession interface {
	StartProvider(context.Context, frame.Start) (frame.Started, error)
	Stdout() io.Reader
	WriteInput([]byte) error
	ToolCalls() <-chan frame.ToolCall
	Reply(frame.ToolReply) error
	Stop(context.Context, int64) (frame.Exited, error)
	Close() error
}
