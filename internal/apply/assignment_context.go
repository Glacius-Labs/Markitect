package apply

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func prepareAssignments(req AssignmentRealizationRequest) ([]run.Manager, string, Candidates, []run.Check, error) {
	candidates, err := NewCandidates(req.Prepared)
	if err != nil {
		return nil, "", candidates, nil, err
	}
	settings, ok := req.Prepared.Settings().Settings()
	if !ok {
		return nil, "", candidates, nil, errors.New("complete resolved assignment settings are required")
	}
	selections, ok := req.Prepared.Structural().Contexts().Selections()
	if !ok {
		return nil, "", candidates, nil, errors.New("complete model context selections are required")
	}
	checked := req.Prepared.Structural().Model().checked
	topology, err := organization.BuildTopology(checked)
	if err != nil {
		return nil, "", candidates, nil, err
	}
	selected, ok := req.Prepared.Settings().Checks()
	if !ok || len(selected) > 0 && (req.Checks == nil || req.CheckRequest == nil) {
		return nil, "", candidates, nil, errors.New("complete required check ports and selections are required")
	}
	var checks []run.Check
	var briefChecks []briefCheck
	for _, check := range selected {
		request, err := req.CheckRequest(check)
		if err != nil {
			return nil, "", candidates, nil, err
		}
		probe := request
		probe.ID = "validate"
		if !probe.Valid() {
			return nil, "", candidates, nil, errors.New("required check request is incomplete")
		}
		declaration := check.Declaration()
		project := declaration.Args.Value
		if len(request.Args) < len(project) || !slices.Equal(request.Args[len(request.Args)-len(project):], project) {
			return nil, "", candidates, nil, errors.New("required check request does not retain the declared arguments")
		}
		required := ""
		if declaration.RequiredBy != nil {
			required = identityText(declaration.RequiredBy.Value)
		}
		checks = append(checks, run.Check{Name: declaration.ID.Value, RequiredBy: required, Owner: ".", Request: request})
		briefChecks = append(briefChecks, briefCheck{Name: declaration.ID.Value, RequiredBy: required, Command: strings.Join(append([]string{request.Executable}, request.Args...), " "), Directory: request.Directory.String()})
	}
	var team []run.Manager
	var modelContext strings.Builder
	modelContext.WriteString("Fixed model, including source provenance and linked Definitions:\n")
	for _, owner := range checked.Models {
		for _, definition := range owner.Definitions {
			writeDefinition(&modelContext, organization.ContextDefinition{Definition: definition}, true, organization.Manager{}, nil)
		}
	}
	for _, manager := range topology.Managers() {
		resolved, ok := settings.Manager(manager.Package)
		if !ok {
			return nil, "", candidates, nil, fmt.Errorf("Manager %s lacks resolved settings", manager.Package)
		}
		selection := selectionOf(selections, manager.Package)
		input := briefInput{Manager: manager, Selection: selection, Settings: resolved, Areas: candidates.Areas(manager.Package), Shard: candidates.Shard(manager.Package), Checks: briefChecks, Children: briefChildren(manager, topology.Managers(), selections, candidates.Areas)}
		team = append(team, run.Manager{Path: manager.Package, Parent: manager.Parent, Children: manager.Children, Role: managerRole(manager, selection), Brief: renderAssignmentBrief(input), Obligations: ownDefinitions(selections, manager.Package), ExecuteProfile: resolved.Execute.Profile.Value.Name, VerifyProfile: resolved.Verify.Profile.Value.Name})
		fmt.Fprintf(&modelContext, "\nManager %s, parent %s, areas %s, registry %s\n", manager.Package, manager.Parent, strings.Join(input.Areas, ", "), input.Shard)
		if manager.Package == organization.Root {
			writeInstructions(&modelContext, "Model assessment and independent Verify", resolved.Verify)
		}
	}
	return team, modelContext.String(), candidates, checks, nil
}

func renderAssignmentBrief(in briefInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Accountable Manager of Package %s. Responsibility: %s\nYour areas: %s\nLocal registry: %s\n%s\n", in.Manager.Package, managerRole(in.Manager, in.Selection), strings.Join(in.Areas, ", "), in.Shard, registryShape("package/id"))
	b.WriteString("Work in your own Git copy. Keep the canonical model, locks, execution, ownership and ignore policy fixed. Responsibility is a working rule: use the owners' realizations through their contracts and route cross-area questions through your parent. Keep file accounting exact. A parent reviews and integrates its children; it cannot waive their responsibilities.\n")
	for _, child := range in.Children {
		fmt.Fprintf(&b, "Required direct child %s owns areas %s; Definitions %s\n", child.Path, strings.Join(child.Areas, ", "), strings.Join(child.Definitions, ", "))
	}
	for _, definition := range in.Selection.Definitions() {
		writeDefinition(&b, definition, definition.Definition.Identity.Package == in.Manager.Package, in.Manager, nil)
	}
	writeProjectChecks(&b, in.Checks)
	writeInstructions(&b, "Implement", in.Settings.Execute)
	return b.String()
}

func assignmentVerifyContext(modelContext string, result AssignmentRealizationResult) string {
	var b strings.Builder
	b.WriteString(modelContext)
	fmt.Fprintf(&b, "\nIntegrated Root result %s, summary: %s\n", result.Root.ResultCommit, result.Root.Summary)
	for _, r := range result.Work.Flow.Results() {
		fmt.Fprintf(&b, "\nManager %s handed off %s from %s: %s\n", r.Manager, r.Handoff.ResultCommit, r.Handoff.StartingCommit, r.Handoff.Summary)
		b.WriteString("Native observations during that work (commands can precede later edits; judge their relevance to the named result):\n")
		for _, o := range r.Observations {
			for _, command := range o.Commands {
				exit := "unreported"
				if command.ExitCode != nil {
					exit = fmt.Sprint(*command.ExitCode)
				}
				fmt.Fprintf(&b, "Command %s; status %s; exit %s; output %s\n", command.Command, command.Status, exit, command.Output)
			}
		}
	}
	for _, check := range result.Checks {
		if check.Commit == result.Root.ResultCommit {
			fmt.Fprintf(&b, "\nRequired check %s on this commit: started=%t exited=%t exit=%d uncertain=%t\n%s\n", check.Name, check.Receipt.Started, check.Receipt.Exited, check.Receipt.ExitCode, check.Receipt.Uncertain, check.Receipt.Output)
		}
	}
	for _, prior := range result.Reviews {
		fmt.Fprintf(&b, "\nPrevious independent review: %s\n", prior.Handoff.Summary)
		for _, issue := range prior.Issues {
			fmt.Fprintf(&b, "Prior required matter for %s, %s: expected %s; observed %s\n", issue.Owner, issue.Obligation, issue.Expected, issue.Observed)
		}
	}
	b.WriteString("Judge relevant follow-up from the actual changes. Earlier passing tests and an unchanged summary cannot approve a different commit.\n")
	return b.String()
}
