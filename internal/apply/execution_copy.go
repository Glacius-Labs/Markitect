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
	if value.Layout != nil {
		layout := *value.Layout
		layout.Roots.Value = slices.Clone(layout.Roots.Value)
		layout.Exceptions = slices.Clone(layout.Exceptions)
		value.Layout = &layout
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
