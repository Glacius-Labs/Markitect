# Engineering constitution

This page states the engineering principles that every change to Markitect keeps. [Architecture](architecture.md) describes how current source implements them, and [CONTRIBUTING](../CONTRIBUTING.md) owns the contribution checks.

## Product purpose

The [vision](vision.md) owns the product thesis: model-first development with delegated realization. People keep one canonical model of the project; Managers realize changes of intent under independent review, verification and guarded Apply. The principles below keep that delegated work bounded and its evidence honest. The goal of less routine human supervision does not weaken them.

## Determinism and explicit inputs

- A fixed model, snapshot and tool version give the same compile, coverage and impact results.
- Relationships and input scope are explicit. Prose, README links and paths create no dependency.
- Impact follows declared relationships. It stays conservative for unknown inputs; nothing narrows it silently.
- The structural Core stays generic. It has no providers, policy language, Git, paths or execution, and it does not depend on adopting repositories.
- New Core semantics need repeated concrete failures, a comparison with solutions in checks or derived views, and a focused generic design. Names, YAML repetition or provider convenience do not justify one.

## Distinct kinds of state

- The committed model is canonical. Observed repository state, inferred proposals, drafts, candidates and generated views stay distinct.
- Typed YAML is the canonical model. Generated Markdown, Codex and Claude files are views of it; change the source and regenerate them.
- Agent proposals and Brownfield findings become intent only when the owner commits the revised model.
- An adopting project's files are opaque declared artifacts ([project artifact boundary](architecture.md#project-artifact-boundary)).

## Manager authority

- Manager authority comes from the active model revision. The Managers, the paths they own and their rules are those of the committed model that a plan binds.
- Local agent settings, instructions and reports grant no authority.
- Each Manager may propose changes only within the write paths its model responsibility allows. Control-plane paths under `.markitect/` are never writable by a role.
- A parent Manager integrates its children's results and keeps the duty to integrate.
- A changed model makes a plan stale. It needs a fresh preview.

## Explicit budgets and role starts

- Every role start counts against explicit limits in `.markitect/runtime.yaml`: depth, starts, retries, parallel roles, duration, cost and candidate bytes.
- Each role start is reserved and recorded in the run before it happens.
- An unknown outcome is not replayed. Recovery inspects the saved attempt and never starts a replacement role ([candidate workspaces and recovery](project-operations.md#candidate-workspaces-and-recovery)).
- Configured cost is an estimate, not billing enforcement. An unmetered role records unknown cost.

## Lifecycle trust boundaries

The lifecycle is inspect, explore, edit, readiness, plan, run, verify, preflight and Apply ([project workflow](project-workflow.md)).

- Inspection is read-only and starts no role.
- A write that has a preview requires the preview's digest. A stale preview is refused.
- A plan binds its base revision, snapshot and model digest. Run and Verify work in owned workspaces; only Apply writes the checkout.
- Apply writes only the latest verified integrated candidate, after preflight, through guarded writes. It does not commit, merge, publish or deploy.
- Check and Verify are strict. An error result stays an error, structural findings block readiness, and a failed or incomplete verification blocks Apply.

## Executor, candidate and independent Verifier

- The method separates the Executor, its candidate and an independent Verifier ([delegated method](vision.md#the-delegated-method)).
- In current source a Manager, or a helper it starts, produces a candidate. An independent reviewer with fresh context assesses each candidate and each integration result, and the verify role checks the integrated candidate.
- Review and verify roles return no file changes.
- Tests the Executor wrote are realizations too; they are not independent assurance on their own ([CPT-002](concepts/register.md#cpt-002-tests-are-realizations-too)).

## Evidence and acceptance

- AI evidence, passing checks and digests are not human acceptance.
- Deterministic outputs, such as the generated model document, are checked byte for byte. Agent candidates vary and need project-owned checks.
- A result establishes its declared scope. It does not prove semantic correctness, complete modeling or business value.
- Agent settings and owned workspaces are not an operating-system sandbox.
- Benefits such as fewer missed updates remain hypotheses until measured on repeated real tasks ([measurement](measurement.md)).

## Intent changes and repairs

- A real intent change updates the model first: edit, check, inspect impact and commit under the repository's policy, then realize it.
- A repair against unchanged intent uses the same model. It needs no invented model edit.
- Both kinds of change use fixed snapshots, declared inputs and bounded evidence.

## The earlier line

The earlier v0.13 Project/Domain line had its own constitution, with Domains, finite policies, exceptions and Copy Me. Its text is kept in a [history record](history/engineering-constitution-v0.13-20261010.md). The line is not developed further, and compatibility is no reason to keep a function, mode or fallback ([DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions)). ARCH-07 moves this repository off it, and ARCH-09 removes it.

## Historical constitution links

These retained anchors route existing links to the [history record](history/engineering-constitution-v0.13-20261010.md).

<a id="kernel-invariants"></a>
The [v0.13 kernel invariants](history/engineering-constitution-v0.13-20261010.md#kernel-invariants) are historical.

<a id="reuse-existing-domain-packages"></a>
[Reusing Domain packages](history/engineering-constitution-v0.13-20261010.md#reuse-existing-domain-packages) is historical.

<a id="example-contract-vertical-slice-usecases"></a>
The [vertical-slice UseCase example](history/engineering-constitution-v0.13-20261010.md#example-contract-vertical-slice-usecases) is historical.

<a id="policy-results-and-explicit-exceptions"></a>
[Policy results and explicit exceptions](history/engineering-constitution-v0.13-20261010.md#policy-results-and-explicit-exceptions) are historical.

<a id="inspect-a-policy-breaking-candidate"></a>
[Inspecting a policy-breaking candidate](history/engineering-constitution-v0.13-20261010.md#inspect-a-policy-breaking-candidate) is historical.

<a id="copy-me-propose-review-adopt"></a>
[Copy Me](history/engineering-constitution-v0.13-20261010.md#copy-me-propose-review-adopt) is historical.

<a id="normal-changes-and-architecture-changes"></a>
The [earlier rule for normal and architecture changes](history/engineering-constitution-v0.13-20261010.md#normal-changes-and-architecture-changes) is historical; the current rule is [Intent changes and repairs](#intent-changes-and-repairs).

<a id="projection-ownership-and-evidence"></a>
[Projection ownership and evidence](history/engineering-constitution-v0.13-20261010.md#projection-ownership-and-evidence) is historical.

<a id="deliberate-boundaries"></a>
The [v0.11.0 deliberate boundaries](history/engineering-constitution-v0.13-20261010.md#deliberate-boundaries) are historical.

<a id="constraint-language-and-specialist-engines"></a>
The [constraint language and specialist engines](history/engineering-constitution-v0.13-20261010.md#constraint-language-and-specialist-engines) of the v0.13 kernel are historical.

<a id="canonical-reset-source-boundary"></a>
The [canonical reset source boundary](history/engineering-constitution-v0.13-20261010.md#canonical-reset-source-boundary) is historical; its Schema and Projection Module split left source with the canonical alpha.
