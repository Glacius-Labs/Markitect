# Two-area operating pilot

This package preserves three separately frozen attempts for one synthetic shared Rule across two .NET areas and a shared Markdown parent. The Rule changes integer quantity bounds from 1..20 to 1..10. Each attempt has its own canonical commits, source bundle, manifest, and protocol. Manifest preparation statuses describe the moment of freezing, not current execution status. The package contains compact prior-result receipts and digest references, not raw actor payloads.

## Attempt history

Attempt 01 supplied the Orders and Fulfillment projection policies without required project-file paths and net10.0 constraints. The coordinator rejected the candidate at the explicit Apply review boundary. The controller returned planned; it did not reject. No candidate was applied, and verification and holdout did not run. The retained result attributes this stop to the fixture setup, not actor noncompliance.

Attempt 02 added explicit buildable .NET 10 project and handler paths, no external packages or sibling-project dependency, and public API/callback/effect boundaries. Apply materialized the five listed owned artifacts. Independent verification passed for Orders and failed for Fulfillment and the parent; controller audit was incomplete. The baseline behavior holdout passed 476 calls. Attempt 02 then stopped without repair because the Rule projection policy prohibited numeric duplication without distinguishing independent authored authority from faithful encoding of the canonical Rule in a target. This was classified as an ambiguous fixture contract, not actor unreliability. attempt-02-result.json records the outcome; attempts-01-02-evidence-index.json provides hashes for retained external evidence.

Attempt 03 corrects only the canonical Rule-to-.NET projection policy. It states that the canonical Rule is the sole authored source of quantity bounds, while each selected projection must faithfully encode those bounds in its representation. Constants, helper classes, and inline predicates are allowed; independent or conflicting bounds are not. The broader effect-boundary Rule remains. Runtime, checks, holdout bounds, and success criteria remained frozen; fresh actors were dispatched, and previous attempt responses were not reused. Attempt 03 is frozen as its own source/runtime/protocol attempt; its prior response payloads remain external. The baseline Verify phase had one bounded same-input retry: the first Orders result omitted four required scope references, Host rejected it and parent verification escalated. transport-retry-baseline-03.json records the unchanged-input retry; no source, runtime, checks, or prompt-template changes were made.

The changed-rule Execute phase also had one bounded same-input retry before Apply. The original transport-retry-changed-03.json blamed unsupported mode 0644. The immutable correction transport-retry-changed-03-attribution.json supersedes that attribution: 0644 on the handler was supported; the rejected Orders.csproj candidate omitted the required mode field, so Host decoded an empty mode and rejected the aggregate Execute. The original receipt is retained unchanged.

Attempt 03 completed the finite sequence, recorded in [result.json](result.json): baseline and changed-rule audits closed, external Orders drift was reconciled under unchanged intent, and an unowned probe made the read-only audit incomplete. After only that injected probe was quarantined, the final audit was complete with no findings (digest `e3bb208ef48d522295a2199e21fa9ec2e4270f7b0f16c0b97be68224dbb1f169`). This is completion after the retained corrections and bounded retries, not first-pass success.

## Prepare an exact source checkout

Requirements: Git and Python 3.9 or later. Choose a new absolute destination outside the Markitect checkout. The helper refuses an existing destination, never deletes or overwrites a path, resolves the selected bundle inside this package, verifies its SHA-256 before cloning, sets repository-local core.autocrlf=false, and verifies the full selected commit. It defaults to Attempt 03.

    python experiments/two-area-operating-pilot/prepare.py --destination C:/tmp/two-area-attempt-03-baseline
    python experiments/two-area-operating-pilot/prepare.py --attempt 3 --revision changed --destination C:/tmp/two-area-attempt-03-inspection
    python experiments/two-area-operating-pilot/prepare.py --attempt 2 --revision baseline --destination C:/tmp/two-area-attempt-02-inspection
    python experiments/two-area-operating-pilot/prepare.py --attempt 1 --revision baseline --destination C:/tmp/two-area-attempt-01-inspection

The Attempt 03 bundle prepares a clean canonical source checkout only. It does not contain the live Attempt 02 generated targets or controller ledger. The actual Attempt 03 continuation retains the existing external fixture, generated targets, and ledger, then advances that same fixture to Attempt 03's baseline source while keeping its runtime and ledger. Do not substitute a fresh prepare.py clone for that retained live state. In that existing checkout, use the manifest branch for the baseline source, then its changed branch after the baseline phase:

    git -C C:/tmp/two-area-live-fixture switch --create codex/two-area-pilot-attempt-03 4644f3b556e742d8d5d2bffd92b228c72c85e02a
    git -C C:/tmp/two-area-live-fixture switch --create codex/two-area-pilot-attempt-03-change 9dac557171a82039370d76f66abcbf2152c46e5a

A separately prepared changed checkout is for inspection only; it cannot retain prior outputs or ledger history. Runtime configuration, record store, queue, private logs, intermediate candidates, and raw holdout runs remain external. [result.json](result.json) records their privacy-safe digests and outcomes. The five actual final artifact files are copied byte-exactly under `final-artifacts/`; their hashes are in that result. These files are evidence samples, not a live adopting checkout.

## Transport and holdout

bridge.py is the frozen transport adapter for externally dispatched collaboration agents. It validates invocation and response echoes and preserves raw, normalized, and emitted response artifacts. The protocol bounds bridge wait to 540 seconds and the Host runner to 600 seconds. Dispatch metadata records configured gpt-6-luna / high selection; it is not provider attestation.

holdout/run_holdout.py checks both area implementations and their composition for the selected maximum (20 for baseline, 10 for changed). Run it from a disposable external copy so its source copies, build logs, and receipts stay outside this package. It requires .NET SDK 10.0.103 and isolated local restore settings. A holdout pass is technical behavior evidence; it does not replace independent leaf verification, parent composition verification, or controller audit.

## Scope and evidence status

This is a small synthetic pilot, not a reliability or productivity study. The bounded broad-rule review is not universal semantic proof. The collaboration transport is experimental; the captured native Codex CLI preflight rejected the requested Luna model under the available ChatGPT-account authentication. Provider version and token usage are unavailable, and prompt/history separation is not OS or filesystem isolation. Coordinator review and Apply remain part of the experiment.

Attempt 01 stopped before Apply; Attempt 02 stopped after failed verification. Attempt 03 closed all four declared stages. Its independent behavior checks recorded 476/0 failures at baseline, 316/0 after the shared change, 316 calls with 34 failed checks for the deliberately injected defect, and 316/0 after correction. Both areas and the parent obtained fresh verification. The drift Executor changed both Orders-owned files: it removed the defect and also stripped their final LF, including the otherwise unchanged project file. Fulfillment and Markdown bytes were preserved. An unnecessary evidence-refresh proposal was rejected without mutation because the Markdown materialization was already current; normal Verify subsequently refreshed parent assurance. This is ordinary drift reconciliation, not a typed persisted-finding repair.

The final unknown-artifact audit left source, artifact bytes, runtime, ledger, queue, and private logs unchanged. The exact injected file was moved to external quarantine, with no exclusion or model change. Final managed state and audit digest matched the completed drift state. Full product-source gates are separate commit-bound evidence.