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
	value.Limits.CreationRoots.Value = slices.Clone(value.Limits.CreationRoots.Value)
	value.Defaults.Manager = executionRoleCopy(value.Defaults.Manager)
	value.Defaults.Reviewer = executionRoleCopy(value.Defaults.Reviewer)
	value.Defaults.BatchNoChangeReviews = executionPointer(value.Defaults.BatchNoChangeReviews)
	value.Defaults.RejectionLimit = executionPointer(value.Defaults.RejectionLimit)
	value.Defaults.CreationRoots.Value = slices.Clone(value.Defaults.CreationRoots.Value)
	value.Packages = executionListPointer(value.Packages)
	if value.Packages != nil {
		for i := range value.Packages.Value {
			p := &value.Packages.Value[i]
			p.Manager = executionRoleCopy(p.Manager)
			p.Reviewer = executionRoleCopy(p.Reviewer)
			p.StepTimeout = executionPointer(p.StepTimeout)
			p.RejectionLimit = executionPointer(p.RejectionLimit)
			p.CreationRoots = executionListPointer(p.CreationRoots)
		}
	}
	value.Scope = executionListPointer(value.Scope)
	value.Checks = executionListPointer(value.Checks)
	if value.Checks != nil {
		for i := range value.Checks.Value {
			value.Checks.Value[i].Args.Value = slices.Clone(value.Checks.Value[i].Args.Value)
			value.Checks.Value[i].RequiredBy = executionPointer(value.Checks.Value[i].RequiredBy)
		}
	}
	value.OwnerApproval = executionListPointer(value.OwnerApproval)
	return value
}
