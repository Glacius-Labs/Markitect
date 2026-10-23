package project_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

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

func TestCandidateAccountingFeedbackIsBoundedDeterministicAndExcludesFileContents(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	actual := raw
	source := controlledSource{
		capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
			return captured(raw), nil
		},
		preview: func(_ context.Context, r project.PreviewRequest) (project.PreviewedInputs, error) {
			return project.PreviewedInputs{Raw: actual, Head: fixtureHead, Revision: r.Revision}, nil
		},
	}
	service := project.NewPreparationService(project.NewModelService(source, source))
	initial, err := service.Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	a, ok := initial.Prepared()
	if !ok {
		t.Fatal("fixture preparation failed", initial.Status())
	}
	p, err := project.Prepare(a)
	if err != nil {
		t.Fatal(err)
	}
	const contents = "private-file-content-canary"
	for i := 0; i < 80; i++ {
		censusFile(t, &actual, fmt.Sprintf("realization/unregistered-%03d-ä.go", i), snapshot.Regular, []byte(contents))
	}
	var previous string
	for i := 0; i < 2; i++ {
		preview, err := service.Preview(context.Background(), project.PreviewRequest{Selection: selection, Revision: fixtureHead}, model.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		attempts := preview.Attempts()
		err = p.ValidateCandidate(attempts[len(attempts)-1])
		if !errors.Is(err, project.ErrCandidateInvalid) {
			t.Fatal("accounting refusal lost its semantic classification", err)
		}
		text := err.Error()
		if len(text) > 8192 || !utf8.ValidString(text) || !strings.Contains(text, "Additional structural diagnostics omitted") || !strings.Contains(text, "file.unexplained") || !strings.Contains(text, "realization/unregistered-000-ä.go") || strings.Contains(text, contents) {
			t.Fatalf("feedback lost its bounds, actionable path or content boundary: bytes=%d %s", len(text), text)
		}
		if i > 0 && text != previous {
			t.Fatal("same fixed candidate returned unstable feedback")
		}
		previous = text
	}
}
