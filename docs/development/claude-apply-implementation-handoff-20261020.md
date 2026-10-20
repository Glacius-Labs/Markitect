# Claude handoff: checked recursive Apply passes

**Prepared assignment, 20 October 2026.** The owner adopted the complete discussed workflow and requested preparation for a Claude implementation session. The [accepted workflow](../design/canonical-model/manager-apply-workflow.md) is the behavior owner; [DEC-064](../concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams) records supersession. This handoff makes that assignment concrete. No implementation session or provider Run has been started by preparing it.

## Start and authority

The preparation branch is `docs/apply-terminology-notes`, in `C:\Users\Consiliari\.codex\worktrees\apply-terminology-notes\Markitect`. It includes integrated Main `a0928a1b7a95ebbc23836e5a5f472b29d233466e`, the preserved discussion commit `06ea1952` and the accepted concept/handoff changes. The preparation chat reports the final full commit SHA. Use the committed branch as the implementation base, not the unrelated original checkout at `C:\Users\Consiliari\Glacius Labs\Markitect`.

On starting Claude, verify the actual source commit, clean state and ancestry; inspect any subsequent Main changes before composing them. Create a separate implementation worktree and non-protected branch, for example `dev/checked-apply-passes`. Do not reset, stash or reuse another session's dirty checkout. Read `AGENTS.md`, the working method, current delivery plan, model, accepted workflow and this handoff. Run actual `markitect check` and `context product`; choose the owning model Packages before writing. The documentation Package owns this preparation, not all future runtime changes.

The accepted product decisions do not need another design-approval round. Choose concrete Go types, wire identifiers and storage within them, preserving simple boundaries. Complete the implementation and its relevant checks; report a real external authority/capability blocker honestly. A new product rule, spending authority, release or model contradiction still requires its proper decision.

This is one bounded Apply implementation assignment, not an Extract redesign, broad trial matrix, pilot release or UI project. Provider execution requires an explicit finite grant for the session and supported profiles/credentials. This handoff grants none and starts no Run. Lack of that grant does not block code, meaningful source tests or source smoke; it limits the runtime claim.

## Required result

`markitect apply` performs strict Model check and File check, then Plan → Prepare → Implement through one common recursive Pass mechanism, then fresh independent Verify of one immutable candidate. Genuine findings return to the necessary normal work Passes; clean Verify reaches guarded Publish/read-back. The entire required Manager organization participates, with enforced local writes, tree-only questions, explicit lifecycle reports/events and configurable bounded local teams.

There is no automatic repair of failed model/accounting gates, no implicit completion on Turn exit, no silent omission of model duties and no author acceptance substituting for independent Verify. No mandatory extra review after every phase or generic workflow framework is requested.

## Delivery sequence

Keep the assignment complete while using reviewable implementation increments. Do not claim the target implemented when only names or scaffolding exist.

| Increment | Concrete work and exit evidence |
| --- | --- |
| 1. Intent and terminology | Update the canonical self-model to the accepted behavior and regenerate official views with fresh `previewDigest`. Rename current competing Compile/Reconcile/controller-Host concepts across source, CLI/config, reports, tests and guides. Neutral shared Pass execution retains Extract's separate behavior. No aliases or old success paths remain. |
| 2. Checked ownership | Strict initial refusal; directory and exact-file specificity for any Manager; repository-relative local registry selectors; protected-input safeguards; single producer of versioned derived views; scoped stable acceptance and whole-Pass checks. Enforce the actual mount/write permissions, including deeper exclusions. |
| 3. Pass and lifecycle engine | Replace implicit first-Turn/done assumptions with explicit basis/progress/wait/completion/problem reports; Ready/Working/Waiting/Done; accepted events, dependencies, parent-first release and upward integration. Add Plan/Prepare/Implement and separate Verify mandates using the same engine. |
| 4. Questions and tools | Provider-neutral lookup/check/send/report, escalation as parent routing, bidirectional questions and cross-branch correlation, Root-to-invoking-AI question/answer channel. Root model-error abort remains mandatory. Preserve finite waiting and stop certainty. |
| 5. Team execution and Verify | Standard Manager/Worker/Reviewer roles with lazy bounded instantiation and configurable specialists. Per-Agent containers, full project reads, delegated local writes, no overlapping writers, fresh read-only reviewers, retained findings and complete exact-candidate Verify after correction. |
| 6. Complete source and runtime evidence | Reconcile active guides and examples, run relevant tests and bounded source smoke, then independently review the complete diff. With a separate finite provider grant, run the selected complete workflow and refusal/correction cases on one frozen build. Preserve failed attempts and unresolved limitations. |

