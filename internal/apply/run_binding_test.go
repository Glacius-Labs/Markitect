package apply_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func runBindingHost(t *testing.T) apply.RunBindingFacts {
	t.Helper()
	owner, err := run.NewPayload("text/plain", "utf-8", []byte("\xef\xbb\xbfApply this captured intent.\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	return apply.RunBindingFacts{
		RepositoryIdentity: "explicit-host-repository-identity", ProductBuild: "explicit-product-build",
		OwnerRequest: owner,
		Allocation:   run.Capacity{Runs: 2, SessionStarts: 70, Steps: 140, ExecutionTime: 7 * time.Hour},
	}
}

func runBindingCapture(t *testing.T, change func(*apply.RawInputs), complete bool) (apply.CapturedSettingsResult, *int, *int) {
	t.Helper()
	selection, actual, host := settingsServiceInputs(t)
	if change != nil {
		change(&actual)
	}
	if complete {
		selection = completeSettingsServiceSelection(t, selection, host)
	}
	captures, metadata := 0, 0
	source := controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		captures++
		input := captured(preparationObservation(t, request.Selection, actual))
		input.Guard = apply.PublicationGuard(fmt.Sprintf("original-guard-%d", captures))
		return input, nil
	}}
	agent := settingsServiceAgent(t, host)
	_, check := hostPorts(t)
	service := apply.NewPreparationService(apply.NewModelService(source, nil))
	result, err := service.CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{YAMLDepth: 96}, hostAgentPort(func(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		metadata++
		return agent(ctx, input)
	}), check)
	if _, ok := result.Prepared(); err != nil || !ok {
		for _, attempt := range result.Attempts() {
			t.Logf("model=%+v execution=%+v files=%+v", attempt.Structural().Model().Compilation().Diagnostics, attempt.Structural().Execution().Diagnostics(), attempt.Structural().Files().Diagnostics())
		}
		t.Fatalf("actual preparation: status=%s error=%v", result.Status(), err)
	}
	return result, &captures, &metadata
}

type runBindingDocument struct {
	Source               organization.SettingSource
	Path, RepositoryPath string
	Mode                 snapshot.Mode
	Bytes                []byte
}

type runBindingView struct {
	Version         uint32
	Scope           *[]organization.Setting[organization.ScopeRule]
	MaxContextBytes *organization.Setting[int64]
	Source          run.SourceBasis
	OwnerRequest    run.BlobRef
	ModelInput      model.Input
	Policy          run.BudgetPolicy
	PolicySource    organization.SettingSource
	Profiles        []organization.ProfileFact
	Documents       []runBindingDocument
	Ceilings        struct {
		StepTimeout    organization.Setting[int64]
		ReviewRounds   organization.Setting[int]
		CreationRoots  []string
		CreationSource organization.SettingSource
	}
}

