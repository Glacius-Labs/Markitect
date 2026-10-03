# Copy Me discovery workstream

This is a bounded future-work specification, not authorization to add discovery behavior to the deterministic core. Read it with the [shared contracts](../development/shared-contracts.md) and [adoption boundaries](../development/adoption-boundaries.md). The coordinator owns selection, snapshot, and workspace handoff; investigation can proceed in parallel, while durable implementation waits for approval of those shared contracts.

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

The Copy Me authoring Workflow shipped in v0.11.0 and remains in published v0.12.0. It guides human-led/provider-neutral analysis of explicitly selected evidence at immutable snapshots. There is no discover CLI or model call in Markitect's deterministic core. The synthetic `examples/engineering-discovery/` helper and its `engineering-discovery-evidence/v1alpha1` and `engineering-discovery-decision/v1alpha1` files demonstrate byte binding and dossier review; they are example-only formats, not reusable published schemas.

The desired future work concerns an explicit, fixed-input discovery workflow with observations, support and counterexamples, duplicate/conflicting evidence, current versus legacy observations, uncertainty, and a separate candidate/decision queue. It must preserve the rule that a candidate is not authority and must not presume a particular CLI or schema.

## Objective

Improve the reviewability and completeness of Copy Me discovery over a coordinator-approved selection. Produce a separate evidence dossier and candidate queue that preserves provenance, counterevidence, uncertainty, and owner decisions. Keep source interpretation and any model-assisted synthesis outside Markitect Core; keep adoption as a separate, explicit owner-reviewed Project change.

## Scope

- This workstream owns generic authoring guidance and, after contract approval, any separate discovery support package or example that validates evidence bindings without judging truth.
- It may investigate ledger completeness, evidence deduplication, conflicts, historical/current classification, candidate lifecycle, and decision invalidation.
- It does not own shared source selection, snapshot identity, or workspace handoff; those are coordinator decisions.
- It does not own adopter-specific policy. Konfyra observations remain inputs to consumer-owned review, not generic Core rules.
- Parallel analysis is permitted on separately assigned immutable evidence selections. Shared candidate/evidence/decision records require a single owner or serialized merge with preserved provenance.

## Current implementation

The normative current workflow is `internal/authoring/resources/workflow-engineering-discovery.yaml`; related authoring and adoption guidance is in `workflow-authoring-change.yaml` and `workflow-constitution-change.yaml`. The runnable synthetic example and helper live under `examples/engineering-discovery/`. `docs/usage.md`, `docs/engineering-constitution.md`, and `docs/validation/real-project-adoption-pilot.md` describe limits and evidence. None establishes a generic discovery API or accepts a human decision as an automatic mutation request.

## Owned subsystem

For future authorized changes, ownership is limited to Copy Me authoring resources and tests in `internal/authoring/`, plus a new, isolated discovery support package or fixture under its explicitly assigned path after the shared schema is approved. Example-only dossier checks stay example-scoped unless a separate product decision adopts their behavior. Do not modify `internal/core`, `internal/app` source selection, ContextRun parsing, snapshot acquisition, generic Project schemas, or adopter source files as part of this workstream.

## Allowed changes

- During investigation: compare the existing workflow/example against approved requirements; document missing evidence fields, lifecycle states, and review handoffs without claiming a frozen format.
- After approval: clarify or extend authoring guidance; add a distinct package/helper that validates exact IDs, paths, hashes, status transitions, and decision bindings; add examples that retain repository identities separately.
- Keep discovery output in an external dossier or isolated staging area outside configured Areas, package contents, generated projections, and canonical consumer files.
- Preserve observations separately from interpretations; include supporting evidence, counterexamples, duplicates, conflicts, date/period, scope, confidence basis, alternatives, and unanswered questions as the approved contract requires.
- Require an explicit, hash-bound human decision for the exact candidate and evidence bytes; make any change to cited inputs invalidate that decision.

## Forbidden changes

