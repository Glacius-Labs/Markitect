# Apply architecture review, 17 October 2026

The owner asked the assistant to act as the project's architect. The owner wanted an assessment of the project and the Apply concept, with emphasis on cleanliness and architectural quality, and asked what is missing and what should be done differently. This page records the findings and the proposed approach. It is a dated assessment, not a measured result. The owner decisions it led to are listed under [owner decisions](#owner-decisions) and in [DEC-040](../../concepts/register.md#dec-040-concept-freeze-until-the-core-runs).

**Basis:**
- **Revision:** Main [`3580b30d`](https://github.com/Glacius-Labs/Markitect/tree/3580b30d37f31fbaba0ff1606cd29ed3e58f44e8). Later merges up to `0674e129` are noted where they change a finding.
- **Method:** four independent read-only reviews. They covered code architecture, documentation, work in flight and process, and the Apply concept family. The main claims were then spot-checked against source.
- **What the numbers are:** counts are measurements at that revision. Cost figures for a 15-Package model are estimates. The judgments are the assistant's.

## Verdict

The product idea and the target architecture are sound, and the Go dependency structure is clean. The project is being built from the outside in. The concept and the infrastructure around the core keep growing, while the core does not yet exist. The core is the Manager task process: work, independent review, rework, integration and acceptance. The [project history](../../history/project-history-20261017.md#assessment-of-the-working-method) named the same pattern as the main weakness, and it continued on 17 October.

## Keep

- **Four separate inputs and records:** model, organization configuration, realization registry and execution history. Binding intent lives only in the model.
- **Participation and process are guaranteed by the Host** ([DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process)). Agents realize and judge; they do not choose participants or rewrite operations.
- **Ownership:** one accountable owner per artifact, one writer at a time, and an overlap is an error.
- **Reviews** record their expectations before they see the candidate.
- **Definition census:** each Definition gets a disposition backed by evidence. The Host checks the final evidence.
- **Code:** the real import graph matches the [Go architecture](manager-apply-go-architecture.md#dependency-direction-and-ports) and is enforced in CI. Interfaces are few and live with their consumers. The model compiler is pure. Core tests are fast.
- **Delivery:** small PRs, a green Main (80 of 80 recent runs), and review that catches real defects.

## Findings

### F1 The core is unspecified and unbuilt while its periphery hardens

- **The core is not specified.** The Manager message tool is "schema pending" ([port contracts](manager-apply-port-contracts.md#session-and-communication-port)). The concept still asks to "define and validate the technical message, readiness, delivery and counter bindings" ([concept](manager-apply-concept.md#remaining-process-detail-and-technical-boundary)). No document defines a task state machine.
- **The core is not built.** `internal/run` implements sessions, binding, budget and restore, but not task execution, review or integration.
- **About 16% of production Go is unreachable from any command** (about 8k lines). This includes `adapters/journal`, the Claude Code codec, most of `run` and of `realization` composition, and context rendering.
- **That code is fixed in place by unit tests** before one end-to-end loop has validated its contracts. `run.Sessions` has no implementation.
- **The concept was revised under the code six times on 17 October.** DEC-037 made #313 (+2,009 lines) obsolete within hours.

**Recommendation:**
- Specify the core on one page: the task state machine, four or five message types, the composition rule and one issue type.
- Then build one Package end to end, as described under [proposed approach](#proposed-approach).
- Treat unreachable code as a draft that this slice may rewrite or delete.

### F2 The composition rules contradict each other

- **The concept** says composition is a disjoint union because each file has one owner and one task writes at a time. An overlap stops the run ([concept](manager-apply-concept.md#apply-lifecycle)).
- **The port contracts** say sequential writing does not establish a common base. A composition without a proven common base is refused and "never silently rebased" ([port contracts](manager-apply-port-contracts.md#candidate-workspace-port)). The [library walkthrough](manager-apply-library-flow.md#consuming-a-delivery-without-losing-local-work) says the same.
- **After a limited delivery**, such as a skeleton, producer and consumer never share a base again. The port contracts require the producer to establish a common base or return the conflict. The walkthrough relies on an explicitly admitted basis advance. So the operation that every parent integration needs has no specified rule.

**Recommendation:**
- Define composition as replaying owned file deltas onto the latest accepted aggregate. Exclusive ownership makes this safe.
- Renew a review only when an interface the subject consumes has changed.
- Keep a single overlap rule: stop.

### F3 The continuity machinery is accidental complexity for the first Apply

The first version specifies a large runtime apparatus: conversation continuity across working-basis versions, quiescence proofs, write barriers, warm retention, eviction order and provider resume ([concept](manager-apply-concept.md#working-bases-and-conversation-continuity), [execution](manager-apply-execution.md#idle-execution-and-resumption)). Every writing Manager meets this, because Explore is read-only and a change of mounts needs a new environment. DEC-039 already says "where supported", and a warm limit of zero is allowed.

**Recommendation:**
- In the first version, every step runs in its own container, and that container stops when the step ends.
  - The stopped container is the proof of quiescence. Warm retention, pause barriers and eviction rules become unnecessary.
  - The Host composes the next workspace version between steps.
  - The conversation resumes from the retained provider home where that is supported. Otherwise the Host supplies an evidence-linked handoff.
- Merge [#352](https://github.com/Glacius-Labs/Markitect/pull/352) showed after the basis revision that both CLIs can resume a conversation in a new container from the retained provider home ([record](../../validation/runtime-capabilities-20261017.md)). This makes the step-bound form cheap.
- While the run waits for the owner, no container runs, and the Host process can exit. `answer` and `resume` continue the run.
- Warm retention can return when a real run shows the need.

### F4 Complete participation costs a lot and does not scale yet

- **Cost per Apply for 15 Packages:**
  - Preparation reserves `2 * (N + P)` session starts for reviews ([execution](manager-apply-execution.md#one-ledger-with-protected-completion-capacity)).
  - With 16 Managers and 5 parents that is 42 reserved review starts plus 16 Explore sessions. The floor is about 58 sessions, even for a one-field change.
  - The estimate for a realistic Apply is about 70 to 90 sessions. A strong single-agent baseline would need 2 to 5.
- **Cost of routing:** cross-area questions travel one tree edge at a time. A question from `lending/reminders` to `members` and its reply cost four intermediate Agent turns: lending, root, root and lending.
- **Computed scope contradicts DEC-039.** [Computed scope](manager-apply-concept.md#toward-computed-scope) says "Managers outside it get no task". DEC-039 says no Manager is skipped.

**Recommendation:**
- Replace computed scope with reuse verified by the Host. When a Manager's bound subject is unchanged, the Host records that Manager's prior reviewed result without starting an Agent. The subject means its context digest, owned tree and consumed interfaces. This keeps every Manager in the census, and it is the way the participation guarantee scales.
- When the target of a question is known, the Host forwards it along recorded edges. The common parent is informed but does not relay it.

### F5 The validation cannot yet show the promised benefit

- **No baseline:** the [playground](manager-apply-playground.md) has no comparison arm.
- **A weak M2:** M2 changes one field in one Package, so it barely tests whether cross-Package obligations are kept.
- **Reviewer independence:** reviewers may share the Manager's profile and see the same compiled context. A missing link blinds both.
- **"Consistent repository" rests on one review:** it depends on a single root integration review that works from summaries.

**Recommendation:**
- Add a strong single-agent baseline with the same oracle and budget.
- Add an M2 that crosses Packages.
- Run reviewers on a different model family, and test them against seeded defects.
- Introduce an integration census: every explicit cross-Package reference gets a disposition in the integration review of its lowest common ancestor.

### F6 Code structure and style

- **`internal/apply` is not the Apply use case.** It holds the YAML decoders for five configuration files, the preparation pipeline, profile observation and run control. It also has twelve parallel Captured/Preview type pairs.
- **Infrastructure is duplicated:**
  - two Git acquisition stacks: `tooling/source` and `adapters/gitrepo`;
  - six strict-YAML node walkers;
  - two digest representations.
- **Errors are opaque.** Production code in `run` returns a bare `ErrInvalid` at 124 sites, and `run.Bind` alone has about 22. When the first real binding fails, the reason will be unknown. One condition at `internal/run/manager.go:327` is 872 characters long.
- **Leftovers, contrary to [DEC-032](../../concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility):**
  - The v1/v2 basis replay is retained for runs that never happened.
  - `organization.Manager.PassThrough` is deprecated but still used by `run.reviewFloor`.
- **The CLI** builds its reports from `map[string]any` and a reflection serializer. `check` on this repository prints 5.5 MB of JSON.

**Recommendation:**
- Split preparation from the Apply use case, and place each decoder beside its consumer.
- Keep one Git stack, one strict-decode helper, one digest type, and one error type that carries a reason.
- Delete the replay and `PassThrough`.
- Use typed report structs.

### F7 The documentation breaks its own one-owner-per-fact rule

- **The rejection limit** is restated in nine active documents. Four of them claim to own it, and the model itself does not state it ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)).
- **"Apply is unavailable"** is repeated in at least twelve places.
- **Volume:** `docs/` holds 5.5 MB against 1.5 MB of production Go. The reading path to understand Apply is about 42k words.
- **History mixed with contracts:** about 970 KB of historical, retired or dated material sits outside `history/`. It includes the replacement map with 26 retired `src/` paths.
- **Register entries are patched instead of replaced.** DEC-039 is 5.4 KB with four refinement bullets. The register defines a Superseded status, but no entry uses it, even where DEC-039 replaces parts of DEC-029 and DEC-036.
- **Links:** CI checks the links of 33 entry documents only. The Apply family is not checked.
- **Contract style:** the contracts are careful prose. They have no rule IDs, and owner narration is woven into the binding rules, so an implementer must decide sentence by sentence what binds.

**Recommendation:**
- One short "built and planned" table, linked from everywhere else.
- One contract per rule, with numbered rule IDs that tests can cite, and links elsewhere.
- Archive material moves to `history/`.
- Register entries hold the statement, reason and owner link. Replaced entries use the existing Superseded status, and a new Retired status covers entries whose subject no longer exists.
- The link check covers all active documents.

### F8 Process overhead and repository hygiene

- **Overhead:**
  - Coordinator handoffs reach 153 lines of session IDs, SHAs and worktree tables, although the [delivery plan](../../implementation-plan.md#sessions) forbids status mirrors.
  - Worktree preservation was lost when the Codex App emptied 250 of 308 Codex worktree folders.
- **Hygiene:**
  - 92 local worktrees, 85 of them at merged commits.
  - About 261 merged branches remain on origin.
  - The draft PRs #264 and #265 are superseded by #266.

**Recommendation:**
- A handoff is a short list of PR links. Pushed commits are the preservation.
- One approved cleanup: close #264 and #265, delete merged and superseded branches, enable automatic branch deletion, and prune merged worktrees without uncommitted changes.

## Proposed approach

1. **Freeze the concept** ([DEC-040](../../concepts/register.md#dec-040-concept-freeze-until-the-core-runs)). No new mechanisms or design areas until the core runs.
2. **Complete the concept without adding scope:**
   - Write the one-page core specification (F1).
   - Resolve F2 and F4, and apply the cuts of F3.
   - Where a proposal changes an adopted item, the owner confirms it first. Such items include:
     - computed scope, which is in DEC-003 and DEC-010;
     - DEC-039's participation and review-reuse rules.
   - Merge the roughly twenty status vocabularies into a few:
     - one effect record with the outcomes done, not done and unknown;
     - one issue type for findings, questions and decisions;
     - the task machine;
     - the run machine;
     - the dispositions.
   - Give each rule one owner and an ID (F7).
3. **Clean up in parallel** where no other session works: documentation (F7), code duplication and leftovers (F6), and repository hygiene (F8).
4. **Build a walking skeleton:**
   - `markitect apply` on library M1 for one Package, first with a controlled fake session and then with real Claude Code and Codex sessions.
   - The steps are Explore, work, review, rework, acceptance and guarded publication.
   - Recursion, integration, the complete Manager tree and the two-Apply demonstration follow.
5. **Measure** the coordination account against a strong baseline (F5) before any later extension.

**Proposed division of work (owner proposal, 17 October 2026):**
- **This architect session:** the concept and the core. That means the Apply design family, `run` task and review execution, `realization` composition, `organization` context rendering, and `apply` orchestration.
- **The Coordinator and its implementers:** the surrounding parts. That means the container runtime and `agentd`, the session adapters, the journal adapter, repository and check adapters, packaging, CI and release tooling, and marketing.
- **Shared files** stay reserved explicitly, as the [delivery plan](../../implementation-plan.md#sessions) describes.

## Owner decisions

On 17 October 2026, after this assessment, the owner made these decisions. The [source](../../concepts/sources/architecture-review-20261017.md) keeps the original wording.
- **Freeze:** the owner froze the concept ([DEC-040](../../concepts/register.md#dec-040-concept-freeze-until-the-core-runs)).
- **Order:** complete the concept and start the cleanup round before implementation. The cleanup round is worthwhile in any case.
- **Division of work:** proposed by the owner, and acknowledged by the Coordinator.

This review asked whether a Codex credential would be provided. Merge #352 answers that: a dedicated runtime login now serves Codex sessions ([record](../../validation/runtime-capabilities-20261017.md)).

## What would change this assessment

- A working one-Package loop shows that a fresh session per step loses information that review cannot recover. That would bring continuity back.
- Measured runs show that complete participation costs far less than estimated.
- A baseline comparison shows no quality advantage at equal budget. That would challenge the method itself ([DEC-009](../../concepts/register.md#dec-009-the-central-evaluation-question)).
