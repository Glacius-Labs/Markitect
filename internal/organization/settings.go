package organization

import (
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// SettingSource identifies an explicitly acquired file and a field within it.
// File is an opaque locator, not a path to read. ContentID binds original file
// bytes. Field is the supplied field locator; empty means the whole file.
// These are caller-supplied provenance facts, not authenticated approval or
// evidence that organization parsed or acquired that file.
type SettingSource struct {
	File      string
	ContentID snapshot.ContentID
	Field     string
}

// Setting is a configuration value with its selection source. A valid source
// makes even false or an empty list explicitly supplied. In an override, a nil
// *Setting means absent; a nonnil pointer replaces or appends as documented.
type Setting[T any] struct {
	Value  T
	Source SettingSource
}

// ProfileFact is an explicitly supplied named profile identity. Organization
// resolves names only; preparation of executables, protocols and provider
// settings belongs to their consumers. Identity is opaque, never wire data.
type ProfileFact struct {
	Name     string
	Identity string
	Source   SettingSource
}

// InstructionFact carries original whole-file bytes, including empty content.
// Source.Field must be empty and ContentID must match Bytes. ID is an exact
// lookup key, not a request to discover or read an instruction file.
type InstructionFact struct {
	ID     string
	Source SettingSource
	Bytes  []byte
}

type RoleDefaults struct {
	Profile      Setting[string]
	Instructions Setting[[]string]
}

type RoleOverride struct {
	Profile      *Setting[string]
	Instructions *Setting[[]string]
}

// SettingsDefaults must supply every field, including empty instruction and
// creation-root lists and false BatchNoChangeReviews. There are no built-ins.
type SettingsDefaults struct {
	Manager              RoleDefaults
	Reviewer             RoleDefaults
	StepTimeout          Setting[time.Duration]
	ReviewRounds         Setting[int]
	CreationRoots        Setting[[]snapshot.DirectoryBoundary]
	BatchNoChangeReviews Setting[bool]
}

// PackageOverride addresses exactly one local Package or Root. An explicit
// empty CreationRoots list replaces the default with no creation area.
// Instructions append; profiles and the other fields replace independently.
// There is deliberately no BatchNoChangeReviews override or role registry.
type PackageOverride struct {
	Package       string
	Manager       RoleOverride
	Reviewer      RoleOverride
	StepTimeout   *Setting[time.Duration]
	ReviewRounds  *Setting[int]
	CreationRoots *Setting[[]snapshot.DirectoryBoundary]
}

// SettingsCeilings constrain every selection, including unused defaults.
// Roots are lexical directory areas, not file entries or write permission.
// A selected directory must be contained inclusively in an allowed directory;
// real/sub and real equal the ceiling real, whereas real-other does not.
// The explicit RootDirectory boundary is supported. No normalization, physical
// existence, symlink resolution or artifact authority is implied.
type SettingsCeilings struct {
	StepTimeout   Setting[time.Duration]
	ReviewRounds  Setting[int]
	CreationRoots Setting[[]snapshot.DirectoryBoundary]
}

// SettingsFacts is the complete explicit input to this small resolution slice.
// Lists preserve duplicate keys for rejection. All facts are validated, even
// unused profiles, instructions and defaults. Acquisition/restricted-YAML
// validation and semantic review of instruction prose remain caller work.
// Conflicting content identities for one File, or conflicting typed values
// attributed to the same file/content/field, are rejected as inconsistent facts.
type SettingsFacts struct {
	Defaults     SettingsDefaults
	Overrides    []PackageOverride
	Ceilings     SettingsCeilings
	Profiles     []ProfileFact
	Instructions []InstructionFact
}

// RoleSettings retains the profile's selection and catalog provenance.
// InstructionSources records contributing lists even when they are empty;
// Instructions appends their resolved facts in declared order, with repeats.
type RoleSettings struct {
	Profile            Setting[ProfileFact]
	InstructionSources []SettingSource
	Instructions       []InstructionFact
}

type ManagerSettings struct {
	Package              string
	Manager              RoleSettings
	Reviewer             RoleSettings
	StepTimeout          Setting[time.Duration]
	ReviewRounds         Setting[int]
	CreationRoots        Setting[[]snapshot.DirectoryBoundary]
	BatchNoChangeReviews Setting[bool]
}

// Settings is a fixed successful resolution for all slots of one Topology.
// Its zero value is unusable and exposes no Manager settings. All returned
// lists and instruction bytes are owned copies. Ceilings explain the bounds
// on every Manager; provenance explains supplied facts, never authenticates
// approval or copies canonical Definitions into execution settings.
type Settings struct {
	modelID     string
	modelDigest string
	managers    []ManagerSettings
	index       map[string]int
	ceilings    SettingsCeilings
}

func (s Settings) ModelID() string     { return s.modelID }
func (s Settings) ModelDigest() string { return s.modelDigest }

func (s Settings) Ceilings() SettingsCeilings {
	result := s.ceilings
	result.CreationRoots.Value = slices.Clone(result.CreationRoots.Value)
	return result
}

// Managers returns Root first, then exact UTF-8 Package order, as in Topology.
func (s Settings) Managers() []ManagerSettings {
	result := make([]ManagerSettings, len(s.managers))
	for i, manager := range s.managers {
		result[i] = cloneManagerSettings(manager)
	}
	return result
}

func (s Settings) Manager(packagePath string) (ManagerSettings, bool) {
	i, found := s.index[packagePath]
	if !found {
		return ManagerSettings{}, false
	}
	return cloneManagerSettings(s.managers[i]), true
}

func cloneManagerSettings(manager ManagerSettings) ManagerSettings {
	manager.CreationRoots.Value = slices.Clone(manager.CreationRoots.Value)
	manager.Manager = cloneRoleSettings(manager.Manager)
	manager.Reviewer = cloneRoleSettings(manager.Reviewer)
	return manager
}

func cloneRoleSettings(role RoleSettings) RoleSettings {
	role.InstructionSources = slices.Clone(role.InstructionSources)
	role.Instructions = slices.Clone(role.Instructions)
	for i := range role.Instructions {
		role.Instructions[i].Bytes = slices.Clone(role.Instructions[i].Bytes)
	}
	return role
}
