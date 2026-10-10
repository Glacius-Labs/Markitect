# A01 native smoke

Validation record, 10 October 2026. This AI-generated summary was made from the retired Product Readiness native acceptance ledger and integration progress log. The originals, with every pin, digest, session and turn ID and the per-attempt accounting, are in git history:

```text
git show 4852f6d7:docs/work-items/product-readiness/evidence/native-acceptance-ledger.yaml
git show 4852f6d7:docs/work-items/product-readiness/integration-progress-20261009.md
```

## Result

Attempt 28 passed the combined A01 native smoke on Markitect source `cdd30b0efc540f151404dabe86b022275dc40d83`. Its batch was `a01-apply-clean-filter-ae0942d9a6954a759666cb99730a3d67`, and it ended in state `applied` at 2026-10-10T12:10:13Z.

| Input | Value |
|---|---|
| Markitect source and validated docs | `cdd30b0efc540f151404dabe86b022275dc40d83` |
| Binary SHA-256 | `d56376cfeba3b9c9a1292557ea005a0430ff81f3576e5d2f1d7a2d33cdbbc0ad` |
| Provider | Codex CLI 0.162.0 App Server, model `gpt-6-luna`, effort `high` |
| Platform | Windows, build 26100, with the MXC sandbox backend and the packaged Codex app paths |
| Project | An owned, disposable Python greeting project at HEAD `275baa9b830f1eee67cc901321a595002ffe0744` |
| Model | Three Managers (Root, Source, Docs), three reviewers and one Verifier |
| Plan | `ac57fa274cf4a21dc8c872cff257ab72`, SHA-256 `7fc506a435fd38519f450914a1b5096d43ead643b30f9260de182b3747431cd4` |
| Readiness and runtime digests | `ee0420a136539cd4925bfd762172957dfdc82a03d60f59912cb55a23033ecde5`; `0b0bea059d135cf436833a5a8327ddef1d95581cd8483911739e039ded14b9f7` |
| Profiles and runtime limits | Writers `:workspace`, reviewers and Verifier `:read-only`, child approval `never`, one hour per role (the product's configured ceiling), 128 MiB event budget |
| Exercise limits per batch | Four hours, 256 start requests including failed and nested ones. The owner's renewal ("Budget darfst du solange machen bis es klappt") removed any cumulative cap, but each batch was pinned and finite. |

What the run showed:

- **Setup and planning:** readiness was positive, and a normal Plan bound a short Work Item and model change.
- **Parallel Managers:** the independent Source and Docs Managers overlapped for 34.4 seconds.
- **Helper:** one genuine fresh Host helper wrote `tests/test_greeting.py` (683 bytes, mode 100644). Its delivery was applied and closed, and its bytes matched the source and the integrated candidate.
- **Independent review:** the Root and Source work reviews passed. Docs passed after one ordinary rework. A separate Root integration review passed.
- **Recovery:** the original Root integration turn completed once. It was resumed on the same binary without a replay and with zero recovery-inspection starts.
- **Full Verify:** passed. This covered the configured check `python -B -m unittest discover -s tests -v`, the initial Verifier, and the Manager audits: Docs 3 of 3 subjects, Source 5 of 5, Root 10 of 10.
- **Guarded Apply:** wrote exactly `README.md`, `docs/greeting.md`, `docs/markitect/project.md`, `src/greeting.py` and `tests/test_greeting.py`. Disk hashes and modes matched the verified candidate.
- **Accounting:** 15 requests (14 role invocations and 1 helper) and 15 provider starts, at about 1.77 million estimated micros. This is a partial lower bound: helper cost was not itemized.

## What it establishes and what not

The pass shows that the core native mechanisms completed one small end-to-end journey on that exact source, binary, platform and project. It covers setup, Plan, parallel Managers, a real helper, independent review, no-replay recovery, Full Verify and guarded Apply.

It does not establish:

- **Acceptance or benefit:** no human semantic acceptance and no productivity or cost benefit.
- **Reliability:** it was the first pass after 27 failed or blocked attempts, each on a newly repaired source. No repeat run exists.
- **Later sources:** none was re-run, including Main `f12ffb00`, where PR #89 merged on 10 October 2026, and anything after it. Hosted CI on the PR #89 head (run [38057729908](https://github.com/Glacius-Labs/Markitect/actions/runs/38057729908)) is separate evidence.
- **Other platforms:** no Linux or other platform was run. The roadmap now makes Linux the first platform (register DEC-013).
- **Sandbox and process limits:** the effective MXC backend identity is unknown. The run is not an exhaustive process census, not an OS sandbox, and not a hard cap on native helper starts.
- **Deferred capabilities:** read-only helper results (backlog RUN-04) and the broader A02 and A03 trials were deferred and never ran.

## Attempts and failure classes

A01 took 28 attempts, from 2026-10-09T20:27Z to 2026-10-10T12:10Z. Attempts 1 to 27 failed or were blocked. Each failure was diagnosed and repaired in source before a fresh, fully pinned batch. No turn was replayed, and no Host guard was weakened. Before A01, a separate earlier proof lease had ended after two CLI failures before any provider start; those are not counted here.

Across the series, cumulative partial accounting was 184 combined requests, 180 provider starts and about 25.06 million estimated micros. This is a lower bound, not an invoice.

| Failure class | Attempts | Typical cause | Repair |
|---|---|---|---|
| Plan and run binding | 1, 15 (Resume) | Plan bound transport-neutral fingerprints, while Run checked the transport fingerprint. A rebuilt binary changes the project digest, so Resume across builds is stale. | Plan binds the transport fingerprint; Resume only on the same binary |
| Permissions and sandbox | 2, 5, 13, 15 | Inherited `:read-only` with `on-request` approval. MXC known-folder lookup failures. Shell writes denied under the packaged `LocalCache` path. | Explicit `:workspace` for writers, child approval `never`, native editor for writes |
| Windows path alias | 2, 3 | Git reported `…\LocalCache\Local` where the journal held `AppData\Local`, and lexical path equality rejected it | `os.SameFile` fallback for the Git-reported candidate directory only |
| Strict output contract | 4, 6, 9, 10, 19, 22 | Wrong DTO shapes, copied digest and base64 content, references the request never supplied, typed fail report without guidance, paraphrased Verifier subjects | Output schemas on `turn/start`, Host-supplied references, short aliases, prompt guidance |
| Review grounding and scope | 11, 12, 13, 14 | Findings outside the exact candidate scope, `pass` with findings, prose grounding without an exact token, missing helper metadata | Reviewer guidance and schema corrections |
| Helper contract and timing | 7, 8, 17, 18, 20 | Helper scope outside the parent's paths. Reviewer adapter given a tool handler without specs. Parent edited while a helper call was pending. Helper ended unknown. A read-only helper returned nothing. | Scope-subset rule, serialized helper calls, file-authoring-only contract |
| Coverage census race | 18 | Repository census changed during the operational-path metadata recheck | Narrow operational-path recheck |
| Recovery identity | 15, 20 | A task-ID-only journal lookup matched three historical journals. A Windows journal append hit a sharing violation. | Bind the exact run ID and input digest |
| Verify evidence and limits | 16, 21, 23, 24 | Unsupported or truncated subjects, missing briefings, the 16 MiB event limit, audits without integration-review evidence | Full-audit scope fixes, a 128 MiB event budget, direct-parent integration reviews supplied to audits |
| Rework order | 26 | Parent integration review re-ran before queued child rework, until the review limit was exhausted | Queued child rework runs first |
| CRLF bytes | 25, 27 | LF Git bytes against CRLF worktree bytes in helper assertions and Apply preflight | Comparison through the Git clean filter; exercised by attempt 28 |

The [Product Readiness lessons survey](../work-items/surveys/product-readiness-lessons-20261010.md) turns these causes into known limits per area and names the backlog packages they inform.
