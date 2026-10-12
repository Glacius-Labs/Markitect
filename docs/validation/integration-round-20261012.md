# Role-round integration — 12 October 2026

This checkpoint records source delivery from the three sessions requested after the [previous integration queue](integration-queue-20261011.md) closed. The initial Main was `1b621b1e06a3b0670d7f07b67a2632320796c7bf`, with complete [CI 38184367647](https://github.com/Glacius-Labs/Markitect/actions/runs/38184367647). The checkpoint Main is `d95d886eb37ee81e72dd9e264be6c5ceedc2e7bc`. TEST-02 remains active; this is not the completed round or a release, provider or method-study acceptance record. The [backlog](../work-items/backlog.yaml) owns current package status.

## Reviewed sources and actual integration

| PR | Full reviewed head | Root composition before merge | Actual Main merge |
|---|---|---|---|
| [233 — initial role status](https://github.com/Glacius-Labs/Markitect/pull/233) | `f69e83e626dbb1a5e9b33c647846ccc166256cc1` | fixed-source documentation checks | `231f3b9af18de4c44415fe299364a15530894aa9` |
| [232 — TEST-03](https://github.com/Glacius-Labs/Markitect/pull/232) | `352118412e7139ebed2e4b260f8cb1db7f474491` | `46990a8995033cadf6e8df750b7db4af09bf8685` | `6cef4af864094f4a46b2807e974f4e81cccba1e8` |
| [231 — CLI-03](https://github.com/Glacius-Labs/Markitect/pull/231) | `f66c4f13189150613be9afea5a30cac07c3ba74e` | `aca81f8319a0c27ab4b8100a14bf5fce95c93585` | `1bf4d36018de8b6ca6d7fa0ad05e8ecbc4caf97a` |
| [234 — merged-package status](https://github.com/Glacius-Labs/Markitect/pull/234) | `2b234babec8e692f8c15c563997808b3d938f002` | `f917d4e6d6195fb02c167dc8ab9f77d2c18dff60` | `c9f1090246bd992eced71530e7fc0afd67391cf3` |
| [235 — CLI-04](https://github.com/Glacius-Labs/Markitect/pull/235) | `173bf5a9e490eb32e232e5e45b7a10be47c158c6` | `5d58ce7a59ca169e6db42835af6790337f968a37` | `d95d886eb37ee81e72dd9e264be6c5ceedc2e7bc` |

All five heads received independent scoped review and passed required Linux Quality/Race: [233](https://github.com/Glacius-Labs/Markitect/actions/runs/38199203141), [232](https://github.com/Glacius-Labs/Markitect/actions/runs/38198514757), [231](https://github.com/Glacius-Labs/Markitect/actions/runs/38197630425), [234](https://github.com/Glacius-Labs/Markitect/actions/runs/38201794737), [235](https://github.com/Glacius-Labs/Markitect/actions/runs/38201888140). Root compiled each code candidate on actual current Main with `go vet ./...`; model, artifact and module checks remained separate. CLI-03 plus TEST-03 retain their 18 additive artifact entries; CLI-04 adds two more paths under the same implementation owner.

Complete Main CI passed on [233's merge](https://github.com/Glacius-Labs/Markitect/actions/runs/38200262740), [232's merge](https://github.com/Glacius-Labs/Markitect/actions/runs/38200515564), and [231's merge](https://github.com/Glacius-Labs/Markitect/actions/runs/38200658844). The later [234 Main run](https://github.com/Glacius-Labs/Markitect/actions/runs/38202488077) and [235 Main run](https://github.com/Glacius-Labs/Markitect/actions/runs/38202568775) were still running when this checkpoint was prepared. Their PR descriptions retain subsequent exact-merge proof; a green PR head does not stand in for a completed Main run.

## TEST-03 coverage and one source comparison

The change preserves all 295 test functions plus `TestMain` and all 51 runtime process flows. Twelve E2E files carry `!runtime_unit`; ordinary untagged Quality/Race includes them. Private child descriptors, arguments, environment, repositories, logs and temporary directories isolate parallel scenarios; the parallel parent tests do not change their environment or working directory. This is test organization, with no production Runtime changes.

The hosted comparison records 70 packages and 2,511 distinct package/test/subtest names: 2,492 pass, 19 skip and zero fail. Baseline and candidate event names and statuses match in both Quality and Race. Observed `projectrun` durations were 123.800 → 101.309 seconds in Quality and 878.142 → 386.181 seconds in Race. This is one hosted sample per source, not a repeated study, guaranteed duration or Windows acceptance.

CI tested synthetic checkout `b571f290d7c06d3c4e1fb08fea446611a8c0a18d`, whose full tree `cd3ed9a3940e2911ba64295de1dd9b57892438aa` equals literal head `352118412e7139ebed2e4b260f8cb1db7f474491`. QA preserves the comparison and original archives under `C:\Users\Consiliari\.codex\worktrees\78bb\test-evidence\linux-candidate-35211841\`. Quality archive SHA-256: `2d96d2debf1d55f8f0f2ee49a05c225d3ae06d1a852c795ea29207a36cdb015e`; Race archive: `084d5ec89e143c0b5e93b8d826e7d3fe6bf7fa6230edd19c18def7cc3ee522fc`.

## CLI-03 and CLI-04 contracts

CLI-03 routes product operations through the application facade and shares its composition factory. AST checks enforce the CLI service boundary and 300-line function limit. Independent review checked digest/no-write behavior, prospective hints, historical projection and the atomic dismissal compare-and-swap that retains provisional state. Constructing the factory starts no provider or workspace; read helpers can construct separate instances.

Its CI checkout `35af055d6bccc69d03eeb246da3a410cd0e71059` has tree `e3a4f768034961de0f970fea1f9eda19c6f4e178`, equal to literal reviewed head `f66c4f13189150613be9afea5a30cac07c3ba74e`. Fresh exact-head Quality/Race and provider-free smoke passed; the focused precommit checks used the same final Go bytes and remain described as such in the PR.

CLI-04 reports the actual Git version and a failed prerequisite below 2.40 before loading the model or inspecting a provider. CLI emits the report with exit 1; MCP returns the same report as nonconforming. Independent review and focused tests cover supported distribution suffixes, old/missing/unrecognized versions, command failure diagnostics and no project writes. Doctor is diagnostic; it does not automatically guard later Plan/Apply calls. Fixed check/Verify and provider-free smoke passed on head `173bf5a9e490eb32e232e5e45b7a10be47c158c6`. Its final current-Main composition tree is `be45609545cf5b385ee09d38e6b692e81c0a9e26`.

## Separate Windows results

QA's original full command `go test ./... -count=1 -timeout=60m -json` on clean literal head `352118412e7139ebed2e4b260f8cb1db7f474491` ended with exit 1: 53 packages passed, 15 had no tests, and `projectbriefing`/`projectrun` each exhausted the cumulative 60-minute package budget. Their active cases had run for 77/65 seconds at the alarm. No `projectrun` parallel `cont` event occurred; that E2E group had not run. This establishes neither an individual one-hour hang nor successful Windows E2E execution.

Complete raw JSON is `C:\Users\Consiliari\.codex\worktrees\78bb\test-evidence\full-suite-35211841.json`, SHA-256 `23ea68f5401560230aa6fe86d8d32a4deaf674508751e6dc1e9bc5277a7cb773`. Reconstructed stdout, machine result and interpretation are in the same directory. QA's branch `dev/runtime-test-isolation` remains clean and unchanged; no retry or repair was started.

CLI-03's separate full Windows run on frozen `f66c4f13189150613be9afea5a30cac07c3ba74e` also ended with exit 1: `projectbriefing`, `projectcli` and `projectrun` each exhausted their 60-minute package budget. The complete log remains at `C:\Users\Consiliari\.codex\worktrees\4fdc\interface-evidence\cli-composition-full-windows-f66c4f13.txt`. CLI-04's own Windows full run on frozen `173bf5a9e490eb32e232e5e45b7a10be47c158c6` is separate and still running at this checkpoint. None is a current-Main Windows PASS. Causes and host-load contribution were not isolated. The earlier curated/truncated `f9ddd795` Git-timeout/update-ref diagnosis remains a distinct preserved record.

## Bughunter boundary and future work

The first BUG-01 sweep completed on clean `fix/helper-cleanup-review` at `1b621b1e06a3b0670d7f07b67a2632320796c7bf` in `C:\Users\Consiliari\.codex\worktrees\2a48\Markitect`, without production changes. Its isolated observation reproduces helper `cleanup-pending` after an applied delta, failed Close and loss of process-local workspace handles. A passing observation test proves the blockade, not successful recovery. Cross-process reconciliation is a new proposed capability with its own intent and acceptance work; current closure guards stay intact. The Census observation remains unconfirmed as a product defect.

The external `helper-cleanup-recovery-proposal.md` under `C:\Users\Consiliari\.codex\visualizations\2026\10\12\01a12dfb-3026-75c0-9b81-6a8a1be53e4f\bug01-evidence\` is **PROPOSED / NOT IMPLEMENTED / NOT ACCEPTED**, SHA-256 `08FF8A68CADC7344EAEA87730EC6737B492D6A3B1936CE5B35952D82AA5C1A12`. No recovery, legacy-record migration, build-pin bypass or live cleanup was authorized.

TEST-02 owns Interface next. TEST-04 is future ready work, TEST-05 waits, and RUN-02/RUN-05 stay proposed. PLAY-06 is ready without starting a Scientist chat. KG-04 remains **PREPARED / NOT RUN**; provider and method studies remain deferred. Government implementation, releases and worktree cleanup are outside this round. Historical and dirty checkouts remain preserved.
