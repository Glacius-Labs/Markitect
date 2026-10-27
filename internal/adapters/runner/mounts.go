package runner

import (
	"fmt"
	"path"
	"strings"
)

// HostPath spells an absolute host path as the engine CLI's bind-mount source.
//
// On Linux and in WSL ("linux" and every other non-Windows goos) the engine
// resolves the path as given, so it must be a clean absolute POSIX path. A WSL
// launcher therefore names a Windows checkout as /mnt/c/...
//
// On Windows the engine CLI is Docker Desktop's docker.exe. Docker Desktop
// accepts a drive path and maps it into its Linux VM itself; the VM-side spelling
// (/run/desktop/mnt/host/c/... with the WSL 2 backend) depends on the backend and
// is never passed. HostPath returns the drive path in Docker Desktop's
// forward-slash form, C:\Users\me\repo becoming C:/Users/me/repo. UNC, device,
// drive-relative and relative paths are refused, and so are WSL paths
// (\\wsl$\...): launch from WSL for a checkout inside a distribution.
//
// Characters that would change the meaning of the engine's comma-separated
// --mount value are refused on every platform.
func HostPath(goos, p string) (string, error) {
	if p == "" || strings.ContainsAny(p, ",\"\x00\r\n") {
		return "", fmt.Errorf("host path %q cannot be passed as an engine mount source", p)
	}
	if goos == "windows" {
		return windowsHostPath(p)
	}
	if !strings.HasPrefix(p, "/") || path.Clean(p) != p || p == "/" {
		return "", fmt.Errorf("host path %q is not a clean absolute path below the root", p)
	}
	return p, nil
}

func windowsHostPath(p string) (string, error) {
	letter := len(p) > 0 && (p[0] >= 'A' && p[0] <= 'Z' || p[0] >= 'a' && p[0] <= 'z')
	if len(p) < 4 || !letter || p[1] != ':' || p[2] != '\\' && p[2] != '/' {
		return "", fmt.Errorf("host path %q is not an absolute Windows drive path below the drive root", p)
	}
	rest := strings.ReplaceAll(p[3:], `\`, "/")
	for _, element := range strings.Split(rest, "/") {
		if element == "" || element == "." || element == ".." || strings.ContainsAny(element, `:*?<>|`) {
			return "", fmt.Errorf("host path %q is not a clean Windows drive path", p)
		}
	}
	return strings.ToUpper(p[:1]) + ":/" + rest, nil
}

// Mount is one engine mount of the runner container.
type Mount struct {
	// Type is "bind" or "volume".
	Type string
	// Source is the engine-side spelling: a translated host path, the engine
	// socket or a volume name.
	Source   string
	Target   string
	ReadOnly bool
}

func (m Mount) arg() string {
	value := "type=" + m.Type + ",source=" + m.Source + ",target=" + m.Target
	if m.ReadOnly {
		value += ",readonly"
	}
	return value
}

// MountSpec names the host side of the runner's mounts.
type MountSpec struct {
	// Checkout is the owner's checkout, mounted read-only.
	Checkout string
	// Invocation holds the translated run configuration and runtime profile, mounted read-only.
	Invocation string
	// Credentials are the host login files, mounted read-only at CredentialPath(1..n).
	Credentials []string
	// Output is the writable hand-back area.
	Output string
	// StateVolume is the named engine volume that keeps the runner's Apply state.
	StateVolume string
}

// Mounts returns the runner's complete mount set. Host paths are spelled for
// goos. The engine socket is the engine's own path: sibling containers that
// Apply starts through it are created by the engine on its host, and they
// receive only named volumes, never a host path.
func Mounts(goos string, s MountSpec) ([]Mount, error) {
	var mounts []Mount
	bind := func(host, target string, readOnly bool) error {
		source, err := HostPath(goos, host)
		if err != nil {
			return err
		}
		mounts = append(mounts, Mount{Type: "bind", Source: source, Target: target, ReadOnly: readOnly})
		return nil
	}
	if err := bind(s.Checkout, CheckoutDir, true); err != nil {
		return nil, fmt.Errorf("checkout: %w", err)
	}
	if err := bind(s.Invocation, InvocationDir, true); err != nil {
		return nil, fmt.Errorf("invocation: %w", err)
	}
	if len(s.Credentials) == 0 {
		return nil, fmt.Errorf("the runner needs at least one dedicated login")
	}
	for i, login := range s.Credentials {
		// Errors never repeat a login path.
		if err := bind(login, CredentialPath(i+1), true); err != nil {
			return nil, fmt.Errorf("dedicated login %d cannot be passed as an engine mount source", i+1)
		}
	}
	mounts = append(mounts, Mount{Type: "bind", Source: EngineSocket, Target: EngineSocket})
	if err := bind(s.Output, OutputDir, false); err != nil {
		return nil, fmt.Errorf("output: %w", err)
	}
	if !engineNamePattern.MatchString(s.StateVolume) {
		return nil, fmt.Errorf("state volume %q is not a plain engine volume name", s.StateVolume)
	}
	mounts = append(mounts, Mount{Type: "volume", Source: s.StateVolume, Target: StateDir})
	return mounts, nil
}

// RunSpec is one runner container.
type RunSpec struct {
	// Name is the unique container name; Label marks it for cleanup sweeps.
	Name, Label string
	// Image is the runner image's engine image ID.
	Image string
	// Base is the checkout commit the runner must clone.
	Base string
	// Model and Dependencies are repository-relative manifest paths.
	Model        string
	Dependencies []string
	Mounts       []Mount
}

// RunArgs returns the engine arguments that start the runner. The runner has
// no network: it reaches the engine through its socket, and the Agent
// containers it starts get their network from the runtime profile.
func RunArgs(s RunSpec) ([]string, error) {
	if !engineNamePattern.MatchString(s.Name) || !engineNamePattern.MatchString(s.Label) {
		return nil, fmt.Errorf("runner container name or label is not a plain engine name")
	}
	if !imageIDPattern.MatchString(s.Image) {
		return nil, fmt.Errorf("runner image %q is not an engine image ID (sha256:<64 hex>)", s.Image)
	}
	if !validObjectID(s.Base) {
		return nil, fmt.Errorf("base %q is not a full commit ID", s.Base)
	}
	if s.Model != "" && !RepositoryRelative(s.Model, false) {
		return nil, fmt.Errorf("runner model %q is not repository-relative", s.Model)
	}
	args := []string{"run", "--rm", "--init", "--pull", "never",
		"--name", s.Name, "--label", "markitect.runner=" + s.Label,
		"--network", "none", "--security-opt", "no-new-privileges"}
	targets := map[string]bool{}
	for _, m := range s.Mounts {
		if m.Type != "bind" && m.Type != "volume" || targets[m.Target] {
			return nil, fmt.Errorf("runner mount %s is invalid or repeated", m.Target)
		}
		targets[m.Target] = true
		args = append(args, "--mount", m.arg())
	}
	args = append(args, s.Image, "--base", s.Base)
	if s.Model != "" {
		args = append(args, "--model", s.Model)
	}
	for _, dependency := range s.Dependencies {
		if !RepositoryRelative(dependency, false) {
			return nil, fmt.Errorf("runner dependency %q is not repository-relative", dependency)
		}
		args = append(args, "--dependency", dependency)
	}
	return args, nil
}