The order expresses dependencies, not permission to replace intermediate stages with stubs, skip a meaningful test or stop at a design-only patch. Split source changes into focused commits or PRs if useful while maintaining an honest current contract.

## Source map and migration hazards

Paths below were inspected at the stated integrated base. Re-read the actual source rather than relying on the table as an immutable symbol index.

| Existing area | Required change or invariant |
| --- | --- |
| `internal/model/compile.go`, `internal/apply/model.go` | Name formal model validation Model check, for example `CheckModel`. Keep model language v1, diagnostics, canonical bytes and model/registry separation intact. Do not reinterpret language rules merely to rename an operation. |
| `internal/apply/file_check_run.go`, `preparation*.go` | Remove `repairableDrift` and every initial accounting exception. A failed initial gate starts no Manager/provider and publishes nothing. Later mandatory Model/File failure also stops, without an automatic accounting repair Pass. |
| `internal/realization/accounting.go`, `layout.go`; `internal/apply/layout.go`, `registry_record.go`, `registry_prepare.go`, `registry_decode.go`, `candidates.go` | Ownership by specificity, deterministic observed-file expansion, local shards, complete accounting, checked version composition. Current exact-file-only registration and Root-only exact-file exceptions do not satisfy the target. Do not weaken the source repository's own exact registrations while migrating target support. |
| `internal/organization/`, `internal/apply/brief.go`, `internal/run/prompts.go` | Fixed required organization, own/context labels and actual owners, project-wide read rights, explicit Root project context, Pass instructions and relevant dependency plans. Empty Packages must not disappear. |
| `internal/apply/reconcile.go`, `internal/run/reconcile.go`, `internal/run/engine.go`, `execute.go`, `turn.go` | Rename complete operation vs neutral shared engine consistently. Replace current many-boolean/first-Turn behavior with the small validated lifecycle. Remove implicit Done on a Turn without a semantic completion report. Parent basis acceptance and subtree completion are distinct. |
| `internal/run/tools.go`, `checktool.go`, `checks.go`, `verify.go` | Minimal semantic tools; Orchestrator-owned gate acceptance; tree-only correlated routing; candidate-bound checks; fresh separate Verify; retain findings across candidates until explicit independent closure. A completed negative review is not a failed reviewer task. |
| `internal/bootstrap/execution.go`, `internal/adapters/cli/command.go`, `execution.go` | Apply questions are no longer deliberately disconnected from the owner channel. Bind question/answer authority and Run IDs, expose pending questions, retain honest cancellation/deadline/restart behavior. Coordinate the CLI and bootstrap seams as one writer. |
| `internal/apply/host_profiles.go`, execution configuration; `internal/adapters/container/`, `claudecode/`, `codexappserver/`, `agentd/` | Coherent Orchestrator/Runtime vocabulary; explicit supported team/model profiles; per-Agent containers and enforceable rights. Preserve provider-required wire names. Reject unsupported capabilities rather than silently using a broader mount or a different model. |
| `internal/adapters/gitrepo/workspace.go`, `publisher.go` | Observe writer stop before capture/harvest, preserve fixed-source guard, exact verified write plan, destination checks and read-back. A partial failed Publish stays partial and unsuccessful; do not add automatic Git commit/merge/push. |
| `internal/bootstrap/` and shared `internal/run/` Extract callers | Shared refactoring must preserve Extract's reviewed behavior and its distinct model-writing authority. Do not make Apply's artifact publication an accidental side effect of Extract. |
| `.markitect/model/product/run/confirmed-run-history.yaml`, `.markitect/model/product/organization/delegated-context.yaml`, `.markitect/model/product/realization/artifact-accounting.yaml`, `.markitect/model/product/cli/honest-read-only-interface.yaml` | Locate all existing owning Definitions for changed contracts. Refine intent instead of writing a file/type inventory into the model. Keep exact paths and reasons in owner registry shards; regenerate views through the official guarded operation. |
| Current runtime/CLI/package guides, `AGENTS.md`, code map, execution examples | Update when code changes, preserving the distinction between implemented contract, accepted target and historical evidence. The current `--host-profile` example is implemented old syntax, not an exception permitting new controller-Host naming. |

