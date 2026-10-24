# A1 native Codex contract proof — 24 October 2026

One bounded native proof of the Run collaboration contract of [V1 concept 1.0](../concepts/v1-concept.md) decisions D10–D13, as required for PR A1 by the [delivery plan](../implementation-plan.md#steps). The coordinator dispatched it under the standing evidence authorization. It ran once and passed. An independent review of the retained files confirmed the result.

## Binding

| Item | Value |
| --- | --- |
| Source | `f76826c3b86104bd59344971cab92be1d93cb259` (A1 branch, contains Main `5169a7c9`), clean checkout |
| Image | `sha256:a9671e7c4dbcbc3a0f5f173d027070db2235d8da0d956fdb8abcdcba525741f5`, built once from that checkout's `packaging/runtime` with `MARKITECT_VERSION` set to the source SHA; `runtime.env` holds exactly the `codex`, `codexSchema` and `agentd` records |
| Provider | Codex App Server 0.162.0, `gpt-6-sol`, effort `high`, dedicated normal account login |
| Test | opt-in `TestRuntimeNativeHierarchy` in `internal/adapters/codexappserver`, one Implement Run with Root, `a`, grandchild `a/deep`, `b` and empty `b/empty` |
| Limits | 600 s work, 48 shared messages, 5 Managers, cleanup 60 s, persistence 10 s, `go test` 15 min, whole packet 2,100 s, one attempt |

## Result

The image build took 52 s of 600. The test step took 307 s of 900 (`go test` reported 304 s). The whole packet took 364 s. The Run completed technically with decision `submitted`, five handoffs with passing checks and all 12 closures confirmed. It used 23 of 48 messages: 14 native responses and 9 questions. The retained candidate is commit `316ce6548020aa6ef37df4f3c49d8a508c635e6a`, tree `7f273d58d8ab6d97fa5b029dc94d558b4416ce36`, read back from a fresh bare repository. Tracked-writer drain, Notes canaries and an unchanged source fixture were also confirmed.

The four required interactions appear in the ordered stream of delivered messages and tool calls:

1. **Answer without a new judgment.** `b/empty`, `b`, `a` and `a/deep` each received messages carrying only a question. Each replied and ended without calling `judge`; their fixed handoffs stayed unchanged.
2. **Question answered before the first judgment.** Root asked `a` and `a` asked Root before either replied. Root replied before its first judgment, and so did `a`. No judgment was demanded.
3. **Judgment newly required after a change.** `a/deep` judged while its tracked writer still ran. After the terminal drain, Run sent "your working state changed after your judgment: e4b33559… became 333902b2…", and `a/deep` judged again.
4. **Invalid child candidate, then parent correction.** `a` received `a/deep`'s handoff `333902b2…` with a failing check (`out/draft-note.txt` unexplained). It rejected it with Findings and delegated a correction. It then received `3cd7bc07…` with a passing check and approved it. `a` rejected directly, so the refused approval of an invalid candidate did not occur natively.

**No-progress rule.** After the state-changed message, `a/deep`'s next response only called `await`, which was refused because nothing could arrive. The following message therefore reported no progress. `a/deep` then asked its parent a question, which reset the count, received one plain continuation and judged. The rule warned once and did not stop the Run; `ErrNoProgress` was not triggered natively.

**Residue and login.** Baseline and final censuses of containers and volumes with the packet's label were empty. The Run, loader and home volumes were absent. The login fingerprint was unchanged. No credential material appears in the retained transcripts or logs.

**Packet defect.** The controller's `files.sha256` hashed itself while it was still being written, so its own entry does not match. All 70 data files match. This is a defect of the controller script, not a change to the evidence.

## Limits

- **One attempt.** One fixture, one provider and model; there is no repetition evidence.
- **Not exercised natively:** declared project checks (the fixture has none), late findings and model blockers, refused approval of an invalid candidate, and refused sibling or level-skipping questions. These have controlled-session evidence only.
- **Not covered:** Assessment and Verify, Apply, publication, account refresh and other providers.
- **Later changes not proven natively.** Commits after `f76826c3` change only a code comment and the wording of Run refusals and diagnostics, which now name a next step (concept §3.7). That wording is not natively proven. Step 4 of the delivery plan requalifies the integrated source.

## Retained evidence

The packet lives on the execution host under `Documents/Codex/Markitect-Native-Contract-Proof/f76826c3`. It holds:

- the controller `run-packet.sh` with its SHA-256, plus `config.json`, `runtime.yaml` and the runtime instruction;
- `attempt-01/`, with per-step controller records, logs, the image ID, `runtime.env`, login fingerprints and `files.sha256`;
- `attempt-01/native-evidence/`, with `hierarchy.json`, `resources.jsonl`, redacted transcripts, Run state and the candidate bundle, and `readback.git`.

`package-result.json` has SHA-256 `3a21b592ddc92048d0a8f2b19ceb88aaf26d329cbd03b0d560595a57aadb4eb6`, and `native-evidence/hierarchy.json` has `4adabaa89e824f6e059f32dd9c0c78c56d1eb4aa8a0259c87a0eae16e796ebdf`.
