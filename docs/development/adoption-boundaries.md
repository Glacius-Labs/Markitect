# Existing-project adoption boundaries

This note separates current Markitect contracts from a proposed preparation and discovery workflow for an existing repository, including the Konfyra consumer inventory. It records ownership and design questions for the coordinator; it does not define a new command, schema, or permission model. The reviewed baseline is integrated PR 59 at `e9550f5c91430c6a65cbbd4ffcdbb49b48272537`.

## Current behavior

### Init

The published v0.12.0 `init` contract starts a new Markitect Project in a repository that has no `markitect.yaml`. It previews a minimal Project plus one Area README. The default Area is `.markitect/areas/<namespace>`; `--path` selects another supported repository-relative location. Preview is read-only. A write recomputes and validates the plan, requires the Git worktree root and a named non-protected branch when in Git, and creates only the Project file and Area README using exclusive creation. The selected Area directory must not already exist. Existing configuration, path aliases, unsafe or ignored targets, and staged or committed target paths are rejected. Partial writes are reported for deliberate recovery; cleanup is not automatic.

Consequently, current `init` does not adopt an existing Project, inventory repository roots, inspect or classify files, create discovery evidence, or set checks, policy, adapters, packages, or provider configuration. Its default path is a convention; `--path` and already configured Project layouts remain supported. See [usage](../usage.md#initialize-a-project), implemented in `internal/app/init*.go` and `cmd/markitect/init.go`.

### Copy Me and evidence

The provider-neutral Copy Me workflow shipped in v0.11.0 and remains part of the published v0.12.0 guidance. It is invoked only when a person requests discovery. It describes owner-agreed scope, exact selected paths at immutable full Git revisions, a committed ContextRun manifest per repository, a preserved context report, and a separate evidence ledger and candidate. Multiple repositories retain separate snapshot identities. The workflow asks authors to distinguish observations from interpretations, retain counterexamples and uncertainty, and keep candidate/evidence material outside configured Areas and packages.

The current v0.12.0 mechanism is authoring guidance plus Markitect's existing ContextRun facility; there is no `discover` CLI or model call in the deterministic core. The synthetic `examples/engineering-discovery/` helper checks fixed-input identities and byte bindings. Its `engineering-discovery-evidence/v1alpha1` and `engineering-discovery-decision/v1alpha1` documents are example-local dossier formats, not published Markitect schemas or a general adoption API. A helper can verify hashes and fields, but cannot authenticate the named reviewer, establish evidence completeness, or enact acceptance. Adoption remains a separate ordinary Project change after explicit human review.

Copy Me does not supersede the [project-artifact boundary](../documentation.md): source code, configuration, infrastructure, CI, schemas, and documentation can be explicit UTF-8 inputs, but are opaque bytes to Markitect. A ContextRun source selection contributes to that run's context and digest; it creates no resource-graph edge. Project-owned source analyzers belong in explicit checks or adapters. Do not infer policy, structure, ownership, dependencies, or a person's preference from unselected files or prose links.

### Release and source status

v0.12.0 is the latest published contract in this baseline. Source-only designs, experiments, and roadmap proposals do not change that release. In particular, the policy-failure analysis document is an unreleased source design; it does not make policy-failing Context or Impact available in v0.12.0. Any later source behavior needs its own integration and release evidence.

## Proposed preparation and discovery boundary

The proposed work separates preparing an adoption workspace from analyzing evidence and from adopting any conclusion. Its intended preparation can help an owner review repository roots and potential evidence scope. A future inventory may distinguish roots detected by tooling from roots selected for analysis and roots explicitly excluded. Those categories are review states, not implicit permission to read or interpret every detected file. The owner must review scope before analysis begins.

For an in-scope Konfyra use, the deliverable is a consumer inventory, coverage account, and report of observed gaps. It may help classify a gap and preserve evidence for human follow-up. It cannot authorize a Markitect Core rule, change Konfyra policy, activate an adapter, or rewrite AGENTS, provider instructions, documentation, tests, or other existing authorities. Existing authorities remain in force during any shadow comparison. A gap is evidence to review, not self-authorizing product input.

Independent investigations may proceed in parallel only where each has bounded source ownership and does not write shared canonical or consumer files. They should use the same owner-approved scope and immutable source identities, or explicitly record separate snapshot identities. Findings must be reconciled in an outer report without flattening provenance. A shared candidate, evidence ledger, decision record, or consumer worktree is a coordination boundary; parallel readers must not silently overwrite one another's selection or decision state.

These are proposed outcomes, not a frozen CLI. The source-selection and snapshot schema is an open coordinator-owned design. Do not infer a `markitect init --existing`, inventory, discovery, resume, or adoption command from this note or copy illustrative syntax into a contract. Keep preparation, selection, discovery, review, and adoption as distinct states until the coordinator approves their interfaces.

## Fixed inputs, scope review, and privacy

Any future analysis should be bound to fixed, reviewable inputs. At minimum, the design needs an immutable identity for each repository snapshot, exact selected paths and byte hashes, the selection decision and its digest, the tool identity, and the context/result identity. A hash establishes byte identity; it does not establish authorization, consent, source legitimacy, completeness, or human acceptance. No glob, live working-tree drift, implicit repository traversal, or current-branch lookup should silently broaden a reviewed selection.

Selection review should make detected, selected, and excluded roots visible, record who supplied the scope and why, and preserve exclusions as first-class information. Changing a selected path, source revision, or evidence bytes should invalidate dependent analysis and require renewed review. Whether exclusions need content hashes, whether directory roots can be selected at all, and how to bind multiple repositories remain design questions.

Treat tickets, reviews, conversations, and personal information with a stricter boundary than repository files. Include them only when the owner explicitly approves the source and purpose, exports the exact material, and applies an agreed redaction and retention policy. Do not fetch provider histories or infer an individual's preferences from activity that was not selected. The current Copy Me workflow already describes explicit exported, redacted UTF-8 evidence with immutable source identity; a future workflow should preserve at least that boundary and define storage/access before collecting additional evidence.

## Candidate is not authority

Keep observations, interpretations, counterexamples, and candidate rules in a separate dossier or staging area outside configured Areas and package contents. Preserve competing explanations, historical differences, sample limits, uncertainty, and gaps in coverage. Do not turn discovery output into a canonical resource, generated projection, active package pin, check, adapter, exception, or implementation change.

A human decision applies to exact candidate and evidence bytes and should record the scope, rationale, decision, and reviewer identity as supplied. A changed candidate or cited evidence requires a new decision. These records provide traceability; hashes and supplied identity do not authenticate the reviewer. Even an explicit acceptance only permits a separate adoption review. The adopting Project owner decides whether to make a normal Project change, then reviews its Domain, resources, pins, checks, impact, and semantic correctness. Until that change is deliberately made and verified, existing project and product authorities remain unchanged.

## Open design questions for the coordinator

- What exactly can preparation detect, and which detected roots require owner selection or exclusion before any read? How are nested repositories, submodules, generated trees, symlinks, and ignored paths handled?
- Is each source identity a Git commit, another immutable snapshot, or both? How are several repositories or roots represented without merging their provenance?
- Which evidence types and encodings are eligible? Are paths always exact, UTF-8 files? How are size limits, secrets, personal information, and explicitly exported external evidence governed?
- What must the selection record bind: path and byte hashes, exclusions, owner/scope review, tool version, ContextRun manifest, context digest, and report digest? Which changes force reselection or rerun?
- Where do preparation reports, candidates, ledgers, and decisions live so they remain outside canonical Areas, packages, generated outputs, and consumer authority files? Who may read or retain them?
- How do parallel investigators partition read-only work, record separate identities, and merge findings without shared-file races or lost counterevidence?
- What human roles can accept, defer, reject, split, or revise a candidate, and what distinct owner review is required before normal Project adoption?
- What is the narrow Konfyra inventory question and coverage denominator? How will observed gaps remain consumer-owned findings rather than implicit Core requirements?
- Which tests establish no mutation during preparation/discovery, exact input binding, visible exclusions, invalidation after source changes, separation from canonical state, and absence of automatic authority or policy changes?

Until these questions are resolved, stable contracts are the existing v0.12.0 Init and ContextRun behavior and the shipped Copy Me guidance. New selection, snapshot, inventory, or decision formats remain proposals.