The old process was executable and has bounded real evidence. Preserve its safety properties and useful ports rather than rebuilding unrelated infrastructure. Delete obsolete paths and their tests together; no compatibility scaffolding, skipped gates or fake pass results.

## Acceptance scenarios

Use meaningful behavior tests at the smallest boundary that proves each property. The following are required properties, not a request to mirror every implementation function with a test.

| Scenario | Required observable evidence |
| --- | --- |
| Invalid formal model | Stable structured diagnostics; no provider acquisition, child dispatch or publication. |
| Invalid initial accounting | Unexplained/missing/duplicate files, dangling Definition links, wrong/ambiguous/no owner and invalid protected inputs all refuse before Agent work. Previously accepted committed drift is not repaired. Gitignore does not hide it. |
| Valid but semantically wrong registered artifact | Its actual owner can repair behavior through the normal Passes, with valid accounting maintained. This remains distinct from invalid initial accounting. |
| Directory ownership and registry expansion | Parent directory, child directory and exact-file overrides resolve identically in lookup, File check and actual write capability. Equal-specificity conflicts fail. Other owners' descendants and protected inputs cannot be registered or written through a broad directory entry. |
| New files and parallel shards | New owned files are validly covered before accepted readiness/completion. Two disjoint Managers can finish concurrently without losing a shard update or claiming a mixed live global view. Later mandatory failed accounting blocks and stops; no auto-repair exception. |
| Parent-first work | Children cannot start on a mere Turn exit, unverified registry edit or unaccepted basis. Parent can subsequently answer questions and integrate children. Pass completion requires every required Package, including empty/intermediate ones. |
| Local concrete plans | Every Manager supplies a real own plan with obligations, method, evidence, inputs/outputs, owners and dependencies. Parent returns an incompatible child plan to that owner. No central plan silently substitutes for a missing child. |
| States and events | Ready dispatches; Waiting wakes only for relevant work; Done requires accepted complete result. Duplicate/stale reports cannot advance work. A normal exit without a report, technical timeout and uncertain writer stop never become semantic success. |
| Questions | Child→parent answers and parent→child questions work while waiting. A cross-branch exchange takes the ancestor route both ways; sibling/tool bypass is rejected. Root clarification can pause and continue the same valid Run; stale/unauthorized answers cannot. |
| Model authority | Root's erroneous/contradictory-model decision stops without publication. A clarification within existing rules can continue. An answer contradicting a rule never changes the bound model. |
| Team bounds | Helpers see all project artifacts, edit only delegated own subsets and cannot write sibling/protected paths. Overlapping writers cannot coexist. Reviewer is fresh/read-only and records findings independently of the author. Extra Architect/cross-cutting roles acquire no global ownership. |
| Separate Verify | Parent integration cannot skip independent Verify. Required checks and reviewers assess the same immutable candidate. A reviewer can finish Done with findings, and the Run cannot succeed. |
| Findings and correction | Preserve a real finding across Turns/candidates until an explicit fresh independent closure. A correction re-enters the needed normal work Passes and receives complete fresh Verify; optional notes do not become hidden model violations. |
| Guarded completion | Only clean Verify and successful exact Publish/read-back yield Success, including honest no-change. Source/destination drift, partial writes, limits/cancellation/unresolved dependency retain distinct failure evidence. No commit/push occurs. |
| Shared-engine regression | Extract retains its separately owned behavior. Read-only CLI preparation still starts no Agents/checks, and formal model/diagnostic/canonical-byte conformance is unchanged by naming changes. |

Test runtime rights at the actual container/adapter boundary as well as in pure decision logic. A fake adapter proving a proposed path filter is not evidence that a mounted repository enforces it. Where that boundary cannot be exercised locally, report the unproved capability and keep execution refusal rather than returning success.

## Validation and complete-workflow evidence

