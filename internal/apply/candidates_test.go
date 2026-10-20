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
	"github.com/Glacius-Labs/Markitect/internal/run"
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

// libraryLayoutFiles is a completely accounted library project under the
// ownership of libraryOwnershipYAML.
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
	addRuntimeDocument(t, &selection, &raw, testRuntimeDocument)
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		return captured(preparationObservation(t, request.Selection, raw)), nil
	}}
	agent, check := runtimePorts(t)
	result, err := apply.NewPreparationService(apply.NewModelService(source, nil)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
	attempt, ok := result.Prepared()
	if err != nil || !ok {
		t.Fatalf("preparation failed: %s %v %+v", result.Status(), err, result.Attempts()[len(result.Attempts())-1].Settings().Diagnostics())
	}
	return attempt
}

func layoutCandidates(t *testing.T) (apply.Candidates, snapshot.Tree) {
	t.Helper()
	selection, raw := executionInputs(t, withOwnership(t, libraryOwnershipYAML))
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

func TestCandidatesLibraryOwnershipOwnersShardsAndAreas(t *testing.T) {
	candidates, start := layoutCandidates(t)
	for name, want := range map[string]string{
		"realization/go.mod":                      ".",
		"realization/cmd/library/main.go":         "library",
		"realization/library/library.go":          "library",
		"realization/lending/borrow.go":           "lending",
		"realization/lending/reminders/x.go":      "lending/reminders",
		"realization/lending/reminders/deep/y.go": "lending/reminders",
		"realization/unknown/file.go":             ".",
		// Nothing declares an owner for these: nobody owns them.
		"docs/guide.md":             "",
		".markitect/execution.yaml": "",
		rootShard:                   ".",
		lendingShard:                "lending",
		remindersShard:              "lending/reminders",
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
		"lending":           {`realization/lending/ except realization/lending/reminders/ (Manager "lending/reminders")`},
		"lending/reminders": {"realization/lending/reminders/"},
		"library":           {"realization/cmd/library/", "realization/library/"},
		".":                 {`realization/ except realization/cmd/library/ (Manager "library"), realization/lending/ (Manager "lending"), realization/library/ (Manager "library")`},
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
	// The registrant must fix its entry, and the file stays unexplained for its
	// actual owner: another owner's entry neither explains nor takes a file.
	if want := []string{"lending registry.outside_area realization/lending/reminders/schedule.go", "lending/reminders file.unexplained realization/lending/reminders/schedule.go"}; !reflect.DeepEqual(summary(findings), want) {
		t.Fatalf("registration in the wrong shard: %v, want %v", summary(findings), want)
	}
	if !strings.Contains(findings[0].Message, `"lending/reminders"`) || !strings.Contains(findings[0].Message, `"lending"`) {
		t.Fatalf("the message must name the actual owner and the registrant: %q", findings[0].Message)
	}
	if !strings.Contains(findings[1].Message, remindersShard) {
		t.Fatalf("the owner must be told where to register the file: %q", findings[1].Message)
	}
	// Root registering a Package's file, and a Package registering the exception area of another.
	crossed := change(t, start, map[string]*string{
		rootShard:    text(shardYAML(shardEntry("realization/go.mod", "build", "library/library"), shardEntry("realization/cmd/library/main.go", "code", "library/callable-interface"))),
		libraryShard: text(shardYAML(shardEntry("realization/library/library.go", "code", "library/library"))),
	})
	if got, want := summary(candidates.Account(crossed)), []string{". registry.outside_area realization/cmd/library/main.go", "library file.unexplained realization/cmd/library/main.go"}; !reflect.DeepEqual(got, want) {
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
			[]string{". registry.duplicate realization/lending/borrow.go", "lending registry.duplicate realization/lending/borrow.go", ". registry.outside_area realization/lending/borrow.go", "lending file.unexplained realization/lending/borrow.go"},
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

func TestCandidatesWithoutOwnershipKeepShardOwnership(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	attempt := preparedAttempt(t, selection, raw)
	candidates, err := apply.NewCandidates(attempt)
	if err != nil {
		t.Fatal(err)
	}
	start := attempt.Structural().Model().Inputs().Raw.Census
	for name, want := range map[string]string{
		"realization/library.md": ".", "realization/lending.go": "lending", "realization/new.go": "", lendingShard: "lending",
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
	// Without declared ownership a Manager may register a new file in its own shard.
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
	// Nobody owns the file: Root is the addressee, and the message does not
	// invite an ownership transfer.
	message := findings[0].Message
	if findings[0].Owner != "." || strings.Contains(message, "its owner") || !strings.Contains(message, "neither a registered artifact") {
		t.Fatalf("message of an unowned file: %+v", findings[0])
	}
}

func TestCandidatesRequireAPassedPreparation(t *testing.T) {
	if _, err := apply.NewCandidates(apply.CapturedSettingsAttempt{}); !errors.Is(err, apply.ErrCandidatesUnavailable) {
		t.Fatalf("zero preparation accepted: %v", err)
	}
	service := func(t *testing.T, execution, stray string) (*apply.PreparationService, apply.InputSelection) {
		selection, raw := executionInputs(t, execution)
		libraryLayoutFiles(t, &selection, &raw)
		censusFile(t, &raw, stray, snapshot.Regular, []byte("package stray\n"))
		addRuntimeDocument(t, &selection, &raw, testRuntimeDocument)
		source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
			return captured(preparationObservation(t, request.Selection, raw)), nil
		}}
		return apply.NewPreparationService(apply.NewModelService(source, nil)), selection
	}
	prepareWith := func(t *testing.T, execution, stray string) apply.CapturedSettingsResult {
		s, selection := service(t, execution, stray)
		agent, check := runtimePorts(t)
		result, _ := s.CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, agent, check)
		return result
	}
	// The initial File check is strict: a committed file that is not registered
	// fails preparation for Apply, whoever would own it. There is no repairable
	// start drift a Manager could fix during the run.
	for name, stray := range map[string]string{
		"a file a Package owns":           "realization/lending/stray.go",
		"a file Root owns":                "realization/stray.go",
		"a file in an unmapped directory": "realization/unmapped/stray.go",
		"a file no area owns":             "docs/stray.md",
	} {
		t.Run(name, func(t *testing.T) {
			result := prepareWith(t, withOwnership(t, libraryOwnershipYAML), stray)
			if _, ok := result.Prepared(); ok || result.Status() == model.Passed || len(result.Attempts()) == 0 {
				t.Fatalf("an unregistered start file passed preparation: %s", result.Status())
			}
			if _, err := apply.NewCandidates(result.Attempts()[len(result.Attempts())-1]); !errors.Is(err, apply.ErrCandidatesUnavailable) {
				t.Fatalf("an unaccounted start was accepted: %v", err)
			}
		})
	}
	// Without declared ownership the same file fails too.
	result := prepareWith(t, documentedExecution(t), "realization/stray.go")
	if len(result.Attempts()) == 0 || result.Status() == model.Passed {
		t.Fatal("an unexplained start file passed preparation without ownership")
	}
	// Strict structural preparation (markitect check) refuses the same file.
	s, selection := service(t, withOwnership(t, libraryOwnershipYAML), "realization/stray.go")
	strict, _ := s.Capture(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if strict.Status() == model.Passed {
		t.Fatal("strict preparation accepted an unregistered file")
	}
}

func TestFileCheckAppliesTheDeclaredOwnershipBeforeAnyWork(t *testing.T) {
	selection, raw := executionInputs(t, withOwnership(t, libraryOwnershipYAML))
	libraryLayoutFiles(t, &selection, &raw)
	passed := checkedFiles(t, selection, raw)
	layout, declared := passed.Layout()
	if owner, owned := layout.OwnerOf(path(t, "realization/cmd/library/main.go")); passed.Status() != model.Passed || !declared || !owned || owner.IsRoot() {
		t.Fatalf("declared ownership unavailable after success: %s %+v", passed.Status(), passed.Diagnostics())
	}
	// An unregistered committed file in a Manager's area fails the initial File
	// check; its finding is addressed to the owner of the path.
	stray := raw
	censusFile(t, &stray, "realization/lending/stray.go", snapshot.Regular, []byte("package lending\n"))
	unregistered := checkedFiles(t, selection, stray)
	accounting, ran := unregistered.Accounting()
	if unregistered.Status() != model.Failed || !ran {
		t.Fatalf("an unregistered committed file passed the initial File check: %s %+v", unregistered.Status(), accounting.Findings())
	}
	found := false
	for _, finding := range accounting.Findings() {
		found = found || finding.Code == "file.unexplained" && finding.Path.String() == "realization/lending/stray.go"
	}
	if !found {
		t.Fatalf("file.unexplained missing: %+v", accounting.Findings())
	}
	censusFile(t, &raw, "realization/lending/reminders/extra.go", snapshot.Regular, []byte("package reminders"))
	wrongShard := shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/borrow"), shardEntry("realization/lending/reminders/extra.go", "code", "lending/reminders/assess-overdue"))
	for i := range raw.Registry {
		if raw.Registry[i].RepositoryPath.String() == lendingShard {
			rewriteFileCheckSource(t, &raw, &raw.Registry[i], wrongShard)
		}
	}
	failed := checkedFiles(t, selection, raw)
	accounting, ran = failed.Accounting()
	if failed.Status() != model.Failed || !ran || len(accounting.Findings()) == 0 {
		t.Fatalf("a registration outside its owner's shard passed preparation: %s %+v", failed.Status(), accounting.Findings())
	}
	found = false
	for _, finding := range accounting.Findings() {
		found = found || finding.Code == "registry.outside_area" && finding.Path.String() == "realization/lending/reminders/extra.go"
	}
	if !found {
		t.Fatalf("outside-area finding missing: %+v", accounting.Findings())
	}
	if _, available := failed.Layout(); available {
		t.Fatal("a failed file check exposed its ownership")
	}
	selection, raw = fileCheckInputs(t)
	if _, declared := checkedFiles(t, selection, raw).Layout(); declared {
		t.Fatal("a project without ownership reported one")
	}
}

func TestCandidatesNoOwnerFilesBlockAndGoToRoot(t *testing.T) {
	candidates, start := layoutCandidates(t)
	stray := change(t, start, map[string]*string{"docs/guide.md": text("# Guide\n"), "realization/Makefile": text("all:\n")})
	findings := candidates.Account(stray)
	want := []string{". layout.no_owner docs/guide.md", ". file.unexplained realization/Makefile"}
	if got := summary(findings); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %v, want %v", got, want)
	}
	if !strings.Contains(findings[0].Message, "needs an area for it, or the file must go") {
		t.Fatalf("message must say how to resolve it: %q", findings[0].Message)
	}
	if !strings.Contains(findings[1].Message, `Register it, or a directory holding it, in the local registry of its owner "."`) || !strings.Contains(findings[1].Message, rootShard) {
		t.Fatalf("message must name the owner's shard: %q", findings[1].Message)
	}
	if candidates.OwnerOf(path(t, "docs/guide.md")) != "" {
		t.Fatal("a file no area covers has an owner")
	}
}

func TestCandidatesWritePlansOfTheLibraryOwnership(t *testing.T) {
	candidates, _ := layoutCandidates(t)
	ro := func(p string) run.PathMount { return run.PathMount{Path: p, Writable: false} }
	rw := func(p string) run.PathMount { return run.PathMount{Path: p, Writable: true} }
	// Nothing declares a whole-repository area, so no Manager has a writable
	// base: only the areas and the Manager's own registry shard are writable, and
	// everything else, .markitect included, stays read-only without a mount.
	for manager, want := range map[string]run.WritePlan{
		".": {Mounts: []run.PathMount{
			rw(".markitect/registry"),
			ro(".markitect/registry/catalog"), ro(".markitect/registry/lending"), ro(".markitect/registry/library"), ro(".markitect/registry/members"),
			rw("realization"), ro("realization/cmd/library"), ro("realization/lending"), ro("realization/library"),
		}},
		"library": {Mounts: []run.PathMount{
			rw(".markitect/registry/library"), rw("realization/cmd/library"), rw("realization/library"),
		}},
		"lending": {Mounts: []run.PathMount{
			rw(".markitect/registry/lending"), ro(".markitect/registry/lending/reminders"),
			rw("realization/lending"), ro("realization/lending/reminders"),
		}},
		"lending/reminders": {Mounts: []run.PathMount{
			rw(".markitect/registry/lending/reminders"), rw("realization/lending/reminders"),
		}},
	} {
		got, err := candidates.WritePlan(manager)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("write plan of %s: %v\n got %+v\nwant %+v", manager, err, got, want)
		}
	}
}

func TestCandidatesWithoutOwnershipRefuseEveryWritePlan(t *testing.T) {
	selection, raw := executionInputs(t, documentedExecution(t))
	libraryLayoutFiles(t, &selection, &raw)
	candidates, err := apply.NewCandidates(preparedAttempt(t, selection, raw))
	if err != nil {
		t.Fatal(err)
	}
	for _, manager := range []string{".", "lending", "library"} {
		if plan, err := candidates.WritePlan(manager); !errors.Is(err, apply.ErrUnsupportedOwnership) || !strings.Contains(err.Error(), "declares no ownership") || len(plan.Mounts) != 0 || plan.Base {
			t.Errorf("%s: a project without declared ownership got a plan: %+v %v", manager, plan, err)
		}
	}
}

// foreignFileOwnership adds an exact file of Root inside the lending directory.
const foreignFileOwnership = libraryOwnershipYAML + "  - path: realization/lending/Makefile\n    owner: .\n    reason: The build file lives with lending but belongs to Root.\n"

func TestCandidatesMountAForeignExactFileInsideAManagersDirectory(t *testing.T) {
	selection, raw := executionInputs(t, withOwnership(t, foreignFileOwnership))
	libraryLayoutFiles(t, &selection, &raw)
	censusFile(t, &raw, "realization/lending/Makefile", snapshot.Regular, []byte("all:\n"))
	for i := range raw.Registry {
		if raw.Registry[i].RepositoryPath.String() == rootShard {
			rewriteFileCheckSource(t, &raw, &raw.Registry[i], shardYAML(shardEntry("realization/go.mod", "build", "library/library"), shardEntry("realization/lending/Makefile", "build", "library/library")))
		}
	}
	attempt := preparedAttempt(t, selection, raw)
	candidates, err := apply.NewCandidates(attempt)
	if err != nil {
		t.Fatal(err)
	}
	if owner := candidates.OwnerOf(path(t, "realization/lending/Makefile")); owner != "." {
		t.Errorf("owner of the exact file %q: the exact file is more specific than its directory", owner)
	}
	if owner := candidates.OwnerOf(path(t, "realization/lending/borrow.go")); owner != "lending" {
		t.Errorf("owner of the directory's other files %q", owner)
	}
	// The directory's owner cannot write the foreign file, and the file's owner
	// can write that one file inside a directory it otherwise only reads.
	lending, err := candidates.WritePlan("lending")
	if err != nil {
		t.Fatal(err)
	}
	if want := (run.PathMount{Path: "realization/lending/Makefile", Writable: false, File: true}); !slices.Contains(lending.Mounts, want) {
		t.Errorf("lending can write the foreign exact file: %+v", lending.Mounts)
	}
	root, err := candidates.WritePlan(".")
	if err != nil {
		t.Fatal(err)
	}
	if want := (run.PathMount{Path: "realization/lending/Makefile", Writable: true, File: true}); !slices.Contains(root.Mounts, want) {
		t.Errorf("Root cannot write its exact file inside lending's directory: %+v", root.Mounts)
	}
	lendingDir := slices.Index(root.Mounts, run.PathMount{Path: "realization/lending"})
	file := slices.Index(root.Mounts, run.PathMount{Path: "realization/lending/Makefile", Writable: true, File: true})
	if lendingDir < 0 || file < lendingDir {
		t.Errorf("a file mount must follow the read-only directory it lies in: %+v", root.Mounts)
	}
	// No other Manager needs the file mounted, and every directory mount stays a
	// directory.
	for _, manager := range []string{"library", "lending/reminders"} {
		plan, err := candidates.WritePlan(manager)
		if err != nil {
			t.Fatal(err)
		}
		for _, mount := range plan.Mounts {
			if mount.Path == "realization/lending/Makefile" || mount.File {
				t.Errorf("%s mounts a file it neither reads nor writes specially: %+v", manager, mount)
			}
		}
	}
	// The stand is accounted: the exact file is registered by its owner Root.
	if findings := candidates.Account(attempt.Structural().Model().Inputs().Raw.Census); len(findings) != 0 {
		t.Fatalf("the start with a foreign exact file is not accounted: %v", summary(findings))
	}
}

func TestCandidatesRefuseAnExactFileAreaThatIsMissingInTheStartStand(t *testing.T) {
	// The exact-file area names a file the start stand does not hold. It cannot
	// be mounted, so the Managers that would need the mount are refused instead
	// of left to trust; a directory area is created, a file is not.
	selection, raw := executionInputs(t, withOwnership(t, foreignFileOwnership))
	libraryLayoutFiles(t, &selection, &raw)
	candidates, err := apply.NewCandidates(preparedAttempt(t, selection, raw))
	if err != nil {
		t.Fatal(err)
	}
	for _, manager := range []string{"lending", "."} {
		_, err := candidates.WritePlan(manager)
		if !errors.Is(err, apply.ErrUnsupportedOwnership) || !strings.Contains(err.Error(), "realization/lending/Makefile") || !strings.Contains(err.Error(), "does not exist in the start stand") {
			t.Errorf("%s: a missing exact file was not refused: %v", manager, err)
		}
	}
	// A Manager that never needs the mount is not affected.
	if _, err := candidates.WritePlan("library"); err != nil {
		t.Errorf("library does not touch the missing file: %v", err)
	}
}

func TestCandidatesAccountDirectoryRegistrationsAndShowThemInTheRegistryView(t *testing.T) {
	candidates, start := layoutCandidates(t)
	// The reminders Manager registers its whole directory with one entry.
	registered := change(t, start, map[string]*string{
		remindersShard: text(shardYAML(shardEntry("realization/lending/reminders/", "code", "lending/reminders/assess-overdue"))),
	})
	if findings := candidates.Account(registered); len(findings) != 0 {
		t.Fatalf("a directory registration was refused: %v", summary(findings))
	}
	// A file added below the registered directory is covered without a new entry.
	grown := change(t, registered, map[string]*string{"realization/lending/reminders/schedule.go": text("package reminders\n")})
	if findings := candidates.Account(grown); len(findings) != 0 {
		t.Fatalf("a new file below a registered directory is unexplained: %v", summary(findings))
	}
	view := candidates.RegistryView(grown)
	var entry []string
	for _, registration := range view.Registrations {
		if registration.Entry == "realization/lending/reminders/" {
			entry = append(entry, registration.Path+" "+registration.Owner)
		}
	}
	slices.Sort(entry)
	if want := []string{"realization/lending/reminders/remind.go lending/reminders", "realization/lending/reminders/remind_test.go lending/reminders", "realization/lending/reminders/schedule.go lending/reminders"}; !slices.Equal(entry, want) {
		t.Errorf("registry view of a directory entry: %q, want %q", entry, want)
	}
	if view.Shards[remindersShard] == "" || view.Shards[rootShard] == "" {
		t.Errorf("the view lacks the shard versions: %v", view.Shards)
	}
	// A Package cannot register a directory that belongs to another Manager.
	foreign := change(t, start, map[string]*string{
		lendingShard: text(shardYAML(shardEntry("realization/lending/borrow.go", "code", "lending/borrow"), shardEntry("realization/lending/reminders/", "code", "lending/borrow"))),
	})
	if got := summary(candidates.Account(foreign)); !slices.Contains(got, "lending registry.outside_area realization/lending/reminders") {
		t.Errorf("a foreign directory registration was accepted: %v", got)
	}
	// An entry for a directory without an eligible file of its registrant is dead.
	dead := change(t, start, map[string]*string{
		libraryShard: text(shardYAML(shardEntry("realization/library/library.go", "code", "library/library"), shardEntry("realization/cmd/library/main.go", "code", "library/callable-interface"), shardEntry("realization/library/empty/", "code", "library/library"))),
	})
	if got := summary(candidates.Account(dead)); !slices.Contains(got, "library registry.missing_file realization/library/empty") {
		t.Errorf("a registration covering no file was accepted: %v", got)
	}
}
