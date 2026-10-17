package journal

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// Source and profile metadata are controlled evidence. The repository, original
// fixture bytes, compilation, preparation, control, SQLite and reducer are real;
// these tests make no claim about production source acquisition or providers.
type integrationSource struct {
	t         *testing.T
	selection apply.InputSelection
	inputs    apply.CapturedInputs
}

func (s *integrationSource) Capture(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
	s.t.Helper()
	if !reflect.DeepEqual(request.Selection, s.selection) {
		s.t.Fatal("unexpected source selection", request.Selection)
	}
	return s.inputs, nil
}

type integrationProfiles struct{ executable, schema apply.ProfileMetadataFile }

func (p integrationProfiles) PrepareAgentProfile(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	return apply.ProfileObservation{
		InputBinding: input.Binding(), Status: model.Passed,
		Contract:    apply.ProfileContract{Name: "controlled-journal-integration-metadata", Version: 1},
		Files:       []apply.ProfileMetadataFile{p.executable},
		Inventories: []apply.ProfileMetadataInventory{{Boundary: p.schema.Location.Boundary, Files: []apply.ProfileMetadataFile{p.schema}}},
	}, nil
}

type integrationFixture struct {
	source   *integrationSource
	prepared apply.CapturedSettingsResult
	facts    apply.RunBindingFacts
	request  apply.ControlRequest
	options  apply.ControlOptions
}

