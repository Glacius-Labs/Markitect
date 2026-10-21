# Apply acceptance after the bounded Codex retest

## Owner adoption, 21 October 2026

The owner discussed why Implement had closed successfully while the known Depot defect remained, then accepted the proposed process correction:

> Das passt. Halte die Änderungen am Apply-Prozess auf Main fest und gib mir einen Prompt für eine neue Codex-Session, die das umsetzt und mit gezielten Nachtests prüft.

Source: the planning chat `Apply-Ablauf nachvollziehen`, following the completed three-attempt Codex retest. The following is a recorder summary of the accepted proposal, not a further verbatim owner quotation:

- Cover material conditions and interactions inside Definitions, rather than treating complete Definition IDs as semantic completeness.
- Require parent-owned integration evidence, including for a Root with no directly owned Definitions.
- Bind Implement acceptance to adequate evidence for its behavioral claims at the actual public entry points. A green suite or a correct internal function alone cannot establish those claims.
- Have fresh Verify reviewers derive coverage and counterexamples independently from the model, including valid inputs the implementation might wrongly reject.
- Compare a separately adjudicated conforming reference with a faulty counterpart; settle the disputed reference obligations before claiming optional changes or preservation failures.
- Reserve finite execution time for independent Verify and safe cleanup. More time alone does not repair missing semantic coverage.

The [workflow](../../design/canonical-model/manager-apply-workflow.md#evidence-backed-acceptance-after-the-codex-retest) owns the accepted target. The [new Codex handoff](../../development/codex-apply-conformance-handoff-20261021.md) defines implementation, targeted checks and an explicit startup-prompt grant. This documentation starts no provider run and does not replenish the completed series.

## Observed basis and limits

The [source-bound retest record](../../validation/apply-codex-conformance-20261021.md) separates the successful model-gap stop from the two incomplete workflows. R03's Implement verdict was satisfied, but the faulty CLI file remained unchanged even in its retained candidate. Verify did not finish; nothing was published. Independent offline review found the necessity of other changes partly ambiguous. This adoption does not classify all additional work as optional or turn incomplete Verify into a false Verified result.
