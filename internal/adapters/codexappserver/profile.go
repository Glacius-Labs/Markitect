package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

var ErrSession = errors.New("Codex turn refused or incomplete")

// Profile is everything a turn needs from one captured Host Agent profile:
// the immutable image, the opaque credential reference the Host resolves, the
// per-message wire limit and the explicit execution settings.
type Profile struct {
	ImageID       string
	CredentialRef string
	MaxWireBytes  int64
	Session       SessionProfile
}

// SessionProfile is the closed execution profile, separate from the static
// image-metadata profile. No field is inferred from the Host environment.
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

// RuntimeCredential is resolved by the Host from exactly CredentialRef. Mode is
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
	StartProvider(context.Context, frame.Start) (frame.Started, error)
	Stdout() io.Reader
	WriteInput([]byte) error
	ToolCalls() <-chan frame.ToolCall
	Reply(frame.ToolReply) error
	Stop(context.Context, int64) (frame.Exited, error)
	Close() error
}
type RuntimeStarter func(context.Context, container.ContainerSpec) (RuntimeSession, error)
type RuntimeHelper func(context.Context, container.ContainerSpec, []string, io.Reader, io.Writer) error

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

func turnSpec(name string, workspace, home string, readOnly bool, p Profile) container.ContainerSpec {
	s := p.Session
	return container.ContainerSpec{Name: name, Image: p.ImageID,
		Labels: map[string]string{"markitect.store": s.Store}, User: "1000:1000", Workdir: "/work",
		Mounts: []container.Mount{{Volume: workspace, Target: "/work", ReadOnly: readOnly, NoCopy: true}, {Volume: home, Target: "/home/agent"}},
		CPUs:   s.CPUs, MemoryBytes: s.MemoryBytes, Pids: s.Pids, TmpBytes: s.TmpBytes, Network: s.Network}
}

func providerArgv(mode string) []string {
	args := []string{"/usr/local/bin/codex", "app-server", "--disable", "apps", "--disable", "memories", "--disable", "multi_agent",
		"--disable", "image_generation", "--disable", "browser_use", "--disable", "computer_use",
		"-c", `mcp_servers={markitect={command="/usr/local/bin/markitect-agentd",args=["mcp"]}}`,
		"-c", `cli_auth_credentials_store="file"`}
	if mode == "api-key" {
		return append(args, "-c", `model_provider="markitect"`, "-c", `model_providers.markitect={name="OpenAI API",base_url="https://api.openai.com/v1",env_key="OPENAI_API_KEY",wire_api="responses",requires_openai_auth=false}`)
	}
	return append(args, "-c", `model_provider="openai"`)
}
