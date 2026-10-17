# Manager realization north star

The owner asked to keep all ideas and variants of the Manager discussion as a north star, so that none is forgotten ([owner discussion](sources/manager-realization-20261016.md#preserve-the-variants-and-start-with-simple-context)). The [Apply concept](manager-apply-concept.md) owns what was selected. This page keeps the direction, the variants that were not selected and the later extensions, each with its status. Recording an idea here does not schedule it.

## Direction

- Managers organize the realization of the canonical model as code, tests, documentation and other artifacts. One Manager per Package; the root is an ordinary Manager ([DEC-025](../../concepts/register.md#dec-025-package-managers-organize-realization)).
- The realization strategy is customizable: a Manager's responsibility stays the same while its team, tools and way of working vary.
- Apply comes first and must work reliably. Every other operation is an extension of that foundation ([DEC-026](../../concepts/register.md#dec-026-apply-comes-first-and-changes-only-artifacts)).
- The owner raised an organizational framing for later: the Manager hierarchy as the application's entry point, with the canonical model as its foundation. It might support a later Government extension. This is an open framing ([OQ-008](../../concepts/register.md#open-questions)); [DEC-004](../../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) keeps realizing a model apart from deciding model changes.

## Team variants

| Variant | Intended behavior | Status |
| --- | --- | --- |
| Manager implements | The Manager implements and organizes review and integration. | Selected as the first strategy, with a fresh reviewer |
| Implementer and reviewer | Agents alternate implementation and review until the reviewer is satisfied. | Later strategy |
| Reviewer per Definition | Reviewers examine the realization of individual Definitions. | Later strategy |
| Teams at the leaves | Only the lowest Managers have executing teams; higher Managers integrate and coordinate. | Later strategy |
| Teams at several levels | A Manager does local work and integrates delegated child work. | Covered by the first strategy |

Configurable teams are also where different perspectives and the four-eyes principle live ([organization and trust](../../concepts/reflections/organization-and-trust-20261016.md#different-perspectives-through-configurable-teams)).

## Hierarchies for code, tests and documentation

One shared hierarchy for all artifact forms is selected. The alternative remains: the same Package hierarchy repeated for separate code, test and documentation branches, each with its own Managers and teams, coordinated at the end. It could support specialized review at the cost of more coordination.

## Context options

Need to know keeps Agents focused; it is not a security boundary. Start simple and optimize only when real use shows a need. The three starting options can be combined:

| Option | Starting point | Status |
| --- | --- | --- |
| Bodies from the own Package | The bodies of the Package's Definitions, each under its filename as heading. | Selected, with the binding `spec` and type contracts |
| See also | Explicitly linked Definitions from other Packages as a compiled section or as pointers. | Selected for direct links |
| Files to read | The relevant source files or their paths, read directly by the Agent. | Available as targeted reads |

The owner also considered whether higher Managers should not know or read descendant files unless someone supplies them. The concept chose summaries first plus recorded targeted reads, with no file-blindness rule. The earlier [context and semantic-index proposals](minimal-language/procedure-proposals.md) remain material for later.

## Missing links and the Optimizer

The owner selected the strict variant as the baseline: authors express the cross-Package connections that context compilation needs ([DEC-027](../../concepts/register.md#dec-027-strict-links-with-an-optional-optimizer)). The compensating variant, in which discovery or broader reading during every realization finds unlinked statements, was not selected.

An optional Optimizer proposes missing links. Each proposal names its source, target and reason, and it may prepare the actual edits as a candidate. Review and integration under the project's delegated authority decide which edits become model content, without manual confirmation of each link. Finding a connection and adopting it are separate steps. The Optimizer's context can reach beyond existing links, since those links are what it investigates. How Managers organize that wider investigation is open. Structural validation can check a link once its form is defined; it cannot prove that all relevant prose is linked.

## Later operations and extensions

Model changes and artifact changes are separate operations. A task can read both, but its change target is explicit and checked against the returned edits.

| Operation | Change target | Intended work |
| --- | --- | --- |
| Compile | None | Check compilability and resolve the model deterministically, without Managers. |
| Model optimize | Model | Managers look for missing connections and return justified link proposals or candidate edits. |
| Check or verify meaning | None | Managers and reviewers assess selected obligations and report evidence and findings. |
| Cleanup or refactor | Model or artifacts, one per operation | Improve organization within the existing requirements and permitted freedom. |
| Bughunt | Artifacts | A prepared instruction: find bugs against the model and fix them, with review and integration. It never rewrites the model to fit faulty artifacts. |
| Apply | Artifacts | Reconcile artifacts with the current model. Implemented first. |

Most operations share one sequence: check and load the model, assemble context, give the operation's instruction and custom instructions to the root, delegate recursively, integrate and check. Custom commands could reuse it by declaring a task prompt, context, configuration and one change target. Command naming (for example `markitect model optimize` against a flat surface), a separate prepare command and the registration format are open.

Further extensions recorded for after the two-Apply demonstration:

| Extension | Idea | Register |
| --- | --- | --- |
| Computed scope | Impact-scoped Apply from the model difference, references and registry ([concept](manager-apply-concept.md#toward-computed-scope)). | Vision mechanism 1 |
| Assess | A full reconciliation that stops before publication and reports drift, for CI and existing projects. | |
| Observed code dependencies | Compare, for example, the Go import graph with the model's references; a difference becomes an Optimizer proposal for a missing link. | [ENH-001](../../concepts/register.md#enh-001-observed-code-dependencies-widen-impact) |
| Model lint | Registry-based signals: Definitions without realization, files realizing many Definitions, Packages near the context limit. | [ENH-002](../../concepts/register.md#enh-002-model-quality-linter-and-metrics) |
| Scenario Definitions | Owner-authored examples with structured inputs and outputs in the model, which tests must demonstrably cover. | [ENH-003](../../concepts/register.md#enh-003-test-to-statement-binding-and-test-review) |
| Richer teams | The team variants above, perspectives and four eyes. | [ENH-006](../../concepts/register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes) |
| Parallel writers | Bounded concurrency on the same ownership and reservation rules. | |
| Kind guidance | Operational task and review recipes keyed by resolved Kind identity, adding no project rules and granting no execution rights. | |
| Model operations | Optimizer, cleanup, bughunt and custom commands as listed above. | [DEC-026](../../concepts/register.md#dec-026-apply-comes-first-and-changes-only-artifacts) |
| Existing-project adoption | An operation that drafts a model and a registry proposal from existing code; the owner reviews and commits them. | |
| Egress allowlist | Restrict a session container's network to its provider and declared hosts. | [DEC-037](../../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers) |
| Experience and doctor | Experience notes and an operational check through the Managers ([experience reflection](../../concepts/reflections/experience-without-implicit-rules-20261016.md)). | [IDEA-001](../../concepts/register.md#idea-001-the-model-as-institutional-memory-for-stateless-agents), [IDEA-017](../../concepts/register.md#idea-017-operational-check-through-the-managers) |
| Approval reuse, cockpit, Government | Reusing earlier approvals, a richer interface and delegated model authority. | [VIS-001](../../concepts/register.md#vis-001-cockpit-personal-ai-and-autonomous-project-organization), [OQ-008](../../concepts/register.md#open-questions) |

The second session adapter and session containers with declared mounts, once listed here, moved into the first version with [DEC-037](../../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers).

## Simplified for the first version

The consolidation of 16 October 2026 replaced these earlier proposals with simpler first-version rules. They remain possible later refinements; the [earlier texts](https://github.com/Glacius-Labs/Markitect/tree/3ebe3c6e6d283b998020350334e24a7303b089b7/docs/design/canonical-model) stay in Git history.

| Earlier proposal | First-version rule |
| --- | --- |
| Bounded budget allocations per Manager, reallocated by the root, with an illustrative 60-step split. The owner asked to keep the budget model including bounded allocations ([source](../../concepts/sources/working-day-discussion-20261016.md#preserve-the-shared-budget-model)). | The Host accounts once per assignment; one completion-reserve rule protects reviews and checks ([execution](manager-apply-execution.md#one-ledger-with-protected-completion-capacity)). Allocations can return if real runs show the need. |
| An optional absolute delivery deadline that keeps running while work waits. | No deadline; caps on active execution time remain. |
| One assignment commissioned for both stages of the two-Apply demonstration. | Each Apply invocation is its own assignment; further runs within an assignment are continuations. |
| A separate final verifier Agent. | The Host's deterministic final evidence check; the root's fresh integration review is the semantic review of the whole. |
| Successor runs replacing an unfinished run in one Superseded handoff. | A run that needs a changed basis finishes Incomplete; a continuation starts after the committed model edit. |
| Four applicability judgments for inherited findings. | Three finding states: open, resolved, not applicable; a changed requirement becomes a new linked finding. |
| Gating active work by derived delivery exposure. | Dependent work starts only from an accepted, composed delivery; a blocked delivery blocks its consumers. |
| Three numbering schemes (P00 to P12, R0 to R6, delivery packages). | Gates P00 to P12 keyed to the delivery packages. |

## Open questions

- Which explicit connections does the strict baseline require, and which reference classes and directions feed context?
- How does the Optimizer present, review and integrate its suggestions, and how is its investigation scoped?
- Which team variants should be tried after the first strategy, and how are review loops concluded or escalated in them?
- Should code, tests and documentation ever get separate hierarchies?
- How are model and artifact commands named, and how are custom commands registered without another execution mechanism?
- How can an organizational entry point support later Government while keeping model and artifact change targets apart?
