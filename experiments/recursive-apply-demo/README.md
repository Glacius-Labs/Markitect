# Recursive Apply library demonstration

This harness runs the real library demonstration of the [implemented recursive Apply](../../docs/design/canonical-model/manager-apply-runtime.md): `markitect apply` on Library M1, the owner's one model change (`maxOpenLoans` 5 → 3), and `markitect apply` on M2 against the actual A1, all inside one hard wall-clock limit. The product orchestrates every Agent turn, check, capture and publication. The scripts build, bind, project the target repository, perform the scripted owner step (commit A1 unchanged, change only the limit), enforce the time limit and clean up. Two scripts also shape Agent input: `ra_bind.sh` writes the Host profile's system prompt, and `ra_runconfig.py` supplies the owner request. No run binds owner decisions: the two boundary decisions that earlier runs bound (unknown fields, due dates after 9999-12-31) are part of the Library model since 20 October 2026 ([DEC-063](../../docs/concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)). The [validation record](../../docs/validation/recursive-apply-demonstration-20261019.md) reports the run of 19 October 2026.

Agent-produced code, transcripts and credentials never enter this repository. Run directories live under `MARKITECT_DEMO_WORK` (default `~/mk-ra`).

## Requirements

Linux (the record used WSL Ubuntu) with Docker, Go and Python 3 on `PATH`, and a dedicated Claude runtime token as `CLAUDE_CODE_OAUTH_TOKEN` in the file named by `MARKITECT_RUNTIME_ENV` (default `~/.markitect/runtime.env`). The scripts never print the token.

## Steps

| Step | Script | Timed |
| --- | --- | --- |
| Build the runtime and project Go images and the CLI from one commit | `ra_build.sh SHA BUILD_DIR` | no |
| Bind an attempt: Host profiles (Sonnet `claude-sonnet-5-5`, high effort, for every Execute and Verify turn), check profiles, token file, frozen scripts and their digests | `ra_bind.sh BUILD_DIR ATTEMPT_DIR` | no |
| Run the demonstration under `timeout` (27.5 min, then cleanup within 30 min) | `ATTEMPT_DIR/frozen/ra_demo.sh ATTEMPT_DIR SRC_DIR` | yes |
| Afterwards, outside the process and the limit: the independent library oracle assesses the published A1 and A2 (`ra_demo.sh` calls it) | `ra_assess.sh ATTEMPT_DIR` | no |
| Summarize evidence; approximate write attribution from transcripts | `analyze.py STATE_DIR REPORT_JSON`, `writes.py STATE_DIR...` | no |

`ra_demo.sh` calls `ra_inner.sh`, which projects the target (`ra_project.sh`: M1 model, dependencies and `library-execution.yaml` only), writes each run configuration (`ra_runconfig.py`, with the owner request and an absolute deadline), runs both Applies and makes the scripted owner commits between them. `ra_cleanup.sh` stops every remaining container. `smoke_apply.sh` (one Apply with a short deadline) and `nochange_apply.sh` (Apply again on the published A2, committed by the script) are separate checks, not part of the timed run.

## Extract on the library

`ra_extract.sh ATTEMPT_DIR SRC_DIR SOURCE_REPO [rebuild|norebuild]` runs the [Extract](../../docs/design/canonical-model/manager-apply-runtime.md#extract-first-bounded-implementation) of case 3 on a library realization, for example the A2 of an earlier demonstration attempt. `ra_extract_project.sh` copies only `realization/` from `SOURCE_REPO`, plus the imported vocabulary, a manifest without local Definitions and `extract-execution.yaml` (layout root `realization`, no exceptions, no required checks; a source may carry its own `extract-execution.yaml`); nothing of the original model is copied. `markitect extract` then runs under `timeout` (27.5 min) while `rx_owner.sh` answers owner questions by script: a question about the loan period gets the scenario's decision (21 days), any other a neutral answer. Outside that limit the script adopts the published model and registry with a scripted commit, standing in for the owner's adoption. With `rebuild`, `rx_rebuild_project.sh` projects a case-1 target from the adopted model alone (no artifacts, no registry; the documented build and test checks), `markitect apply` rebuilds the realization and the independent oracle judges it with the M2 policy. The rebuild uses the documented library checks of `library-execution.yaml`, not the checks an Extract proposed. No Extract finding is passed to Apply; that handoff is not implemented.

The sources of the bench: `plain` is a published A2; `rx_seed.py` plants a known defect against clear documented intent and a loan-period conflict (`seeded`), or the conflict alone between two documents that claim no decision (`conflict`). `extract-checklist.md` is the fixed rubric an independent assessor scores each extracted model with; `rx_collect.py` summarizes the attempts. The second project is Readinglog: its source is a Git repository holding `experiments/test-projects/readinglog/template/realization` as `realization/`, scored with `extract-checklist-readinglog.md`. After binding, `rx_python_profile.py ATTEMPT_DIR IMAGE` puts the agent profile on the Python example runtime image (`packaging/runtime/examples/python`, built on the attempt's base image) and offers the check profile `readinglog-python`; the run uses `norebuild`. `MARKITECT_STORE` gives an attempt its own container label (`ra_bind.sh`), which the run configuration's workspace helper and `ra_cleanup.sh` follow, so parallel sessions never stop each other's containers. `MARKITECT_AGENT_MODEL` selects another Claude model for every role.

## Project controls and the oracle

`library-execution.yaml` declares the persistent layout (root `realization`, exception `realization/cmd/library` → `library`), the budget (20 minutes per Apply) and two required checks: `go build ./cmd/library` and `go test ./...`. With the layout, ownership is enforced: each Execute turn can write only its own areas and registry shard.

The independent library oracle of `experiments/library-playground` is not part of the Apply ([DEC-059](../../docs/concepts/register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle)). It observes once after each Apply, through `ra_assess.sh` and the hardened assessment harness of `experiments/runtime-baseline`, and its results never reach the Agents. The [first record](../../docs/validation/recursive-apply-demonstration-20261019.md) ran it as a required check; that setup is retired.
