# Copy Me discovery workstream

**Status:** current source adds `copy-me` record validation outside Core and a common externally prepared handoff. Source verification is in progress; this is not a CI or release claim. Published v0.12.0 remains unchanged. This page preserves the original workstream rationale; [Selective adoption handoff](../design/selective-adoption-handoff.md) is the canonical source contract. See [shared contracts](../development/shared-contracts.md) and [adoption boundaries](../development/adoption-boundaries.md).

## Context

The Copy Me authoring Workflow shipped in v0.11.0 and remains in published v0.12.0. Current source adds an explicit `copy-me` command backed by `internal/copyme`, which validates supplied handoff/queue/decision data without changing the author's Project or adopting candidates.

The Copy Me authoring Workflow remains provider-neutral guidance for human-led interpretation of explicitly selected evidence. The new CLI validates caller-supplied bytes; it does not crawl/discover evidence, call a model, or add semantics to the deterministic core. The synthetic `examples/engineering-discovery/` helper and its `engineering-discovery-evidence/v1alpha1` and `engineering-discovery-decision/v1alpha1` files remain example-only formats, not the new shared record contract or published schemas.

The bounded implementation separates captured observations, candidate interpretations, and supplied decisions over fixed inputs. A candidate remains outside canonical Project state and is not an authority; adoption requires a separate owner-reviewed Project change.

## Objective

Maintain reviewability of Copy Me records over the common owner-supplied handoff. Preserve provenance, counterevidence, uncertainty, and decision byte bindings. Keep interpretation/model assistance outside Markitect Core and adoption as a separate, explicit owner-reviewed Project change.

## Scope

- This workstream owns Copy Me authoring guidance and bounded record validation in `internal/copyme`; common selection/capture and workspace handoff are owned by `internal/adoption` and the preparation packages.
- Candidate and decision records are validated structurally and by exact byte/digest references, not judged for truth or completeness.
- It does not own adopter-specific policy. Konfyra observations remain inputs to consumer-owned review, not generic Core rules.
- Parallel analysis is permitted on separately assigned immutable evidence selections. Shared candidate/evidence/decision records require a single owner or serialized merge with preserved provenance.

## Current implementation

The published authoring workflow remains `internal/authoring/resources/workflow-engineering-discovery.yaml`; related guidance is in `workflow-authoring-change.yaml` and `workflow-constitution-change.yaml`. Current source adds the separate CLI/package `cmd/markitect/adoption.go` and `internal/copyme`. Canonical handoff/record semantics are in [Selective adoption handoff](../design/selective-adoption-handoff.md); Usage is being updated by the coordinator. Neither the CLI nor supplied reviewer claims create an automatic mutation request.

## Owned subsystem

The current bounded ownership is Copy Me authoring guidance and `internal/copyme` record validation. `internal/adoption`, selective Git acquisition, app orchestration, and CLI dispatch own the shared preparation/handoff side. Core/Project schema changes, adopters' source files, and actual candidate adoption remain outside this slice.

## Allowed changes

- Maintain authoring guidance against the implemented fixed-input contract without implying model analysis or automatic discovery.
- Keep candidate and decision data in the external preparation workspace; validate exact supplied bytes and record bindings.
- Keep discovery output in an external dossier or isolated staging area outside configured Areas, package contents, generated projections, and canonical consumer files.
- Preserve observations separately from interpretations; include supporting evidence, counterexamples, duplicates, conflicts, date/period, scope, confidence basis, alternatives, and unanswered questions as the approved contract requires.
- Require an explicit, hash-bound human decision for the exact candidate and evidence bytes; make any change to cited inputs invalidate that decision.

## Forbidden changes

- Do not treat this workstream page as the current command or record-format reference; use the canonical design and Usage.
- Do not broaden scope, authenticate reviewer identity, interpret source code in Core, or make a candidate canonical.
- Do not add source inspection, repository crawling, provider-history retrieval, model calls, semantic inference, or human identity authentication to Markitect Core.
- Do not silently include unselected or excluded files, flatten multiple repositories into one identity, remove counterexamples, or turn frequency in a selected sample into proof of intent.
- Do not place candidates in configured Areas or packages, render them as authoritative, change Domain definitions/pins/adapters/checks, or edit adopter policy as an effect of discovery.
- Do not treat an accepted dossier as adoption. The normal Project change and semantic review remain separate.

