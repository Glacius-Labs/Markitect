package apply_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// extractExecution is the execution document of a project at the start of an
// Extract: budget, profiles, the ownership of the artifacts as far as the
// project knows it (the overview file and the artifact tree belong to Root) and
// a check without requiredBy.
const extractExecution = `apiVersion: markitect.execution/v1
strategy: direct
budget:
  executionTime: 25m
  turnTimeout: 10m
  agentTurns: 120
  passes: 12
profiles:
  execute: library-agent
  verify: library-agent
packages: {}
ownership:
  - path: README.md
    owner: .
  - path: realization/
    owner: .
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    timeout: 2m
`

// extractOwnershipProposed is the ownership the structure proposal declares:
// each Package gets the directory of its artifacts.
const extractOwnershipProposed = `ownership:
  - path: README.md
    owner: .
  - path: realization/
    owner: .
  - path: realization/catalog/
    owner: catalog
  - path: realization/cmd/library/
    owner: library
    reason: The program entry belongs to the library responsibility.
  - path: realization/lending/
    owner: lending
  - path: realization/lending/reminders/
    owner: lending/reminders
  - path: realization/library/
    owner: library
  - path: realization/members/
    owner: members
`

// extractExecutionProposed is the same document with the ownership the
// structure proposal adds: each Package gets the directory of its artifacts.
const extractExecutionProposed = `apiVersion: markitect.execution/v1
strategy: direct
budget:
  executionTime: 25m
  turnTimeout: 10m
  agentTurns: 120
  passes: 12
profiles:
  execute: library-agent
  verify: library-agent
packages: {}
` + extractOwnershipProposed + `checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    timeout: 2m
`

// extractProposedCheck is a check the structure proposal may add, with the
// profile it names.
func extractProposedCheck(profile string) string {
	return "  - id: project-vet\n    profile: " + profile + "\n    directory: realization\n    args: [vet, ./...]\n    timeout: 2m\n"
}

var extractArtifacts = map[string]string{
	"README.md":                                    "Library\n",
	"realization/go.mod":                           "module library\n",
	"realization/library/library.go":               "package library\n",
	"realization/cmd/library/main.go":              "package main\n",
	"realization/members/member.go":                "package members\n",
	"realization/catalog/copy.go":                  "package catalog\n",
	"realization/lending/borrow.go":                "package lending\n",
	"realization/lending/reminders/remind.go":      "package reminders\n",
	"realization/lending/reminders/remind_test.go": "package reminders\n",
}

