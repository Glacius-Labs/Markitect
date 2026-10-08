# Real-agent operating-model proof protocol

Protocol ID: operating-model-proof/v1.

Status: frozen before the first real-agent trial. This protocol evaluates fresh behavior from the accepted Markitect operating model. Source commit, fixture commit, runtime configuration, and real Executor/Verifier receipts are separate identities and must all be recorded.

## Historical evidence retained

The protocol-v1 checkpoint remains **C5 FAIL for targeted acquisition**. Its two runs, raw logs, manifests, fixture digests, and classifications remain immutable. C1–C4 and C6–C13 fresh trials remain **NOT RUN** wherever the existing [capability matrix](../../docs/research/capability-matrix.md) says so. The frozen checkpoint's C5 three-projection trial remains NOT RUN. This protocol creates new run IDs; it never edits, upgrades, or replaces prior evidence. Every prior FAIL, BLOCKED, INVALID, and NOT RUN stays visible alongside later outcomes.

## Frozen setup and authority boundary

- Product source for the first trial is integrated source e348607c1ad4aaf7dfef118b77e0885ce8ae021f, or a later full SHA frozen before any invocation. Record Markitect CLI version, source SHA, executable SHA-256 and build command.
- Use a private Git clone of the public examples/operating-model fixture only. Record fixture origin SHA, exact fixture HEAD, Git object format, line endings, and every source/Module/Schema/ProjectionPolicy input digest. Never use Konfyra, another private adopter, or an unapproved holdout.
- Give every independent trial a fresh fixture. Commit all fixture inputs; keep the source tree clean; bind base, candidate, and evidence revisions to real full Git commit IDs. Commit every mutation. Never synthesize revisions, model outputs, ProjectionRecords, VerificationResults, or receipts.
- Use a fresh Executor process/session and a separately fresh Verifier process/session. Bind each invocation to canonical revision, exact scope, ProjectionPolicy bytes and IDs, target context, Module pin/version, owned artifacts and runtime. The Verifier receives independently selected obligations and artifact bytes, without Executor transcript, reasoning or executor-authored tests.
- Initial provider binding: Codex CLI 0.130.0, model gpt-5.5, reasoning effort high, Python 3.13, and internal/tooling/codexrunner/runner.py with the native Windows x64 codex.exe, identified by SHA-256. Record literal command/arguments and config digests. Verify actual executable size and digest; the native executable is about 235 MiB and the complete runtime-file set must fit the configured 256 MiB ceiling. A missing/mismatched provider is BLOCKED or INVALID; never substitute a fake actor.
- Use fresh RecordStore and private-log paths outside the fixture, its Git common directory and target roots. Raw Codex JSONL, stderr, prompts, candidate bytes and private logs never enter the repository. Persist only sanitized receipts containing actual digests and observed metrics; provider usage fields are included only when actually returned.
- Apply requires the exact reviewed-run digest and explicit controller-apply --write. Verification is a separate invocation. Technical evidence never means owner acceptance.

## Frozen claims and controls

Claim wording follows the existing matrix. Run every control under a distinct ID and bind it to exact revisions. A control never substitutes for a positive trial.

| Claim | Positive trial | Required control / classification |
|---|---|---|
| C3 — source-ontology-agnostic projection | Add a previously unseen source Schema/Kind with sufficient target policy; obtain a real bounded Executor candidate, independent verification, and selected project checks. | Test absent policy and syntactically valid but insufficient/contradictory policy. Both must escalate before materialization. A plausible candidate does not rescue either control. |
| C4 — reconcile-first operation | Commit canonical intent, then run proposal → Executor → reviewed apply → independent verification; record convergence and ledger selection. | Stale plan, changed source after execute, or failed verifier remains visible and cannot count as convergence. Keep partial/incomplete attempts. |
| C5 — minimal impact | Change only Orders intent/policy. Measure Orders and Billing leaf work separately, each selected representation, and conservative/global freshness separately. | Include Billing-only and unchanged-representation controls. Count proposal, invocation, write, and evidence-refresh separately. Preserve the historical targeted-acquisition FAIL. |
| C8 — drift repair | Keep canonical intent byte-identical, mutate one owned target artifact in a fresh fixture, then propose/execute/review/apply/verify repair. | Add an unowned file under configured roots and an explicit exclusion. Report unknown and excluded paths distinctly; neither can be silently overwritten/deleted or imply full-repository coverage. |
| C9 — recursive verification | Run leaf and parent scopes with each scope's own implementation and checks. | The required trial needs leaves to pass while a parent-owned composition check fails. The current three-scope fixture does not supply this: its parent is deterministic Markdown, while its composition check invokes both leaves. A wrong leaf that fails parent behavior while passing leaf checks would also violate leaf canonical meaning. Thus the current arrangement cannot prove C9; keep C9 BLOCKED until a separate real parent-owned composition implementation/control exists. Do not weaken C9 or use this fixture's leaf mutation as semantic proof. |
| C10 — Executor / Verifier independence | Use fresh independent pairs with separately selected obligations and artifact bytes. | Challenge with controlled wrong candidates: one defect caught by deterministic checks and one requiring semantic judgment. Measure misses and correlated failures. Verifier never sees Executor transcript. Preserve every failed, incomplete, invalid or escalated attempt. |

