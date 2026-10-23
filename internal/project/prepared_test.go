package project_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestFixedProjectRejectsExternalInstructionMutationAndDistinguishesManifestFailure(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	location := project.SourceLocation{Boundary: "external-guidance", Path: path(t, "guide.md")}
	file, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("original external guidance"))
	if err != nil {
		t.Fatal(err)
	}
	selection.Instructions = append(selection.Instructions, location)
	raw.Instructions = append(raw.Instructions, project.SourceFile{Location: location, File: file})
	actual := raw
	var observationErr error
	source := controlledSource{
		capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
			return captured(raw), nil
		},
		preview: func(_ context.Context, r project.PreviewRequest) (project.PreviewedInputs, error) {
			return project.PreviewedInputs{Raw: actual, Head: fixtureHead, Revision: r.Revision}, observationErr
		},
	}
	service := project.NewPreparationService(project.NewModelService(source, source))
	captured, err := service.Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	a, ok := captured.Prepared()
	if err != nil || !ok {
		t.Fatal(captured.Status(), err)
	}
	p, err := project.Prepare(a)
	if err != nil {
		t.Fatal(err)
	}
	preview := func() project.PreviewPreparationAttempt {
		t.Helper()
		r, _ := service.Preview(context.Background(), project.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
		attempts := r.Attempts()
		return attempts[len(attempts)-1]
	}
	if err = p.ValidateCandidate(preview()); err != nil {
		t.Fatal("unchanged fixed input refused", err)
	}
	changed, _ := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("mutated external guidance"))
	actual.Instructions = append([]project.SourceFile(nil), raw.Instructions...)
	actual.Instructions[len(actual.Instructions)-1].File = changed
	if err = p.ValidateCandidate(preview()); !errors.Is(err, project.ErrCandidateInvalid) {
		t.Fatal("external input drift was not rejected", err)
	}
	actual = raw
	for _, incomplete := range []bool{false, true} {
		_, _, inspection := partialManifest(t, incomplete)
		observationErr = project.NewManifestPreflightError(selection.Local, inspection)
		err = p.ValidateCandidate(preview())
		if err == nil || errors.Is(err, project.ErrCandidateInvalid) == incomplete {
			t.Fatalf("known invalid vs incomplete manifest classification: incomplete=%v error=%v", incomplete, err)
		}
	}
}
