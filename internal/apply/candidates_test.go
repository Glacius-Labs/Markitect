package apply_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const (
	rootShard      = ".markitect/registry/registry.yaml"
	libraryShard   = ".markitect/registry/library/registry.yaml"
	lendingShard   = ".markitect/registry/lending/registry.yaml"
	remindersShard = ".markitect/registry/lending/reminders/registry.yaml"
)

func shardYAML(entries ...string) string {
	text := "apiVersion: markitect.registry/v1\nartifacts:\n"
	if len(entries) == 0 {
		return "apiVersion: markitect.registry/v1\nartifacts: []\n"
	}
	for _, entry := range entries {
		text += entry
	}
	return text
}

func shardEntry(path, form, realizes string) string {
	return "  - path: " + path + "\n    form: " + form + "\n    realizes: [" + realizes + "]\n    why: Fixture artifact.\n"
}

// libraryLayoutFiles is a completely accounted library project under the layout
// roots [realization] with the exception realization/cmd/library -> library.
func libraryLayoutFiles(t *testing.T, selection *apply.InputSelection, raw *apply.RawInputs) {
	t.Helper()
	for name, content := range map[string]string{
		"realization/go.mod":                           "module library\n",
		"realization/library/library.go":               "package library\n",
		"realization/cmd/library/main.go":              "package main\n",
		"realization/lending/borrow.go":                "package lending\n",
		"realization/lending/reminders/remind.go":      "package reminders\n",
		"realization/lending/reminders/remind_test.go": "package reminders\n",
	} {
		censusFile(t, raw, name, snapshot.Regular, []byte(content))
	}
	for name, content := range map[string]string{
		rootShard:      shardYAML(shardEntry("realization/go.mod", "build", "library/library")),
		libraryShard:   shardYAML(shardEntry("realization/library/library.go", "code", "library/library"), shardEntry("realization/cmd/library/main.go", "code", "library/callable-interface")),
		lendingShard:   shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/borrow")),
		remindersShard: shardYAML(shardEntry("realization/lending/reminders/remind.go", "code", "lending/reminders/assess-overdue"), shardEntry("realization/lending/reminders/remind_test.go", "test", "lending/reminders/assess-overdue")),
	} {
		registrySource(t, selection, raw, name, content, true)
	}
}

func preparedAttempt(t *testing.T, selection apply.InputSelection, raw apply.RawInputs) apply.CapturedSettingsAttempt {
	t.Helper()
	addHostDocument(t, &selection, &raw, testHostDocument)
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		return captured(preparationObservation(t, request.Selection, raw)), nil
	}}
	agent, check := hostPorts(t)
	result, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
	attempt, ok := result.Prepared()
	if err != nil || !ok {
		t.Fatalf("preparation failed: %s %v %+v", result.Status(), err, result.Attempts()[len(result.Attempts())-1].Settings().Diagnostics())
	}
	return attempt
}

func layoutCandidates(t *testing.T) (apply.Candidates, snapshot.Tree) {
	t.Helper()
	selection, raw := executionInputs(t, withLayout(t, libraryLayoutYAML))
	libraryLayoutFiles(t, &selection, &raw)
	attempt := preparedAttempt(t, selection, raw)
	candidates, err := apply.NewCandidates(attempt)
	if err != nil {
		t.Fatal(err)
	}
	return candidates, attempt.Structural().Model().Inputs().Raw.Census
}

// change returns the tree with the given files set (nil content deletes them).
func change(t *testing.T, tree snapshot.Tree, changes map[string]*string) snapshot.Tree {
	t.Helper()
	var entries []snapshot.Entry
	for _, entry := range tree.Entries() {
		content, touched := changes[entry.Path().String()]
		if !touched {
			entries = append(entries, entry)
			continue
		}
		delete(changes, entry.Path().String())
		if content == nil {
			continue
		}
		replaced, err := snapshot.NewEntry(entry.Path(), entry.Mode(), []byte(*content))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, replaced)
	}
	for name, content := range changes {
		if content == nil {
			t.Fatalf("cannot delete absent file %s", name)
		}
		added, err := snapshot.NewEntry(path(t, name), snapshot.Regular, []byte(*content))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, added)
	}
	result, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func text(s string) *string { return &s }

