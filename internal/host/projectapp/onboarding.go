package projectapp

import (
	"github.com/Glacius-Labs/Markitect/internal/host/projectonboarding"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

type OnboardOperation struct {
	Root           string                    `json:"root"`
	Options        projectonboarding.Options `json:"options"`
	Write          bool                      `json:"write"`
	ExpectedDigest string                    `json:"expectedDigest,omitempty"`
}

func (o Operations) Onboard(operation OnboardOperation) (projectonboarding.Plan, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectonboarding.Plan{}, err
	}
	project, err := projectwork.Load(operation.Root, "")
	if err != nil {
		return projectonboarding.Plan{}, err
	}
	plan, err := projectonboarding.Preview(operation.Root, project.Report.ModelDigest, operation.Options)
	if err != nil {
		return projectonboarding.Plan{}, err
	}
	if operation.Write {
		return projectonboarding.Apply(operation.Root, plan, operation.ExpectedDigest)
	}
	return plan, nil
}
