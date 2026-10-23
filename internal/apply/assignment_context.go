package apply

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

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
	var b assignmentPromptProjection
	fmt.Fprintf(&b, "\nIntegrated Root result %s, summary: %s\n", result.Root.ResultCommit, assignmentExcerpt(result.Root.Summary, 2048))
	for _, check := range result.Checks {
		if check.Commit == result.Root.ResultCommit {
			fmt.Fprintf(&b, "\nRequired check %s on this commit: started=%t exited=%t exit=%d uncertain=%t\n%s\n", check.Name, check.Receipt.Started, check.Receipt.Exited, check.Receipt.ExitCode, check.Receipt.Uncertain, assignmentExcerpt(string(check.Receipt.Output), 4096))
		}
	}
	for _, prior := range result.Reviews {
		fmt.Fprintf(&b, "\nPrevious independent review: %s\n", assignmentExcerpt(prior.Handoff.Summary, 2048))
		for _, issue := range prior.Issues {
			fmt.Fprintf(&b, "Prior required matter for %s, %s: expected %s; observed %s\n", assignmentExcerpt(issue.Owner, 512), assignmentExcerpt(issue.Obligation, 1024), assignmentExcerpt(issue.Expected, 2048), assignmentExcerpt(issue.Observed, 4096))
		}
	}
	results := result.Work.Flow.Results()
	for _, r := range results {
		fmt.Fprintf(&b, "\nManager %s handed off %s from %s: %s\n", r.Manager, r.Handoff.ResultCommit, r.Handoff.StartingCommit, assignmentExcerpt(r.Handoff.Summary, 2048))
	}
	fmt.Fprint(&b, "Native observations during the handed-off work (commands can precede later edits; judge their relevance to the named result):\n")
	for _, r := range results {
		for _, o := range r.Observations {
			for _, command := range o.Commands {
				exit := "unreported"
				if command.ExitCode != nil {
					exit = fmt.Sprint(*command.ExitCode)
				}
				fmt.Fprintf(&b, "%s at handoff %s: command %s; status %s; exit %s; output %s\n", r.Manager, r.Handoff.ResultCommit, assignmentExcerpt(command.Command, 2048), assignmentExcerpt(command.Status, 512), exit, assignmentExcerpt(command.Output, 4096))
			}
		}
	}
	return modelContext + b.text.String() + "\nJudge relevant follow-up from the actual changes. Earlier passing tests and an unchanged summary cannot approve a different commit. Prompt excerpts do not replace full retained observations; inspect the named artifacts and run focused checks where needed.\n"
}

const assignmentTruncation = "\n[truncated; full observation retained by Host]\n"

func assignmentExcerpt(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := limit - len(assignmentTruncation)
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + assignmentTruncation
}

// Only the prompt projection is bounded; native result/check/review values are
// kept in full. The fixed model context is supplied separately without clipping.
type assignmentPromptProjection struct {
	text    strings.Builder
	clipped bool
}

func (p *assignmentPromptProjection) Write(data []byte) (int, error) {
	n := len(data)
	if p.clipped {
		return n, nil
	}
	room := (32 << 10) - len(assignmentTruncation) - p.text.Len()
	if n > room {
		for room > 0 && !utf8.RuneStart(data[room]) {
			room--
		}
		p.text.Write(data[:room])
		p.text.WriteString(assignmentTruncation)
		p.clipped = true
	} else {
		p.text.Write(data)
	}
	return n, nil
}
