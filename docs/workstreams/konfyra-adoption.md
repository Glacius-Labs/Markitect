# Konfyra consumer inventory workstream

This is a bounded consumer investigation specification, not authorization to change Konfyra or Markitect. Follow the instructions and authority boundaries in the actual Konfyra checkout. Read it with Markitect's [shared contracts](../development/shared-contracts.md) and [adoption boundaries](../development/adoption-boundaries.md). The coordinator owns shared selection, snapshot, and handoff design. Inventory investigation can proceed in parallel on approved, read-only scopes; no Markitect source implementation is assigned here.

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

Konfyra is a consumer project with its own taxonomy, policy, instructions, and delivery authority. Markitect's v0.12.0 input contract can bind exact UTF-8 paths and bytes, but treats them as opaque and does not derive Konfyra semantics. A ContextRun selection contributes to task context and digest without creating graph edges. Project-owned analysis belongs in explicitly governed project checks or adapters, which must not be activated by this investigation.

The intended outcome is an owner-scoped consumer inventory, coverage account, and report of observed adoption gaps. This workstream assesses what is present and what the declared inventory covers; it does not decide that a gap is a Markitect product requirement or authorize a policy cutover.

## Objective

Produce a bounded, evidence-linked inventory/report for an explicitly approved Konfyra scope. Distinguish detected, selected, and excluded roots; state coverage and blind spots; classify observations and gaps for human review; preserve the current Konfyra authorities and operating rules throughout.

## Scope

- This workstream owns read-only evidence gathering and inventory/reporting about the approved Konfyra checkout and task question.
- It may inspect in-scope project artifacts, instructions, tests, configuration, and declared contracts as allowed by Konfyra's own instructions and the explicit owner-approved scope.
- It does not own Markitect Init, Copy Me implementation, shared schema, or Markitect source files.
- It does not create a Konfyra policy, authority, implementation change, or product requirement. Any future change needs the authority of the project that owns it.
- Parallel investigators may cover disjoint approved roots or questions. Each must use a fixed source identity, record exclusions, and avoid writes to the consumer checkout or shared evidence files.

## Current implementation

There is no Markitect feature that inventories or semantically adopts Konfyra. Current Markitect `init` is for a new Project and refuses an existing `markitect.yaml`. Copy Me is authoring guidance over explicitly selected fixed inputs; it does not crawl repositories, call a model, authenticate review, or adopt a candidate. The existing consumer's own tooling and instructions govern this investigation; Markitect's artifact boundary remains opaque-input only.

## Owned subsystem

This workstream owns only a reviewable inventory/report in the explicitly assigned external dossier or investigation area. It owns no files under Markitect product source and no files in the Konfyra checkout. If durable storage location or permission to retain an inventory is not clear, record the findings in the assigned workstream report and ask the project owner before writing them into a consumer repository.

## Allowed changes

- During the authorized investigation: read only approved sources in the actual Konfyra checkout, following its instructions and scope; record fixed source identity, roots detected, roots selected, roots excluded, reasons, coverage denominator, and observed evidence.
- Produce an external, reviewable inventory and gap report with exact evidence pointers and uncertainty. Classify a gap as observed missing coverage, ambiguous ownership, conflicting authority, stale evidence, excluded/uninspected scope, or another evidence-backed category; do not convert the classification into policy.
- Use only public, non-secret, non-sensitive content that the owner has authorized for the stated purpose. Redact any accidentally encountered credential or personal data from notes and do not copy it into evidence.
- Ask the coordinator to route a generic product hypothesis to a separate product-design workstream only after Konfyra owner review; preserve project-specific details as Konfyra-owned.

## Forbidden changes