// summary renders findings as "owner code path" lines for exact comparison.
func summary(findings []apply.CandidateFinding) []string {
	var lines []string
	for _, finding := range findings {
		if finding.Message == "" {
			lines = append(lines, "MISSING MESSAGE for "+finding.Code)
		}
		lines = append(lines, finding.Owner+" "+finding.Code+" "+finding.Path)
	}
	return lines
}

func TestCandidatesLibraryLayoutOwnersShardsAndAreas(t *testing.T) {
	candidates, start := layoutCandidates(t)
	for name, want := range map[string]string{
		"realization/go.mod":                      ".",
		"realization/cmd/library/main.go":         "library",
		"realization/library/library.go":          "library",
		"realization/lending/borrow.go":           "lending",
		"realization/lending/reminders/x.go":      "lending/reminders",
		"realization/lending/reminders/deep/y.go": "lending/reminders",
		"realization/unknown/file.go":             ".",
		"docs/guide.md":                           ".",
		".markitect/execution.yaml":               ".",
		rootShard:                                 ".",
		lendingShard:                              "lending",
		remindersShard:                            "lending/reminders",
	} {
		if got := candidates.OwnerOf(path(t, name)); got != want {
			t.Errorf("owner of %s = %q, want %q", name, got, want)
		}
	}
	for manager, want := range map[string]string{
		".": rootShard, "library": libraryShard, "lending": lendingShard, "lending/reminders": remindersShard,
	} {
		if got := candidates.Shard(manager); got != want {
			t.Errorf("shard of %s = %q, want %q", manager, got, want)
		}
	}
	for manager, want := range map[string][]string{
		"lending":           {"realization/lending/** except realization/lending/reminders/**"},
		"lending/reminders": {"realization/lending/reminders/**"},
		"library":           {"realization/library/**", "realization/cmd/library/**"},
		".":                 {"shared files in realization/ such as realization/go.mod", "every other file of the repository outside the roots and exceptions"},
		"missing":           nil,
	} {
		if got := candidates.Areas(manager); !reflect.DeepEqual(got, want) {
			t.Errorf("areas of %s = %q, want %q", manager, got, want)
		}
	}
	if findings := candidates.Account(start); len(findings) != 0 {
		t.Fatalf("the captured start is not accounted: %v", summary(findings))
	}
}

func TestCandidatesAccountNewFilesAndRegistrationsByTheirOwner(t *testing.T) {
	candidates, start := layoutCandidates(t)
	// A new file registered by its area's Manager is accounted.
	good := change(t, start, map[string]*string{
		"realization/lending/reminders/schedule.go": text("package reminders\n"),
		remindersShard: text(shardYAML(
			shardEntry("realization/lending/reminders/remind.go", "code", "lending/reminders/assess-overdue"),
			shardEntry("realization/lending/reminders/remind_test.go", "test", "lending/reminders/assess-overdue"),
			shardEntry("realization/lending/reminders/schedule.go", "code", "lending/reminders/assess-overdue"))),
	})
	if findings := candidates.Account(good); len(findings) != 0 {
		t.Fatalf("accounted candidate refused: %v", summary(findings))
	}
	// A new file without registration belongs to the Manager of its area.
	unregistered := change(t, start, map[string]*string{"realization/lending/reminders/schedule.go": text("package reminders\n"), "realization/cmd/library/flags.go": text("package main\n"), "realization/Makefile": text("all:\n")})
	findings := candidates.Account(unregistered)
	if want := []string{". file.unexplained realization/Makefile", "lending/reminders file.unexplained realization/lending/reminders/schedule.go", "library file.unexplained realization/cmd/library/flags.go"}; !reflect.DeepEqual(summary(findings), want) {
		t.Fatalf("unexplained files: %v, want %v", summary(findings), want)
	}
	if !strings.Contains(findings[1].Message, remindersShard) || !strings.Contains(findings[2].Message, libraryShard) {
		t.Fatalf("message must name the owner's shard: %q", findings[1].Message)
	}
	// The parent registered the child's file in its own shard: the registrant must fix it.
	wrong := change(t, start, map[string]*string{
		"realization/lending/reminders/schedule.go": text("package reminders\n"),
		lendingShard: text(shardYAML(
			shardEntry("realization/lending/borrow.go", "code", "lending/borrow"),
			shardEntry("realization/lending/reminders/schedule.go", "code", "lending/borrow"))),
	})
	findings = candidates.Account(wrong)
	if want := []string{"lending registry.outside_area realization/lending/reminders/schedule.go"}; !reflect.DeepEqual(summary(findings), want) {
		t.Fatalf("registration in the wrong shard: %v, want %v", summary(findings), want)
	}
	if !strings.Contains(findings[0].Message, `"lending/reminders"`) || !strings.Contains(findings[0].Message, `"lending"`) {
		t.Fatalf("the message must name the actual owner and the registrant: %q", findings[0].Message)
	}
	// Root registering a Package's file, and a Package registering the exception area of another.
	crossed := change(t, start, map[string]*string{
		rootShard:    text(shardYAML(shardEntry("realization/go.mod", "build", "library/library"), shardEntry("realization/cmd/library/main.go", "code", "library/callable-interface"))),
		libraryShard: text(shardYAML(shardEntry("realization/library/library.go", "code", "library/library"))),
	})
	if got, want := summary(candidates.Account(crossed)), []string{". registry.outside_area realization/cmd/library/main.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("exception area registered by Root: %v, want %v", got, want)
	}
}

