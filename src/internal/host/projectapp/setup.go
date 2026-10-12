package projectapp

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

// SetupOperation previews or writes the exact guarded runtime edit. Options
// remain explicit; discovery never starts a provider role.
type SetupOperation struct {
	Root           string               `json:"root"`
	Options        projectsetup.Options `json:"options"`
	Write          bool                 `json:"write"`
	ExpectedDigest string               `json:"expectedDigest,omitempty"`
}

type DoctorOperation struct {
	Root    string               `json:"root"`
	Options projectsetup.Options `json:"options"`
}

func (o Operations) Setup(operation SetupOperation) (projectsetup.Preview, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectsetup.Preview{}, err
	}
	project, err := projectwork.Load(operation.Root, "")
	if err != nil {
		return projectsetup.Preview{}, err
	}
	proposal, err := projectsetup.PreviewEdit(project, operation.Options)
	if err != nil {
		return projectsetup.Preview{}, err
	}
	mutation := projectvalue.Mutation{
		APIVersion: projectvalue.APIVersion,
		BaseDigest: project.Digest,
		Actor:      projectvalue.HumanActor,
		Goal:       "Configure the project-local agent runtime from explicitly selected local tools",
		Files:      []projectvalue.FileChange{{Path: projectvalue.RuntimePath, Content: proposal.RuntimeContent}},
	}
	plan, err := projectwork.PlanEdit(project, mutation)
	if err != nil {
		return projectsetup.Preview{}, fmt.Errorf("validate runtime edit proposal: %w", err)
	}
	preview := projectsetup.Preview{
		APIVersion: proposal.APIVersion, Discovery: proposal.Discovery, Roles: proposal.Roles,
		PricingBasis: proposal.PricingBasis, Mutation: mutation, EditPlan: plan,
	}
	if operation.Write {
		if operation.ExpectedDigest != preview.EditPlan.Digest {
			return projectsetup.Preview{}, fmt.Errorf("expectedDigest does not match the exact runtime edit plan digest %s", preview.EditPlan.Digest)
		}
		applied, err := projectwork.ApplyEdit(operation.Root, preview.EditPlan, preview.EditPlan.BaseDigest)
		if err != nil {
			return projectsetup.Preview{}, err
		}
		preview.EditPlan = applied
	}
	return preview, nil
}

func (o Operations) Doctor(operation DoctorOperation) (projectsetup.DoctorReport, error) {
	return o.doctor(operation, readGitVersion)
}
