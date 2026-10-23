package project

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

func instructionLists(execution AuthoredExecution) []*organization.Setting[[]string] {
	var lists []*organization.Setting[[]string]
	if execution.Instructions != nil {
		for i := range execution.Instructions.Value {
			lists = append(lists, &execution.Instructions.Value[i].Files)
		}
	}
	return lists
}

func cloneExecution(value AuthoredExecution) AuthoredExecution {
	value.Instructions = executionListPointer(value.Instructions)
	if value.Instructions != nil {
		for i := range value.Instructions.Value {
			value.Instructions.Value[i].Files.Value = slices.Clone(value.Instructions.Value[i].Files.Value)
		}
	}
	value.Ownership = executionListPointer(value.Ownership)
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
