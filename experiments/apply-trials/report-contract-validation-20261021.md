# Offline report-contract correction, 21 October 2026

This separately re-evaluates existing inputs. It preserves [A's historical record](../../docs/validation/session-a-b0-b1-runs-20261021.md), its ledger and old scores. No product attempt, Agent, provider, oracle execution or hosted CI was started. A owns the production report/stdout contract; this change owns only B's evaluator and associated regression/evidence files.

## Source and input bindings

Integrated base: `23591c4a5ff0038d87630c71cd0f3ad445202fe2`. Composed with Main `3fd86624a00563e6dfb73a4c9aa87dd70fbb730d` (#578/#579). Corrected evaluator source: `596d3f3252330b12dd50638744674d6f2709b695`.

The secret-free WSL archive `~/session-a/evidence/b0b1-attempts-20261021.tar.gz` has SHA256 `038d314634e191c5ed6e5de6027fe8daf80d73bdee12d2d87e70f3ddfa4ab75e`. Its `SHA256SUMS` verified the archive, ledger and build bindings. [The committed projections](report-contract-extracts.json) identify original archive members and full digests, including the separately persisted `report.json` where present. They retain actual values for the fields used here. Negative mutations in [the tests](test_report_contract.py) are synthetic controls, not product attempts.

| Existing stdout report (`ATTEMPT/out/apply.json`) | SHA256 |
| --- | --- |
| `SA-B0-SN-2` (#3) | `86f9f04450df51d22f450fee0322ae75f0fc1278dd36c5f13879ffa1802ebbbb` |
| `SA-B0-SQV-2` (#4) | `75b4a541e367fd13926bb9df0807cb83fcbd92fe59ec377336b6c19284944206` |
| `SA-B0-SN-3` (#5) | `5f660b89214d53f3c5b0679858c2cd6ae03254f823f899b3e6f3fb98e418b827` |
| `SA-B1-SN-1` (#6) | `29350c8b1f568a2be945417c96fbb60e0d434f802223a3eddad68043f18b71dc` |
| `SA-B1-SQV-1` (#7) | `45cc74f92fe08e4bded9ba58fadd96f31264c4ec80b0b69949687b73275a8dcb` |

Full originals, extracted historical sources, reproduction outputs and new offline outputs are outside the repository at `C:\Users\Consiliari\.codex\evidence\markitect-b-report-contract-20261021`. Scoring uses zero published changed paths: #4/#7's historical assessments report none, #5 records `no_change`, and #3/#6 never published. The four retained Run repositories were also observed clean. This establishes no source-oracle conclusion.

## Two reproduced gaps

The actual B0/B1 kit's `assess.py` matches the file in both product commits `b9cdd6d893fa9487a09016a3520c3c3442660c90` and `fd94e365f9c1c060c9419be67b70ca072b6c1128`: SHA256 `2753c03ce5797209ebaf8f446c83fdf745e67a77e02ab4c3580d831702fffd77`. B's integrated #564 head `86fd6586253215f30ce02ae863aa89fe1d10d817` has `assess.py` SHA256 `d7609205a5c458262a68d2ed15cd8337493c9e678452c350cdb1ce70b3658c15`. Both reproduce both failures on the bound originals using only `summarize_report` and `judge_outcome`:

1. #4/#7 have a sealed review assessment, Root's accepted `needs_correction` result and open `model_gap` findings on its candidate. `assess.py` reports `met: false`, `defectNamed: true`: its reason-prefix list predates `ModelAssessmentReason`. #564's `evaluate.model_stop` already recognizes this wording but does not require those structured bindings.
2. #5 raises `AttributeError: 'str' object has no attribute 'get'`. Registry diagnostics use a string `Source`; file/settings diagnostics use an object with `File`. The parser also fails on #3/#6. #5 retains an earlier failed preparation attempt before its later passed preparation; both must remain visible without confusing the final preparation status.

## Correction and separate outcome re-evaluation

Both evaluators now share one stop recognizer. Assessment requires the producer's terminal conclusion, successful preparation, a sealed review assessment with Root's accepted correction verdict, and an open genuine model finding bound to a reviewed candidate and named in that conclusion. Reassessment can confirm an earlier finding on that same candidate. DEC-064's Root abort remains supported through its matching structured Root/model/abort problem. Text prefixes, technical Incomplete/timeout/capture errors and unsupported old conclusions alone cannot establish current closed-Run success. DEC-063's reason-only Incomplete ending and old owner-question expectations remain historical. External owner questions, any publication (including `no_change` on a gap stop) or changed paths fail the stop criterion. Case scoring also requires the expected correction class and defect terms.

Independent review found that the old 400-character problem preview also broke strict legacy Root-abort matching. Structured problem text is now retained in full; a regression covers a valid statement longer than 500 characters. The no-Run #3 projection also retains the original absence of a `run` key.

Diagnostic formatting accepts both actual Source forms. `preparationDiagnostics` retains complete records, attempt indexes, diagnostic kinds and unfamiliar shapes; only the readable `preparation` preview remains capped at 2,000 characters.

| Input | Old kit / integrated #564 `assess.py` | Corrected outcome-only result |
| --- | --- | --- |
| #4, #7 S-QV | `met: false`, defect named | `met: true`; A4-gap outcome `met / justified-stop` |
| #5 S-N | Parser exception | `met: true`, Verified / `no_change` |
| #3 infrastructure, #6 capture | Parser exception | `met: false`; no model-stop success |

Corrected `assess.py` SHA256: `2fdf877f88d0170c60e15f3eb3bf9fd75f53243927bdaf6928fcb86c525b37d3`; `evaluate.py`: `c3396f7e1409a983e34090d780d45d8b6eed59d0fb47e940d4fbd4d36d8098f3`. These results exercise parsing and outcome scoring only. They do not rerun `assess()`/`evaluate()`, artifact tests, behavioral oracles, event/process evaluation or providers, and establish no new complete scenario result or product acceptance.

## Local validation

From the repository root:

```text
python -B -m unittest discover -s experiments/apply-trials -p test_report_contract.py
python -B experiments/apply-trials/freeze.py check
go run ./cmd/markitect check --repo .
```

The 13 focused tests pass: actual positive/technical reports, missing or mismatched Pass/candidate/Root/finding bindings, disputes, unsupported classes, publication, external questions with/without answers, real diagnostic shapes and retention beyond the preview limit. Full-original and projected inputs produce equal stop/case outcomes and diagnostic records. The refreshed lock changes only evaluator bindings: all scenario/control versions, budgets, packet heads and inputs remain unchanged. Independent complete-diff review and final integration/read-back are recorded in the delivery PR/chat, separately from these local observations.
