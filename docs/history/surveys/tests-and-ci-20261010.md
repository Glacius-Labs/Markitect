# Survey: tests and CI

AI-generated, read-only, 10 October 2026, against main `02c7e529`. Input for TEST-01 to TEST-05 and CI-01 to CI-05. Register DEC-013 makes Linux the required gate.

## Main causes of CI pain

- **The suite runs twice per job.** The dogfood step `markitect verify --repo .` runs the declared check `go-tests` from `markitect.yaml`, which repeats `go test ./... -count=1 -timeout=60m`.
- **Windows is 10 to 25 times slower** on the new `project*` packages. They use git fixtures and re-exec the test binary, serially.
- **Smoke tests are about 400 lines of bash and PowerShell**, written once per OS in `ci.yaml` and again in `release.yaml`. They drift from the CLI.
- **Main runs get cancelled.** Concurrency `ci-${workflow}-${ref}` with cancel-in-progress also applies to main. Main run 38067569269 was cancelled by the next push.

## Inventory

- **Go:** 303 test files, about 71.5k test lines against 95.5k production lines, in 61 packages.
  - **Fast tier** (each package under 10 s on Windows): core, modules, compat, tooling, mcp, records, recordstore.
  - **Integration tier** (git fixtures and subprocess re-exec): host (350 tests), projectrun (256 tests, 12.8k lines), projectcli, projectbriefing, projectadoption, projectonboarding, projectwork, projectworkspace, projectapp, codexappserver (fake App Server via `TestMain`), agentexec.
  - **Fixture end-to-end:** `src/harness/examples` (56 tests) builds binaries inside tests.
- **Parallelism and tiers:** only one `t.Parallel()` in the whole suite, 175 `t.Setenv` calls, no `testing.Short()`, no tier build tags.
- **Python:** the `clauderunner` and `codexrunner` protocol tests and `scripts/test_check_docs.py` run in CI. The example tests run only through the harness, if at all.
- **CLI coverage (best effort):**
  - No argv-level test exists for `project deliver`, `onboard`, `coverage`, `doctor`, `index`, `cleanup`, `reconcile`, `resume` or `verify`.
  - The MCP tests never name `project_preflight`, `project_repair`, `project_resume` or `project_verify`.
  - Nothing enforces completeness.

## CI timings

| Run | Windows job | Windows `go test` | Windows dogfood verify | Linux job |
|---|---|---|---|---|
| 38061655290 (main `f12ffb00`) | 68.5 min | 1940 s | 1966 s (suite again) | 7.7 min |
| 38064068682 (PR) | 49.7 min | 1372 s | 1462 s | 8.1 min |
| 37836181702 (8 Oct, before the `project*` packages) | 15.2 min | – | – | 3.0 min |

**Windows against Linux, same run:**

| Package | Windows | Linux |
|---|---|---|
| projectrun | 1467 s | 112 s |
| projectcli | 606 s | 40 s |
| host | 433 s | 44 s |
| projectbriefing | 365 s | 24 s |
| projectadoption | 278 s | 24 s |
| projectonboarding | 238 s | 11 s |

Of 200 runs since 1 October, 159 succeeded, 29 were cancelled and 10 failed.

**Smoke tests:**
- **Packaged smoke:** Go, git, jq and unzip.
- **Onboarding replay:** `scripts/onboarding/Measure-Adoption.ps1`, Windows only.
- **A01 native smoke:** manual. It needs an authenticated Codex CLI, `gpt-6-luna` and the Windows MXC sandbox.

## Flakiness and environment sensitivity

**Failures seen in CI:**
- The Windows onboarding replay exited with 2.
- `TestGuardedWrite*` failed inside the dogfood verify's temp snapshot ("guarded apply root differs from captured repository root"), likely path aliasing.
- An `agentexec` spawn-timeout test failed on Windows.
- A Linux `TempDir RemoveAll … .git: directory not empty` failure.
- App Server timeout subtests expired while the version probe ran under load.

**Drift and fixture bugs:**
- The packaged smoke called the removed top-level `init`.
- A CRLF Apply fixture did not set `core.autocrlf`.

**Root causes visible in the code:**
- There are 33 separate git test helpers, and only 3 files isolate git config. The runner's `core.autocrlf=true` leaks into fixtures; 12 files toggle autocrlf by hand.
- The process end-to-end tests pass behaviour to the re-executed test binary through environment variables, which prevents `t.Parallel`.
- Declared checks get an allowlisted environment (`projectrun/verify.go`, `explicitEnvironment`), so TEMP and GOCACHE are missing unless declared.
- "The directory name is invalid": the materialized candidate path can exceed Windows' working-directory limit under deep paths. A fix and `check_directory_test.go` are in pull request #92.
- The workspace default `os.UserCacheDir()/Markitect/workspaces` lands in MSIX-redirected AppData under the Codex desktop app.

## Reusable building blocks

- **`projectrun/process_e2e_test.go`:** plan, run, review findings, repair, rework, resume, verify, apply, plus cost and retry ceilings.
- **`operations_e2e_test.go`:** full verify, no-op, cleanup, reconcile, guarded apply.
- **`repair_e2e_test.go`** and **`resume_native_e2e_test.go`**.
- **Pull request #92:**
  - `markitect-exchange-executor` and the `unmetered` cost mode;
  - `byo_executor_e2e_test.go` with mixed role profiles and fault injection (out-of-scope write, forgotten file, review finding with repair, integration failure).

  Together these enable black-box tests of the real binary with a scripted responder.
- **Playground:** a container image and the provider-free fake-agent smoke (`tests/smoke_docker.py`).

## Proposed test tiers

| Tier | Content | When |
|---|---|---|
| L0 unit | pure packages | every PR |
| L1 integration | git and process fixtures through a hermetic test kit | every PR (Linux) |
| L2 CLI contract | black-box per verb and MCP tool against one built binary | every PR |
| L3 method end-to-end | exchange executor with scripted scenarios | every PR (Linux) |
| L4 container runtime | playground fake agent nightly; real providers manual with cost cap | nightly, manual |
| L5 studies | case studies, A01 | manual |

Target: a Linux PR job of about 8 to 10 minutes. Windows follows DEC-013.

## Collision zones

- **runtime zone:** `projectrun` and `projectsetup` (pull request #92).
- **ci zone:** `.github/workflows/ci.yaml`, `markitect.yaml` (the `go-tests` check), `release.yaml`.
- **interface zone:** `projectcli/run.go`, shared with #92 and with CLI-02.
