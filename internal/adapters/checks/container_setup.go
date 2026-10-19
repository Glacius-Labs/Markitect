package checks

import (
	"errors"
	"path"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
)

// ContainerHostProfile is the explicit content of one container Check Host
// profile. Setup writes it into the closed section that the metadata decoder
// reads back.
type ContainerHostProfile struct {
	EngineExecutable string
	Profile          ContainerProfile
}

// ContainerHostLimits are the resource bounds of a container check.
type ContainerHostLimits struct {
	MilliCPUs   int64 `yaml:"milliCPUs"`
	MemoryBytes int64 `yaml:"memoryBytes"`
	Processes   int64 `yaml:"processes"`
}

// ContainerHostSection is the closed `container` section of a Host document in
// the field order and spelling of the decoder.
type ContainerHostSection struct {
	ProfileVersion   int                 `yaml:"profileVersion"`
	EngineExecutable string              `yaml:"engineExecutable"`
	ImageID          string              `yaml:"imageID"`
	Executable       string              `yaml:"executable"`
	ArgumentPrefix   []string            `yaml:"argumentPrefix"`
	Env              []string            `yaml:"env"`
	Limits           ContainerHostLimits `yaml:"limits"`
	MaxOutputBytes   int64               `yaml:"maxOutputBytes"`
}

// Section checks the basic shape of the profile and returns its Host document
// section. Lists are never null. It performs no engine or image access; the
// metadata preparation that follows observes the image and the executable.
func (p ContainerHostProfile) Section() (ContainerHostSection, error) {
	c := p.Profile
	if !filepath.IsAbs(p.EngineExecutable) || filepath.Clean(p.EngineExecutable) != p.EngineExecutable {
		return ContainerHostSection{}, errors.New("container check profile: the engine executable requires a clean absolute native path")
	}
	if !containerImageID(c.ImageID) {
		return ContainerHostSection{}, errors.New("container check profile: an immutable lowercase sha256 engine image ID is required")
	}
	if len(c.Executable) > 256 || !strings.HasPrefix(c.Executable, "/") || c.Executable == "/" || path.Clean(c.Executable) != c.Executable || strings.ContainsAny(c.Executable, "\\\x00") {
		return ContainerHostSection{}, errors.New("container check profile: the executable requires a clean absolute POSIX image file path of at most 256 bytes")
	}
	if c.Limits.MilliCPUs <= 0 || c.Limits.MemoryBytes <= 0 || c.Limits.Processes <= 0 || c.MaxOutputBytes <= 0 || c.MaxOutputBytes > container.MaxCommandOutputBytes {
		return ContainerHostSection{}, errors.New("container check profile: resource and output limits must be positive and within their bounds")
	}
	for _, entry := range c.Env {
		if key, _, ok := strings.Cut(entry, "="); !ok || !containerEnvKey(key) {
			return ContainerHostSection{}, errors.New("container check profile: environment requires literal KEY=value entries")
		}
	}
	section := ContainerHostSection{
		ProfileVersion: 1, EngineExecutable: p.EngineExecutable, ImageID: c.ImageID, Executable: c.Executable,
		ArgumentPrefix: append([]string{}, c.ArgumentPrefix...), Env: append([]string{}, c.Env...),
		Limits:         ContainerHostLimits{MilliCPUs: c.Limits.MilliCPUs, MemoryBytes: c.Limits.MemoryBytes, Processes: c.Limits.Processes},
		MaxOutputBytes: c.MaxOutputBytes,
	}
	return section, nil
}
