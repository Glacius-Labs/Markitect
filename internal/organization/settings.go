package organization

import (
	"slices"

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

// SettingsDefaults must supply every field, including empty instruction lists.
// There are no built-ins. Execute and Verify are the two Agent roles of every
// Manager: its own work and its fresh Verify reviewer. Team adds the helper
// roles every Manager may instantiate for delegated tasks.
type SettingsDefaults struct {
	Execute RoleDefaults
	Verify  RoleDefaults
	Team    TeamDefaults
}

// TeamDefaults is the local team template of every Manager: its helper roles
// and how many may be active for one Manager at once.
type TeamDefaults struct {
	MaxHelpers Setting[int]
	Helpers    []HelperDefaults
}

// HelperDefaults is one helper role. A nil Profile binds the Manager's own
// Execute profile; Writes says whether the role may receive a delegated write
// subset of its Manager's area.
type HelperDefaults struct {
	Role         Setting[string]
	Writes       Setting[bool]
	Profile      *Setting[string]
	Instructions *Setting[[]string]
}

// PackageOverride addresses exactly one local Package or Root. Instructions
// append; profiles replace. There is deliberately no other override.
type PackageOverride struct {
	Package string
	Execute RoleOverride
	Verify  RoleOverride
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
	Profiles     []ProfileFact
	Instructions []InstructionFact
}

// RoleSettings retains the profile's selection and catalog provenance.
// InstructionSources records contributing lists even when they are empty;
// Instructions appends their resolved facts in declared order, with repeats.
// Each occurrence's outer Source identifies the selecting list field; its
// Value.Source identifies the original whole-file instruction content.
type RoleSettings struct {
	Profile            Setting[ProfileFact]
	InstructionSources []SettingSource
	Instructions       []Setting[InstructionFact]
}

// HelperSettings is one resolved helper role of a Manager.
type HelperSettings struct {
	Role   string
	Writes bool
	RoleSettings
}

type ManagerSettings struct {
	Package    string
	Execute    RoleSettings
	Verify     RoleSettings
	Helpers    []HelperSettings
	MaxHelpers int
}

// Settings is a fixed successful resolution for all slots of one Topology.
// Its zero value is unusable and exposes no Manager settings. All returned
// lists and instruction bytes are owned copies. Provenance explains supplied
// facts, never authenticates approval or copies canonical Definitions into
// execution settings.
type Settings struct {
	configuration settingsValidation // Fixed validated assertions, including unused facts.
	modelID       string
	modelDigest   string
	managers      []ManagerSettings
	index         map[string]int
}

func (s Settings) ModelID() string     { return s.modelID }
func (s Settings) ModelDigest() string { return s.modelDigest }

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
	manager.Execute = cloneRoleSettings(manager.Execute)
	manager.Verify = cloneRoleSettings(manager.Verify)
	manager.Helpers = slices.Clone(manager.Helpers)
	for i := range manager.Helpers {
		manager.Helpers[i].RoleSettings = cloneRoleSettings(manager.Helpers[i].RoleSettings)
	}
	return manager
}

func cloneRoleSettings(role RoleSettings) RoleSettings {
	role.InstructionSources = slices.Clone(role.InstructionSources)
	role.Instructions = slices.Clone(role.Instructions)
	for i := range role.Instructions {
		role.Instructions[i].Value.Bytes = slices.Clone(role.Instructions[i].Value.Bytes)
	}
	return role
}
