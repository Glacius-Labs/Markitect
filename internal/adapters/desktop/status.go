package desktop

import (
	"context"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Status is what the hub shows about a project: whether Markitect is set up
// in it and what markitect check finds in its working tree.
type Status struct {
	Initialized bool   `json:"initialized"`
	Check       *Check `json:"check,omitempty"`
	// Problem says why the project could not be read, such as a folder that
	// is gone or no longer a Git repository.
	Problem string `json:"problem,omitempty"`
}

// Check is the result of markitect check: the same load, the same errors.
type Check struct {
	OK          bool     `json:"ok"`
	ErrorCount  int      `json:"errorCount"`
	Errors      []string `json:"errors"` // the first maxErrors
	Definitions int      `json:"definitions"`
	Packages    int      `json:"packages"`
	Files       int      `json:"files"`
}

const maxErrors = 20

func readStatus(ctx context.Context, repos Repos, p Project) Status {
	ws, err := repos.Open(ctx, p.Path)
	if err != nil {
		return Status{Problem: err.Error()}
	}
	pr, err := project.Load(ws)
	if err != nil {
		return Status{Problem: err.Error()}
	}
	if !slices.Contains(pr.Files, project.ManifestFile) {
		return Status{}
	}
	c := &Check{OK: len(pr.Errors) == 0, ErrorCount: len(pr.Errors), Errors: []string{}}
	for _, d := range pr.Errors {
		if len(c.Errors) == maxErrors {
			break
		}
		c.Errors = append(c.Errors, d.String())
	}
	if pr.Model != nil {
		c.Definitions = len(pr.Model.ModelDefinitions())
		c.Packages = len(pr.Model.Packages)
	}
	if pr.Registry != nil {
		c.Files = len(pr.Registry.Owner)
	}
	return Status{Initialized: true, Check: c}
}