func TestCandidatesRegistryFaultsGoToTheShardsManager(t *testing.T) {
	candidates, start := layoutCandidates(t)
	for name, test := range map[string]struct {
		changes map[string]*string
		want    []string
	}{
		"deleted registered file": {
			map[string]*string{"realization/lending/borrow.go": nil},
			[]string{"lending registry.artifact_missing " + lendingShard},
		},
		"shard is not YAML": {
			map[string]*string{lendingShard: text("apiVersion: [unterminated")},
			[]string{"lending registry.yaml " + lendingShard, "lending file.unexplained realization/lending/borrow.go"},
		},
		"unknown definition": {
			map[string]*string{lendingShard: text(shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/missing")))},
			[]string{"lending registry.definition_missing " + lendingShard, "lending file.unexplained realization/lending/borrow.go"},
		},
		"record for an absent file": {
			map[string]*string{lendingShard: text(shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/borrow"), shardEntry("realization/lending/ghost.go", "code", "lending/borrow")))},
			[]string{"lending registry.artifact_missing " + lendingShard},
		},
		"shard of a Package outside the model": {
			map[string]*string{".markitect/registry/nope/registry.yaml": text(shardYAML())},
			[]string{". registry.unknown_owner .markitect/registry/nope/registry.yaml"},
		},
		"file in the registry directory": {
			map[string]*string{".markitect/registry/notes.txt": text("notes")},
			[]string{". registry.layout .markitect/registry/notes.txt"},
		},
		"one path in two shards": {
			map[string]*string{rootShard: text(shardYAML(shardEntry("realization/go.mod", "build", "library/library"), shardEntry("realization/lending/borrow.go", "code", "lending/borrow")))},
			[]string{". registry.duplicate realization/lending/borrow.go", "lending registry.duplicate realization/lending/borrow.go", ". registry.outside_area realization/lending/borrow.go"},
		},
		"a Manager removes its shard": {
			map[string]*string{remindersShard: nil},
			[]string{"lending/reminders file.unexplained realization/lending/reminders/remind.go", "lending/reminders file.unexplained realization/lending/reminders/remind_test.go"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := summary(candidates.Account(change(t, start, test.changes)))
			slices.Sort(got)
			want := slices.Clone(test.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("findings %q, want %q", got, want)
			}
		})
	}
}

func TestCandidatesProtectedInputsAreNotManagerWork(t *testing.T) {
	candidates, start := layoutCandidates(t)
	var model string
	for _, entry := range start.Entries() {
		if strings.HasPrefix(entry.Path().String(), ".markitect/model/") {
			model = entry.Path().String()
			break
		}
	}
	if model == "" {
		t.Fatal("fixture has no canonical model file")
	}
	for name, test := range map[string]struct {
		changes map[string]*string
		want    []string
	}{
		"model changed":                  {map[string]*string{model: text("changed: true\n")}, []string{". input.changed " + model}},
		"model deleted":                  {map[string]*string{model: nil}, []string{". input.deleted " + model}},
		"model file added":               {map[string]*string{".markitect/model/new.yaml": text("id: x\n")}, []string{". input.added .markitect/model/new.yaml"}},
		"execution changed":              {map[string]*string{".markitect/execution.yaml": text("changed\n")}, []string{". input.changed .markitect/execution.yaml"}},
		"execution deleted":              {map[string]*string{".markitect/execution.yaml": nil}, []string{". input.deleted .markitect/execution.yaml"}},
		"ignore added":                   {map[string]*string{".markitect/ignore.yaml": text("apiVersion: markitect.ignore/v1\nfiles: []\n")}, []string{". input.added .markitect/ignore.yaml"}},
		"unprotected file in .markitect": {map[string]*string{".markitect/notes.md": text("notes")}, []string{". file.unexplained .markitect/notes.md"}},
	} {
		t.Run(name, func(t *testing.T) {
			got := summary(candidates.Account(change(t, start, test.changes)))
			if !slices.Equal(got, test.want) {
				t.Fatalf("findings %q, want %q", got, test.want)
			}
		})
	}
}

func TestCandidatesProtectedFindingsQuoteTheOriginal(t *testing.T) {
	candidates, start := layoutCandidates(t)
	execution, _ := start.Lookup(path(t, ".markitect/execution.yaml"))
	for name, test := range map[string]struct {
		changes map[string]*string
		quoted  bool
	}{
		"changed": {map[string]*string{".markitect/execution.yaml": text("changed\n")}, true},
		"deleted": {map[string]*string{".markitect/execution.yaml": nil}, true},
		"added":   {map[string]*string{".markitect/model/new.yaml": text("id: x\n")}, false},
	} {
		t.Run(name, func(t *testing.T) {
			findings := candidates.Account(change(t, start, test.changes))
			if len(findings) != 1 {
				t.Fatalf("findings %v", summary(findings))
			}
			if got := strings.Contains(findings[0].Message, "BEGIN\n"+string(execution.Bytes())+"\nEND"); got != test.quoted {
				t.Fatalf("original quoted = %v, want %v: %q", got, test.quoted, findings[0].Message)
			}
		})
	}
	large := make([]byte, 8<<10+1)
	for i := range large {
		large[i] = 'a'
	}
	binary := []byte{0xff, 0xfe}
	for name, content := range map[string][]byte{"too large": large, "not UTF-8": binary} {
		entry, err := snapshot.NewEntry(path(t, "x"), snapshot.Regular, content)
		if err != nil {
			t.Fatal(err)
		}
		quoted := 0
		got := apply.OriginalText(entry, "input.changed", &quoted)
		if strings.Contains(got, "BEGIN") || !strings.Contains(got, entry.ContentID().String()) || !strings.Contains(got, strconv.Itoa(len(content))) {
			t.Errorf("%s: want size and SHA-256 only, got %q", name, got)
		}
	}
	entry, _ := snapshot.NewEntry(path(t, "x"), snapshot.Regular, large[:8<<10])
	quoted := 0
	if got := apply.OriginalText(entry, "input.deleted", &quoted); !strings.Contains(got, "BEGIN") {
		t.Errorf("an input of exactly 8 KiB must be quoted, got a digest")
	}
	// All quotes of one candidate stay within 32 KiB: four 8 KiB originals are
	// quoted, the fifth gets its digest.
	for i := 2; i <= 5; i++ {
		got := apply.OriginalText(entry, "input.deleted", &quoted)
		if strings.Contains(got, "BEGIN") != (i <= 4) {
			t.Errorf("original %d: quoted %v with %d bytes quoted before", i, strings.Contains(got, "BEGIN"), quoted)
		}
	}
}

func TestCandidatesWithoutLayoutKeepShardOwnership(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	attempt := preparedAttempt(t, selection, raw)
	candidates, err := apply.NewCandidates(attempt)
	if err != nil {
		t.Fatal(err)
	}
	start := attempt.Structural().Model().Inputs().Raw.Census
	for name, want := range map[string]string{
		"realization/library.md": ".", "realization/lending.go": "lending", "realization/new.go": ".", lendingShard: "lending",
	} {
		if got := candidates.OwnerOf(path(t, name)); got != want {
			t.Errorf("owner of %s = %q, want %q", name, got, want)
		}
	}
	if got, want := candidates.Areas("lending"), []string{"the files it registers in " + lendingShard}; !reflect.DeepEqual(got, want) {
		t.Errorf("areas = %q, want %q", got, want)
	}
	if findings := candidates.Account(start); len(findings) != 0 {
		t.Fatalf("start refused: %v", summary(findings))
	}
	// Without a layout any Manager may register a new file in its own shard.
	registered := change(t, start, map[string]*string{
		"realization/extra.go": text("package x\n"),
		lendingShard:           text(shardYAML(shardEntry("realization/lending.go", "code", "lending/borrow"), shardEntry("realization/extra.go", "code", "lending/borrow"))),
	})
	if findings := candidates.Account(registered); len(findings) != 0 {
		t.Fatalf("registration in own shard refused: %v", summary(findings))
	}
	unregistered := change(t, start, map[string]*string{"realization/extra.go": text("package x\n")})
	findings := candidates.Account(unregistered)
	if got, want := summary(findings), []string{". file.unexplained realization/extra.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %q, want %q", got, want)
	}
	// Root is the addressee but not the owner: the creator registers in its own shard.
	message := findings[0].Message
	if strings.Contains(message, "its owner") || !strings.Contains(message, "created this file registers it in its own registry shard") || !strings.Contains(message, "asks its children") {
		t.Fatalf("undeclared message invites an ownership transfer: %q", message)
	}
}

func TestCandidatesRequireAPassedPreparation(t *testing.T) {
	if _, err := apply.NewCandidates(apply.CapturedSettingsAttempt{}); !errors.Is(err, apply.ErrCandidatesUnavailable) {
		t.Fatalf("zero preparation accepted: %v", err)
	}
	selection, raw := executionInputs(t, withLayout(t, libraryLayoutYAML))
	libraryLayoutFiles(t, &selection, &raw)
	censusFile(t, &raw, "realization/stray.go", snapshot.Regular, []byte("package stray\n"))
	addHostDocument(t, &selection, &raw, testHostDocument)
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		return captured(preparationObservation(t, request.Selection, raw)), nil
	}}
	agent, check := hostPorts(t)
	result, _ := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
	if len(result.Attempts()) == 0 || result.Status() == model.Passed {
		t.Fatal("an unaccounted start must not pass preparation")
	}
	if _, err := apply.NewCandidates(result.Attempts()[len(result.Attempts())-1]); !errors.Is(err, apply.ErrCandidatesUnavailable) {
		t.Fatalf("unaccounted start accepted: %v", err)
	}
}

func TestFileCheckAppliesTheDeclaredLayoutBeforeAnyWork(t *testing.T) {
	selection, raw := executionInputs(t, withLayout(t, libraryLayoutYAML))
	libraryLayoutFiles(t, &selection, &raw)
	passed := checkedFiles(t, selection, raw)
	layout, declared := passed.Layout()
	if passed.Status() != model.Passed || !declared || layout.OwnerOf(path(t, "realization/cmd/library/main.go")).IsRoot() {
		t.Fatalf("declared layout unavailable after success: %s %+v", passed.Status(), passed.Diagnostics())
	}
	censusFile(t, &raw, "realization/lending/reminders/extra.go", snapshot.Regular, []byte("package reminders"))
	wrongShard := shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/borrow"), shardEntry("realization/lending/reminders/extra.go", "code", "lending/reminders/assess-overdue"))
	for i := range raw.Registry {
		if raw.Registry[i].RepositoryPath.String() == lendingShard {
			rewriteFileCheckSource(t, &raw, &raw.Registry[i], wrongShard)
		}
	}
	failed := checkedFiles(t, selection, raw)
	accounting, ran := failed.Accounting()
	if failed.Status() != model.Failed || !ran || len(accounting.Findings()) == 0 {
		t.Fatalf("a registration outside its owner's shard passed preparation: %s %+v", failed.Status(), accounting.Findings())
	}
	found := false
	for _, finding := range accounting.Findings() {
		found = found || finding.Code == "registry.outside_area" && finding.Path.String() == "realization/lending/reminders/extra.go"
	}
	if !found {
		t.Fatalf("outside-area finding missing: %+v", accounting.Findings())
	}
	if _, available := failed.Layout(); available {
		t.Fatal("a failed file check exposed its layout")
	}
	selection, raw = fileCheckInputs(t)
	if _, declared := checkedFiles(t, selection, raw).Layout(); declared {
		t.Fatal("a project without a layout reported one")
	}
}
