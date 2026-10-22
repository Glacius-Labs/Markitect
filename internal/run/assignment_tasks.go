package run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// The Codex channel checks the complete JSON frame, where characters such as
// '&' can expand sixfold. A 128 KiB raw part stays below its 1 MiB frame limit
// even with that expansion and the fixed review prompt around it. All parts go
// to the same Manager conversation before a decision is offered.
const reviewChangeChunkBytes = 128 << 10
const reviewSummaryBytes = 8 << 10

func reviewTextParts(value, label string) ([]string, error) {
	if !utf8.ValidString(value) {
		return nil, fmt.Errorf("%s contains invalid UTF-8 and cannot be delivered for parent review", label)
	}
	if len(value) <= reviewChangeChunkBytes {
		return []string{value}, nil
	}
	parts := make([]string, 0, 1+len(value)/reviewChangeChunkBytes)
	for len(value) > reviewChangeChunkBytes {
		cut := reviewChangeChunkBytes
		if line := strings.LastIndexByte(value[:cut], '\n'); line >= cut-(16<<10) {
			cut = line + 1
		} else {
			for cut > 0 && !utf8.RuneStart(value[cut]) {
				cut--
			}
		}
		parts = append(parts, value[:cut])
		value = value[cut:]
	}
	return append(parts, value), nil
}

func (s *assignmentScheduler) taskPrompt(path, task string) string {
	manager := s.team[path]
	prompt := fmt.Sprintf("Your responsibility is %s. %s\nTask from your parent: %s\nWork in your own Git copy. Complete your own responsibility and summarize changes, actual tests and open matters; your current Git stand is the handoff.", manager.Role, manager.Brief, task)
	if len(manager.Children) > 0 {
		prompt += fmt.Sprintf("\nEstablish the shared sibling interfaces in your own workspace before calling delegate_child. This is part of this Implement task, with no separate plan run or stage. Each required direct child is a separate Markitect Manager with its own container and working copy. Only delegate_child delegates through the Markitect Host; do not substitute a Codex or other provider-native subagent. Give each direct child the finalized interfaces and bounded responsibility it needs. The Host records your delegate_child calls during this response; after every required delegation, finish this response with Go. Only then does the Host commit your stand and deliver all child tasks concurrently. Do not wait for or consult a child during this Go response; ask_child and corrections can follow after its first task starts.\nRequired direct children: %s. Give each child a concrete task using delegate_child. The Host will later present each fixed child result for your review.", strings.Join(manager.Children, ", "))
	}
	return prompt
}

func (s *assignmentScheduler) work(ctx context.Context, path, task, stand string) (AssignmentHandoff, error) {
	prompt := s.taskPrompt(path, task)
	if stand != "" {
		if err := s.flow.serial(ctx, path, func(ctx context.Context) error {
			updateErr := s.flow.deliver(ctx, s.team[path].Parent, path, stand)
			if updateErr != nil && !errors.Is(updateErr, ErrAssignmentUpdateConflict) {
				return updateErr
			}
			prompt += "\nParent Git stand delivered for this task: " + stand
			if updateErr != nil {
				prompt += "\nResolve the normal Git merge conflicts in your own copy before completing this task: " + updateErr.Error()
			}
			return nil
		}); err != nil {
			return AssignmentHandoff{}, err
		}
	}
	var actions []assignmentAction
	var result AssignmentHandoff
	_, err := s.interact(ctx, path, prompt, func(ctx context.Context, _ TurnResult, decisions []assignmentAction) error {
		seen := make(map[string]bool)
		for _, action := range decisions {
			if action.name != DelegateChildTool || s.team[action.child].Parent != path || seen[action.child] {
				return invalidDecision("delegate each required direct child exactly once with a concrete task; invalid delegation %q", action.child)
			}
			seen[action.child] = true
		}
		var missing []string
		for _, child := range s.team[path].Children {
			if !seen[child] {
				missing = append(missing, child)
			}
		}
		if len(missing) > 0 {
			return invalidDecision("missing required child tasks: %s. Use delegate_child for every direct child in this response", strings.Join(missing, ", "))
		}
		actions = decisions
		if len(s.team[path].Children) == 0 {
			var err error
			result, err = s.flow.Submit(ctx, path)
			return err
		}
		return nil
	})
	if err != nil || len(s.team[path].Children) == 0 {
		return result, err
	}
	if err := s.children(ctx, path, actions); err != nil {
		return AssignmentHandoff{}, err
	}
	return s.correctTask(ctx, path, "All direct-child results are integrated. Check the composition and relevant tests in your Git copy. If completed sibling work requires a later task, use return_child now for that owning child; its same copy will first receive your integrated stand. Otherwise summarize your result, observations and open matters for your parent.")
}

func (s *assignmentScheduler) review(ctx context.Context, parent, child string) error {
	for {
		correction, err := s.reviewResult(ctx, parent, child)
		if err != nil || correction == "" {
			return err
		}
		if err := s.correct(ctx, parent, child, correction); err != nil {
			return err
		}
	}
}

