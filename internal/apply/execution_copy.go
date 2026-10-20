package apply

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func executionPointer[T any](value *organization.Setting[T]) *organization.Setting[T] {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func executionListPointer[T any](value *organization.Setting[[]T]) *organization.Setting[[]T] {
	result := executionPointer(value)
	if result != nil {
		result.Value = slices.Clone(result.Value)
	}
	return result
}

func executionRoleCopy(value organization.RoleOverride) organization.RoleOverride {
	value.Profile = executionPointer(value.Profile)
	value.Instructions = executionListPointer(value.Instructions)
	return value
}

// instructionLists returns every authored instruction reference list: the
// project's Execute and Verify lists, the Package overrides' and the team's
// helper roles'.
func instructionLists(execution AuthoredExecution) []*organization.Setting[[]string] {
	lists := []*organization.Setting[[]string]{execution.Defaults.Execute.Instructions, execution.Defaults.Verify.Instructions}
	if execution.Packages != nil {
		for _, override := range execution.Packages.Value {
			lists = append(lists, override.Execute.Instructions, override.Verify.Instructions)
		}
	}
	if execution.Team != nil {
		for _, helper := range execution.Team.Helpers {
			lists = append(lists, helper.Instructions)
		}
	}
	return lists
}

func cloneExecution(value AuthoredExecution) AuthoredExecution {
	value.Defaults.Execute = executionRoleCopy(value.Defaults.Execute)
	value.Defaults.Verify = executionRoleCopy(value.Defaults.Verify)
	value.Packages = executionListPointer(value.Packages)
	if value.Packages != nil {
		for i := range value.Packages.Value {
			p := &value.Packages.Value[i]
			p.Execute = executionRoleCopy(p.Execute)
			p.Verify = executionRoleCopy(p.Verify)
		}
	}
	value.Ownership = executionListPointer(value.Ownership)
	if value.Team != nil {
		team := *value.Team
		team.Helpers = slices.Clone(team.Helpers)
		for i := range team.Helpers {
			team.Helpers[i].Profile = executionPointer(team.Helpers[i].Profile)
			team.Helpers[i].Instructions = executionListPointer(team.Helpers[i].Instructions)
		}
		value.Team = &team
	}
	value.Scope = executionListPointer(value.Scope)
	value.Checks = executionListPointer(value.Checks)
	if value.Checks != nil {
		for i := range value.Checks.Value {
			value.Checks.Value[i].Args.Value = slices.Clone(value.Checks.Value[i].Args.Value)
			value.Checks.Value[i].RequiredBy = executionPointer(value.Checks.Value[i].RequiredBy)
		}
	}
	return value
}
