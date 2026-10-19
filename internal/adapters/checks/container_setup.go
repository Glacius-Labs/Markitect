package checks

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

// Section returns the Host document section of the profile. Lists are never
// null. It validates nothing: the metadata decoder owns every rule, and setup
// runs the written document through it before any file is written.
func (p ContainerHostProfile) Section() ContainerHostSection {
	c := p.Profile
	return ContainerHostSection{
		ProfileVersion: 1, EngineExecutable: p.EngineExecutable, ImageID: c.ImageID, Executable: c.Executable,
		ArgumentPrefix: append([]string{}, c.ArgumentPrefix...), Env: append([]string{}, c.Env...),
		Limits:         ContainerHostLimits{MilliCPUs: c.Limits.MilliCPUs, MemoryBytes: c.Limits.MemoryBytes, Processes: c.Limits.Processes},
		MaxOutputBytes: c.MaxOutputBytes,
	}
}
