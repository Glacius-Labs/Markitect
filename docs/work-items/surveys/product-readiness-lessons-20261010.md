# Survey: Product Readiness lessons

AI-generated distillation, 10 October 2026, against main `4852f6d7`. Input for RUN-02 to RUN-04, MCP-02, TEST-01 to TEST-05, CI-01 to CI-05, BUG-01, AGENT-02, ARCH-01, ARCH-07, PLAY-07 and IDEA-01.

It condenses two sets of retired records:

- **Product Readiness:** the program guide, P01 to P11, the P03, P04 and P05 handoffs, the 9 to 10 October integration progress log and the native acceptance ledger.
- **3 October preparation:** the workstream specifications, and the parallelizability audit, risk register and baseline under `docs/development/`.

It keeps known technical limits, open defects and recurring failure causes. It does not repeat the attempt-by-attempt evidence.

These are inputs, not decisions. Facts are as recorded on their sources up to Main `f12ffb00`. Items marked *(on main)* were spot-checked on `4852f6d7`; check the rest against current main before relying on them. The originals stay in git history, for example `git show 4852f6d7:docs/work-items/product-readiness/integration-progress-20261009.md`. Other results live elsewhere:

- the A01 result: the [A01 validation record](../../validation/a01-native-smoke-20261010.md);
- the R01 to R03 research: the [research page](../../research/optional-research-r01-r03-20261009.md).

## What the program left on main

PR #89 merged the program at `f12ffb00` on 10 October 2026. It delivered:

- shared application operations for CLI and MCP;
- private Git candidate workspaces;
- the Codex App Server adapter and the stdio MCP adapter;
- Host-scheduled Managers with dependency-aware concurrency, fresh reviewers and the Host helper tool `markitect_start_helper`;
- durable journals with no-replay recovery;
- the `src/` layout.

Its open follow-ups moved to the backlog: P11 became RUN-04, R03-F01 became AGENT-03, and KG01 to KG03 became KG-00 to KG-03.

## Runtime and Codex App Server

- **Pinned protocol.** The adapter accepts only `codex-cli 0.162.0`, and its protocol identity embeds the digests of the generated v2 schema bundles *(on main: `codexappserver/process.go`)*. Every Run and Recover probes `--version` first; there is no protocol-version handshake. A Codex upgrade means regenerating schemas and changing the fingerprint. [MCP-02, RUN-02]
- **No hard cap on native helper starts or depth.** `agents.max_concurrent_threads_per_session` bounds concurrent children only. `agents.max_depth` is not a supported key and is not sent *(on main)*. Native collab starts are seen only after dispatch, so an over-limit start is interrupted, not prevented. Only Host-reserved `markitect_start_helper` calls are counted before launch. [RUN-02, MCP-02]
- **Loaded instructions are unknown.** The server reports instruction paths, not the bytes it loaded. The adapter records current-byte digests separately and leaves the effective instruction digest empty. [MCP-02, AGENT-02]
- **Permissions must be explicit.** An empty profile inherits the user's settings. In A01, inherited `:read-only` with `on-request` stalled the first writing Manager on a file-change approval. The adapter declines that request, as it declines every approval, authentication, input or tool request it does not know. Writers need `:workspace`; reviewers and Verifiers need `:read-only`; child approval is `never`. [RUN-02, RUN-03]
- **Uncertainty stops automation.** A lost request or an unconfirmed timeout after dispatch is `unknown`. Nothing retries or replays it automatically. Process ownership uses a Windows Job Object or a Linux process group; other platforms refuse to start. [RUN-02, TEST-04]
- **Event budget.** Events are bounded per connection, not per line. 16 MiB was too small for a full Manager audit (attempt 21). The A01 runtime used 128 MiB, and the client caps it at 256 MiB *(on main)*. [RUN-02]
- **Fingerprints bind everything.** A Plan binds the transport fingerprint: protocol, App Server settings, and dynamic tool specs and limits. Any change to a tool description stales saved plans. Build identity is part of the project digest, so a rebuilt binary cannot resume a run planned with another build (attempt 15). [RUN-02, whose acceptance removes executor fingerprints from plans]
- **Executable.** The npm `codex.ps1` wrapper cannot be used; the adapter needs the native `codex.exe`. [RUN-03, CI-05]
- **Race detector.** `go test -race` never ran on the development machine, because CGO was disabled. [TEST-01, CI-02]

## Model output contract: the most frequent A01 failure cause

Seven of the 27 failed A01 attempts ended because a role's output did not match the strict decoder. Several more ended on review grounding. The Host validator was never weakened. Each fix moved more of the contract into the request:

