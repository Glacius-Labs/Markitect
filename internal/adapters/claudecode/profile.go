package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var ErrSession = errors.New("Claude Code session refused or incomplete")

// Fixed container paths. The image installs the CLI and agentd; agentd's
// private initialization writes the instruction files into the private home.
const (
	claudeExecutable       = "/usr/local/bin/claude"
	privateHome            = "/home/agent"
	configDirectory        = privateHome + "/.claude"
	systemPromptPath       = privateHome + "/.markitect/system-prompt.md"
	appendSystemPromptPath = privateHome + "/.markitect/append-system-prompt.md"
)

// builtinTools is the fixed native tool set: shell and file tools. Native
// subagents, web access, skills and background work stay unavailable.
const builtinTools = "Bash,Read,Edit,Write,Glob,Grep"

// alwaysPresentTools are built-in tools that --tools cannot remove while MCP
// tools remain, as the CLI reference states for EndConversation.
var alwaysPresentTools = []string{"EndConversation"}

// Profile is everything a session needs from one captured Agent profile of
// the Runtime profile: the immutable image, the opaque credential reference
// the Orchestrator resolves, the per-line stream limit and the explicit
// execution settings.
type Profile struct {
	ImageID       string
	CredentialRef string
	MaxWireBytes  int64
	Session       SessionProfile
}

// SessionProfile is the closed execution profile. No field is inferred from
// the Orchestrator environment.
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

var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:\[\]-]{0,127}$`)

func validateSessionProfile(profile SessionProfile) error {
	if profile.Version != 1 || !modelName.MatchString(profile.Model) || !utf8.ValidString(profile.BaseInstructions) || !utf8.ValidString(profile.DeveloperInstructions) || !oneOf(profile.Effort, "low", "medium", "high", "xhigh", "max") || !volumeName.MatchString(profile.Store) || profile.CPUs <= 0 || math.IsNaN(profile.CPUs) || math.IsInf(profile.CPUs, 0) || profile.MemoryBytes <= 0 || profile.Pids <= 0 || profile.TmpBytes <= 0 || profile.Network == "host" || profile.Network != "" && !volumeName.MatchString(profile.Network) {
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

// RuntimeSession is the container Session surface used by this adapter. One
// container serves one Manager; each response starts a fresh provider process
// in it, and Stop collects that process's exit before the next one starts.
type RuntimeSession interface {
	ImageID() string
	Initialize(context.Context, frame.Initialize) error
	ClearPrivateHome(context.Context) error
	StartProvider(context.Context, frame.Start) (frame.Started, error)
	Stdout() io.Reader
	WriteInput([]byte) error
	CloseInput() error
	ToolCalls() <-chan frame.ToolCall
	Reply(frame.ToolReply) error
	Stop(context.Context, int64) (frame.Exited, error)
	Census(context.Context) ([]frame.Process, error)
	Close() error
}
type RuntimeStarter func(context.Context, container.ContainerSpec) (RuntimeSession, error)

var volumeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
var imageName = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// strictValue decodes one closed JSON object: no duplicate keys, malformed
// Unicode or unknown fields.
func strictValue(data []byte, v any) error {
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

func sessionSpec(name string, location run.CopyLocation, home string, p Profile) (container.ContainerSpec, error) {
	if !location.Valid() || location.Directory != "" || !volumeName.MatchString(location.Volume) || !container.ValidSubpath(location.Subpath) || !volumeName.MatchString(home) || home == location.Volume {
		return container.ContainerSpec{}, ErrSession
	}
	s := p.Session
	return container.ContainerSpec{Name: name, Image: p.ImageID, Labels: map[string]string{"markitect.store": s.Store, "markitect.native-session": name}, User: "1000:1000", Workdir: "/work", Mounts: []container.Mount{{Volume: location.Volume, Subpath: location.Subpath, Target: "/work", NoCopy: true}, {Volume: home, Target: privateHome}}, CPUs: s.CPUs, MemoryBytes: s.MemoryBytes, Pids: s.Pids, TmpBytes: s.TmpBytes, Network: s.Network}, nil
}

// mcpConfig names the session helper as the only MCP server.
const mcpConfig = `{"mcpServers":{"markitect":{"type":"stdio","command":"/usr/local/bin/markitect-agentd","args":["mcp"]}}}`

// providerArgv starts one non-interactive response. The first response
// creates the conversation under the adapter's session ID; every later one
// resumes it. The harness replaces the default system prompt; project
// settings, CLAUDE.md, skills, slash commands and other MCP servers are not
// loaded, and the permission mode lets native tools run without prompts in
// the container, which is the sandbox.
func providerArgv(session string, resume bool, p SessionProfile) []string {
	conversation := []string{"--session-id", session}
	if resume {
		conversation = []string{"--resume", session}
	}
	args := []string{claudeExecutable, "--print", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose"}
	args = append(args, conversation...)
	return append(args, "--model", p.Model, "--effort", p.Effort,
		"--system-prompt-file", systemPromptPath, "--append-system-prompt-file", appendSystemPromptPath,
		"--permission-mode", "bypassPermissions", "--permission-prompts", "none",
		"--tools", builtinTools, "--disable-slash-commands", "--setting-sources", "user",
		"--strict-mcp-config", "--mcp-config", mcpConfig)
}

// authStatusArgv asks the CLI which login it would use, as JSON.
func authStatusArgv() []string { return []string{claudeExecutable, "auth", "status"} }

// providerEnvironment is the fixed provider environment. It pins the private
// configuration directory and switches off CLAUDE.md and memory loading,
// background work, built-in subagents, MCP tool deferral, updates and other
// nonessential traffic, and asks Claude Code to strip credentials from the
// environments of the processes it starts. It carries no credential.
func providerEnvironment() []string {
	return []string{
		"HOME=" + privateHome,
		"CLAUDE_CONFIG_DIR=" + configDirectory,
		"CLAUDE_CODE_DISABLE_CLAUDE_MDS=1",
		"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
		"CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1",
		"CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS=1",
		"CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING=1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		"DISABLE_AUTOUPDATER=1",
		"CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1",
		"ENABLE_TOOL_SEARCH=false",
		// The tools' size annotation governs text results; this also lifts
		// the token cap should the annotation not apply.
		"MAX_MCP_OUTPUT_TOKENS=200000",
	}
}

// defaultToolTimeout is Claude Code's documented MCP tool timeout, set
// explicitly when a response has no deadline.
const defaultToolTimeout = 100_000_000 * time.Millisecond

// toolTimeout is the MCP_TOOL_TIMEOUT of one response. The longest host await
// is bounded by the Run deadline: Run cancels every served tool call when its
// context ends, and a response's context carries that deadline. The timeout
// lies a minute beyond it, so Claude Code never abandons a call the host is
// still serving; the Run deadline ends it first.
func toolTimeout(ctx context.Context, now time.Time) string {
	timeout := defaultToolTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = max(deadline.Sub(now), 0) + time.Minute
	}
	return "MCP_TOOL_TIMEOUT=" + strconv.FormatInt(timeout.Milliseconds(), 10)
}

// environmentNames returns only the names of KEY=VALUE entries, for evidence.
func environmentNames(env []string) []string {
	names := make([]string, len(env))
	for i, entry := range env {
		names[i], _, _ = strings.Cut(entry, "=")
	}
	return names
}