func runBindingRead(t *testing.T, binding run.Binding) runBindingView {
	t.Helper()
	var view runBindingView
	if err := json.Unmarshal(binding.Payload().Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	return view
}

func TestCapturedRunBindingRetainsFinalActualPreparationAndOriginalDocuments(t *testing.T) {
	result, captures, metadata := runBindingCapture(t, nil, false)
	a, _ := result.Prepared()
	modelStage := a.Structural().Model()
	inputs := modelStage.Inputs()
	if *captures != 2 || len(result.Attempts()) != 2 || inputs.Guard == result.Attempts()[0].Structural().Model().Inputs().Guard {
		t.Fatal("fixture must retain a distinct earlier incomplete acquisition")
	}
	facts := runBindingHost(t)
	calls, observed := *captures, *metadata
	binding, err := result.BindRun(facts)
	if err != nil || !binding.Valid() {
		t.Fatalf("BindRun: %v", err)
	}
	view := runBindingRead(t, binding)
	if view.Source.Repository != facts.RepositoryIdentity || view.Source.Repository == modelStage.Request().Selection.Repository || view.Source.ProductBuild != facts.ProductBuild || view.Source.Revision != inputs.Head || view.Source.Guard != string(inputs.Guard) || view.Source.Operation != "artifact-apply/v1" || view.OwnerRequest != facts.OwnerRequest.Ref() {
		t.Fatal("explicit Host facts or final captured source binding changed")
	}
	wantModel := model.Input{Root: inputs.Raw.Local.Source, Limits: modelStage.RequestedLimits()}
	for _, dependency := range inputs.Raw.Dependencies {
		wantModel.Dependencies = append(wantModel.Dependencies, dependency.Source)
	}
	if !reflect.DeepEqual(view.ModelInput, wantModel) {
		t.Fatal("full original model closure, lock entries or compilation limits changed")
	}
	execution, _ := a.Structural().Execution().Document()
	if view.Version != 3 || view.MaxContextBytes == nil || *view.MaxContextBytes != execution.Limits.MaxContextBytes {
		t.Fatal("selected context limit/provenance not bound")
	}
	if execution.Scope == nil {
		if view.Scope != nil {
			t.Fatal("absent authored scope invented")
		}
	} else if view.Scope == nil || len(*view.Scope) != len(execution.Scope.Value) || len(*view.Scope) > 0 && !reflect.DeepEqual(*view.Scope, execution.Scope.Value) {
		t.Fatal("authored scope/provenance not bound")
	}
	wantPolicy := run.BudgetPolicy{
		Ceilings:       run.Capacity{Runs: execution.Limits.MaxRuns.Value, SessionStarts: execution.Limits.MaxSessionStarts.Value, Steps: execution.Limits.MaxSteps.Value, ExecutionTime: execution.Limits.MaxExecutionTime.Value},
		Assignment:     facts.Allocation,
		Reserve:        run.Capacity{SessionStarts: execution.CompletionReserve.SessionStarts.Value, Steps: execution.CompletionReserve.Steps.Value, ExecutionTime: execution.CompletionReserve.ExecutionTime.Value},
		MaxStepTimeout: execution.Limits.MaxStepTimeout.Value, HostOperationTimeout: execution.Limits.HostOperationTimeout.Value,
	}
	if view.Policy != wantPolicy || view.Policy.Assignment.Runs == execution.Assignment.Runs.Value || view.PolicySource.Field != "" || view.PolicySource.File != inputs.Raw.Execution[0].RepositoryPath.String() {
		t.Fatal("caps/reserve/timeouts or explicit allocation/provenance changed")
	}
	settings, _ := a.Settings().Settings()
	ceilings := settings.Ceilings()
	if view.Ceilings.StepTimeout.Value != int64(ceilings.StepTimeout.Value) || view.Ceilings.StepTimeout.Source != ceilings.StepTimeout.Source || view.Ceilings.ReviewRounds != ceilings.RejectionLimit || view.Ceilings.CreationSource != ceilings.CreationRoots.Source {
		t.Fatal("actual resolved ceilings/provenance changed")
	}
	var originals []apply.SourceFile
	for _, group := range [][]apply.SourceFile{inputs.Raw.Execution, inputs.Raw.Registry, inputs.Raw.Ignore, inputs.Raw.HostProfiles, inputs.Raw.Instructions} {
		originals = append(originals, group...)
	}
	if len(view.Documents) != len(originals)+1 {
		t.Fatal("an original registry/ignore/Host/instruction/product document was omitted")
	}
	for i, file := range originals {
		doc := view.Documents[i]
		if doc.Path != file.File.Path().String() || doc.RepositoryPath != file.RepositoryPath.String() || doc.Mode != file.File.Mode() || !bytes.Equal(doc.Bytes, file.File.Bytes()) || doc.Source.Field != "" || doc.Source.ContentID != file.File.ContentID() {
			t.Fatalf("original acquired document %d changed", i)
		}
	}
	product := view.Documents[len(originals)]
	if product.Source != a.Defaults().ProductSource() || product.RepositoryPath != "" || !bytes.Equal(product.Bytes, a.Defaults().ProductInput().Bytes) {
		t.Fatal("logical embedded product document lost its original bytes/source")
	}
	for _, profile := range a.Settings().ProfileObservations() {
		identity, _ := profile.Identity()
		found := false
		for _, fact := range view.Profiles {
			found = found || fact.Name == profile.Input().Name() && fact.Identity == identity && fact.Source == profile.Source()
		}
		if !found {
			t.Fatal("original Agent/Check static profile identity or field source omitted")
		}
	}
	again, err := result.BindRun(facts)
	if err != nil || again.Ref() != binding.Ref() || *captures != calls || *metadata != observed {
		t.Fatal("assembly is unstable or called acquisition/metadata again")
	}
}

func TestCapturedRunBindingChangesWithExplicitHostFactsAndOriginalInputs(t *testing.T) {
	result, _, _ := runBindingCapture(t, nil, true)
	facts := runBindingHost(t)
	baseline, err := result.BindRun(facts)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"repository", "build", "owner", "allocation"} {
		t.Run(field, func(t *testing.T) {
			changed := facts
			switch field {
			case "repository":
				changed.RepositoryIdentity += "-other"
			case "build":
				changed.ProductBuild += "-other"
			case "owner":
				changed.OwnerRequest, _ = run.NewPayload("text/plain", "utf-8", []byte("Changed original owner request."))
			case "allocation":
				changed.Allocation.Steps--
			}
			binding, err := result.BindRun(changed)
			if err != nil || binding.Ref() == baseline.Ref() {
				t.Fatalf("changed %s did not change basis: %v", field, err)
			}
		})
	}
	for _, original := range []string{"execution", "host-document", "instruction"} {
		t.Run(original, func(t *testing.T) {
			changed, _, _ := runBindingCapture(t, func(raw *apply.RawInputs) {
				switch original {
				case "execution":
					rewriteFileCheckSource(t, raw, &raw.Execution[0], string(raw.Execution[0].File.Bytes())+"\n# Original configuration changed.\n")
				case "host-document":
					raw.HostProfiles[0].File, err = snapshot.NewEntry(raw.HostProfiles[0].File.Path(), raw.HostProfiles[0].File.Mode(), append(raw.HostProfiles[0].File.Bytes(), []byte("\n# Original Host configuration changed.\n")...))
				case "instruction":
					raw.Instructions[0].File, err = snapshot.NewEntry(raw.Instructions[0].File.Path(), raw.Instructions[0].File.Mode(), []byte("Changed original external instruction.\r\n"))
				}
				if err != nil {
					t.Fatal(err)
				}
			}, true)
			binding, err := changed.BindRun(facts)
			if err != nil || binding.Ref() == baseline.Ref() {
				t.Fatalf("changed %s bytes did not change basis: %v", original, err)
			}
		})
	}
}

