package project_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

// requiredBy identifies an obligation; private check execution also needs a
// declared directory entirely inside the Manager's modeled responsibility.
func TestChecksForConfinesExistingObligationsToOwnedDirectorySubtrees(t *testing.T) {
	for _, tc := range []struct {
		name, ownership, directory, requiredBy string
		selected                               []string
	}{
		{"owned directory", libraryOwnershipYAML, "realization/library", "library/callable-interface", []string{"library"}},
		{"project suite", libraryOwnershipYAML, "realization", "library/callable-interface", nil},
		{"repository root", "ownership:\n  - {path: './', owner: library}\n", ".", "library/callable-interface", nil},
		{"missing layout", "", "realization/library", "library/callable-interface", nil},
		{"unowned directory", "ownership:\n  - {path: realization/library/, owner: library}\n", "elsewhere", "library/callable-interface", nil},
		{"nested exact foreign file", libraryOwnershipYAML + "  - {path: realization/library/build.go, owner: .}\n", "realization/library", "library/callable-interface", nil},
		{"nested foreign directory", libraryOwnershipYAML + "  - {path: realization/library/peer/, owner: lending}\n", "realization/library", "library/callable-interface", nil},
		{"nested exact own file", libraryOwnershipYAML + "  - {path: realization/library/build.go, owner: library}\n", "realization/library", "library/callable-interface", []string{"library"}},
		{"path prefix is not containment", libraryOwnershipYAML + "  - {path: realization/library-other/, owner: lending}\n", "realization/library", "library/callable-interface", []string{"library"}},
		{"eligible aggregate ancestor", libraryOwnershipYAML, "realization/lending", "lending/reminders/assess-overdue", []string{"lending"}},
		{"owned grandchild", libraryOwnershipYAML, "realization/lending/reminders", "lending/reminders/assess-overdue", []string{"lending", "lending/reminders"}},
		{"nested exact ancestor file", libraryOwnershipYAML + "  - {path: realization/lending/reminders/build.go, owner: lending}\n", "realization/lending/reminders", "lending/reminders/assess-overdue", []string{"lending"}},
		{"without obligation stays root only", libraryOwnershipYAML, "realization/library", "", nil},
		{"owner does not add unrelated eligibility", libraryOwnershipYAML, "realization/lending", "library/callable-interface", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := "apiVersion: markitect.execution/v2\n" + tc.ownership + "checks:\n  - id: project-tests\n    profile: library-tests\n    directory: " + tc.directory + "\n    args: [test, ./...]\n    timeout: 2m\n"
			if tc.requiredBy != "" {
				content += "    requiredBy: " + tc.requiredBy + "\n"
			}
			selection, raw := executionInputs(t, content)
			registrySource(t, &selection, &raw, ".markitect/registry/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n", true)
			source := controlledSource{capture: func(context.Context, project.CaptureRequest) (project.CapturedInputs, error) {
				return captured(raw), nil
			}}
			result, err := project.NewPreparationService(project.NewModelService(source, source)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
			attempt, ok := result.Prepared()
			if err != nil || !ok {
				t.Fatal(result.Status(), err)
			}
			p, err := project.Prepare(attempt)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(p.ChecksFor("."), p.Checks()) {
				t.Fatal("Root lost or changed declared suite")
			}
			var selected []string
			for _, manager := range []string{"library", "lending", "lending/reminders"} {
				if len(p.ChecksFor(manager)) != 0 {
					selected = append(selected, manager)
				}
			}
			if !reflect.DeepEqual(selected, tc.selected) {
				t.Fatalf("private check Managers=%v, want %v", selected, tc.selected)
			}
		})
	}
}
