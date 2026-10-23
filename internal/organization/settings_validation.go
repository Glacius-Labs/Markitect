package organization

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"reflect"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type settingsValidation struct {
	files map[string]snapshot.ContentID
	// Used only to reject contradictory supplied assertions about one field.
	// This is not a parsed document or an execution configuration property bag.
	fields map[SettingSource]any
}

// clone keeps later consistency checks from mutating the fixed settings basis.
// Values retained by field are immutable facts or copied lists.
func (v settingsValidation) clone() settingsValidation {
	return settingsValidation{files: maps.Clone(v.files), fields: maps.Clone(v.fields)}
}

func (v settingsValidation) source(source SettingSource, field bool) error {
	if source.File == "" || source.ContentID == (snapshot.ContentID{}) || field && source.Field == "" {
		return fmt.Errorf("missing configuration source file, content identity or field: %+v", source)
	}
	if previous, found := v.files[source.File]; found && previous != source.ContentID {
		return fmt.Errorf("conflicting content identities for configuration source %q", source.File)
	}
	v.files[source.File] = source.ContentID
	return nil
}

func (v settingsValidation) field(source SettingSource, value any) error {
	if err := v.source(source, true); err != nil {
		return err
	}
	if previous, found := v.fields[source]; found && !reflect.DeepEqual(previous, value) {
		return fmt.Errorf("conflicting supplied values for configuration field %q in %q", source.Field, source.File)
	}
	v.fields[source] = value
	return nil
}

func (v settingsValidation) catalogs(facts SettingsFacts) (map[string]ProfileFact, map[string]InstructionFact, error) {
	profiles := make(map[string]ProfileFact, len(facts.Profiles))
	for _, profile := range facts.Profiles {
		if profile.Name == "" || profile.Identity == "" {
			return nil, nil, fmt.Errorf("profile requires an exact name and identity")
		}
		if _, found := profiles[profile.Name]; found {
			return nil, nil, fmt.Errorf("duplicate profile %q", profile.Name)
		}
		if err := v.field(profile.Source, profile); err != nil {
			return nil, nil, err
		}
		profiles[profile.Name] = profile
	}
	instructions := make(map[string]InstructionFact, len(facts.Instructions))
	for _, instruction := range facts.Instructions {
		if instruction.ID == "" || instruction.Source.Field != "" {
			return nil, nil, fmt.Errorf("instruction requires an exact ID and whole-file source")
		}
		if _, found := instructions[instruction.ID]; found {
			return nil, nil, fmt.Errorf("duplicate instruction %q", instruction.ID)
		}
		if err := v.source(instruction.Source, false); err != nil {
			return nil, nil, err
		}
		if instruction.Source.ContentID != snapshot.ContentID(sha256.Sum256(instruction.Bytes)) {
			return nil, nil, fmt.Errorf("instruction %q bytes disagree with source content identity", instruction.ID)
		}
		instructions[instruction.ID] = instruction
	}
	return profiles, instructions, nil
}

func (v settingsValidation) role(defaults RoleDefaults, override RoleOverride, profiles map[string]ProfileFact, instructions map[string]InstructionFact) (RoleSettings, error) {
	profile := replaceSetting(defaults.Profile, override.Profile)
	if err := v.field(profile.Source, profile.Value); err != nil {
		return RoleSettings{}, err
	}
	fact, found := profiles[profile.Value]
	if !found {
		return RoleSettings{}, fmt.Errorf("unresolved profile %q", profile.Value)
	}
	result := RoleSettings{Profile: Setting[ProfileFact]{Value: fact, Source: profile.Source}}
	lists := []Setting[[]string]{defaults.Instructions}
	if override.Instructions != nil {
		lists = append(lists, *override.Instructions)
	}
	for _, list := range lists {
		if err := v.field(list.Source, append([]string{}, list.Value...)); err != nil {
			return RoleSettings{}, err
		}
		result.InstructionSources = append(result.InstructionSources, list.Source)
		for _, id := range list.Value {
			instruction, found := instructions[id]
			if !found {
				return RoleSettings{}, fmt.Errorf("unresolved instruction %q", id)
			}
			result.Instructions = append(result.Instructions, Setting[InstructionFact]{Value: instruction, Source: list.Source})
		}
	}
	return result, nil
}

// team resolves the helper roles of one Manager: each binds its own profile or
// the Manager's Execute profile, and its own instructions.
func (v settingsValidation) team(team TeamDefaults, execute RoleSettings, profiles map[string]ProfileFact, instructions map[string]InstructionFact) ([]HelperSettings, int, error) {
	if len(team.Helpers) == 0 {
		return nil, 0, nil
	}
	if err := v.field(team.MaxHelpers.Source, team.MaxHelpers.Value); err != nil {
		return nil, 0, err
	}
	if team.MaxHelpers.Value <= 0 {
		return nil, 0, fmt.Errorf("a team with helper roles needs a positive maxHelpers")
	}
	seen := map[string]bool{}
	var helpers []HelperSettings
	for _, h := range team.Helpers {
		if h.Role.Value == "" || h.Role.Value == "manager" || h.Role.Value == "reviewer" || seen[h.Role.Value] {
			return nil, 0, fmt.Errorf("helper role %q is empty, reserved or repeated", h.Role.Value)
		}
		seen[h.Role.Value] = true
		if err := v.field(h.Role.Source, h.Role.Value); err != nil {
			return nil, 0, err
		}
		if err := v.field(h.Writes.Source, h.Writes.Value); err != nil {
			return nil, 0, err
		}
		role := RoleSettings{Profile: execute.Profile}
		if h.Profile != nil {
			if err := v.field(h.Profile.Source, h.Profile.Value); err != nil {
				return nil, 0, err
			}
			fact, found := profiles[h.Profile.Value]
			if !found {
				return nil, 0, fmt.Errorf("helper role %q: unresolved profile %q", h.Role.Value, h.Profile.Value)
			}
			role.Profile = Setting[ProfileFact]{Value: fact, Source: h.Profile.Source}
		}
		if h.Instructions != nil {
			if err := v.field(h.Instructions.Source, append([]string{}, h.Instructions.Value...)); err != nil {
				return nil, 0, err
			}
			role.InstructionSources = append(role.InstructionSources, h.Instructions.Source)
			for _, id := range h.Instructions.Value {
				instruction, found := instructions[id]
				if !found {
					return nil, 0, fmt.Errorf("helper role %q: unresolved instruction %q", h.Role.Value, id)
				}
				role.Instructions = append(role.Instructions, Setting[InstructionFact]{Value: instruction, Source: h.Instructions.Source})
			}
		}
		helpers = append(helpers, HelperSettings{Role: h.Role.Value, Writes: h.Writes.Value, RoleSettings: role})
	}
	return helpers, team.MaxHelpers.Value, nil
}