- Final messages use `turn/start.outputSchema`.
- The Host supplies executor `evidenceRefs`.
- Verifiers pick request-derived short aliases that the adapter maps back.
- Report invocations return empty `candidateFiles`, because the Host harvests the bytes.
- Helpers return plain-text files, not the input artifact's digest and base64 shape.
- Reviewer guidance separates actionable defects from positive findings.
- The project-run review prompt covers the typed fail report.

Each observed provider error deserves a regression case:

- string arrays where typed objects were expected;
- digest and base64 content copied from the input;
- references the request never supplied, including truncated strings despite an enum;
- `status: pass` with findings;
- findings outside the exact candidate scope;
- prose grounding without an exact token;
- paraphrased subject IDs;
- omitted helper metadata.

[TEST-04 fault injection; RUN-02, where external candidates and verdicts get the same validation; BUG-01]

## Workspaces and helpers

- **Workspace model.** Each role gets a private local clone (`--no-hardlinks --dissociate --no-checkout`) with real history, outside the source repository and without an origin. Ambient Git configuration, fsmonitor, filters and hooks are ignored. This is cooperative isolation, not an OS sandbox. [RUN-02, ARCH-06]
- **Rejected repository shapes.** Symlinks, gitlinks and submodules, special files, unsafe paths and case aliases fail visibly. Windows symlink behavior is untested, because the test skipped without symlink privilege. [TEST-01, CI-05]
- **Bounds.** Read context is limited to 100,000 files, 256 MiB per file and 1 GiB in total. The overlay is limited to 10,000 changes, 64 MiB per file and 256 MiB in total. Rename scratch space is limited to 256 MiB. [RUN-02]
- **Renames.** Rename provenance follows Git similarity; a fully rewritten move is a delete plus an add. [RUN-02]
- **Races.** Harvest detects ordinary races, not an adversarial writer. The coverage census does not detect a pure worktree byte change made after the selected bytes were captured (attempt 18 analysis). [BUG-01]
- **Helper contract.** `markitect_start_helper` authors files only, within a subset of the parent's write paths. It has no report-only result for read-only analysis (the former P11). The parent must not edit while a helper call is pending: if the target changed, the Host rejects the helper's delta (attempt 17), so the tool description now says to serialize. [RUN-04]
- **Open defect: helper cleanup after a failed Close.** If `Close` fails after a validated delta was applied, the Host records `cleanup-pending` with the handle, receipt, delta and an unknown reservation *(on main: `projectrun/helper.go`)*. Nothing can then reconcile it:
  - Resume has no helper-journal reader;
  - `GitService.Close` needs in-memory state that is lost;
  - `ReopenCandidate` fails its baseline check, because the parent already holds the change.

  The guard blocks the unknown state, and the negative test `TestHelperAppliedDeltaCleanupAndPersistenceFailuresRemainRecoverable` covers only that block. [BUG-01, RUN-02, TEST-04]
- **No adoption across processes.** Reopening a workspace requires a journal-confirmed terminal turn and every tracked child terminal. Nothing discovers workspaces automatically. [RUN-02]

## Recovery and resume

- **Exact turn only.** Recovery reads the exact saved turn (`thread/resume`, `thread/read`) and never starts a new one. A running or missing turn stays uncertain. A01 proved this only on the same binary. [TEST-04]
- **Bind journals to the run, not the task.** A lookup by task ID alone matched three historical journals (attempt 15), and the private A01 observer had the same flaw. The fix binds the receipt's run ID and input digest. [TEST-04, BUG-01]
- **Windows file sharing.** A journal append hit a sharing violation and left a review turn in an unknown terminal state (attempt 15). [CI-05, TEST-04]
- **Rework order.** A parent integration review must wait for queued child rework. In attempt 26 it re-ran first until the review limit was exhausted; fixed in `0c8d3873`. [TEST-04]

## Verify

- **Audits failed on missing evidence, not defects.** Full Verify runs the configured checks, an initial Verifier and one audit per Manager. In attempts 16, 21, 23 and 24 the audits failed on unsupported or truncated subjects, missing briefings, and documentation-review or integration-review evidence missing from the audit snapshot. An auditor needs the review evidence the run already produced. [TEST-04]
- **Unknown flags.** `project verify` rejected an unsupported `--plan` flag before any start (attempt 27). The bad-flag handling worked; the calling agent had used a flag that does not exist. Help and generated guidance must stay in step with the verbs. [TEST-02, AGENT-02]

## Windows, CRLF and paths

