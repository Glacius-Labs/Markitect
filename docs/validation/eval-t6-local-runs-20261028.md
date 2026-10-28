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
| Login | The owner's Claude subscription login, copied per run without its refresh token; the stand-in does not use it |
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

To be filled in after the runs.