- Do not modify, stage, commit, publish, or otherwise write Markitect product source or the Konfyra checkout under this workstream.
- Do not change, replace, weaken, or claim adoption of Konfyra policy, AGENTS/provider instructions, docs, tests, configuration, architecture decisions, or delivery controls.
- Do not activate a Markitect package, Domain, check, adapter, ContextRun policy, or integration in Konfyra.
- Do not collect, display, log, or retain secrets, credentials, tokens, private customer data, or unapproved personal information. Do not use credentials as evidence.
- Do not scan beyond owner-approved roots, infer authority from file naming or README links, or treat absence from a selected inventory as proof of absence repository-wide.
- Do not publish the inventory or raw internal paths/evidence externally. Any public communication requires separate explicit authorization and review.

## Dependencies

- **Hard:** actual Konfyra checkout instructions and authority; explicit owner-approved question, source scope, and read-only boundary; fixed source identity for every inspected checkout/repository.
- **Hard shared dependency for future Markitect integration:** coordinator-approved selection, snapshot, privacy, and workspace-handoff contract. No such integration is part of this workstream.
- **Soft:** potential use of Markitect as an opaque exact-file inventory or later ContextRun input; external scripts or a report template; breadth beyond the initial approved roots. None is required to conduct a bounded read-only inventory.
- **External:** permission to retain, publish, or act on results remains with the Konfyra owner and relevant project governance. Inventory findings grant no authority.

## Design questions

- What exact consumer question is the inventory answering, and who owns the decision that the evidence is sufficient?
- Which roots may be detected, read, excluded, or reported? What sensitive/configuration/customer-data boundaries does Konfyra require?
- What fixed commit or snapshot identifies the checkout and any linked repositories? How are local uncommitted changes handled or excluded?
- What is the coverage denominator: directories, authoritative files, tests, resource declarations, policy surfaces, or another explicit set? How are generated, vendored, archived, and inaccessible files represented?
- Which findings are objective inventory facts, which are interpretations, and which are gaps requiring owner judgment? How will negative findings be limited to the inspected scope?
- Where may the report be retained, who may see it, and what evidence must be redacted or omitted?
- If an observed gap suggests generic Markitect work, how will that proposal be separated from the Konfyra-specific requirement and reviewed by both owners?

## Required tests

This is read-only investigation, so do not add product tests or alter either checkout. Before reporting, independently check that:

- Every inventory row maps to an in-scope path and fixed source identity; selected and excluded sets reconcile to the stated detected scope or list unresolved roots.
- The denominator and inclusion/exclusion rules are explicit and allow a reviewer to reproduce coverage counts.
- Each gap cites evidence and is labeled as fact, inference, uncertainty, or uninspected scope; no repository-wide absence claim exceeds inspected coverage.
- No secret, credential, private customer content, or unapproved personal information appears in the report or evidence bundle.
- The working trees remain unchanged and no adapter, policy, check, package, or Markitect integration was enabled.

## Required evidence

Record the approved question and scope; actual checkout identity and fixed commit/snapshot; detected, selected, and excluded roots with rationale; inventory method and coverage denominator; exact in-scope evidence pointers; gap classifications with confidence and limits; files or systems that could not be inspected; privacy/redaction review; and a read-only working-tree check. Do not bundle raw sensitive content. State explicitly that the report does not constitute policy approval, product acceptance, or authorization to implement.

## Exit criteria

- The Konfyra owner has approved the question, scope, and read-only evidence boundary.
- The inventory states fixed identities, detected/selected/excluded scope, denominator, coverage, and limitations.
- Observed gaps are traceable and separated from inference and proposed generic product hypotheses.
- Both Markitect and Konfyra working trees remain unchanged; no policy, authority, package, check, or adapter was activated.
- The result is retained only in an approved location and reviewed by the owner before any follow-on action.

## Completion questions

- Did the investigation follow the actual Konfyra instructions and explicit owner scope?
- Can the owner reproduce the inventory coverage and see excluded/uninspected areas?
- Are gaps evidence-backed and clearly separated from inferences and generic Markitect proposals?
- Were secrets, credentials, private data, and unapproved personal information excluded from retained evidence?
- Did the work leave Konfyra's existing authorities and both working trees unchanged?