- Do not implement now solely on this workstream specification; no implementation authorization is implied.
- Do not invent a stable discover CLI, workflow command, evidence/decision schema, or selection/snapshot record before coordinator approval.
- Do not add source inspection, repository crawling, provider-history retrieval, model calls, semantic inference, or human identity authentication to Markitect Core.
- Do not silently include unselected or excluded files, flatten multiple repositories into one identity, remove counterexamples, or turn frequency in a selected sample into proof of intent.
- Do not place candidates in configured Areas or packages, render them as authoritative, change Domain definitions/pins/adapters/checks, or edit adopter policy as an effect of discovery.
- Do not treat an accepted dossier as adoption. The normal Project change and semantic review remain separate.

## Dependencies

- **Hard current boundary:** use only explicit fixed-snapshot evidence; source bytes are opaque inputs to Markitect; no implicit graph edges or semantic source parsing.
- **Hard future dependency:** coordinator-approved selection, snapshot, privacy, and workspace-handoff contract before durable implementation or schema publication.
- **Hard adoption dependency:** explicit human review of exact candidate/evidence bytes followed by a separate adoption decision in the owning Project.
- **Soft choices:** a model-assisted authoring agent, a separate validator helper, choice of dossier serialization, and whether multiple candidates share a ledger. None is a current CLI/API contract.
- **External:** evidence owner approval, redaction, retention, and adopter project authority are supplied by the relevant human/project process; a helper cannot authenticate them.

## Design questions

- What are the required identities for a repository, snapshot, ContextRun, exact selected path/bytes, tool, selection, compiled context, and final report?
- How are detected, selected, and excluded roots represented and reviewed without treating inventory as permission to read everything?
- How should duplicate evidence, conflicting examples, date drift, legacy/current status, missing counterevidence, and uncertain attribution be recorded?
- What candidate states and decision transitions are needed (accept, reject, defer, split, revise), and how is a decision invalidated by changed evidence?
- How are external reviews, tickets, or conversations explicitly exported, redacted, source-identified, and retained without fetching provider history or collecting unapproved personal information?
- What is the boundary between a generic evidence validator and project-owned semantic checks? Which formats remain illustrative until adoption?
- How are parallel researchers assigned separate source/evidence ownership and how are findings reconciled without losing contrary evidence?

## Required tests

After contract approval, test that:

- Every evidence record resolves to an approved repository/revision, exact selected path, matching byte hash, and selection/report identity.
- Changes to selected files, candidate text, evidence ledger, source revision, or selection invalidate the relevant compiled evidence and prior decision.
- Unselected/excluded roots do not appear in the dossier or context; multiple repositories retain distinct provenance.
- Duplicate and conflicting evidence remain visible and deterministic; the helper never decides which interpretation is true.
- Decision records bind exact bytes and supported status transitions, while output explicitly does not authenticate the reviewer or perform adoption.
- Running the workflow/helper changes no Project resource, package pin, Domain, adapter, check, generated output, or adopter source file.

## Required evidence

Retain the exact tool/build identity; owner-approved scope; per-repository immutable revision and snapshot identity; exact path and byte hashes; selection, ContextRun, context, and report digests as approved; ledger entries for support, counterexamples, duplicates, conflicts, and exclusions; candidate and decision byte hashes; and deterministic helper results. Record what was not inspected, redacted, or verified. Keep private or secret source material out of evidence artifacts unless separately authorized and governed; never persist credentials.

## Exit criteria

- Shared selection/snapshot/privacy/handoff contracts are approved before any durable schema or CLI work.
- The existing v0.12.0 authoring contract and example-only status are preserved or their changes receive explicit review.
- Discovery yields a provenance-preserving dossier and candidate queue without mutating canonical state or representing a candidate as authority.
- Tests establish fixed-input binding, decision invalidation, counterevidence retention, privacy/selection boundaries, and no automatic adoption.
- The adopter's owner separately reviews any accepted candidate before a normal Project change.

## Completion questions

- Can each statement be traced to exact selected evidence at an immutable source identity?
- Are support, counterexamples, duplicates, conflicts, legacy/current status, and uncertainty preserved?
- Does every candidate remain outside canonical state until a separate owner-approved change?
- Does the decision apply to the exact evidence and candidate bytes and become stale when either changes?
- Have privacy, exclusions, uninspected scope, and the limits of any validator/helper been recorded?
