package checks

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
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
func fixtureModel(t *testing.T, from, boundary string) apply.ModelSource {
	t.Helper()
	root, err := snapshot.DirectoryAt(fixturePath(t, "model"))
	if err != nil {
		t.Fatal(err)
	}
	r := apply.ModelSource{Selection: apply.ModelSelection{Boundary: boundary, ManifestPath: fixturePath(t, "model.yaml")}, Root: root, Source: model.SourceModel{Origin: "controlled:" + boundary, ManifestPath: "model.yaml"}}
	var files []snapshot.Entry
	err = fs.WalkDir(os.DirFS(from), ".", func(name string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		e := model.Entry{Path: name, Kind: model.Directory}
		if !dir.IsDir() {
			e.Kind = model.Regular
			e.Bytes, err = os.ReadFile(filepath.Join(from, filepath.FromSlash(name)))
			if err != nil {
				return err
			}
			files = append(files, fixtureEntry(t, name, e.Bytes))
		}
		r.Source.Entries = append(r.Source.Entries, e)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r.Files = fixtureTree(t, files)
	return r
}

// Acquisition is controlled test data, not a real repository adapter or clean
// HEAD proof. Model, Defaults, Registry and FileCheck run their actual producers.
type fixtureSource struct{ raw apply.RawInputs }

func (s fixtureSource) Capture(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) {
	return apply.CapturedInputs{Raw: s.raw, Head: strings.Repeat("a", 40), Guard: "controlled-test-capture"}, nil
}

func m1Parents(t *testing.T, process string) (apply.CapturedFileCheck, apply.CapturedDefaultsPreparation) {
	t.Helper()
	return m1SectionParents(t, "process", process)
}

func m1SectionParents(t *testing.T, section, process string) (apply.CapturedFileCheck, apply.CapturedDefaultsPreparation) {
	t.Helper()
	base := "../../../docs/design/canonical-model"
	raw := apply.RawInputs{Local: fixtureModel(t, base+"/library-example/m1", "controlled-local"), Dependencies: []apply.ModelSource{fixtureModel(t, base+"/library-example/vocabulary", "controlled-vocabulary"), fixtureModel(t, base+"/minimal-language/exchange-example/dependencies/core", "controlled-core")}}
	area, err := snapshot.DirectoryAt(fixturePath(t, ".markitect"))
	if err != nil {
		t.Fatal(err)
	}
	raw.Local.RepositoryArea = area
	var census []snapshot.Entry
	for _, file := range raw.Local.Files.Entries() {
		census = append(census, fixtureEntry(t, ".markitect/"+file.Path().String(), file.Bytes()))
	}
	execution := fixtureExecution
	selection := apply.InputSelection{Repository: "controlled-m1-repository", Local: raw.Local.Selection}
	for _, dependency := range raw.Dependencies {
		selection.Dependencies = append(selection.Dependencies, dependency.Selection)
	}
	for _, source := range []struct {
		name, data string
		group      string
	}{
		{".markitect/execution.yaml", execution, "execution"},
		{".markitect/registry/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n", "registry"},
	} {
		file := fixtureEntry(t, source.name, []byte(source.data))
		census = append(census, file)
		location := apply.SourceLocation{Boundary: "controlled-repository-source", Path: file.Path()}
		observation := apply.SourceFile{Location: location, File: file, RepositoryPath: file.Path()}
		if source.group == "execution" {
			raw.Execution = []apply.SourceFile{observation}
			selection.Execution = []apply.SourceLocation{location}
		} else {
			raw.Registry = []apply.SourceFile{observation}
			selection.Registry = []apply.SourceLocation{location}
		}
	}
	raw.Census = fixtureTree(t, census)
	runtimeDoc := "apiVersion: markitect.runtime/v1\nagentProfiles:\n  library-agent: {codexAppServer: {profileVersion: 1}}\ncheckProfiles:\n  library-tests:\n    " + section + ":\n"
	for _, line := range strings.Split(strings.TrimSuffix(process, "\n"), "\n") {
		runtimeDoc += "      " + line + "\n"
	}
	location := apply.SourceLocation{Boundary: "controlled-runtime-source", Path: fixturePath(t, "runtime.yaml")}
	raw.RuntimeProfiles = []apply.SourceFile{{Location: location, File: fixtureEntry(t, "runtime.yaml", []byte(runtimeDoc))}}
	selection.RuntimeProfiles = []apply.SourceLocation{location}
	m, err := apply.NewModelService(fixtureSource{raw}, nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || m.Checked().Status != model.Passed {
		t.Fatalf("M1 model: %v %+v", err, m.Checked().Diagnostics)
	}
	files := m.PrepareRegistry().CheckFiles()
	defaults := m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults())
	if files.Status() != model.Passed || defaults.Status() != model.Passed {
		t.Fatalf("M1 parents: files=%s %+v defaults=%s %+v", files.Status(), files.Diagnostics(), defaults.Status(), defaults.Diagnostics())
	}
	return files, defaults
}

type recordingPreparer struct {
	input   apply.ProfileInput
	observe bool
}

func (r *recordingPreparer) PrepareCheckProfile(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	r.input = input
	if r.observe {
		return (MetadataPreparer{}).PrepareCheckProfile(ctx, input)
	}
	return apply.ProfileObservation{InputBinding: input.Binding(), Status: model.NotCompleted}, nil
}
func processInput(t *testing.T, inner string) apply.ProfileInput {
	t.Helper()
	files, defaults := m1Parents(t, inner)
	recorder := &recordingPreparer{}
	_, err := files.PrepareSettings(context.Background(), defaults, nil, recorder)
	if err != nil || recorder.input.Namespace() != "checkProfiles" {
		t.Fatalf("actual M1 profile input missing: %v", err)
	}
	return recorder.input
}

// fixtureExecution is the execution document of the controlled Library fixture.
const fixtureExecution = `apiVersion: markitect.execution/v1
strategy: shared-workspace-v1
budget:
  executionTime: 25m
  turnTimeout: 10m
  agentTurns: 120
  rounds: 4
profiles:
  execute: library-agent
  verify: library-agent
packages:
  lending:
    profiles: {verify: library-agent}
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    requiredBy: library/callable-interface
    timeout: 2m
`
