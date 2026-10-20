package apply

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func prepareRuntimeSettings(ctx context.Context, fileStatus model.Status, defaults settingsPreparation, checked model.Result, raw RawInputs, execution executionDecoding, sameParent bool, agents AgentProfilePreparer, checks CheckProfilePreparer) (r runtimeSettingsPreparation, err error) {
	r.status = runtimeStatus(fileStatus, defaults.Status())
	defer func() {
		slices.SortFunc(r.diagnostics, func(a, b RuntimeProfileDiagnostic) int {
			ap, bp := model.Position{}, model.Position{}
			if a.Position != nil {
				ap = *a.Position
			}
			if b.Position != nil {
				bp = *b.Position
			}
			return cmp.Or(cmp.Compare(a.Source.File, b.Source.File), cmp.Compare(a.Source.Field, b.Source.Field), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Message, b.Message), cmp.Compare(ap.Line, bp.Line), cmp.Compare(ap.Column, bp.Column))
		})
		slices.SortFunc(r.required, func(a, b SourceLocation) int {
			return cmp.Or(cmp.Compare(a.Boundary, b.Boundary), cmp.Compare(a.Path.String(), b.Path.String()))
		})
		r.required = slices.Compact(r.required)
	}()
	if !sameParent {
		r.issue(model.Failed, "parent_mismatch", organization.SettingSource{}, "FileCheck and Defaults must retain the same complete model, raw inputs, request and capture/preview binding.")
		return r, nil
	}
	if fileStatus != model.Passed {
		r.issue(fileStatus, "file_check_not_passed", organization.SettingSource{}, "The bound FileCheck has not passed; inspect FileCheck diagnostics.")
	}
	if defaults.Status() != model.Passed {
		r.issue(defaults.Status(), "defaults_not_passed", organization.SettingSource{}, "The bound declarative defaults have not passed; inspect Defaults diagnostics.")
	}
	envelope := decodeRuntime(raw)
	r.status = runtimeStatus(r.status, envelope.status)
	r.diagnostics = append(r.diagnostics, envelope.diagnostics...)
	if envelope.status != model.Passed {
		return r, nil
	}
	baseStatus := runtimeStatus(runtimeStatus(fileStatus, defaults.Status()), execution.Status())
	agentStatus, checkStatus := baseStatus, baseStatus
	agentNames := make(map[string]bool)
	checkProfiles := make(map[string]PreparedProfile)
	var profileFacts []organization.ProfileFact
	for _, input := range envelope.profiles {
		profile, portErr := observeRuntimeProfile(ctx, input, agents, checks)
		err = errors.Join(err, portErr)
		r.profiles = append(r.profiles, profile)
		r.diagnostics = append(r.diagnostics, profile.observation.Diagnostics...)
		r.required = append(r.required, profile.observation.RequiredInstructions...)
		r.status = runtimeStatus(r.status, profile.observation.Status)
		if input.namespace == "agentProfiles" {
			agentNames[input.name] = true
			agentStatus = runtimeStatus(agentStatus, profile.observation.Status)
			if identity, ok := profile.Identity(); ok {
				profileFacts = append(profileFacts, organization.ProfileFact{Name: input.name, Identity: identity, Source: profile.Source()})
			}
		} else {
			checkProfiles[input.name] = profile
			checkStatus = runtimeStatus(checkStatus, profile.observation.Status)
		}
	}
	if contextErr := ctx.Err(); contextErr != nil {
		err = errors.Join(err, contextErr)
		agentStatus = runtimeStatus(agentStatus, model.NotCompleted)
		checkStatus = runtimeStatus(checkStatus, model.NotCompleted)
		r.issue(model.NotCompleted, "cancelled", organization.SettingSource{}, contextErr.Error())
	}
	declarations, complete := defaults.Declarations()
	if complete {
		refs := []organization.Setting[string]{declarations.Defaults.Execute.Profile, declarations.Defaults.Verify.Profile}
		for _, h := range declarations.Defaults.Team.Helpers {
			if h.Profile != nil {
				refs = append(refs, *h.Profile)
			}
		}
		for _, override := range declarations.Overrides {
			if override.Execute.Profile != nil {
				refs = append(refs, *override.Execute.Profile)
			}
			if override.Verify.Profile != nil {
				refs = append(refs, *override.Verify.Profile)
			}
		}
		for _, ref := range refs {
			if !agentNames[ref.Value] {
				agentStatus = model.Failed
				r.issue(model.Failed, "agent_profile_missing", ref.Source, fmt.Sprintf("No Agent profile named %q exists in the Agent namespace.", ref.Value))
			}
		}
	}
	if agentStatus == model.Passed && complete {
		facts, instructionErr := runtimeProjectInstructions(raw)
		topology, resolveErr := organization.BuildTopology(checked)
		if resolveErr == nil {
			resolveErr = instructionErr
		}
		var settings organization.Settings
		settingsFacts := organization.SettingsFacts{Defaults: declarations.Defaults, Overrides: declarations.Overrides, Profiles: profileFacts, Instructions: facts}
		if resolveErr == nil {
			settings, resolveErr = organization.ResolveSettings(topology, settingsFacts)
		}
		if resolveErr != nil {
			r.issue(model.Failed, "resolution", organization.SettingSource{}, resolveErr.Error())
		} else {
			budget := execution.authored.Budget
			r.settings, r.facts, r.budget = &settings, &settingsFacts, &budget
		}
	}
	// Check declarations are retained exactly, with no second inheritance,
	// timeout or argument policy. No process request is constructed here.
	var selections []CheckSelection
	if execution.authored.Checks != nil {
		for _, declaration := range execution.authored.Checks.Value {
			profile, found := checkProfiles[declaration.Profile.Value]
			if !found {
				checkStatus = model.Failed
				r.issue(model.Failed, "check_profile_missing", declaration.Profile.Source, fmt.Sprintf("No Check profile named %q exists in the Check namespace.", declaration.Profile.Value))
				continue
			}
			if _, ok := profile.Identity(); ok {
				selections = append(selections, CheckSelection{declaration: declaration, profile: profile})
			}
		}
	}
	if checkStatus == model.Passed {
		r.checks, r.checksComplete = selections, true
	}
	r.status = runtimeStatus(r.status, runtimeStatus(agentStatus, checkStatus))
	return r, err
}

func runtimeProjectInstructions(raw RawInputs) ([]organization.InstructionFact, error) {
	var facts []organization.InstructionFact
	seen := make(map[string]SourceFile)
	for _, file := range raw.Instructions {
		id := file.RepositoryPath.String()
		if id == "" {
			continue
		} // Runtime-only instructions remain adapter-owned.
		if prior, found := seen[id]; found {
			if prior != file {
				return nil, fmt.Errorf("Conflicting acquired instruction observations for %q.", id)
			}
			continue
		}
		seen[id] = file
		facts = append(facts, organization.InstructionFact{ID: id, Source: runtimeSource(file, ""), Bytes: file.File.Bytes()})
	}
	return facts, nil
}
