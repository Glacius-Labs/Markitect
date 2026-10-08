# Copy Me evidence review and candidate queue: earlier design proposal

**Status:** Historical design proposal. The current source has a `copy-me` command and bounded record validation outside Core. The canonical source contract and workspace boundary are in [Selective adoption handoff](selective-adoption-handoff.md); CLI invocation is documented in [Usage](../usage.md). Implementation verification is in progress; no CI or publication claim is made here. Published v0.12.0 remains unchanged.

## Earlier decision request

The original request was for a bounded read-only validator outside Markitect Core. Its former suggestion that no CLI was needed is superseded: source now supplies an explicit `copy-me` command over the externally prepared handoff. The example's `engineering-discovery-evidence/v1alpha1` and `engineering-discovery-decision/v1alpha1` formats remain example-local, not the new shared record contract.

Copy Me consumes permission scope; it does not select, broaden, or infer it. The handoff must contain explicit stable repository IDs, full immutable commit IDs, exact selected paths, snapshot identities and byte hashes. It must not provide a discovered root list or an implicit path scan. A name or owner field is supplied evidence, not authenticated identity or authority.

## Minimum handoff fields

The coordinator-owned handoff should be immutable input to the validator and keep each repository as a separate source identity. Its minimum fields are:

- A handoff format/version, stable handoff ID and digest, owner-supplied scope/approval reference, purpose, relevant period, allowed evidence types, exclusions, privacy/redaction constraints, and retention boundary. The helper records these supplied values and cannot authenticate their source.
- For each repository: owner-supplied stable repository ID, canonical locator when allowed, full Git commit ID, snapshot identity/digest, and the exact selected paths. Every path has an explicit inclusion reason and expected byte hash (and file mode if the snapshot contract makes mode material). Excluded paths/roots are listed when supplied; nothing is inferred from omission.
- If the owner explicitly supplies a Markitect ContextRun: exact manifest path/hash, selection digest, Markitect tool version and executable/build digest, completed report identity/digest, and compiled context digest. These are conditional fields, not prerequisites: selected opaque artifacts can be reviewed for a repository with no Markitect Project or ContextRun. Repository runs remain separate; the outer handoff links them without flattening identities.
- A bounded coverage record over the owner-defined questions or selected items: examined, no evidence found, unavailable, redacted/withheld, or uninspected, with reason where applicable. This record makes missing coverage visible but does not imply completeness of an unenumerated repository.

The selection digest must bind the repository identity and revision, selected exact paths and expected byte hashes, and the applicable run manifest when one is supplied. Snapshot, report, and context identities bind their own recorded bytes/results when present. The coordinator hands the validator only the exact selected byte blobs (or a coordinator-approved selective byte source) identified by repository, revision, and path. The validator compares those bytes with the handoff hashes; it must not call `app.Load(root, revision)`, materialize the whole repository, or require canonical Project state. It does not acquire Git state, read beyond the handoff, or prove the handoff was authorized.

## Evidence, candidate, and decision records

Keep three records with separate purposes. Observations record what selected evidence contains. Candidates interpret observations. Decisions record a reviewer-supplied disposition. A candidate is never canonical policy.

An evidence ledger entry minimally identifies an evidence ID, repository/run and selected-path reference, exact source-byte hash, line/range or other stable anchor, evidence stance (`supports`, `counterexample`, `qualifies`, or an explicitly approved equivalent), and an observation note. Excerpts are optional and may appear only when the owner-approved privacy contract permits them; an excerpt never substitutes for source identity and hash. Date/period and selection reason are retained where material. Every candidate reference must resolve to an entry. The ledger records duplicate and conflict relationships as links between evidence IDs; it never removes duplicates, resolves conflicts, or selects the “best” interpretation.

A candidate has a stable candidate ID and exact candidate-byte digest, proposed rule, intended scope and conditions, classification, explicit support/counterexample/qualifying evidence IDs, confidence with its basis, alternatives, uncertainty, and unanswered questions. Frequency claims include the selected-sample numerator, denominator, inclusion rule, and time window; they describe only that sample. Missing counterevidence and incomplete coverage remain explicit. The validator checks references and required shape only; it cannot judge the interpretation or truth of a proposed rule.

