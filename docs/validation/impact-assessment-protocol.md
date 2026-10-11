# KG-04 impact recall and width protocol

**Status: PREPARED / NOT RUN.** This is a setup protocol, not a result or an acceptance report. No expected must-change list has been authored or frozen, no impact comparison has been run, and no product threshold is set. No model change is enacted by this document.

The direct user delegation on 11 October 2026 authorizes this setup with the Shop example and Markitect itself, an independent pre-result must-change author, and descriptive per-case reporting. It does not supply a measurement, a pass threshold, or product acceptance. The existing KG-04 backlog entry is unchanged by this document.

At setup on 11 October 2026, the observed Markitect Main was `7f1fde0df4519f8205a45e5d5c917df7d58d5cbd`; this is the pinned tool/source revision for both proposed cases. Main has since advanced, but a later commit does not silently replace either case snapshot or this tool pin.

## Cases and pinned baselines

| Case | Target repository baseline | Proposed single model change | Expected must-change files |
|---|---|---|---|
| Shop cancellation | Standalone Shop fixture commit `069f0c1035b5e1c730539af323d10eac9a268c9e`; its root tree is `d7f5224423e7d8105ebfce3f583e51b6342c890a`, exactly the `examples/project-world` subtree from Markitect Main `7f1fde0df4519f8205a45e5d5c917df7d58d5cbd`. The Statement owner is Manager `commerce.sales.orders/orders`. | In the Orders-owned Statement `commerce.sales.orders/cancel-order`, edit only `spec.description` to require trimming the supplied reason, rejecting it if empty after trimming or longer than 160 Unicode code points, and persisting the trimmed value with the order cancellation in the same transaction that releases the reservation. Existing shipment, reservation-count, and rollback constraints remain. | **NOT YET CREATED OR FROZEN.** |
| Markitect relationship effects | Markitect repository at Main `7f1fde0df4519f8205a45e5d5c917df7d58d5cbd`. The Statement owner is Manager `product/product`. | In the product-owned Statement `product/explicit-relationships`, edit only `spec.description` to add: “For Statements, `uses` adds the target to context and Manager ownership routing; `requires` adds the target obligations and declared Artifact/Check coverage to impact.” Keep its existing statements about prose, links, paths, fixed-snapshot results, conservative unknown impact, and canonical output ownership. These are the current modeled relation effects, not a proposal to change runtime behavior. | **NOT YET CREATED OR FROZEN.** |

The Shop target is a standalone Git repository seeded from the named source subtree because the example is nested inside Markitect and its guide requires an isolated repository for Git-backed project commands. The local preparation copy is `C:\Users\Consiliari\.codex\kg04\shop-case`, with no remote configured; it was clean at setup and had the seed commit and root tree shown above. Its initial commit is a local, provider-free setup artifact and does not modify the model. This host path is provenance for the prepared copy, not a portable fixture location. Before any use, independently verify the repository identity, clean status, full `HEAD` SHA, root tree SHA, and equality to the named source subtree; if that prepared copy is unavailable, re-create and re-register the target snapshot before proceeding. No checks, tests, impact commands, or agent runs were executed in that target repository. Both table entries are proposed scenario changes only; neither asserts newly accepted product intent.

The two cases stay separate. Each has its own BASE, candidate revision, expected list, impact output, metrics, and limitations. The Markitect tool/source revision remains the same full SHA for both. If either target changes before its comparison, discard the prepared record and re-register the new full BASE; never rebase an expected list silently.

## Roles and freeze order

Four responsibilities stay separate:

1. **Target custodian** supplies the exact BASE, the task wording, and the one-file model-change candidate. The custodian does not author or revise the expected list.
2. **Independent assessor** has read-only access to the exact target BASE and task wording. The assessor may inspect source, model, and repository files, but does not change the target, execute its tests or code, or invoke a provider. Before seeing any impact, explanation, plan, briefing, or affected-Manager output, the assessor writes an exact repository-relative list of files that must change to fulfill the task. The assessor is not the candidate author, target custodian, impact operator, or person integrating implementation work.
3. **Impact operator** verifies the frozen-record hash and target BASE, then runs the fixed-source Markitect CLI against the exact BASE and candidate. The operator does not alter the expected list.
4. **Recorder** compares the sealed list with the saved impact output and records the prescribed counts and every path. The recorder does not move paths between sets to improve a score.

Before any impact command is run, the assessor's record must contain the case ID, exact task text, target repository identity, full BASE SHA, proposed model change, the non-empty list of exact file paths, the assessor identity, and a freeze timestamp. Store the record outside the target repository so it cannot change BASE. Serialize paths as UTF-8, repository-relative, slash-separated paths; use one exact file per entry, not directory globs. Mark each expected path as either present in the BASE file inventory or a planned new file absent from BASE. The assessor must name a planned new file exactly; directory globs are not allowed.