func TestCapturedRunBindingMapsAuthoredScopeAndContextLimit(t *testing.T) {
	for _, scope := range []string{"", "scope: []\n", "scope: [{kind: lending/RunScopedRule, property: lending/runApplies}]\n"} {
		t.Run(fmt.Sprint(len(scope)), func(t *testing.T) {
			result, _, _ := runBindingCapture(t, func(raw *apply.RawInputs) {
				for _, d := range []struct{ name, text string }{
					{"lending/run-applies", "imports: [{package: markitect/core, as: Core}]\nid: runApplies\nkind: Core.Property\nbody: Explicit run scope.\nspec: {type: string, minCount: 0, maxCount: -1}\n"},
					{"lending/run-scoped-rule", "imports: [{package: markitect/core, as: Core}]\nid: RunScopedRule\nkind: Core.Kind\nbody: Scoped run obligation.\nspec: {properties: [runApplies]}\n"},
					{"lending/reminders/run-global", "imports: [{package: lending}]\nid: run-global\nkind: lending/RunScopedRule\nbody: Global initial run responsibility.\nspec: {runApplies: ['.']}\n"},
				} {
					e := model.Entry{Path: "model/" + d.name + ".yaml", Kind: model.Regular, Bytes: []byte(d.text)}
					raw.Local.Source.Entries = append(raw.Local.Source.Entries, e)
					file, err := snapshot.NewEntry(path(t, e.Path), snapshot.Regular, e.Bytes)
					if err != nil {
						t.Fatal(err)
					}
					raw.Local.Files, err = snapshot.NewTree(append(raw.Local.Files.Entries(), file))
					if err != nil {
						t.Fatal(err)
					}
					repoPath := e.Path
					if !raw.Local.RepositoryArea.IsRoot() {
						repoPath = raw.Local.RepositoryArea.String() + "/" + repoPath
					}
					censusFile(t, raw, repoPath, snapshot.Regular, e.Bytes)
				}
				rewriteFileCheckSource(t, raw, &raw.Execution[0], string(raw.Execution[0].File.Bytes())+"\n"+scope)
			}, true)
			basis, err := result.BindRun(runBindingHost(t))
			if err != nil {
				t.Fatal(err)
			}
			before := basis.Payload().Bytes()
			view := runBindingRead(t, basis)
			prepared, _ := result.Prepared()
			execution, _ := prepared.Structural().Execution().Document()
			if view.MaxContextBytes == nil || *view.MaxContextBytes != execution.Limits.MaxContextBytes {
				t.Fatal("context limit source lost")
			}
			if scope == "" {
				if view.Scope != nil {
					t.Fatal("scope invented")
				}
			} else if view.Scope == nil || len(*view.Scope) != len(execution.Scope.Value) || len(*view.Scope) > 0 && !reflect.DeepEqual(*view.Scope, execution.Scope.Value) {
				t.Fatal("typed authored scope or source lost")
			}
			if view.Scope != nil && len(*view.Scope) > 0 {
				(*view.Scope)[0].Source.File = "changed"
			}
			if !bytes.Equal(before, basis.Payload().Bytes()) {
				t.Fatal("scope aliases bound bytes")
			}
		})
	}
}