Each decision is a separate immutable record for one exact candidate revision. It records the supplied reviewer identity, date, rationale, scope, status (`accept`, `reject`, `defer`, `split`, or `revise`), candidate byte digest, full evidence/queue byte digest, and handoff digest. If status `split` or `revise` points to follow-on candidate IDs, the validator may check that those IDs exist in the same dossier; it cannot decide whether the split or revision is adequate. A later review appends a new decision instead of rewriting the old one. Any changed selected input, handoff, ledger/queue, or candidate bytes makes the prior decision stale. A valid digest proves byte correspondence only: it neither authenticates the reviewer nor approves adoption.

## Ownership and test proposal from the earlier review

After coordinator approval, assign this isolated Go package and its package-local tests:

- `internal/copyme/handoff.go` — typed view of the approved owner-supplied handoff and selected-byte inputs; no source acquisition or scope selection.
- `internal/copyme/evidence.go` — evidence, coverage, duplicate, and conflict reference validation.
- `internal/copyme/candidate.go` — candidate shape and evidence-reference validation.
- `internal/copyme/decision.go` — allowed disposition values and exact-byte freshness binding.
- `internal/copyme/validate.go` — deterministic orchestration and diagnostics over explicit bytes supplied by the caller.
- `internal/copyme/validate_test.go` — focused structural and byte-binding cases.

Keep any later runnable end-to-end fixture under `examples/copy-me-review/`, with a clearly synthetic claim and separate positive and safety cases. Do not change the existing `examples/engineering-discovery/` format in this slice unless the coordinator explicitly assigns that migration. The package must not import `internal/core`, `internal/app`, or adopter source packages; it does not add root schemas, Project dependencies, public CLI/API, generated views, or a Core operator.

The focused test set should prove that the validator rejects an unknown repository/run/path, abbreviated revision, selected bytes that differ from an expected path hash, mismatched snapshot/selection digests, and, when a ContextRun is supplied, mismatched manifest/report/context identities. It also rejects dangling evidence references, missing required support/counterexample linkage when the approved contract requires both, dropped duplicate/conflict or coverage entries, and stale candidate/evidence/handoff bytes. It should accept an opaque selected-artifact handoff with no Project or ContextRun, preserve conflicts and uncertainty in deterministic output, and show the reviewer identity is unauthenticated. A read-only fixture test should show candidate review leaves any supplied Project resources, Areas, pins, Domains, adapters, checks, and generated outputs unchanged. These are safety properties, not evidence that an interpretation is accurate or useful.

## Existing evidence and claim limits

The shipped Copy Me Workflow already calls for owner-agreed scope before reading, per-repository full revisions and ContextRuns, exact path hashes, support and counterexamples, explicit frequency denominators, separation of observation and interpretation, uncertainty/alternatives/questions, an external staging dossier, and a human decision over candidate and ledger hashes. The example helper demonstrates a single synthetic repository's byte-binding checks and explicitly does not authenticate its synthetic `accepted` reviewer or adopt policy. Its success is a format/safety demonstration, not discovery-quality evidence.

The real-project adoption pilot provides a separate bounded safety case: one owner-supplied task and committed Copy Me ContextRun selected exact artifacts, including counterevidence; the selected source files stayed opaque; one narrow assisted change built successfully with no runtime or database calls. The record states that model/token telemetry was not instrumented. It does not establish Copy Me interpretation accuracy, broad coverage, lower effort, productivity benefit, or general adoption. Keep synthetic helper success, this single assisted implementation, and any future human review outcomes as distinct evidence classes.

## Boundaries retained in the implemented source contract

The implemented source slice still excludes source crawling, provider-history retrieval, model calls, secret/personal-data collection, implicit root discovery, semantic source-code interpretation in Core, authentication, and adoption. The explicit `copy-me` command validates supplied bytes; it does not publish a new Project schema or automatically activate packages/dependencies. A separate owner-reviewed Project change remains the only adoption path. This historical proposal is superseded by [Selective adoption handoff](selective-adoption-handoff.md) and the current [Copy Me workstream](../workstreams/copy-me.md) wherever details differ.
