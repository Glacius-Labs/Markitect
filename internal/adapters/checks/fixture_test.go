package checks

import (
	"context"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"strings"
	"testing"
)

func fixturePath(t *testing.T, name string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func fixtureEntry(t *testing.T, name string, data []byte) snapshot.Entry {
	t.Helper()
	e, err := snapshot.NewEntry(fixturePath(t, name), snapshot.Regular, data)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func fixtureTree(t *testing.T, entries []snapshot.Entry) snapshot.Tree {
	t.Helper()
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func runtimeCheckInputs(t *testing.T, fields string) project.RawInputs {
	t.Helper()
	document := "apiVersion: markitect.runtime/v1\nagentProfiles: {}\ncheckProfiles:\n  library-tests:\n    container:\n"
	for _, line := range strings.Split(strings.TrimSuffix(fields, "\n"), "\n") {
		document += "      " + line + "\n"
	}
	file := fixtureEntry(t, "runtime.yaml", []byte(document))
	location := project.SourceLocation{Boundary: "controlled-runtime", Path: file.Path()}
	return project.RawInputs{RuntimeProfiles: []project.SourceFile{{Location: location, File: file}}}
}

type recordingPreparer struct{ input runtimeconfig.ProfileInput }

func (r *recordingPreparer) PrepareCheckProfile(_ context.Context, input runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
	r.input = input
	return runtimeconfig.ProfileObservation{InputBinding: input.Binding(), Status: model.NotCompleted}, nil
}
