package apply

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// RunBindingFacts are explicit Host assertions, not facts inferred from source
// selection or authored assignment defaults. RepositoryIdentity is an opaque
// repository key; supplying it establishes no durable StoreID, live Control,
// execution authority or physical identity verification. ProductBuild identifies
// the actual product build according to the Host's explicit assertion.
type RunBindingFacts struct {
	RepositoryIdentity string
	ProductBuild       string
	OwnerRequest       run.Payload
	// OwnerDecisions are the owner's earlier decisions, bound for every task
	// of the run (DEC-053).
	OwnerDecisions []string
	Allocation     run.Capacity
}

// BindRun assembles a fixed basis from only the final successful committed
// preparation. It acquires nothing, dispatches nothing and creates no Run or
// journal entry. Preview results deliberately have no corresponding method.
// run.Bind owns binding/accounting/budget validation; producer stages retain
// compilation, configuration resolution and acquisition responsibility.
func (r CapturedSettingsResult) BindRun(facts RunBindingFacts) (run.Binding, error) {
	input, err := r.runInput(facts)
	if err != nil {
		return run.Binding{}, err
	}
	return run.Bind(input)
}

// stepPreparation returns the producer inputs of the bound run, which each
// step preparation compiles and resolves again.
func (r CapturedSettingsResult) stepPreparation(facts RunBindingFacts) (run.StepPreparation, error) {
	input, err := r.runInput(facts)
	if err != nil {
		return run.StepPreparation{}, err
	}
	a, _ := r.Prepared()
	settings, ok := a.Settings().SettingsFacts()
	if !ok {
		return run.StepPreparation{}, fmt.Errorf("%w: resolved settings facts required", run.ErrInvalid)
	}
	return run.StepPreparation{Binding: input, Settings: settings}, nil
}

func (r CapturedSettingsResult) runInput(facts RunBindingFacts) (run.BindingInput, error) {
	a, ok := r.Prepared()
	if !ok || a.Error() != nil {
		return run.BindingInput{}, fmt.Errorf("%w: complete captured settings required", run.ErrInvalid)
	}
	s := a.Structural()
	m := s.Model()
	inputs := m.Inputs()
	registry, registryOK := s.Registry().Registry()
	classes, classesOK := s.Files().Classifications()
	protections, protectionsOK := s.Files().Protections()
	settings, settingsOK := a.Settings().Settings()
	execution, executionOK := s.Execution().Document()
	if !registryOK || !classesOK || !protectionsOK || !settingsOK || !executionOK {
		return run.BindingInput{}, fmt.Errorf("%w: incomplete captured binding inputs", run.ErrInvalid)
	}
	input := run.BindingInput{
		MaxContextBytes: execution.Limits.MaxContextBytes,
		Source: run.SourceBasis{
			Repository: facts.RepositoryIdentity, Revision: inputs.Head,
			Guard: string(inputs.Guard), ProductBuild: facts.ProductBuild,
			Operation: "artifact-apply/v1",
		},
		OwnerRequest:   facts.OwnerRequest,
		OwnerDecisions: facts.OwnerDecisions,
		ModelInput: model.Input{
			Root: inputs.Raw.Local.Source, Limits: m.RequestedLimits(),
		},
		Compilation: m.Compilation(), Census: inputs.Raw.Census,
		Registry: registry, Classifications: classes, Protections: protections,
		Settings: settings,
		Policy: run.BudgetPolicy{
			Ceilings: run.Capacity{
				Runs: execution.Limits.MaxRuns.Value, SessionStarts: execution.Limits.MaxSessionStarts.Value,
				Steps: execution.Limits.MaxSteps.Value, ExecutionTime: execution.Limits.MaxExecutionTime.Value,
			},
			Assignment: facts.Allocation,
			Reserve: run.Capacity{
				SessionStarts: execution.CompletionReserve.SessionStarts.Value,
				Steps:         execution.CompletionReserve.Steps.Value, ExecutionTime: execution.CompletionReserve.ExecutionTime.Value,
			},
			MaxStepTimeout:       execution.Limits.MaxStepTimeout.Value,
			HostOperationTimeout: execution.Limits.HostOperationTimeout.Value,
		},
		PolicySource: execution.APIVersion.Source,
	}
	if execution.Checks != nil {
		for _, check := range execution.Checks.Value {
			if check.RequiredBy != nil {
				input.RequiredChecks = append(input.RequiredChecks, check.ID.Value)
			}
		}
	}
	if execution.Scope != nil {
		input.Scope = append([]organization.Setting[organization.ScopeRule]{}, execution.Scope.Value...) // Preserve explicit empty versus absent.
	}
	input.PolicySource.Field = "" // The actual whole execution document, not an invented budget field.
	for _, dependency := range inputs.Raw.Dependencies {
		input.ModelInput.Dependencies = append(input.ModelInput.Dependencies, dependency.Source)
	}
	for _, profile := range a.Settings().ProfileObservations() {
		identity, complete := profile.Identity()
		if !complete {
			return run.BindingInput{}, fmt.Errorf("%w: incomplete static profile identity", run.ErrInvalid)
		}
		// Keep both Agent and Check metadata identities. Their original catalog
		// field sources distinguish namespaces even when names coincide.
		input.Profiles = append(input.Profiles, organization.ProfileFact{
			Name: profile.Input().Name(), Identity: identity, Source: profile.Source(),
		})
	}
	for _, group := range []struct {
		files []SourceFile
		host  bool
	}{
		{inputs.Raw.Execution, false}, {inputs.Raw.Registry, false}, {inputs.Raw.Ignore, false},
		{inputs.Raw.HostProfiles, true}, {inputs.Raw.Instructions, true},
	} {
		for _, file := range group.files {
			source := organization.SettingSource{File: file.RepositoryPath.String(), ContentID: file.File.ContentID()}
			if group.host {
				source = hostSource(file, "")
			}
			input.Documents = append(input.Documents, run.Document{Source: source, File: file.File, RepositoryPath: file.RepositoryPath})
		}
	}
	// This is a logical entry for actual embedded bytes, not an observed disk
	// file. Keep the product's original opaque source separately from its name.
	productPath, err := snapshot.NewPath("strategy-defaults.yaml")
	if err != nil {
		return run.BindingInput{}, err
	}
	product, err := snapshot.NewEntry(productPath, snapshot.Regular, a.Defaults().ProductInput().Bytes)
	if err != nil {
		return run.BindingInput{}, err
	}
	input.Documents = append(input.Documents, run.Document{Source: a.Defaults().ProductSource(), File: product})
	return input, nil
}
