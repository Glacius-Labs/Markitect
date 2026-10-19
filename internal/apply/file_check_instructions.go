package apply

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func (c *fileChecker) instructions(execution AuthoredExecution) {
	selected := make(map[snapshot.Path]bool)
	for _, source := range c.raw.Instructions {
		if source.RepositoryPath.String() == "" {
			continue
		}
		selected[source.RepositoryPath] = sameCensusFile(c.raw.Census, source.RepositoryPath.String(), source.File)
	}
	lists := []*organization.Setting[[]string]{execution.Defaults.Execute.Instructions, execution.Defaults.Verify.Instructions}
	if execution.Packages != nil {
		for _, override := range execution.Packages.Value {
			lists = append(lists, override.Execute.Instructions, override.Verify.Instructions)
		}
	}
	for _, list := range lists {
		if list == nil {
			continue
		}
		for i, ref := range list.Value {
			field := list.Source.Field + "/" + fmt.Sprint(i)
			path, err := snapshot.NewPath(ref)
			if err != nil {
				c.issue(model.Failed, "files.instruction_path", mustExecutionPath(), field, nil, err.Error())
				continue
			}
			c.protectFile(path, realization.MarkitectInput, "Project instruction declared at "+field+".")
			if _, found := c.raw.Census.Lookup(path); !found {
				c.issue(model.Failed, "files.instruction_missing", mustExecutionPath(), field, nil, "Declared project instruction is absent from the complete repository Census: "+ref)
			} else if !selected[path] {
				c.issue(model.NotCompleted, "files.instruction_uncaptured", mustExecutionPath(), field, nil, "Declared project instruction requires a captured repository source: "+ref)
			}
		}
	}
}

func mustExecutionPath() snapshot.Path { path, _ := snapshot.NewPath(executionPath); return path }