## Dependencies

- **Hard current boundary:** use only explicit fixed-snapshot evidence; source bytes are opaque inputs to Markitect; no implicit graph edges or semantic source parsing.
- **Implemented source dependency:** fixed scope, selective capture, privacy claims, and workspace handoff follow the shared [design](../design/selective-adoption-handoff.md).
- **Hard adoption dependency:** explicit human review of exact candidate/evidence bytes followed by a separate adoption decision in the owning Project.
- **Out of scope for this source slice:** model-assisted synthesis or provider calls. The CLI and closed records exist in source; optional authoring UX beyond them would require a separate product decision. The current contract keeps repository identities distinct and validates supplied queue/decision data.
- **External:** evidence owner approval, redaction, retention, and adopter project authority are supplied by the relevant human/project process; a helper cannot authenticate them.

## Questions retained from the earlier proposal

The former proposal raised the questions below before the shared record contract was set. Current command and record behavior is defined in [Selective adoption handoff](../design/selective-adoption-handoff.md); this historical checklist does not reopen those contracts.

- What are the required identities for a repository, snapshot, ContextRun, exact selected path/bytes, tool, selection, compiled context, and final report?
- How are detected, selected, and excluded roots represented and reviewed without treating inventory as permission to read everything?
- How should duplicate evidence, conflicting examples, date drift, legacy/current status, missing counterevidence, and uncertain attribution be recorded?
- What candidate states and decision transitions are needed (accept, reject, defer, split, revise), and how is a decision invalidated by changed evidence?
- How are external reviews, tickets, or conversations explicitly exported, redacted, source-identified, and retained without fetching provider history or collecting unapproved personal information?
- What is the boundary between a generic evidence validator and project-owned semantic checks? Which formats remain illustrative until adoption?
- How are parallel researchers assigned separate source/evidence ownership and how are findings reconciled without losing contrary evidence?

## Verification expectations

Candidate validation should establish that:

- Every evidence record resolves to an approved repository/revision, exact selected path, matching byte hash, and selection/report identity.
- Changes to selected files, candidate text, evidence ledger, source revision, or selection invalidate the relevant compiled evidence and prior decision.
- Unselected/excluded roots do not appear in the dossier or context; multiple repositories retain distinct provenance.
- Duplicate and conflicting evidence remain visible and deterministic; the helper never decides which interpretation is true.
- Decision records bind exact bytes and supported status transitions, while output explicitly does not authenticate the reviewer or perform adoption.
- Running the workflow/helper changes no Project resource, package pin, Domain, adapter, check, generated output, or adopter source file.

## Required evidence

Retain the exact tool/build identity; owner-approved scope; per-repository immutable revision and snapshot identity; exact path and byte hashes; selection, ContextRun, context, and report digests as approved; ledger entries for support, counterexamples, duplicates, conflicts, and exclusions; candidate and decision byte hashes; and deterministic helper results. Record what was not inspected, redacted, or verified. Keep private or secret source material out of evidence artifacts unless separately authorized and governed; never persist credentials.

## Exit criteria

- The exact candidate passes focused and required repository/CI gates; evidence is recorded separately from this workstream summary.
- Supplied records remain bound to exact handoff/queue bytes, and the CLI performs no discovery, model call, authentication, or adoption.
- Published v0.12.0 remains unchanged; a later release requires separate immutable publication evidence.
- An adopter's owner separately reviews any candidate before a normal Project change.

## Completion questions

- Can each statement be traced to exact selected evidence at an immutable source identity?
- Are support, counterexamples, duplicates, conflicts, legacy/current status, and uncertainty preserved?
- Does every candidate remain outside canonical state until a separate owner-approved change?
- Does the decision apply to the exact evidence and candidate bytes and become stale when either changes?
- Have privacy, exclusions, uninspected scope, and the limits of any validator/helper been recorded?
