# Managed project operations validation

Date: 9 October 2026. This is local source evidence for `codex/model-first-operations`, based on the user-supplied operation documents at `ae6396dcc11777b773763b8fa6f27e4237a63462`. It does not update an installed release.

## Candidate and attempts

The implementation commit is `ff5e8c0b367044fefb51c7acdf7e361cf972e276`. The final tested source is `ce0f836dcf5401351493dd8cdb467b9b64259e42`; its only additional changes are the reviewer test fixtures and roadmap prose. Production code, adapters and the Shop example are byte-identical between these commits.

| Frozen-source attempt | Result |
|---|---|
| `ff5e8c0`: `go test ./... -count=1 -timeout=30m` | FAILED: 55 packages passed; two reviewer tests supplied synthetic revision names and incomplete briefing bundles rejected by the newly hardened store |
| Focused correction | PASS: the three reviewer tests now generate validated briefings from real fixed Git commits, retaining dismissal, stale-binding, unbriefed-model and draft-context assertions |
| `ce0f836`: `go test ./... -count=1 -timeout=30m` | PASS: 56 packages passed, no failing tests |

No production guard was relaxed to repair these test fixtures. The failed first suite is retained separately; its log SHA-256 is `7d03346f2c2ebd9f8459a14eba3ae14a36ec5afe06fdbc7b0ad38a9895db523b`. Earlier development checks are not substituted for final-candidate evidence.

The passing full-suite log for `ce0f836` has SHA-256 `f591a66c0a51c2cdf7ee9549bc86a4482e10cd1c6a2cdfc5b25bf717e676cc25`. The final evidence commit adds documentation and artifact accounting only; the full-suite result remains bound to its actual tested source revision.

The first documentation artifact-accounting preview rejected this new report because its tooling entry lacked a managed-root entry. The exact report path was added to the roots before the final documentation gate; the failed preview log is retained separately.

## Executed gates

| Check | Revision | Result |
|---|---|---|
| Full fresh Go suite | `ce0f836` | PASS: 56 packages |
| `go vet ./...` | `ce0f836` | PASS |
| Fixed root `check`, engineering-change `context`, and impact from the document baseline | `ce0f836` | PASS |
| Managed-artifact and module checks | `ce0f836` | PASS |
| Schema output and 20 contribution-guide CLI example commands | `ff5e8c0` | PASS |
| Codex adapter protocol suite | `ff5e8c0` | 41 tests PASS |
| Claude adapter protocol suite | `ff5e8c0` | 22 tests PASS |

The CLI example commands cover minimal and repository-layout Projects, canonical engineering, constitution, discovery, software architecture, delivery-target equality, benchmark v2 and the package consumer. Adapter tests use controlled protocol inputs, not provider accounts.

Source gate logs, command manifests and completed-log hashes are retained locally under `%TEMP%/markitect-operations-20261009/`. These local logs are not hosted CI evidence or a portable evidence bundle. The two full-suite attempts have separate source-named log files.

## Integrated operation scenarios

The Go subprocess fixtures exercise an Orders/Inventory/root Manager tree and actual candidate files. They cover:

- Targeted implementation remains impact-scoped while final Verify assesses every Manager and detects drift in an unimpacted sibling.
- An unclassified ordinary file outside the old inventory roots blocks full closure.
- Dismissed accepted-change events remain in the Manager and Reviewer context.
- Cleanup and Reconcile can produce a justified no-op, pass full Manager/check verification and guarded Apply, while an intervening untracked file makes the old Apply stale.
- An implementation change regenerates the configured readable model document; Verify and Apply bind the exact final source and document bytes.
- Full Apply rejects stale or tampered coverage, runtime, check, Manager and briefing bindings.
- Strictness adds requirements; mandatory subjects, checks and Managers remain required. Missing usage or exhausted budgets produce incomplete evidence. Known over-budget costs remain recorded; overflow is explicit.
- Census and compiled model use the same captured snapshot, including legacy coverage inspection. Draft proposals remain classified without invalidating their own source basis.
- Briefing writes bind fixed source revisions and provenance; reverse ancestry, malformed nested records and altered preview bundles fail. Revision-bound history handles explicit model reverts and rejects later unbriefed model changes.
- Onboarding supports an unborn feature branch, preserves existing native instructions and modes, retains skill frontmatter and is idempotent.

These are finite protocol, consistency and write-safety tests. They do not prove that an AI understood or exhaustively implemented the model.

## Public Shop smoke

A candidate built from `ff5e8c0` was used with a fresh disposable Git copy of `examples/project-world`. The source checkout remained clean.

| Observation | Result |
|---|---|
| Python cancellation application suite | 12/12 PASS |
| Project index | 6 Managers, 10 Statements, 5 Artifacts, 1 Check |
| Check, coverage, index, root and Orders context | PASS |
| Coverage before document / after document / after onboarding | 39 / 40 / 45 entries; accounted and conforming |
| Document preview versus generated file | Exact equality: 17,626 UTF-8 bytes |
| Native onboarding | First guarded write creates 5 files; second preview/write leaves all 5 unchanged |
| Briefing inspection | Empty notifications; no fabricated accepted changes |

The commands and raw outputs are under `%TEMP%/markitect-project-world-smoke-9719337d6feb4a1bbbd3bd8ca4334795/logs/`. One preliminary Python discovery invocation used the wrong working directory and stopped before running tests; the corrected command ran from the disposable fixture root and passed all 12. The Shop runtime remains empty, so this smoke starts no provider and makes no Shop Cleanup/Reconcile or autonomous engineering claim; operation execution is covered separately by the controlled subprocess fixtures.

## Remaining boundaries and assessment

The [operations guide](../project-operations.md) owns the current CLI. Accepted-change briefing creation is explicit: an empty store with an omitted baseline still needs bootstrap/mandatory-first-brief work. Durable Explore/open-decision/readiness state and iterative Brownfield return to modeling remain planned. Technical ownership remapping requires an explicit accepted model edit.

Native onboarding supplies repository instructions. Controlled local execution and scoped payloads do not establish OS isolation or prevent an independently writable agent from bypassing Markitect.

Live Codex/Claude provider trials, Scientist comparison cells, human acceptance, hosted CI and release publication were NOT RUN as part of this validation. Existing historical evidence remains separate.

The design is a closer fit to the requested model-first workflow: it covers unchanged responsibilities, whole-repository gaps, model-change context and one final integrated candidate. This is an architectural assessment. A measured quality, cost or productivity advantage over Classic or conventional agentic coding remains unproven and needs a fair independent comparison.

