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
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var ErrSession = errors.New("Codex session refused or incomplete")

// SessionProfile is the closed, admitted execution profile, separate from the
// static image-metadata profile. No field is inferred from the Host environment.
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

// NewSessionProfile validates explicit execution settings and binds their closed
// v1 JSON bytes for SessionAdmission.Profile. It does not prepare image metadata
// or infer any setting from the Host environment.
func NewSessionProfile(profile SessionProfile) (run.Payload, error) {
	if err := validateSessionProfile(profile); err != nil {
		return run.Payload{}, err
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return run.Payload{}, ErrSession
	}
	return run.NewPayload("application/vnd.markitect.codex-session-profile+json", "utf-8", data)
}

func validateSessionProfile(profile SessionProfile) error {
	if profile.Version != 1 || strings.TrimSpace(profile.Model) == "" || !utf8.ValidString(profile.Model) || !utf8.ValidString(profile.BaseInstructions) || !utf8.ValidString(profile.DeveloperInstructions) || !oneOf(profile.Effort, "none", "minimal", "low", "medium", "high", "xhigh") || !volumeName.MatchString(profile.Store) || profile.CPUs <= 0 || math.IsNaN(profile.CPUs) || math.IsInf(profile.CPUs, 0) || profile.MemoryBytes <= 0 || profile.Pids <= 0 || profile.TmpBytes <= 0 || profile.Network == "host" || profile.Network != "" && !volumeName.MatchString(profile.Network) {
		return ErrSession
	}
	return nil
}

// RuntimeCredential is resolved by the Host from exactly CredentialRef. Mode is
// api-key or dedicated-runtime-login. RuntimeLogin must originate from a
// dedicated runtime login, never a desktop login. Its refresh token is removed
// before a session copy is installed; this adapter does not refresh credentials.
type RuntimeCredential struct {
	Mode         string
	APIKey       string
	RuntimeLogin json.RawMessage
}
type CredentialResolver func(context.Context, string) (RuntimeCredential, error)

// RuntimeSession is the existing container Session surface used by this adapter.
// StartProvider/Stop own their internal control envelopes; these typed calls
// cannot establish a raw control-wire identity.
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

func readSessionProfile(a run.SessionAdmission) (SessionProfile, []frame.ToolSpec, error) {
	p := a.Plan
	if a.Receipt.BatchID == "" || a.Invocation == "" || !a.Profile.Valid() || a.Profile.Ref() != p.Profile || !a.Tools.Valid() || a.Tools.Ref() != p.Tools || !volumeName.MatchString(string(p.ID)) || !volumeName.MatchString(p.Workspace) || !volumeName.MatchString(p.ProviderHome) || p.Workspace == p.ProviderHome || !imageName.MatchString(p.ImageID) || p.CredentialRef == "" || p.Candidate == ([32]byte{}) || p.MaxWireBytes <= 0 || p.MaxWireBytes > maxProtocolMessage || (p.Role != "manager" && p.Role != "reviewer") || p.Role == "reviewer" && (!p.ReadOnly || p.Resume != "") {
		return SessionProfile{}, nil, ErrSession
	}
	if a.Profile.Ref().MediaType != "application/vnd.markitect.codex-session-profile+json" || a.Profile.Ref().Encoding != "utf-8" || a.Tools.Ref().MediaType != "application/vnd.markitect.tools+json" || a.Tools.Ref().Encoding != "utf-8" {
		return SessionProfile{}, nil, ErrSession
	}
	var profile SessionProfile
	if strictValue(a.Profile.Bytes(), &profile) != nil || validateSessionProfile(profile) != nil {
		return SessionProfile{}, nil, ErrSession
	}
	var catalogue struct {
		Version int              `json:"version"`
		Tools   []frame.ToolSpec `json:"tools"`
	}
	if strictValue(a.Tools.Bytes(), &catalogue) != nil || catalogue.Version != 1 || catalogue.Tools == nil {
		return SessionProfile{}, nil, ErrSession
	}
	seen := map[string]bool{}
	for _, t := range catalogue.Tools {
		if t.Name == "" || seen[t.Name] || strings.TrimSpace(t.Description) == "" {
			return SessionProfile{}, nil, ErrSession
		}
		if _, err := protocolObject(t.InputSchema); err != nil {
			return SessionProfile{}, nil, ErrSession
		}
		seen[t.Name] = true
	}
	return profile, catalogue.Tools, nil
}

func sessionSpec(p run.SessionPlan, profile SessionProfile) container.ContainerSpec {
	return container.ContainerSpec{Name: "markitect-" + string(p.ID), Image: p.ImageID,
		Labels: map[string]string{"markitect.store": profile.Store}, User: "1000:1000", Workdir: "/work",
		Mounts: []container.Mount{{Volume: p.Workspace, Target: "/work", ReadOnly: p.ReadOnly, NoCopy: true}, {Volume: p.ProviderHome, Target: "/home/agent"}},
		CPUs:   profile.CPUs, MemoryBytes: profile.MemoryBytes, Pids: profile.Pids, TmpBytes: profile.TmpBytes, Network: profile.Network}
}

func sessionArgv(mode string) []string {
	args := []string{"/usr/local/bin/codex", "app-server", "--disable", "apps", "--disable", "memories", "--disable", "multi_agent",
		"--disable", "image_generation", "--disable", "browser_use", "--disable", "computer_use",
		"-c", `mcp_servers={markitect={command="/usr/local/bin/markitect-agentd",args=["mcp"]}}`,
		"-c", `cli_auth_credentials_store="file"`}
	if mode == "api-key" {
		return append(args, "-c", `model_provider="markitect"`, "-c", `model_providers.markitect={name="OpenAI API",base_url="https://api.openai.com/v1",env_key="OPENAI_API_KEY",wire_api="responses",requires_openai_auth=false}`)
	}
	return append(args, "-c", `model_provider="openai"`)
}
