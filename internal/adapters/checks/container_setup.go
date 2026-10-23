package checks

// ContainerRuntimeProfile is the explicit content of one container Check Runtime
// profile. Setup writes it into the closed section that the metadata decoder
// reads back.
type ContainerRuntimeProfile struct {
	EngineExecutable string
	Profile          ContainerProfile
}

// ContainerRuntimeLimits are the resource bounds of a container check.
type ContainerRuntimeLimits struct {
	MilliCPUs   int64 `yaml:"milliCPUs"`
	MemoryBytes int64 `yaml:"memoryBytes"`
	Processes   int64 `yaml:"processes"`
}

// ContainerRuntimeSection is the closed `container` section of a Runtime profile document in
// the field order and spelling of the decoder.
type ContainerRuntimeSection struct {
	ProfileVersion   int                    `yaml:"profileVersion"`
	EngineExecutable string                 `yaml:"engineExecutable"`
	ImageID          string                 `yaml:"imageID"`
	Executable       string                 `yaml:"executable"`
	ArgumentPrefix   []string               `yaml:"argumentPrefix"`
	Env              []string               `yaml:"env"`
	Limits           ContainerRuntimeLimits `yaml:"limits"`
	MaxOutputBytes   int64                  `yaml:"maxOutputBytes"`
}

// Section returns the Runtime profile document section of the profile. Lists are never
// null. It validates nothing: the metadata decoder owns every rule, and setup
// runs the written document through it before any file is written.
func (p ContainerRuntimeProfile) Section() ContainerRuntimeSection {
	c := p.Profile
	return ContainerRuntimeSection{
		ProfileVersion: 1, EngineExecutable: p.EngineExecutable, ImageID: c.ImageID, Executable: c.Executable,
		ArgumentPrefix: append([]string{}, c.ArgumentPrefix...), Env: append([]string{}, c.Env...),
		Limits:         ContainerRuntimeLimits{MilliCPUs: c.Limits.MilliCPUs, MemoryBytes: c.Limits.MemoryBytes, Processes: c.Limits.Processes},
		MaxOutputBytes: c.MaxOutputBytes,
	}
}
