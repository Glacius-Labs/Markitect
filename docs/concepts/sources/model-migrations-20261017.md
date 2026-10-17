# Model restructuring and migration — 17 October 2026

Recorder's note: direct owner discussion in the Codex chat **Commands**, thread `01a14ac0-d4c1-7cb0-a070-d677f0033b45`, following the [Optimize workflow](commands-extract-optimize-20261017.md#optimize-workflow-clarification). Owner quotations retain their original wording. The assistant summary below identifies the proposal accepted by the final documentation request; it is not verbatim owner text. The [command design](../../design/canonical-model/commands.md#model-restructuring-as-an-explicit-migration) owns the current workflow. No implementation or runtime evidence was produced.

## Owner direction

> Lass uns die Modellumbauten genauer durchgehen. Optimize sollte auch Packages verschieben oder zusammenlegen können, solange die Bedeutung erhalten bleibt. Wie machen wir das mit Identitäten, Managern und Registry sauber?

## Proposed workflow

Recorder's summary of the assistant response:

- Treat the operation as an explicit model migration, jointly reviewing model changes and necessary registry/configuration bindings while leaving realization artifacts unchanged.
- Keep v1 identities as Package path plus local ID. Record an exact old-to-new mapping and migrate imports, typed references, body links and Package scopes. No new global identity layer, inferred renames or hidden redirects. Package merging initially relocates Definitions; collisions do not authorize collapsing their meanings.
- Preserve obligations, subjects, conditions, exceptions and relationship meanings. A scope that would broaden after a move requires another structure or a decision.
- Prepare and review against the existing organization; derive the future tree and activate it only after adoption with fresh bindings. Retain old task and review history. Do not silently combine incompatible Manager settings, rights or budgets.
- Transfer artifact accountability and migrate registry references without requiring artifact path moves. Every artifact keeps one accountable owner. A reviewed move with a new justified basis must not be confused with removed intent and cause accidental deletion.
- Review and validate the complete candidate, then assess unchanged realizations against its exact model. Retain known deviations. Necessary registry and configuration migration is an explicit extension for this future model operation, not permission for current artifact Apply to mutate protected inputs or repair artifacts.

## Owner acceptance and Main publication

> Klingt gut. Halte diesen Ablauf für Modellumbauten inklusive Identitätsmigration, Manager-Neuaufbau und Registry-/Konfigurationsanpassungen in den Docs auf main fest. Weiterhin keine Implementierung.

Recorder's interpretation: this accepts the described migration direction and authorizes the bounded documentation delivery on Main. It resolves the earlier open question of whether Optimize may prepare Package moves and merges when meaning is preserved. It does not change the canonical language identity rules, add an execution API or implementation assignment, waive independent review or authorize provider, Docker, Apply or study runs. Exact staging, publication and recovery formats remain future producer work.
