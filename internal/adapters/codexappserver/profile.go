package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var ErrSession = errors.New("Codex turn refused or incomplete")

// Profile is everything a turn needs from one captured Agent profile of the Runtime profile:
// the immutable image, the opaque credential reference the Orchestrator resolves, the
// per-message wire limit and the explicit execution settings.
type Profile struct {
	ImageID       string
	CredentialRef string
	MaxWireBytes  int64
	Session       SessionProfile
}

// SessionProfile is the closed execution profile, separate from the static
// image-metadata profile. No field is inferred from the Orchestrator environment.
type SessionProfile struct {
	Version               int     `json:"version"`
	Model                 string  `json:"model"`
	Effort                string  `json:"effort"`
	BaseInstructions      string  `json:"baseInstructions"`
	DeveloperInstructions string  `json:"developerInstructions"`
	Store                 string  `json:"store"`
	CPUs                  float64 `json:"cpus"`
	MemoryBytes           int64   `json:"memoryBytes"`
	Pids                  int     `json:"pids"`
	TmpBytes              int64   `json:"tmpBytes"`
	Network               string  `json:"network"`
}

func validateSessionProfile(profile SessionProfile) error {
	if profile.Version != 1 || strings.TrimSpace(profile.Model) == "" || !utf8.ValidString(profile.Model) || !utf8.ValidString(profile.BaseInstructions) || !utf8.ValidString(profile.DeveloperInstructions) || !oneOf(profile.Effort, "none", "minimal", "low", "medium", "high", "xhigh") || !volumeName.MatchString(profile.Store) || profile.CPUs <= 0 || math.IsNaN(profile.CPUs) || math.IsInf(profile.CPUs, 0) || profile.MemoryBytes <= 0 || profile.Pids <= 0 || profile.TmpBytes <= 0 || profile.Network == "host" || profile.Network != "" && !volumeName.MatchString(profile.Network) {
		return ErrSession
	}
	return nil
}

func validateProfile(p Profile) error {
	if !imageName.MatchString(p.ImageID) || p.CredentialRef == "" || p.MaxWireBytes <= 0 || p.MaxWireBytes > maxProtocolMessage {
		return ErrSession
	}
	return validateSessionProfile(p.Session)
}

// RuntimeCredential is resolved by the Orchestrator from exactly CredentialRef. Mode is
// api-key or dedicated-runtime-login. RuntimeLogin must originate from a
// dedicated runtime login, never a desktop login. Its refresh token is removed
// before a turn copy is installed; this adapter does not refresh credentials.
type RuntimeCredential struct {
	Mode         string
	APIKey       string
	RuntimeLogin json.RawMessage
}
type CredentialResolver func(context.Context, string) (RuntimeCredential, error)

// RuntimeSession is the container Session surface used by this adapter.
type RuntimeSession interface {
	Initialize(context.Context, frame.Initialize) error
	ClearPrivateHome(context.Context) error
	StartProvider(context.Context, frame.Start) (frame.Started, error)
	Stdout() io.Reader
	WriteInput([]byte) error
	ToolCalls() <-chan frame.ToolCall
	Reply(frame.ToolReply) error
	Stop(context.Context, int64) (frame.Exited, error)
	Close() error
}
type RuntimeStarter func(context.Context, container.ContainerSpec) (RuntimeSession, error)

var volumeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
var imageName = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func strictValue(data []byte, v any) error {
	// The codec's object reader rejects duplicate keys and malformed Unicode.
	if _, err := protocolObject(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return ErrSession
	}
	return nil
}

func turnSpec(name string, turn run.Turn, p Profile) container.ContainerSpec {
	s := p.Session
	return container.ContainerSpec{Name: name, Image: p.ImageID,
		Labels: map[string]string{"markitect.store": s.Store}, User: "1000:1000", Workdir: "/work",
		Mounts: append(workspaceMounts(turn), container.Mount{Volume: turn.Home, Target: "/home/agent"}),
		CPUs:   s.CPUs, MemoryBytes: s.MemoryBytes, Pids: s.Pids, TmpBytes: s.TmpBytes, Network: s.Network}
}

// workspaceMounts is the workspace volume at /work followed by the turn's
// overlays: each a directory of the same volume at its own path below /work.
func workspaceMounts(turn run.Turn) []container.Mount {
	mounts := []container.Mount{{Volume: turn.Workspace, Target: "/work", ReadOnly: turn.ReadOnly, NoCopy: true}}
	for _, m := range turn.Mounts {
		mounts = append(mounts, container.Mount{Volume: turn.Workspace, Target: "/work/" + m.Path, Subpath: m.Path, ReadOnly: !m.Writable, NoCopy: true})
	}
	return mounts
}

// validMounts reports whether every overlay path is a distinct clean relative
// directory path.
func validMounts(mounts []run.PathMount) bool {
	seen := map[string]bool{}
	for _, m := range mounts {
		if !container.ValidSubpath(m.Path) || seen[m.Path] {
			return false
		}
		seen[m.Path] = true
	}
	return true
}

func writable(mounts []run.PathMount) bool {
	return slices.ContainsFunc(mounts, func(m run.PathMount) bool { return m.Writable })
}

func providerArgv(mode string) []string {
	// Luna's model metadata can select V2 despite features.multi_agent=false.
	// agents.enabled=false overrides that metadata; explicit V2 must be off too.
	args := []string{"/usr/local/bin/codex", "app-server", "--disable", "apps", "--disable", "memories",
		"--disable", "multi_agent", "--disable", "multi_agent_v2", "-c", "agents.enabled=false",
		"--disable", "image_generation", "--disable", "browser_use", "--disable", "computer_use",
		"-c", `mcp_servers={markitect={command="/usr/local/bin/markitect-agentd",args=["mcp"]}}`,
		"-c", `cli_auth_credentials_store="file"`}
	if mode == "api-key" {
		return append(args, "-c", `model_provider="markitect"`, "-c", `model_providers.markitect={name="OpenAI API",base_url="https://api.openai.com/v1",env_key="OPENAI_API_KEY",wire_api="responses",requires_openai_auth=false}`)
	}
	return append(args, "-c", `model_provider="openai"`)
}