// extractInputs is the library case 3: the artifacts under the root
// realization, the imported vocabulary in the repository, a manifest and lock
// with no local Definitions and the execution document.
func extractInputs(t *testing.T) (apply.InputSelection, apply.RawInputs) {
	t.Helper()
	const base = "../../docs/design/canonical-model"
	local := t.TempDir()
	for _, name := range []string{"model.yaml", "model-lock.yaml"} {
		data, err := os.ReadFile(filepath.Join(base, "library-example/m1", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(local, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(local, "model"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "model", ".gitkeep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	raw := apply.RawInputs{Local: fixtureModel(t, local, ".markitect"), Dependencies: []apply.ModelSource{
		fixtureModel(t, base+"/library-example/vocabulary", ".markitect/dependencies/software"),
		fixtureModel(t, base+"/minimal-language/exchange-example/dependencies/core", ".markitect/dependencies/core"),
	}}
	raw.Local.RepositoryArea = area(t, ".markitect")
	var census []snapshot.Entry
	add := func(prefix string, source apply.ModelSource) {
		for _, entry := range source.Files.Entries() {
			file, err := snapshot.NewEntry(path(t, prefix+entry.Path().String()), entry.Mode(), entry.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			census = append(census, file)
		}
	}
	add(".markitect/", raw.Local)
	for i, dependency := range raw.Dependencies {
		raw.Dependencies[i].RepositoryArea = area(t, dependency.Selection.Boundary)
		add(dependency.Selection.Boundary+"/", dependency)
	}
	var err error
	raw.Census, err = snapshot.NewTree(census)
	if err != nil {
		t.Fatal(err)
	}
	selection := apply.InputSelection{Repository: "controlled-repository", Local: raw.Local.Selection}
	for _, dependency := range raw.Dependencies {
		selection.Dependencies = append(selection.Dependencies, dependency.Selection)
	}
	censusFile(t, &raw, ".markitect/execution.yaml", snapshot.Regular, []byte(extractExecution))
	for name, content := range extractArtifacts {
		censusFile(t, &raw, name, snapshot.Regular, []byte(content))
	}
	selection.Execution = []apply.SourceLocation{{Boundary: selection.Repository, Path: path(t, ".markitect/execution.yaml")}}
	entry, _ := raw.Census.Lookup(path(t, ".markitect/execution.yaml"))
	raw.Execution = []apply.SourceFile{{Location: selection.Execution[0], File: entry, RepositoryPath: entry.Path()}}
	addRuntimeDocument(t, &selection, &raw, testRuntimeDocument)
	return selection, raw
}

// extractAttempt prepares the library case 3 for Extract.
func extractAttempt(t *testing.T) apply.CapturedSettingsAttempt {
	t.Helper()
	selection, raw := extractInputs(t)
	return prepareExtract(t, selection, raw)
}

func prepareExtract(t *testing.T, selection apply.InputSelection, raw apply.RawInputs) apply.CapturedSettingsAttempt {
	t.Helper()
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		return captured(preparationObservation(t, request.Selection, raw)), nil
	}}
	agent, check := runtimePorts(t)
	result, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureExtract(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
	attempt, ok := result.Prepared()
	if err != nil || !ok {
		t.Fatalf("extract preparation failed: %s %v %+v", result.Status(), err, result.Attempts()[len(result.Attempts())-1].Settings().Diagnostics())
	}
	return attempt
}

// overview is the overview Definition of a Package.
func overview(id, body string, imports ...string) string {
	text := "imports:\n  - package: software\n    as: Software\n"
	for _, p := range imports {
		text += "  - package: " + p + "\n    as: " + strings.ToUpper(p[strings.LastIndex(p, "/")+1:][:1]) + p[strings.LastIndex(p, "/")+2:] + "\n"
	}
	return text + "id: " + id + "\nkind: Software.Concept\nbody: >-\n  " + body + "\nspec: {}\n"
}

// proposedStructure is the model files and execution document of a structure
// proposal for the library: five Packages, one overview Definition each.
func proposedStructure() map[string]string {
	return map[string]string{
		".markitect/model/library/library.yaml":             strings.Replace(overview("library", "The whole small library as one program that offers members, copies and lending."), "id: library\n", "id: library\ntitle: Small library\n", 1),
		".markitect/model/members/members.yaml":             overview("members", "Registered members and their status."),
		".markitect/model/catalog/catalog.yaml":             overview("catalog", "Physical copies of books and their availability."),
		".markitect/model/lending/lending.yaml":             overview("lending", "Loans of copies to members."),
		".markitect/model/lending/reminders/reminders.yaml": overview("reminders", "Overdue notices derived from open loans, see [[Lending.lending]].", "lending"),
		".markitect/execution.yaml":                         extractExecutionProposed,
	}
}

// proposedTree is the start with the structure proposal applied.
func proposedTree(t *testing.T, start snapshot.Tree) snapshot.Tree {
	t.Helper()
	changes := map[string]*string{}
	for name, content := range proposedStructure() {
		changes[name] = text(content)
	}
	return change(t, start, changes)
}

// registeredModel is the proposal plus the registration of every artifact in
// its owner's shard.
func registeredModel(t *testing.T, start snapshot.Tree) snapshot.Tree {
	t.Helper()
	changes := map[string]*string{
		rootShard:    text(shardYAML(shardEntry("README.md", "doc", "library/library"), shardEntry("realization/go.mod", "build", "library/library"))),
		libraryShard: text(shardYAML(shardEntry("realization/library/library.go", "code", "library/library"), shardEntry("realization/cmd/library/main.go", "code", "library/library"))),
		".markitect/registry/members/registry.yaml": text(shardYAML(shardEntry("realization/members/member.go", "code", "members/members"))),
		".markitect/registry/catalog/registry.yaml": text(shardYAML(shardEntry("realization/catalog/copy.go", "code", "catalog/catalog"))),
		lendingShard:   text(shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/lending"))),
		remindersShard: text(shardYAML(shardEntry("realization/lending/reminders/remind.go", "code", "lending/reminders/reminders"), shardEntry("realization/lending/reminders/remind_test.go", "test", "lending/reminders/reminders"))),
	}
	return change(t, proposedTree(t, start), changes)
}
