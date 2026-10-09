package projectapp

import (
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
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
	preview, err := projectsetup.PreviewEdit(project, operation.Options)
	if err != nil {
		return projectsetup.Preview{}, err
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
	if err := requireRoot(operation.Root); err != nil {
		return projectsetup.DoctorReport{}, err
	}
	project, err := projectwork.Load(operation.Root, "")
	if err != nil {
		return projectsetup.DoctorReport{}, err
	}
	return projectsetup.Doctor(project, operation.Options)
}
