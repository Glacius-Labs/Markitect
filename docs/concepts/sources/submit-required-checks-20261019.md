# A submission must pass its required checks (19 October 2026)

Source of [DEC-055](../register.md#dec-055-a-submission-must-pass-its-required-checks).

## Evidence

Bound library run R2-A1-12 (Main c0eb797c, gpt-6-luna) ended Incomplete after 31 minutes and 40 steps, because `members` reached the rejection limit with three counted rejections. All three were mechanical:

- The first two candidates could not run the required `project-tests` check: it failed at setup because the shared Go module did not exist yet. Before the second submission, `members` asked the root about the module and then submitted "as-is", knowing that the check would fail.
- The third candidate did not compile: `realization/members.go` declared `Member` a second time, in the same Go package as the accepted `realization/library.go`.

The gaps rose (6, 7, 8), so [DEC-054](../register.md#dec-054-a-rejection-that-finds-fewer-gaps-does-not-count) did not apply. Core rule T8 lets a review pass only when the required checks pass, so none of these candidates could have been accepted. Yet each reached review and consumed a counted rejection.

The same run showed that refusing an outcome call early works. The capture check of PR #457 refused `library`'s submit, which had left a `go build` binary behind, and the Manager corrected it within the same step.

## Decision

The Architect decided under the owner's standing authorization ("Ja setze alles um was du für sinnvoll hälst") and the instruction that the Orchestrator, not the prompts, closes process gaps ([DEC-051](../register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process)). When a work or integration step submits, the Host runs the required checks on the observed workspace and refuses the submit with the failing check's output. The Manager then fixes the build in the same step, or waits for the file it needs.
