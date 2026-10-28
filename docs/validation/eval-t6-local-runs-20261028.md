# Evaluation suite T6: local judge and native-mode runs

**Status: run on 28 October 2026 ([results](#results)).** Bounded local evidence for step T6 of the [concept 2.0 build](../implementation-plan.md#concept-20-build): the new series judges in their containers, and the native arm mode with its scripted stand-in on this machine. No provider session runs and no model is used; the only provider contact is the escape probe's TLS handshake with the allowed endpoint, without a request, as in the [E1 smoke](eval-e1-sandbox-smoke-20261028.md). It is not a counted evaluation and no comparison is drawn from it.

## What runs

| Item | Value |
|---|---|
| Source | `tools/evaluation` and `tools/evaluation-judges` at `REV`, the commit named under [results](#results) on branch `dev/relaxed-chatterjee-15dcd3` |
| Judges | `tools/evaluation-judges/library` and `tools/evaluation-judges/makerspace`, each exported once from `REV` with `judges.py export --rev REV` |
| Judge image | `golang@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195` (`golang:1.27.1-bookworm`), already on this engine; nothing is pulled |
| Sandbox image | `sha256:e0b20d8a8c88a0a4b37f151af45618e764f8687b023b10bcb6d3b5148c5e5dad`, the E1 image, for the egress gateway and the stand-in's Run container |
| Series configuration | [`tools/evaluation/sandbox/series.json`](../../tools/evaluation/sandbox/series.json), as in E1 |
| Login | The stand-in uses none. The suite still requires a login file that outlives the run; the counted runs below used a dummy test login with no real token, because the owner's login on disk had expired (attempt 2) |
| Hidden material | Each native run names `EVAL\gridshare` and `EVAL\hidden` with `--hidden-absent`; both are absent |
| Engine lock | `C:\Users\Consiliari\AppData\Roaming\Claude\side-session-notes\local_cbc94143-9a8e-46c4-9c73-b1b58817893e\NATIVE-RUN.lock`, taken by every step that starts a container |
| Limits | Judges: each container's own limit from its `judge.json` (40 and 60 minutes). Native runs: a 600-second deadline each, the series' egress allow list |

`EVAL` is `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation` and `OUT` is `EVAL\t6-local`. `TMP` and `TEMP` are `C:\Users\Consiliari\AppData\Local\Temp`.

**(a) Library series self-test:** 12 judge containers (five references, seven mutants). **(b) Makerspace self-test:** 47 judge containers (ten references, six scope and drift mutants, 31 oracle-family mutants).

```text
python -I -B tools/evaluation/judges.py selftest --source tools/evaluation-judges/NAME --judge OUT\judge-NAME --out OUT\selftest-NAME --lock LOCK
```

Each passes when `selftest.json` says `"passed": true`.

**(c) Native stand-in on Windows:** four runs, each a gateway and one or two Run containers.

```text
python -I -B tools/evaluation/suite.py run --task OUT\task-MODE --model claude:claude-sonnet-5-5 --out OUT\native-MODE \
    --native SANDBOX --hidden-absent EVAL\gridshare --hidden-absent EVAL\hidden --lock LOCK \
    -- python -I -B tools/evaluation/native_standin.py --image SANDBOX [--probe]
```

| Run | Expected |
|---|---|
| `change` | Delivered, with `STANDIN.md` changed; its Run container on the internal network only; valid for counting; a result |
| `nothing` | Delivered with no change; valid; a result |
| `escape` | The second container on the engine's default network is flagged; not valid; invalid |
| `probe` | The escape probe as an agent in a Run container: every escape blocked, the allowed endpoint reached |

**Abort:** any step that fails to clean up, or leaves containers or networks with the evaluation label, stops the sequence; the operator removes them by label. **Cleanup:** the suite removes every container and network it or the stand-in started; outputs stay under `OUT`.

**Not run here:** the same native runs on WSL Linux. Docker Desktop's WSL integration is off for the Ubuntu distribution on this machine, so WSL has no engine; the offline tests ran in WSL against the fake engine.

## Results

Judges at `REV` = `28cd02cd05b3e01f892cdbd00388dc33e8921152` (the reviewed head; the judge files and `judges.py`/`judging.py` are unchanged at the final head). Native runs at `7f411598b234cfaec054d03af6b4b049606ac3ce`, which changes `native.py` only. Outputs under `OUT`; nothing was left on the engine (no container or network with an evaluation label afterwards).

| Item | Result |
|---|---|
| (a) Library series self-test | **Passed**, 12 of 12: every reference M1 to M5 passes all cases, its build, its tests, its scope and has no drift; each mutant fails where seeded. Judge folder digest `580bf049…` |
| (b) Makerspace self-test | **Passed**, 47 of 47: references W0 to C9, the six scope and drift mutants and all 31 oracle-family mutants (each fails a case of its family). Judge folder digest `fd5bccc9…` |
| (c) `change` | Delivered `STANDIN.md`; one Run container, on the internal network only; valid; a result |
| (c) `nothing` | Delivered with no changed path; valid; a result |
| (c) `escape` | The container on `bridge` is flagged as an escape; not valid; classified invalid |
| (c) `probe` | The escape probe in a Run container: `contained`, the allowed endpoint reached through the gateway (one connection), GitHub's server name refused; no result, as designed |

**Attempts, all kept under `OUT` and its siblings:**

1. At `8f830a20` (before the independent review): (a) 12 of 12 and (b) 47 of 47 passed; the four native runs behaved as expected. Superseded because the review changed the judges' accounting and the native check (`EVAL\t6-local-8f830a20`).
2. At `28cd02cd`, native runs with the owner's login: all four refused before starting anything, because the login on disk had expired. The suite worked as intended (`OUT\native-expired-login`).
3. At `28cd02cd`, native runs with the dummy login: the gateway was not recognized by its ID, because Docker 29's events carry the ID only in `Actor.ID`; it was counted as an arm container and removed before its log was read, so every run was invalid. Fixed in `7f411598`, whose fake engine now writes events the same way (`OUT\native-gateway-id-bug`).
4. At `7f411598`, the native runs above.

**Linux:** the offline tests (124, including the Library series self-test and the Makerspace sample through the fake engine) pass in a clean clone in WSL Ubuntu. The native runs on Linux were not run: WSL has no engine on this machine (Docker Desktop's WSL integration is off for Ubuntu).
