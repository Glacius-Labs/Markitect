package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var ErrSession = errors.New("Claude session refused or incomplete")

// SessionProfile is the closed, admitted execution profile, separate from the
// static image-metadata profile. No field is inferred from the Host environment.
type SessionProfile struct {
	Version      int      `json:"version"`
	Model        string   `json:"model"`
	Effort       string   `json:"effort"`
	SystemPrompt string   `json:"systemPrompt"`
	BuiltinTools []string `json:"builtinTools"`
	Store        string   `json:"store"`
	CPUs         float64  `json:"cpus"`
	MemoryBytes  int64    `json:"memoryBytes"`
	Pids         int      `json:"pids"`
	TmpBytes     int64    `json:"tmpBytes"`
	Network      string   `json:"network"`
}

// RuntimeCredential is resolved by the Host from exactly CredentialRef.
// Token must be a long-lived dedicated runtime Claude token, never desktop auth.
type RuntimeCredential struct{ Token string }
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

var volumeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
var imageName = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func strictValue(data []byte, v any) error {
	// The codec's object reader rejects duplicate keys and malformed Unicode.
	if err := validJSON(data); err != nil {
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
	if a.Receipt.BatchID == "" || a.Invocation == "" || !a.Profile.Valid() || a.Profile.Ref() != p.Profile || !a.Tools.Valid() || a.Tools.Ref() != p.Tools || !volumeName.MatchString(string(p.ID)) || !volumeName.MatchString(p.Workspace) || !volumeName.MatchString(p.ProviderHome) || p.Workspace == p.ProviderHome || !imageName.MatchString(p.ImageID) || p.CredentialRef == "" || p.Candidate == ([32]byte{}) || p.MaxWireBytes <= 0 || p.MaxWireBytes > maxLineBytes || (p.Role != "manager" && p.Role != "reviewer") || p.Role == "reviewer" && (!p.ReadOnly || p.Resume != "") {
		return SessionProfile{}, nil, ErrSession
	}
	var profile SessionProfile
	if strictValue(a.Profile.Bytes(), &profile) != nil || profile.Version != 1 || strings.TrimSpace(profile.Model) == "" || profile.Effort != "high" || !volumeName.MatchString(profile.Store) || profile.CPUs <= 0 || profile.MemoryBytes <= 0 || profile.Pids <= 0 || profile.TmpBytes <= 0 || profile.Network == "host" || profile.Network != "" && !volumeName.MatchString(profile.Network) {
		return SessionProfile{}, nil, ErrSession
	}
	var catalogue struct {
		Version int              `json:"version"`
		Tools   []frame.ToolSpec `json:"tools"`
	}
	if strictValue(a.Tools.Bytes(), &catalogue) != nil || catalogue.Version != 1 || catalogue.Tools == nil {
		return SessionProfile{}, nil, ErrSession
	}
	if a.Profile.Ref().MediaType != "application/vnd.markitect.claude-session-profile+json" || a.Profile.Ref().Encoding != "utf-8" || a.Tools.Ref().MediaType != "application/vnd.markitect.tools+json" || a.Tools.Ref().Encoding != "utf-8" || strings.TrimSpace(profile.SystemPrompt) == "" || profile.BuiltinTools == nil {
		return SessionProfile{}, nil, ErrSession
	}
	builtin := map[string]bool{}
	for _, name := range profile.BuiltinTools {
		if (name != "Read" && name != "Write") || builtin[name] || p.ReadOnly && name != "Read" {
			return SessionProfile{}, nil, ErrSession
		}
		builtin[name] = true
	}
	seen := map[string]bool{}
	for _, t := range catalogue.Tools {
		if !volumeName.MatchString(t.Name) || seen[t.Name] || strings.TrimSpace(t.Description) == "" {
			return SessionProfile{}, nil, ErrSession
		}
		if _, err := object(t.InputSchema); err != nil {
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

func sessionArgv(p run.SessionPlan, profile SessionProfile, tools []frame.ToolSpec) []string {
	allowed := append([]string{}, profile.BuiltinTools...)
	for _, tool := range tools {
		allowed = append(allowed, "mcp__markitect__"+tool.Name)
	}
	args := []string{"/usr/local/bin/claude", "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{"markitect":{"type":"stdio","command":"/usr/local/bin/markitect-agentd","args":["mcp"]}}}`,
		"--tools", strings.Join(profile.BuiltinTools, ","), "--allowedTools", strings.Join(allowed, ","), "--permission-mode", "dontAsk", "--model", profile.Model, "--effort", profile.Effort,
		"--setting-sources", "", "--system-prompt", profile.SystemPrompt}
	if p.Resume != "" {
		args = append(args, "--resume", string(p.Resume))
	}
	return args
}
