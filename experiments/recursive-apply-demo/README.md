# Recursive Apply library demonstration

This harness runs the real library demonstration of the [implemented recursive Apply](../../docs/design/canonical-model/manager-apply-runtime.md): `markitect apply` on Library M1, the owner's one model change (`maxOpenLoans` 5 → 3), and `markitect apply` on M2 against the actual A1, all inside one hard wall-clock limit. The product orchestrates every Agent turn, check, capture and publication; the scripts only build, bind, project the target repository, perform the owner's commit and model change, enforce the time limit and clean up. The [validation record](../../docs/validation/recursive-apply-demonstration-20261019.md) reports the run of 19 October 2026.

Agent-produced code, transcripts and credentials never enter this repository. Run directories live under `MARKITECT_DEMO_WORK` (default `~/mk-ra`).

## Requirements

Linux (the record used WSL Ubuntu) with Docker, Go and Python 3 on `PATH`, and a dedicated Claude runtime token as `CLAUDE_CODE_OAUTH_TOKEN` in the file named by `MARKITECT_RUNTIME_ENV` (default `~/.markitect/runtime.env`). The scripts never print the token.

## Steps

| Step | Script | Timed |
| --- | --- | --- |
| Build the runtime, project Go and oracle check images and the CLI from one commit | `ra_build.sh SHA BUILD_DIR` | no |
| Bind an attempt: Host profiles (Sonnet `claude-sonnet-5-5`, high effort, for every Execute and Verify turn), check profiles, token file, frozen scripts and their digests | `ra_bind.sh BUILD_DIR ATTEMPT_DIR` | no |
| Run the demonstration under `timeout` (27.5 min, then cleanup within 30 min) | `ATTEMPT_DIR/frozen/ra_demo.sh ATTEMPT_DIR SRC_DIR` | yes |
| Assess A1 and A2 with the independent oracle | `ra_assess.sh ATTEMPT_DIR` | no |
| Summarize evidence; approximate write attribution from transcripts | `analyze.py STATE_DIR REPORT_JSON`, `writes.py STATE_DIR...` | no |

`ra_demo.sh` calls `ra_inner.sh`, which projects the target (`ra_project.sh`: M1 model, dependencies and `library-execution.yaml` only), writes each run configuration (`ra_runconfig.py`, with the bound `owner-decisions.json` and an absolute deadline), runs both Applies and makes the owner's two commits between them. `ra_cleanup.sh` stops every remaining container. `smoke_apply.sh` (one Apply with a short deadline) and `nochange_apply.sh` (Apply again on the accepted A2) are separate checks, not part of the timed run.

## Project controls and the oracle check

`library-execution.yaml` declares the persistent layout (root `realization`, exception `realization/cmd/library` → `library`), the budget (20 minutes per Apply) and two required checks: `go test ./...` and the independent library oracle. The oracle runs as a check image built from `oracle/` and the evaluator files of `experiments/library-playground`, outside every Agent's write area. `oracle/check.py` selects the m1 or m2 profile from the bound loan policy in the candidate and prints only case IDs and pass/fail.