- **Packaged path alias.** Under the Codex desktop package, the local app-data folder appears as `…\Packages\OpenAI.Codex_…\LocalCache\Local`. Lexical path equality therefore rejected owned workspaces twice. The fix accepts `os.SameFile` only for the Git-reported candidate directory *(on main: `projectworkspace/git_reopen.go`)*; containment, the source root and Close keep strict checks. Shell writes under that path were denied, while the native editor worked. [CI-05]
- **CRLF.** With `core.autocrlf=true`, worktree bytes differ from Git blobs. This broke plan bytes, exploration (false positives), helper byte assertions (attempt 25) and Apply preflight (attempt 27). The fix compares through Git's clean filter. It rejects dirty content, `-text` newline changes, and assume-unchanged or skip-worktree paths. Test fixtures must set `core.autocrlf` explicitly; hosted Linux run 38051318813 failed because one did not. [TEST-01, CI-05]
- **Modes.** On Windows, executable bits come from the Git index, so new executables must be staged with their mode. `inspectFile` normalizes read-only and writable files to `0444` and `0644`. Filesystem-mode tests skip on Windows. [TEST-01, CI-05]
- **MXC sandbox.** Several Windows sandbox problems appeared. The effective backend identity remained unknown. [CI-05, RUN-03]
  - Child shells failed known-folder lookups (`ProgramFiles`, `ProgramData`, HRESULT `0x80070003`) when the child environment lacked Codex path variables.
  - A runtime ACL update failed with a sharing violation.
  - Temporary paths differ between shell calls, so scratch must be created, used and removed in one call.
  - Windows denied removing one scratch directory.
- **Long paths.** The 3 October baseline replay failed on deep temporary package paths until a shorter root was used. [TEST-01]

## Accounting and limits

- **Partial lower bound.** Native accounting is a partial lower bound:
  - missing usage stays unknown, never zero;
  - known arithmetic overflow is recorded separately from missing usage;
  - root usage is the App Server thread total;
  - helper child cost is not itemized;
  - a role's cost counts once across run states;
  - estimates are not invoices.

  [PLAY-06, PLAY-07, register OQ-003]
- **Start requests and role time.** Start requests include failed and nested requests. Product configuration caps a role at one hour; readiness rejected 7,200 seconds. [RUN-02]
- **Scale.** The A01 series used 184 combined requests, 180 provider starts and about 25.06 million estimated micros before its first pass. A study budget must include repair iterations, not only the measured runs. [PLAY-07]

## Tests, CI and records

- **Timing under load.** On Windows the full suite took about 30 minutes, with projectrun at 1,400 to 1,800 seconds. Several tests broke under that load:
  - a Delivery end-to-end test needed a 15-minute window;
  - App Server timeout subtests expired during the version probe;
  - a process fixture was cancelled between subprocess completion and durable result consumption.

  [TEST-03, TEST-05]
- **Gates broken by bookkeeping.** Several gates failed on bookkeeping rather than behavior:
  - new files failed managed-artifact accounting until they were registered (P03, P04, P05, and recovery and cost tests);
  - duplicate keys in a YAML record under `docs/` failed `markitect check`, because the legacy compiler parses YAML there;
  - the configured digest of `ci.yaml` failed after every workflow edit;
  - the packaged smoke still called the removed `init`.

  [ARCH-01, ARCH-07, CI-01, CI-03]
- **Records.** In two days the progress log grew to about 400 lines and the ledger to about 2,500, with status flags such as `no_twentieth_actual_claimed`. They are replaced by one status owner (the backlog), one dated validation record per result, and evidence in pull requests. [OPS-04, OPS-05]

## Earlier preparation records (3 October)

- **Risk register.** The risk register's 18 classifications stay in the [risk triage report](../../validation/parallel-wave-risk-triage.md). Still open there, and relevant to the delegated method:
  - Impact recall and noise in real work (PR #59 found 2 direct policy subjects among 69 conservatively affected resources);
  - context selection quality (an ADR was omitted);
  - the human usefulness of explanations;
  - model upkeep;
  - architecture fossilization;
  - adoption completeness.

  [PLAY-07, KG-03, IDEA-02, IDEA-03]
- **Alignment rule.** The workstream map's alignment rule still fits DEC-016: "Deprioritize expansions without a recurring human task to replace: additional projections/adapters for their own sake, automatic discovery/adoption, a scheduler or background operator, interface proliferation, and speculative Core expressiveness." [IDEA-01]
- **External command adapters.** GitHub and Azure DevOps command adapters were specified but never built. The wave produced only offline consumers, which ARCH-09 removes with the v0.13 line. Any remote Apply would first need:
  - target ownership;
  - detection of cross-adapter target aliases and overlaps, which nothing enforces today;
  - credentials;
  - concurrency;
  - handling of partial failure.

  A command runs with caller authority and can have side effects even in observe, plan or verify. An executable digest is not publisher provenance. [IDEA-01, ARCH-09]
- **Core change bar.** The bar for Core changes is kept in the [parallel-work guide](../../development/parallel-work.md#module-assignments): two distinct vocabularies, a finite alternative and deterministic tests.
