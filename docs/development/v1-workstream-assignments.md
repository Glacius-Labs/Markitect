# V1 bounded implementation assignments

Assignments for steps 2 and 3 of the [delivery plan](../implementation-plan.md#steps), which bring the source to [V1 concept 1.0](../concepts/v1-concept.md). The completed S1–S5 assignments are preserved at [Main `334e7c31`](https://github.com/Glacius-Labs/Markitect/blob/334e7c3133ff294c0b827e623d80185e3d3ac9af/docs/development/v1-workstream-assignments.md).

## Common dispatch rules

Each dispatch records the exact integrated SHA, branch and worktree, dirty state, writer and exclusive paths before any edit. Writers run in parallel as far as their exclusive scopes are disjoint (DEC-070). A writer edits only its listed paths; a needed change elsewhere goes back to the coordinator. Every PR carries exact registry updates for added or removed artifacts, actual local check results and an independent complete-diff review on the frozen head. Concept decisions are not reopened in implementation; a contradiction found in source becomes an open question with its evidence.

Tests use controlled sessions and real local Git. The full native requalification belongs to step 4. Step 2 additionally includes one bounded native contract proof with Codex under its own dispatch, before the dependent contract is fixed (see PR A1).

## Stream A: Run contract

**Scope:** `internal/run`, `internal/harness`, `internal/workflow`, `internal/apply`, `internal/adapters/runstore`, the Codex adapter's tool, session and harness-rendering files with their tests, `internal/adapters/gitrepo/publication.go` for its use of approval validation, the real-Git vertical tests in `internal/adapters/gitrepo` that drive Run through controlled sessions, the Run and Manager-collaboration passages of `docs/project-workflow.md` and `docs/architecture.md`, and the registry shards of these Packages. Project gains only what the `check` tool needs, coordinated in the dispatch.

**PR A1 – collaboration (D10–D13):**

1. Explicit per-Manager assignment state in Run replaces the separate booleans. Each transition follows a named event: assignment received, judgment recorded, handoff fixed, review recorded, correction delegated, late finding. The state is a means, not the goal: keep it as small as the events require, and judge it by clearer events and fewer special cases for the agent.
2. `ask` accepts only the direct parent or a direct child; the refusal names the allowed addressees.
3. In Implement a submitted child candidate is fixed, checked and delivered to the parent with its check result whether or not the check passes. Approval and integration of an invalid candidate are refused with the diagnostic. Invalid means failing a check that applies to that child state; integration still outstanding elsewhere is not invalidity, and checks that did not run are not reported as passed. Root receives its own completion diagnostic. Assessment and Verify handoffs stay judgments about the fixed input; their check results are information, not an integration gate.
4. A `check` tool returns the deterministic check of the caller's current copy for optional self-correction, using the same mechanism as the handoff check.
5. A judgment is bound to the assignment and the assessed state (concept §13.2). A question-or-answer message needs no judgment, also before the first one. The end of a native response is not a completion attempt: a response without a judgment leaves the assignment active and Run continues it without demanding a judgment merely because the response ended. Completion validity is checked when the Manager judges and before Run relies on an earlier judgment (Root completion); an invalid completion receives its concrete reason, and an earlier judgment never completes a changed assessed state. The Codex adapter keeps reporting the judgment recorded in a message; Run owns validity.
6. Findings in native tool schemas carry no `manager` field; the host sets the origin.
7. Harness rules and host messages keep role, organization rules, context, task and answer format. Remove the explanations of active continuations, drains and late-message handling.

Tests: one controlled-session test per counterexample in concept §13.3, including a question answered before the first judgment (no prompt for a judgment, no failure) and completion with an earlier judgment after the assessed state changed (refused); a child state that only lacks a sibling's integration is not invalid; sibling and grandchild-to-Root questions refused (V03); an invalid child submission reaching the parent and being refused for integration, then corrected (V05, scenario 26.4); a test that harness text and host messages contain no host-state explanations (V17); the existing full-hierarchy, correction, conflict, late-rejection and failure tests keep their assertions.

Continuation without a judgment: after a response that only handled communication, the assignment neither ends falsely nor stalls. Run continues it purposefully, without a loop of empty continuation messages; a Manager that repeatedly makes no progress ends honestly within finite limits, never as success. An existing judgment stays valid until an actual change of the assessed state or new mandatory Findings, and pure communication never counts as new domain work. The independent review of A1 judges whether the assignment state makes the behavior simpler to understand, not only whether the new tests pass.

Native contract proof: once the collaboration contract works with controlled sessions, and before PR A1 is frozen for integration, run one bounded Codex proof under its own dispatch on a small fixture. It covers an answer without a new judgment, a question answered before the first judgment, a judgment that becomes newly required after a change, and an invalid child candidate followed by parent correction. The full Library requalification stays in step 4.

**PR A2 – closed-result validation (D15):** move the validation of closed results from `internal/workflow` into the Run contract; workflow and publication call it and no longer name Run resources.

**PR A3 – Notes pages (D17):** first reproduce the size conflict with stored Notes whose total exceeds one agentd frame, then choose the smallest read interface: bounded, stably ordered pages, each within the frame, with no silent truncation or loss. Sort key, page size, filter and the handling of a single very large Note are chosen and justified in the PR.

## Stream B: Provider scope

**Scope:** `internal/adapters/claudecode`, the Claude parts of `internal/runtimeconfig`, `internal/bootstrap`, `internal/adapters/container` and `internal/tooling/release/source_layout.go`, the provider guide, the Claude passages of `docs/architecture.md`, configuration examples, the README provider row and the affected registry entries, architecture maps and code map as assigned in the dispatch; for B1 also `internal/adapters/codexappserver/execution_profile_test.go` (provider-mismatch case only) and the Claude sentences of `docs/project-workflow.md`; for B2 also `packaging/runtime`, `internal/adapters/codexappserver/metadata.go`, `metadata_test.go`, `metadata_decode_test.go` and the runtime.env fixture in `internal/bootstrap/main_test.go`.

**PR B1 – remove the Claude path (D18):** remove the Claude adapter package, its profile decoding, metadata preparation, credential resolution and Bootstrap wiring, and the Claude branches in examples and guides. The runtime profile accepts only the Codex section. Git history keeps the removed code for the later Claude adapter.

Tests: full build, vet, architecture and accounting gates; configuration example tests; a runtime profile with a Claude section is refused with a clear diagnostic. B1 may also change the provider-mismatch case in `internal/adapters/codexappserver/execution_profile_test.go` to a neutral provider and remove the Claude sentences from `docs/project-workflow.md`.

**PR B2 – runtime image (D18):** in parallel with A1, whose Codex adapter files are disjoint from the metadata parser, and before the step 4 native requalification, remove Claude Code from `packaging/runtime` (the Claude Code install and version pin, the `claude=` record the Dockerfile writes to `/opt/markitect/runtime.env`, the Claude environment setting and the README) and the matching record from the runtime metadata parser in `internal/adapters/codexappserver/metadata.go` with its fixtures and Bootstrap tests. The changed image is the one step 4 qualifies. B2 keeps the shared test symbols `fixtureMetadata` and `fakeImages` and changes only their values, because A1 tests use them.