func (s *assignmentScheduler) reviewResult(ctx context.Context, parent, child string) (correction string, err error) {
	changes, handoff, err := s.flow.Changes(ctx, parent, child)
	if err != nil {
		return "", err
	}
	parts, err := reviewTextParts(changes, "child Git difference")
	if err != nil {
		return "", err
	}
	summary := handoff.Summary
	if len(summary) > reviewSummaryBytes {
		summaryParts, err := reviewTextParts(summary, "child summary")
		if err != nil {
			return "", err
		}
		for i, part := range summaryParts {
			prompt := fmt.Sprintf("Fixed child summary from %s, result %s, part %d of %d:\n%s\nRead this part. More review material follows in this same conversation. Do not decide yet; accept_child and return_child are available only after the final Git-difference part.", child, handoff.ResultCommit, i+1, len(summaryParts), part)
			if _, err := s.interact(ctx, parent, prompt, nil); err != nil {
				return "", err
			}
		}
		summary = "[Full child summary delivered in preceding messages.]"
	}
	for i, part := range parts[:len(parts)-1] {
		prompt := fmt.Sprintf("Fixed Git difference from direct child %s, result %s from %s, part %d of %d:\n%s\nRead this part. More parts follow in this same conversation. Do not decide yet; accept_child and return_child are available only after the final part.", child, handoff.ResultCommit, handoff.StartingCommit, i+1, len(parts), part)
		if _, err := s.interact(ctx, parent, prompt, nil); err != nil {
			return "", err
		}
	}
	changeLabel := "Actual Git changes"
	if len(parts) > 1 {
		changeLabel = fmt.Sprintf("Actual Git changes, final part %d of %d (earlier parts are in this conversation)", len(parts), len(parts))
	}
	prompt := fmt.Sprintf("Review your direct child %s's fixed result %s from %s. Child summary: %s\nObserved commands delivered with this result:\n%s\n%s:\n%s\nInspect conformance and tests. Binary changes are named by Git without embedding their opaque contents. Use accept_child with only child to accept this presented result, or return_child with a concrete correction. The Host binds the exact reviewed handoff; do not supply a commit. Your decision must be yours.", child, handoff.ResultCommit, handoff.StartingCommit, summary, s.commandObservations(child, handoff.ResultCommit), changeLabel, parts[len(parts)-1])
	var conflict error
	_, err = s.interact(ctx, parent, prompt, func(ctx context.Context, _ TurnResult, actions []assignmentAction) error {
		if len(actions) != 1 || actions[0].child != child || (actions[0].name != AcceptChildTool && actions[0].name != ReturnChildTool) {
			return invalidDecision("decide exactly once on the currently reviewed child %q: accept_child with only child, or return_child with child and concrete correction", child)
		}
		if actions[0].name == ReturnChildTool {
			correction = actions[0].correction
			return nil
		}
		_, integrateErr := s.flow.Accept(ctx, parent, child, handoff.ResultCommit)
		if errors.Is(integrateErr, ErrIntegrationConflict) {
			conflict = integrateErr
			return nil
		}
		return integrateErr
	})
	if err != nil || conflict == nil {
		return correction, err
	}
	_, err = s.interact(ctx, parent, fmt.Sprintf("Git found a real conflict integrating %s at %s: %v. Give the owning child a concrete correction using return_child.", child, handoff.ResultCommit, conflict), func(_ context.Context, _ TurnResult, actions []assignmentAction) error {
		if len(actions) != 1 || actions[0].name != ReturnChildTool || actions[0].child != child {
			return invalidDecision("route this Git conflict to %q using return_child with a concrete correction", child)
		}
		correction = actions[0].correction
		return nil
	})
	return correction, err
}

func (s *assignmentScheduler) correct(ctx context.Context, parent, child, correction string) error {
	var stand string
	if err := s.flow.serial(ctx, parent, func(ctx context.Context) error {
		var err error
		stand, err = s.flow.Workspace(parent).Commit(ctx, "Parent integrated stand for later child task")
		return err
	}); err != nil {
		return err
	}
	if err := s.flow.serial(ctx, child, func(ctx context.Context) error {
		updateErr := s.flow.deliver(ctx, parent, child, stand)
		if updateErr != nil && !errors.Is(updateErr, ErrAssignmentUpdateConflict) {
			return updateErr
		}
		correction += "\nParent Git stand delivered for this task: " + stand
		if updateErr != nil {
			correction += "\nResolve the normal Git merge conflicts in your own copy before completing this task: " + updateErr.Error()
		}
		return nil
	}); err != nil {
		return err
	}
	_, err := s.correctTask(ctx, child, correction)
	return err
}

// A completed native response needs no submission tool. Concrete child returns
// are ordinary follow-up tasks; otherwise the Host fixes this Manager's result.
func (s *assignmentScheduler) correctTask(ctx context.Context, path, prompt string) (AssignmentHandoff, error) {
	for {
		var actions []assignmentAction
		var result AssignmentHandoff
		_, err := s.interact(ctx, path, prompt, func(ctx context.Context, _ TurnResult, decisions []assignmentAction) error {
			seen := make(map[string]bool)
			for _, action := range decisions {
				if action.name != ReturnChildTool || s.team[action.child].Parent != path || seen[action.child] {
					return invalidDecision("use return_child once for each direct owner needing a concrete correction, or summarize your own completed result")
				}
				s.flow.mu.Lock()
				submitted := s.flow.team[action.child].handoff.ResultCommit != ""
				s.flow.mu.Unlock()
				if !submitted {
					return invalidDecision("child %q has no completed handoff to return; complete its initial task first", action.child)
				}
				seen[action.child] = true
			}
			actions = decisions
			if len(actions) == 0 {
				var err error
				result, err = s.flow.Submit(ctx, path)
				return err
			}
			return nil
		})
		if err != nil || len(actions) == 0 {
			return result, err
		}
		for _, action := range actions {
			if err := s.correct(ctx, path, action.child, action.correction); err != nil {
				return AssignmentHandoff{}, err
			}
			if err := s.review(ctx, path, action.child); err != nil {
				return AssignmentHandoff{}, err
			}
		}
		prompt = "Corrected child results are integrated. Check the changed composition and relevant tests; preserve unaffected work. Summarize the corrected Git result and any open matters. If a concrete required correction remains with a child, use return_child for that owner."
	}
}
