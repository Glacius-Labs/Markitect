# Existing-project adoption boundaries

This note preserves the PR #59 preparation contract and its version-bound examples. Current model-first adoption is documented in the [Brownfield workflow](../project-workflow.md#existing-repositories) and [Project operations](../project-operations.md). Resolve commands against the selected source or release before using the examples below.

This note separates the published Markitect contract, current source-only selective preparation/Copy Me behavior, and adopter-owned adoption. The canonical implementation contract is [Selective adoption handoff](../design/selective-adoption-handoff.md). Published v0.12.0 remains unchanged; source verification and release status are not established by this note. The reviewed historical baseline is integrated PR 59 at `e9550f5c91430c6a65cbbd4ffcdbb49b48272537`.

## Current behavior

### Init

The published v0.12.0 `init` contract starts a new Markitect Project in a repository that has no `markitect.yaml`. It previews a minimal Project plus one Area README. The default Area is `.markitect/areas/<namespace>`; `--path` selects another supported repository-relative location. Preview is read-only. A write recomputes and validates the plan, requires the Git worktree root and a named non-protected branch when in Git, and creates only the Project file and Area README using exclusive creation. The selected Area directory must not already exist. Existing configuration, path aliases, unsafe or ignored targets, and staged or committed target paths are rejected. Partial writes are reported for deliberate recovery; cleanup is not automatic. The newer source-only `prepare` workflow is a separate command and does not change these `init` semantics.

Consequently, `init` itself does not adopt an existing Project or alter checks, policy, adapters, packages, or provider configuration. Its default path is a convention; `--path` and already configured Project layouts remain supported. The independent source command `prepare` performs only the explicitly reviewed selection/capture described in the [shared contract](../design/selective-adoption-handoff.md). See [Usage](../usage.md#initialize-a-project) for published `init` behavior; the coordinator is adding current-source CLI usage separately.

### Copy Me and evidence

The provider-neutral Copy Me workflow shipped in v0.11.0 and remains part of the published v0.12.0 guidance. It is invoked only when a person requests discovery. It describes owner-agreed scope, exact selected paths at immutable full Git revisions, a committed ContextRun manifest per repository, a preserved context report, and a separate evidence ledger and candidate. Multiple repositories retain separate snapshot identities. The workflow asks authors to distinguish observations from interpretations, retain counterexamples and uncertainty, and keep candidate/evidence material outside configured Areas and packages.

The published v0.12.0 Copy Me mechanism is authoring guidance plus Markitect's ContextRun facility. Current source additionally provides a `copy-me` command that validates the prepared handoff and explicitly supplied queue/decision records outside Core; it does not discover evidence or call a model. The synthetic `examples/engineering-discovery/` helper and its `engineering-discovery-evidence/v1alpha1` and `engineering-discovery-decision/v1alpha1` documents remain example-local, not the new shared record contract or published schemas. A helper can verify hashes and fields, but cannot authenticate the named reviewer, establish evidence completeness, or enact acceptance. Adoption remains a separate ordinary Project change after explicit human review.

Copy Me does not supersede the [project-artifact boundary](../documentation.md): source code, configuration, infrastructure, CI, schemas, and documentation can be explicit UTF-8 inputs, but are opaque bytes to Markitect. A ContextRun source selection contributes to that run's context and digest; it creates no resource-graph edge. Project-owned source analyzers belong in explicit checks or adapters. Do not infer policy, structure, ownership, dependencies, or a person's preference from unselected files or prose links.

### Release and source status

v0.12.0 remains the published contract in this baseline. Selective `prepare`/`copy-me` support now exists in source outside Core and without a mandatory Project migration; it does not change that release. Policy-failure analysis is also unreleased source behavior. Any later release claim requires its own immutable publication evidence.

## Implemented source boundary

The source workflow separates preparation from evidence interpretation and adoption. The owner supplies repository identities, revisions, exact paths, inclusion reasons, exclusions and review/privacy/retention claims. Preparation does not inventory candidate roots or expand scope from discovered content. It captures only selected blobs into a separate external workspace; Copy Me consumes those bytes and caller-supplied records. Full details belong to the [selective adoption contract](../design/selective-adoption-handoff.md), not this summary.

For any adopter—including Konfyra—an evidence dossier can help preserve coverage and observed gaps for human follow-up. It cannot authorize a Markitect Core rule, change adopter policy, activate an adapter, or rewrite existing authorities. Existing authorities remain in force during review. A gap is evidence to discuss, not self-authorizing product input.

Independent interpretation may proceed in parallel only over the same explicit prepared handoff or separately identified evidence selections. Each investigator's outputs remain distinct until deliberately reconciled; shared queue/decision bytes are reviewed inputs whose digests invalidate dependent decisions when changed. Workspace provenance does not authorize a consumer edit.

The contract is a source implementation, not a v0.12.0 release change. The command and record interfaces are defined in the canonical design and Usage documentation as it is integrated. Do not infer an `init --existing`, repository inventory, discovery, resume, or adoption command; none is implemented by this slice. Preparation, interpretation, review, and adoption remain separate states.

## Fixed inputs, scope review, and privacy

Current preparation binds analysis to fixed, reviewable inputs: repository identity and commit, exact selected paths/bytes, scope, capture and handoff digests, with dependent decision records bound to the handoff and supplied queue bytes. See [Selective adoption handoff](../design/selective-adoption-handoff.md) for field semantics. A hash establishes byte identity; it does not establish authorization, consent, source legitimacy, completeness, or human acceptance. No glob, live working-tree drift, implicit repository traversal, or current-branch lookup silently broadens a reviewed selection.

The owner supplies selected repository roots and exact file paths plus explicit exclusions; the tool does not inventory or classify detected roots. Scope/revision/path/evidence changes alter the bound handoff and stale dependent records. Directories are not evidence selections; multiple repositories remain distinct identities. Exact record semantics are in the shared design.

Treat tickets, reviews, conversations, and personal information with a stricter boundary than repository files. Include them only when the owner explicitly supplies the exact material and purpose under an agreed redaction and retention policy. Do not fetch provider histories or infer an individual's preferences from unselected activity. The current source contract accepts only explicitly supplied bytes and records privacy/retention claims; those fields are not authentication, access control, or provider guarantees. See the [selective handoff design](../design/selective-adoption-handoff.md).

## Candidate is not authority

Keep observations, interpretations, counterexamples, and candidate rules in a separate dossier or staging area outside configured Areas and package contents. Preserve competing explanations, historical differences, sample limits, uncertainty, and gaps in coverage. Do not turn discovery output into a canonical resource, generated projection, active package pin, check, adapter, exception, or implementation change.

A human decision applies to exact candidate and evidence bytes and should record the scope, rationale, decision, and reviewer identity as supplied. A changed candidate or cited evidence requires a new decision. These records provide traceability; hashes and supplied identity do not authenticate the reviewer. Even an explicit acceptance only permits a separate adoption review. The adopting Project owner decides whether to make a normal Project change, then reviews its Domain, resources, pins, checks, impact, and semantic correctness. Until that change is deliberately made and verified, existing project and product authorities remain unchanged.

## Remaining evidence and adoption limits

The design resolves command shape, fixed input scope, external workspace boundaries and digest binding; see the canonical contract for the specific rules. Remaining review concerns are operational evidence, privacy/retention decisions supplied by each owner, source/CI validation, and whether any adopter separately wants to adopt a resulting candidate. No source status answers those product or human-approval questions. Do not claim CI completion or publication from the presence of implementation files.