Follow [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change). Do not run two Go commands concurrently in one worktree or edit while its gate runs. Start with focused meaningful tests, broaden for changed shared contracts, then freeze and independently review the complete diff. For this cross-layer assignment, include full Go tests, vet/build, architecture/modules gates, structural working-tree and fixed-HEAD checks, generated-view freshness and relevant CLI/adapter checks. Record the exact commands, SHA and actual outcomes. Hosted CI and integration follow the current delivery plan; local tests do not imply either.

After canonical model edits, preview the official readable view, then use the returned `view.previewDigest` for the guarded write. Register every new artifact exactly in its owner's source registry and check links. Regenerate the code map officially if package mappings change. Use a fresh external work directory for `go run ./tools/smoke`; it starts no real Agent and does not prove Apply.

With a separate explicit finite provider grant, use one frozen build for a bounded Library sequence: model-only M1 → A1 → actual M2 → A2 → conforming no-change Apply, with source and outside-oracle evidence at each stand. Add only the targeted refusals, routed clarification, retained-finding correction and enforceable team/ownership cases needed to prove the changed contract. State provider/model/profile, limits, completion, intervention and unproved cases. Haiku and Luna are intended profiles; do not claim either is validated from the other's result.

The current Library canonical model includes the closeout decisions. Do not resurrect deleted `owner-decisions.json` or feed the independent evaluation contract/oracle to Agents. Preserve `experiments/apply-trials/` and its separately owned preparation work. Its old drift-repair expectations are not current acceptance under DEC-064; adjust selected expectations through the owning scope before a new run, without rewriting historical observations. No unlimited matrix is authorized.

Actual source checks, container enforcement, real provider Runs, hosted CI, Main composition, pilot packaging and human acceptance must remain separately reported. An unrun provider scenario is an explicit remaining evidence gap, never a simulated success. This handoff does not claim the new design increases reliability before those observations exist.

## Completion report

Report the implemented workflow and remaining limits against the accepted concept, exact source/head/base SHAs, focused/full test outcomes, independent review findings and their disposition, actual provider/container evidence if authorized, and any honest unproved property. Include current guides, canonical view and changed execution/config examples. Do not end with only a rename, a new state enum, a plan or an unsupported claim that Apply is trustworthy.

## Copyable Claude startup prompt

```text
Arbeite die akzeptierte Apply-Änderung vollständig aus und implementiere sie.

Ausgangspunkt ist der lokal vorbereitete Branch docs/apply-terminology-notes:
C:\Users\Consiliari\.codex\worktrees\apply-terminology-notes\Markitect
Verifiziere dessen tatsächlichen Commit, sauberen Zustand und integrierte Basis.
Erzeuge daraus einen eigenen Worktree auf dev/checked-apply-passes; bewahre alle
anderen Sessions. Lies zuerst AGENTS.md, den aktuellen Delivery Plan,
docs/concepts/register.md (DEC-064),
docs/design/canonical-model/manager-apply-workflow.md und
docs/development/claude-apply-implementation-handoff-20261020.md.

Diese Produktentscheidungen sind übernommen. Setze Konzept, kanonisches Modell,
Code, Tools, Konfiguration, Tests und aktive Dokumentation konsistent um.
Nutze den Handoff als abgegrenzten Auftrag mit Prüfszenarien. Wähle technische
Details selbst innerhalb des Konzepts; frage nicht erneut nach dessen Freigabe.
Keine automatische Reparatur fehlgeschlagener Model-/File-check-Gates.
Keine konkurrierenden Compile-/Reconcile-/Controller-Host-Begriffe.
Keine implizite Fertigmeldung durch Turn-Ende. Keine ausgelassenen Pflichten.

Führe sinnvolle Quelltests und die erforderlichen Prüfungen aus. Starte reale
Provider-Runs nur mit einer ausdrücklichen endlichen Freigabe für diese Session;
deren Fehlen verhindert die Implementierung und Quellprüfung nicht. Bewahre
Trial-Arbeiten und alte Evidenz. Hole vor einem PR den vorgeschriebenen
unabhängigen Review ein. Berichte Implementierung, lokale Prüfung, tatsächliche
Container-/Provider-Evidenz, CI und Owner-Abnahme getrennt. Kein Release.
```
