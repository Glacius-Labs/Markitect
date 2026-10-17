package container

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// AgentdPath is where the runtime image installs the session helper.
const AgentdPath = "/usr/local/bin/markitect-agentd"

var (
	namePattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	imageIDPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	// containerIDPattern matches the full container ID an engine writes.
	containerIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	labelPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,127}$`)
	userPattern        = regexp.MustCompile(`^([1-9][0-9]{0,9}):([1-9][0-9]{0,9})$`)
)

// Mount gives a container one named volume.
type Mount struct {
	Volume   string
	Target   string
	ReadOnly bool
}

// SessionSpec describes one session container. Every container runs agentd
// as a non-root user with a read-only root filesystem, no capabilities and
// no privilege escalation; only its mounts and /tmp are writable.
type SessionSpec struct {
	// Name is the unique container name.
	Name string
	// Image is the engine image ID, the content digest "sha256:...".
	Image string
	// Labels mark the container, at least with its Host store.
	Labels map[string]string
	// User is the numeric "uid:gid"; uid 0 and gid 0 are refused.
	User string
	// Mounts are the only volumes the container receives.
	Mounts []Mount
	// Workdir is the provider's working directory inside the container.
	Workdir string
	// CPUs, MemoryBytes and Pids bound the container. Each must be positive.
	CPUs        float64
	MemoryBytes int64
	Pids        int
	// TmpBytes bounds the writable temporary directory.
	TmpBytes int64
	// Network names the engine network; empty selects the engine default.
	// The host network is refused.
	Network string
}

// runArgs validates the spec and returns the engine's run arguments.
func (s SessionSpec) runArgs() ([]string, error) {
	if !namePattern.MatchString(s.Name) {
		return nil, fmt.Errorf("container name %q is not a plain engine name", s.Name)
	}
	if !imageIDPattern.MatchString(s.Image) {
		return nil, fmt.Errorf("image %q is not an engine image ID (sha256:<64 hex>)", s.Image)
	}
	if !userPattern.MatchString(s.User) {
		return nil, fmt.Errorf("user %q is not a numeric uid:gid outside the root user and group", s.User)
	}
	if !absolute(s.Workdir) {
		return nil, fmt.Errorf("working directory %q is not an absolute container path", s.Workdir)
	}
	if s.CPUs <= 0 || s.MemoryBytes <= 0 || s.Pids <= 0 || s.TmpBytes <= 0 {
		return nil, fmt.Errorf("container %s needs positive CPU, memory, process and temporary-space limits", s.Name)
	}
	if len(s.Labels) == 0 {
		return nil, fmt.Errorf("container %s needs at least its Host store label", s.Name)
	}
	if s.Network != "" && (!namePattern.MatchString(s.Network) || s.Network == "host") {
		return nil, fmt.Errorf("network %q is not a plain isolated engine network", s.Network)
	}
	args := []string{
		"run", "--interactive", "--rm", "--init",
		"--name", s.Name,
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--user", s.User,
		"--tmpfs", "/tmp:rw,nosuid,nodev,mode=1777,size=" + strconv.FormatInt(s.TmpBytes, 10),
		"--workdir", s.Workdir,
		"--cpus", strconv.FormatFloat(s.CPUs, 'f', -1, 64),
		"--memory", strconv.FormatInt(s.MemoryBytes, 10),
		"--pids-limit", strconv.Itoa(s.Pids),
	}
	if s.Network != "" {
		args = append(args, "--network", s.Network)
	}
	labels, err := labelArgs(s.Labels)
	if err != nil {
		return nil, err
	}
	args = append(args, labels...)
	targets := map[string]bool{"/tmp": true}
	for _, m := range s.Mounts {
		if !namePattern.MatchString(m.Volume) {
			return nil, fmt.Errorf("volume %q is not a plain engine name", m.Volume)
		}
		if !absolute(m.Target) || strings.ContainsAny(m.Target, ",=") {
			return nil, fmt.Errorf("mount target %q is not a plain absolute container path", m.Target)
		}
		if targets[m.Target] {
			return nil, fmt.Errorf("mount target %q is used twice", m.Target)
		}
		targets[m.Target] = true
		mount := "type=volume,src=" + m.Volume + ",dst=" + m.Target
		if m.ReadOnly {
			mount += ",readonly"
		}
		args = append(args, "--mount", mount)
	}
	return append(args, "--entrypoint", AgentdPath, s.Image, "serve"), nil
}

func labelArgs(labels map[string]string) ([]string, error) {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var args []string
	for _, k := range keys {
		v := labels[k]
		if !labelPattern.MatchString(k) || strings.ContainsAny(v, "\n\r\x00") {
			return nil, fmt.Errorf("label %q is not a plain engine label", k)
		}
		args = append(args, "--label", k+"="+v)
	}
	return args, nil
}

func absolute(p string) bool {
	return strings.HasPrefix(p, "/") && path.Clean(p) == p
}