Seal the exact record bytes with SHA-256 and preserve the hash, full BASE SHA, assessor identity, and timestamp in an access-controlled, append-only audit log before impact output exists. The log must provide a trustworthy timestamp or an independent witness to the freeze; a Git author date or filesystem modification time alone is not proof of ordering. Before running impact, the operator verifies the log entry and recomputes SHA-256 over the preserved bytes. Preserve the audit-log reference and raw list with the result. A hash proves byte identity, not authorship or time. If the sequence or identity cannot be established independently, report the case as invalid and do not calculate metrics.

The assessor may inspect the baseline source and accepted model to form an independent expectation, but must not access Markitect's impact output or derived scope before freezing. Any accidental exposure invalidates that list; do not patch it after exposure. Select a new independent assessor and freeze a new record before any later run.

## Fixed comparison procedure

For each case, prepare one committed candidate that changes only the named Statement's `spec.description` and the deterministic project view regenerated from that model. Do not change implementation files, other model resources, artifact paths, checks, runtime settings, or ownership. Record its full candidate SHA. The must-change list covers downstream realization files only. Track the canonical model input and its generated view as **control files**: `.markitect/model/commerce/sales/orders/cancel-order.yaml` and `docs/markitect/project.md` for Shop; `.markitect/model/product/explicit-relationships.yaml` and `docs/markitect/project.md` for Markitect. If either control file appears in impact, report it separately as a control inclusion; retain it in complete width, but exclude it from downstream excess. If structural check fails or the candidate contains any other change, mark the case invalid and stop.

After the list is sealed, use the pinned Markitect source revision to run `check` on BASE and candidate, then run `impact` and `impact --explain` with the exact full BASE and candidate revisions. Save the command lines, tool/source SHA, stdout, stderr, exit codes, impact digest, explanation digest, and SHA-256 of each raw output. Do not invoke a provider, `plan`, `run`, or `verify`; this comparison is provider-free and does not test an implementation.

Use the impact's complete file set. From `impact --explain`, select `kind=file` elements and retain each `change` or `context` class; neither class is dropped from width, and class counts are over file elements only. Freeze the BASE and candidate Git file inventories with their SHA-256 digests before impact. Validate every expected existing path against BASE; add each exact assessor-frozen planned new path to a separate, hash-bound expected-new-path inventory. For matching, interpret a reported trailing-slash path as a directory prefix and expand it against the union of BASE, candidate, and expected-new-path inventories. A path without a trailing slash is an exact file path and contributes only if it is present in that union. Thus, a planned new file contributes to `I` and width when an exact impact path names it or an impacted directory prefix contains it. The candidate inventory also ensures a regenerated control view absent from BASE remains countable. Preserve the inventories, their digests, raw impact paths, and expanded set so normalization can be audited. Any path that cannot be mapped to these inventories is recorded as unmatched; do not silently count it in width or recall, and mark the case invalid if it is needed to interpret the frozen expectation.

For each case, let `E` be the sealed expected must-change file set and `I` the normalized file set covered by Markitect's impact paths. Report:

- **Recall:** `|E ∩ I| / |E|`, with numerator and denominator. `E` must be non-empty; otherwise the case is invalid and recall is undefined.
- **Missed files:** the exact paths in `E − I`.
- **Width:** `|I|`, the count of distinct concrete repository file paths included by the impact after the fixed directory-prefix expansion.
- **Control files:** any paths from the case's fixed control-file set that occur in `I`; these remain included in width and are reported separately.
- **Downstream excess files:** the exact paths in `I − E − C`, and their count, where `C` is the fixed control-file set for that case. This avoids describing the required canonical input and deterministic view as excess while retaining the complete impact width.

Also report the raw number of impact file entries, the `change` and `context` counts, the `unknown` entries and findings, and every matching or expansion rule applied. Do not combine the two cases into a single score. Report only observations; there is no pass/fail threshold, aggregate success claim, or product-acceptance conclusion.

If either command fails, returns an invalid or incomplete result, reports unknown impact, or cannot be bound to the exact revisions, preserve that evidence and label the case accordingly. Do not interpret an empty result as zero impact or as success. Do not repair the candidate or expected list after the impact output is known.

## Evidence and limits

The final record, if a later authorized run occurs, must bind each case's full BASE and candidate SHAs, the pinned tool/source SHA, the exact model change, the pre-impact expected-list record and hash, the BASE and candidate inventory digests, the expected-new-path inventory, the raw outputs and hashes, the formulas, and the per-case path lists. This protocol does not claim that the two tasks represent all model changes, that a manually authored list is universal ground truth, or that recall predicts semantic correctness, repository quality, productivity, cost, or product acceptance.

Current case state remains **PREPARED / NOT RUN**: both must-change lists are absent and unfrozen, there are no impact outputs, no thresholds, and no product conclusions.