func integrationPath(t *testing.T, name string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func integrationFile(t *testing.T, name, content string) snapshot.Entry {
	t.Helper()
	e, err := snapshot.NewEntry(integrationPath(t, name), snapshot.Regular, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func integrationTree(t *testing.T, files []snapshot.Entry) snapshot.Tree {
	t.Helper()
	tree, err := snapshot.NewTree(files)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func newIntegrationFixture(t *testing.T) integrationFixture {
	t.Helper()
	repo := testkit.NewRepo(t)
	selection := apply.InputSelection{Repository: repo.Dir,
		Local: apply.ModelSelection{Boundary: "fixture-model", ManifestPath: integrationPath(t, "model.yaml")}}
	root, _ := snapshot.DirectoryAt(integrationPath(t, "model"))
	area, _ := snapshot.DirectoryAt(integrationPath(t, ".markitect"))
	raw := apply.RawInputs{Local: apply.ModelSource{Selection: selection.Local, Root: root, RepositoryArea: area,
		Source: model.SourceModel{Origin: "local", ManifestPath: "model.yaml", Entries: integrationCore()}}}
	var census, modelFiles []snapshot.Entry
	for _, entry := range raw.Local.Source.Entries {
		modelFiles = append(modelFiles, integrationFile(t, entry.Path, string(entry.Bytes)))
		name := ".markitect/" + entry.Path
		repo.Write(name, string(entry.Bytes))
		census = append(census, integrationFile(t, name, string(entry.Bytes)))
	}
	raw.Local.Files = integrationTree(t, modelFiles)
	add := func(name, data string) apply.SourceFile {
		repo.Write(name, data)
		file := integrationFile(t, name, data)
		census = append(census, file)
		return apply.SourceFile{Location: apply.SourceLocation{Boundary: "fixture-repository", Path: file.Path()}, File: file, RepositoryPath: file.Path()}
	}
	add(".gitignore", ".markitect/state/\n")
	for _, group := range []struct {
		name, data string
		selected   *[]apply.SourceLocation
		files      *[]apply.SourceFile
	}{
		{".markitect/execution.yaml", integrationExecution, &selection.Execution, &raw.Execution},
		{".markitect/registry/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n", &selection.Registry, &raw.Registry},
		{".markitect/ignore.yaml", "apiVersion: markitect.ignore/v1\nfiles:\n  - path: .gitignore\n    reason: Fixture Git control policy.\n", &selection.Ignore, &raw.Ignore},
		{".markitect/host.yaml", "apiVersion: markitect.host/v1\nagentProfiles:\n  library-agent:\n    codexAppServer: {profileVersion: 1}\ncheckProfiles: {}\n", &selection.HostProfiles, &raw.HostProfiles},
	} {
		file := add(group.name, group.data)
		*group.files = append(*group.files, file)
		*group.selected = append(*group.selected, file.Location)
	}
	raw.Census = integrationTree(t, census)
	head := repo.Commit("normative Core journal integration fixture")
	source := &integrationSource{t: t, selection: selection, inputs: apply.CapturedInputs{Raw: raw, Head: head, Guard: "original-fixture-guard"}}
	metadata := func(boundary, name, data string) apply.ProfileMetadataFile {
		return apply.ProfileMetadataFile{Location: apply.SourceLocation{Boundary: boundary, Path: integrationPath(t, name)},
			ContentID: snapshot.ContentID(sha256.Sum256([]byte(data))), Size: int64(len(data)), Mode: snapshot.Regular}
	}
	profiles := integrationProfiles{metadata("controlled-executable", "program", "static executable fixture"), metadata("controlled-schema", "schema.json", "static schema fixture")}
	prepared, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, profiles, nil)
	if _, ok := prepared.Prepared(); err != nil || !ok {
		for _, a := range prepared.Attempts() {
			t.Logf("compile=%+v execution=%+v registry=%+v files=%+v defaults=%+v settings=%+v", a.Structural().Model().Compilation(), a.Structural().Execution().Diagnostics(), a.Structural().Registry().Diagnostics(), a.Structural().Files().Diagnostics(), a.Defaults().Diagnostics(), a.Settings().Diagnostics())
		}
		t.Fatalf("fixture preparation: %s %v", prepared.Status(), err)
	}
	owner, err := run.NewPayload("text/plain", "utf-8", []byte("Prepare the captured normative Core without dispatching work.\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	facts := apply.RunBindingFacts{RepositoryIdentity: "journal-integration-repository", ProductBuild: "journal-integration-build", OwnerRequest: owner,
		Allocation: run.Capacity{Runs: 2, SessionStarts: 70, Steps: 140, ExecutionTime: 7 * time.Hour}}
	if _, err := prepared.BindRun(facts); err != nil {
		t.Fatal("fixture binding", err)
	}
	options := apply.ControlOptions{StateDirectory: filepath.Join(testkit.TempDir(t), "store"), RestoreLimits: run.RestoreLimits{PageSize: 1, Batches: 20, Events: 40, Bytes: 16 << 20}}
	return integrationFixture{source: source, prepared: prepared, facts: facts, options: options,
		request: apply.ControlRequest{Repository: repo.Dir, RepositoryIdentity: facts.RepositoryIdentity, Revision: head, Guard: source.inputs.Guard, StateDirectory: options.StateDirectory}}
}

func (f integrationFixture) open(t *testing.T) *apply.ControlledRun {
	t.Helper()
	result, err := apply.NewControlService(f.source, New()).Open(context.Background(), f.prepared, f.facts, f.options)
	c, ok := result.Controlled()
	if err != nil || !ok {
		t.Fatal("open integrated control", err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	return c
}

// Same finite configuration fields as the accepted configuration example,
// without a library-specific Package override or an executable check selection.
const integrationExecution = `apiVersion: markitect.execution/v1
strategy: recursive-apply-v1
limits:
  maxRuns: 5
  maxSessionStarts: 160
  maxSteps: 320
  maxExecutionTime: 16h
  maxStepTimeout: 10m
  hostOperationTimeout: 2m
  maxRejectionLimit: 3
  maxContextBytes: 400000
  maxSessionSteps: 40
  maxTransientRetries: 2
  creationRoots: [realization]
assignment:
  runs: 3
  sessionStarts: 80
  steps: 160
  executionTime: 8h
completionReserve:
  sessionStarts: 24
  steps: 40
  executionTime: 2h
defaults:
  profiles: {manager: library-agent, reviewer: library-agent}
  stepTimeout: 10m
  creationRoots: [realization]
`

// Normative language-contract section 3.1 effective content, also exercised by
// internal/model/core_fixture_test.go. Core itself is the local model here;
// storage integration needs neither a new domain vocabulary nor dependencies.
func integrationCore() []model.Entry {
	entries := []model.Entry{
		{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: markitect-core\nversion: '1.0'\nroot: model\ndependencies: {}\n")},
		{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nmodels: {}\n")},
	}
	for _, d := range []struct{ path, id, kind, body, spec string }{
		{"kind", "Kind", "Kind", "Deklariert einen Typ. properties benennt die erlaubten Felder der spec seiner Definitionen. Der body einer Typdefinition beschreibt deren gemeinsame Bedeutung und geltende Festlegungen.", "properties: [properties]"},
		{"property", "Property", "Kind", "Deklariert einen Feldvertrag mit Werttyp und Kardinalität. Der body beschreibt die Bedeutung seiner Werte oder Beziehungen. Bei type object benennt properties die erlaubten Felder des Objektwerts.", "properties: [name, type, minCount, maxCount, values, target, properties]"},
		{"name", "name", "Property", "Optionaler ausdrücklich abweichender nichtleerer Feldname. Ohne Angabe gilt die vollständige lokale ID der Property als Feldname. Der effektive Feldname muss innerhalb jedes verwendenden Vertrags eindeutig sein.", "type: string, minCount: 0, maxCount: 1"},
		{"type", "type", "Property", "Der vom Core verstandene Werttyp dieser Property.", "type: enum, minCount: 1, maxCount: 1, values: [string, boolean, integer, number, enum, object, reference]"},
		{"min-count", "minCount", "Property", "Nichtnegative Mindestanzahl vorhandener Werte.", "type: integer, minCount: 1, maxCount: 1"},
		{"max-count", "maxCount", "Property", "Höchstanzahl vorhandener Werte. Der Wert -1 erlaubt beliebig viele Werte innerhalb der Werkzeuggrenzen. Andere Werte müssen mindestens minCount entsprechen. Genau 1 bezeichnet einen Einzelwert; andere Höchstwerte eine Liste.", "type: integer, minCount: 1, maxCount: 1"},
		{"values", "values", "Property", "Erlaubte Stringwerte einer enum-Property. Bei type enum ist eine nichtleere Liste eindeutiger Werte erforderlich. Bei anderen Werttypen ist diese Angabe nicht zulässig.", "type: string, minCount: 0, maxCount: -1"},
		{"target", "target", "Property", "Der verlangte Kind der referenzierten Definitionen. Bei type reference ist diese Angabe erforderlich. Bei anderen Werttypen ist sie nicht zulässig.", "type: reference, target: Kind, minCount: 0, maxCount: 1"},
		{"properties", "properties", "Property", "Referenziert die erlaubten Feldverträge des hier beschriebenen Objekts. Die effektiven Feldnamen müssen innerhalb dieses Vertrags eindeutig sein.", "type: reference, target: Property, minCount: 0, maxCount: -1"},
	} {
		entries = append(entries, model.Entry{Path: "model/markitect/core/" + d.path + ".yaml", Kind: model.Regular,
			Bytes: []byte(fmt.Sprintf("id: %s\nkind: %s\nbody: %s\nspec: {%s}\n", d.id, d.kind, d.body, d.spec))})
	}
	return entries
}