Order is fixed: C3 controls and positive; C4; C5 Orders/Billing comparison; C8; C9 only after its blocker is resolved; C10. A shared setup is permitted only for byte-identical inputs, mutation, claim and runtime; it does not merge outcomes. Any blocked/not-run trial remains recorded as such.

## Measurements and classifications

Record only observable quantities:

- Markitect source/CLI SHA and version; fixture provenance and full revisions; repository identity; config and canonical input digests; Module pins; runtime-file digests/sizes; Executor/Verifier fingerprints; Projection/Definition/scope IDs; target preimage digest/mode; proposal, run, plan, candidate, ledger and verification digests; and actual record/result IDs.
- For each fresh invocation: role/session ID, context bytes/digest, supplied artifact bytes/digest, output bytes/digest, wall time, retries, diagnostics, tool calls, and provider-reported tokens when actually supplied. Missing telemetry is unavailable, never zero.
- Keep Orders, Billing and other projection work separate from no-op decisions, escalations, changed files/bytes, unknowns, exclusions, fixed checks, parent checks, stale-plan refusals, repository inventory, and global/conservative freshness requirements.
- Time human review only under a defined direct timing protocol. Do not infer attention from elapsed automation time. Report monetary cost only from an authoritative receipt/invoice mapped to the exact invocation; tokens alone are not currency.
- Each record includes claim, setup, protocol ID, source/runtime, full fixture revisions, mutation, expectation, controls, exact commands and exit codes, observation, metrics, paths touched, evidence hashes, human interventions, status, limitation and follow-up.

Statuses are PASS, PARTIAL, FAIL, BLOCKED, INVALID or NOT RUN. INVALID evidence remains and a corrected attempt gets a new ID. PARTIAL identifies the unproven subclaim. Ownership, checks, executor receipt, safe writes, passing children, verifier independence, semantics, acceptance and economic benefit remain separate claims.

The driver writes a sanitized summary under this experiment directory. Full CLI output and candidate payload needed for review/apply stay in a separate external run directory; provider logs stay in the external private-log directory. Do not commit those private materials.

## Explicit CLI contract

The source-bound harness invokes only these actions:

- markitect canonical --action controller-propose --repo FIXTURE --config CONFIG --runtime RUNTIME --base FULL_BASE --revision FULL_CANDIDATE
- markitect canonical --action controller-execute --repo FIXTURE --config CONFIG --runtime RUNTIME --base FULL_BASE --revision FULL_CANDIDATE
- markitect canonical --action controller-apply --repo FIXTURE --config CONFIG --runtime RUNTIME --base FULL_BASE --revision FULL_CANDIDATE --plan REVIEWED_RUN --expect RUN_DIGEST --write
- markitect canonical --action controller-verify --repo FIXTURE --config CONFIG --runtime RUNTIME --base FULL_SOURCE --revision FULL_EVIDENCE [--write]

Proposal and execute are read-only. Apply requires explicit write intent and the saved reviewed-run digest. Verify uses --base for canonical source and --revision for immutable evidence; it appends verification only with --write. The harness fails closed on mismatched options, malformed output or missing bindings. It never emulates an action or fabricates output.

## Amendments

This file is frozen after the first trial. Any change to claims, controls, runtime, fixture semantics, metrics or classification creates the next protocol version with date and reason. Existing run records remain bound to their original version.