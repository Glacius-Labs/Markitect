package projectrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"go.yaml.in/yaml/v3"
)

const maxRuntimeConfigBytes = 1 << 20

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var forbiddenEnvironmentPrefixes = []string{
	"GIT_", "SSH_", "AWS_", "AZURE_", "GH_", "GITHUB_", "MCP_",
}

var forbiddenEnvironmentNames = map[string]bool{
	"CODEX_HOME": true, "CLAUDE_CONFIG_DIR": true, "HOME": true,
	"USERPROFILE": true, "APPDATA": true, "LOCALAPPDATA": true,
}

// UnmarshalYAML requires a human-readable duration with a unit.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return fmt.Errorf("duration must be a string such as 30s or 5m")
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(node.Value))
	if err != nil || parsed <= 0 {
		return fmt.Errorf("duration must be a positive Go duration: %q", node.Value)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// LoadRuntime reads and strictly validates the project-owned runtime config.
// It never resolves or starts a provider. Isolation claims are rejected here
// until a host-controlled launcher supplies verified evidence.
func LoadRuntime(root string) (Runtime, error) {
	var config Runtime
	if strings.TrimSpace(root) == "" {
		return config, fmt.Errorf("project root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return config, fmt.Errorf("resolve project root: %w", err)
	}
	if err := rejectReparsePath(root, RuntimePath); err != nil {
		return config, err
	}
	path := filepath.Join(root, filepath.FromSlash(RuntimePath))
	file, err := os.Open(path)
	if err != nil {
		return config, fmt.Errorf("open %s: %w", RuntimePath, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return config, fmt.Errorf("stat %s: %w", RuntimePath, err)
	}
	if !info.Mode().IsRegular() {
		return config, fmt.Errorf("%s must be a regular file", RuntimePath)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRuntimeConfigBytes+1))
	if err != nil {
		return config, fmt.Errorf("read %s: %w", RuntimePath, err)
	}
	if len(data) > maxRuntimeConfigBytes {
		return config, fmt.Errorf("%s exceeds %d bytes", RuntimePath, maxRuntimeConfigBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("decode %s: %w", RuntimePath, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return config, fmt.Errorf("%s must contain exactly one YAML document", RuntimePath)
		}
		return config, fmt.Errorf("decode trailing YAML in %s: %w", RuntimePath, err)
	}
	if err := ValidateRuntime(config); err != nil {
		return config, err
	}
	return config, nil
}

// ValidateRuntime checks configuration shape and finite bounds independently
// of the project model. Executable identity is checked later by Fingerprint.
func ValidateRuntime(config Runtime) error {
	if config.APIVersion != APIVersion {
		return fmt.Errorf("runtime apiVersion must be %q", APIVersion)
	}
	if config.Mode != ModeControlledLocal && config.Mode != ModeIsolated {
		return fmt.Errorf("runtime mode must be %q or %q", ModeControlledLocal, ModeIsolated)
	}
	if config.RequireIsolation || config.Mode == ModeIsolated {
		return fmt.Errorf("requested isolated execution is unavailable: no verified host-controlled launcher is configured")
	}
	if len(config.Agents) == 0 {
		return fmt.Errorf("runtime must declare at least one manager agent")
	}
	for managerID, agent := range config.Agents {
		if strings.TrimSpace(managerID) == "" {
			return fmt.Errorf("runtime agent manager ID must not be empty")
		}
		if strings.TrimSpace(agent.Command) == "" {
			return fmt.Errorf("runtime agent %q must declare command", managerID)
		}
		if strings.TrimSpace(agent.Model) == "" || strings.TrimSpace(agent.ProviderVersion) == "" {
			return fmt.Errorf("runtime agent %q must declare model and providerVersion", managerID)
		}
		if agent.Timeout <= 0 || agent.Timeout > Duration(10*time.Minute) || agent.MaxStdoutBytes <= 0 || agent.MaxStderrBytes <= 0 {
			return fmt.Errorf("runtime agent %q must declare positive timeout (at most 10m) and output limits", managerID)
		}
		if agent.ModelOptions != nil {
			if _, err := json.Marshal(agent.ModelOptions); err != nil {
				return fmt.Errorf("runtime agent %q modelOptions must be JSON-compatible: %w", managerID, err)
			}
		}
		if agent.Pricing.InputMicrosPerMillion < 0 || agent.Pricing.OutputMicrosPerMillion < 0 ||
			(agent.Pricing.InputMicrosPerMillion == 0 && agent.Pricing.OutputMicrosPerMillion == 0) {
			return fmt.Errorf("runtime agent %q must declare nonnegative input/output pricing with at least one positive rate", managerID)
		}
		seen := map[string]bool{}
		for _, name := range agent.Environment {
			if !environmentName.MatchString(name) || seen[name] {
				return fmt.Errorf("runtime agent %q has an invalid or duplicate environment variable name %q", managerID, name)
			}
			upper := strings.ToUpper(name)
			if forbiddenEnvironmentNames[upper] {
				return fmt.Errorf("runtime agent %q may not inherit environment variable %q", managerID, name)
			}
			for _, prefix := range forbiddenEnvironmentPrefixes {
				if strings.HasPrefix(upper, prefix) {
					return fmt.Errorf("runtime agent %q may not inherit environment variable %q", managerID, name)
				}
			}
			seen[name] = true
		}
	}
	if config.Verifier != nil {
		if _, err := config.Verifier.AgentConfig(); err != nil {
			return fmt.Errorf("invalid runtime verifier: %w", err)
		}
	}
	if config.Review != nil {
		if len(config.Review.Agents) == 0 || config.Review.MaxRounds < 1 || config.Review.MaxRounds > 3 ||
			config.Review.MaxManagerRounds < 1 || config.Review.MaxManagerRounds > 3 {
			return fmt.Errorf("runtime review must declare reviewer agents and rounds within 1..3")
		}
		for managerID, agent := range config.Review.Agents {
			if strings.TrimSpace(managerID) == "" {
				return fmt.Errorf("runtime reviewer Manager ID must not be empty")
			}
			if _, err := agent.AgentConfig(); err != nil {
				return fmt.Errorf("invalid runtime reviewer %q: %w", managerID, err)
			}
		}
	}
	l := config.Limits
	if l.MaxDepth < 1 || l.MaxDepth > 32 || l.MaxStarts < 1 || l.MaxStarts > 256 ||
		l.MaxRetries < 0 || l.MaxRetries > 3 || l.MaxParallel < 1 || l.MaxParallel > 16 ||
		l.MaxDuration <= 0 || l.MaxDuration > Duration(24*time.Hour) || l.MaxCostMicros <= 0 ||
		l.MaxCandidateFileBytes <= 0 || l.MaxCandidateFileBytes > 8<<20 ||
		l.MaxCandidateBytes < l.MaxCandidateFileBytes || l.MaxCandidateBytes > 32<<20 {
		return fmt.Errorf("runtime limits must be finite and within supported bounds (depth 1..32, starts 1..256, retries 0..3, parallel 1..16, duration <=24h, candidate <=32 MiB)")
	}
	return nil
}

// AgentConfig builds the explicit subprocess config for one manager. The
// configured environment contains names only; agentexec resolves values.
func (a Agent) AgentConfig() (agentexec.Config, error) {
	if err := ValidateRuntime(Runtime{APIVersion: APIVersion, Mode: ModeControlledLocal,
		Agents: map[string]Agent{"manager": a},
		Limits: Limits{MaxDepth: 1, MaxStarts: 1, MaxParallel: 1, MaxDuration: Duration(time.Minute), MaxCostMicros: 1,
			MaxCandidateFileBytes: 1, MaxCandidateBytes: 1}}); err != nil {
		return agentexec.Config{}, err
	}
	var modelOptions []byte
	if a.ModelOptions != nil {
		encoded, err := json.Marshal(a.ModelOptions)
		if err != nil {
			return agentexec.Config{}, fmt.Errorf("encode modelOptions: %w", err)
		}
		modelOptions = encoded
	}
	env := append([]string{}, a.Environment...)
	sort.Strings(env)
	return agentexec.Config{
		Command: a.Command, Args: append([]string(nil), a.Args...), Model: a.Model,
		ModelOptions: modelOptions, ProviderVersion: a.ProviderVersion,
		Timeout: time.Duration(a.Timeout), MaxStdoutBytes: a.MaxStdoutBytes,
		MaxStderrBytes: a.MaxStderrBytes, RuntimeFiles: append([]agentexec.RuntimeFile(nil), a.RuntimeFiles...),
		EnvironmentAllowlist: &env,
	}, nil
}

func rejectReparsePath(root, relative string) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect project root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("project root must be a real directory")
	}
	current := root
	for _, segment := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) && current == filepath.Join(root, ".markitect") {
			return fmt.Errorf("%s does not exist", relative)
		}
		if statErr != nil {
			return fmt.Errorf("inspect %s: %w", relative, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s contains a symlink", relative)
		}
		if current != filepath.Join(root, filepath.FromSlash(relative)) && !info.IsDir() {
			return fmt.Errorf("%s parent must be a directory", relative)
		}
	}
	return nil
}
