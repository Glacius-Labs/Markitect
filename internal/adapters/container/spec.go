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

// Mount gives a container one named volume, or with Subpath one directory of
// it. A Subpath directory must exist in the volume when the container starts.
type Mount struct {
	Volume   string
	Target   string
	Subpath  string
	ReadOnly bool
	// NoCopy prevents Docker from populating an empty volume with files from
	// the image at Target. Use it when the volume already represents exact bytes.
	NoCopy bool
}

// ContainerSpec describes one session or helper container. Every container
// runs as a non-root user with a read-only root filesystem, no capabilities
// and no privilege escalation; only its mounts and /tmp are writable.
type ContainerSpec struct {
	// Name is the unique container name.
	Name string
	// Image is the engine image ID, the content digest "sha256:...".
	Image string
	// Labels mark the container, at least with its Runtime store.
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

// HoldUser runs the init process of a held container: a user apart from the
// Agents', so their processes can neither signal nor trace it.
const HoldUser = "65534:65534"

// runArgs validates the spec and returns the engine's run arguments for the
// entry command: an absolute path inside the image and its arguments.
func (s ContainerSpec) runArgs(entry []string) ([]string, error) {
	return s.args(false, entry)
}

// args builds the run arguments of a session container or, when held, of a
// detached reusable container whose own init process is entry, run as
// HoldUser; the Agents' sessions join it through exec as the spec's user.
func (s ContainerSpec) args(held bool, entry []string) ([]string, error) {
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
		return nil, fmt.Errorf("container %s needs at least its Runtime store label", s.Name)
	}
	if s.Network != "" && (!namePattern.MatchString(s.Network) || s.Network == "host") {
		return nil, fmt.Errorf("network %q is not a plain isolated engine network", s.Network)
	}
	args := []string{"run", "--interactive", "--rm", "--init"}
	user := s.User
	if held {
		args, user = []string{"run", "--detach", "--rm"}, HoldUser
	}
	args = append(args,
		"--name", s.Name,
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--user", user,
		// Executable on purpose: the workspace is executable anyway, and
		// toolchains such as Go run their build products from /tmp.
		"--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size="+strconv.FormatInt(s.TmpBytes, 10),
		"--workdir", s.Workdir,
		"--cpus", strconv.FormatFloat(s.CPUs, 'f', -1, 64),
		"--memory", strconv.FormatInt(s.MemoryBytes, 10),
		"--pids-limit", strconv.Itoa(s.Pids),
	)
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
		if m.Subpath != "" {
			if !ValidSubpath(m.Subpath) {
				return nil, fmt.Errorf("volume subpath %q is not a plain relative path", m.Subpath)
			}
			mount += ",volume-subpath=" + m.Subpath
		}
		if m.NoCopy {
			mount += ",volume-nocopy"
		}
		if m.ReadOnly {
			mount += ",readonly"
		}
		args = append(args, "--mount", mount)
	}
	if len(entry) == 0 || !absolute(entry[0]) {
		return nil, fmt.Errorf("container %s needs an absolute entry command", s.Name)
	}
	args = append(args, "--entrypoint", entry[0], s.Image)
	return append(args, entry[1:]...), nil
}

// ValidSubpath reports whether p is a clean relative path inside a volume: not
// empty, not ".", no "..", no empty or backslash-separated elements and nothing
// that could add engine mount options.
func ValidSubpath(p string) bool {
	if p == "" || p == "." || p == ".." || strings.HasPrefix(p, "../") || path.IsAbs(p) || path.Clean(p) != p {
		return false
	}
	return !strings.ContainsAny(p, `\,=`+"\x00\n\r")
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
