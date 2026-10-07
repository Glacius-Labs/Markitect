# Government G1 validation, 2026-10-07

This isolated Worker candidate realizes the G1 design from Classic `1ea5c76f55526fc4d721e865885436153f48b497` and pure design transfer `691ce1dee3484371c4aea725db4ed5cc3f2e3719`. It is an implementation of the authorized Government intent, not a Classic release or an adopting-project acceptance. The final local commit is reported in the Worker chat; the source tree containing this report is the reviewable candidate.

## Delivered behavior

- Versioned purpose-bearing Area, Ressort, Mandate, Capability, Responsibility, Artifact, Realization and Constitution Definitions compile with arbitrary domain Schemas through the unchanged structural Core. Host checks exact identity, recursive organization, delegation subsets and unique path writers.
- A native read-only inventory binds explicit roots and reasoned exclusion/ignored-admission boundaries. It includes untracked/ignored metadata, leaves ignored bytes unread without explicit admission, and refuses links/reparse targets and submodule traversal. It fails closed on Git classification errors. Observation stays provisional and separate from accepted intent.
- Read-only orders bind the active prior Constitution digest, resolve domain-reference/shared-file closure, responsible Areas, writers, mandates, parent integration reviews and cabinet. Unknown or contradictory scope, insufficient delegation and unavailable selected bytes block work. Unrelated unknown files remain visible in a scoped proposal.
- Inert G2 record constructors bind material, evidence, votes and decision without a hash cycle. They check prior-authority equality, frozen mandate/digest/slot binding, complete explicit assent and fresh candidate/evidence/round. They do not run agents or authorize promotion.

The [executable fixture](../../examples/government/README.md) uses actual Go implementation/test files and shared Markdown. Its pinned Constitution is `sha256:9c94d00054effeff3477850f1872c250d0ea7b9f5602e384fe6a6853148fe70a`. Positive planning returns `planned-scoped`/exit 0. The non-delegated model-amendment order returns `blocked`/exit 1 with `authority.missing`. Inspection returns exit 1 because the real unassigned file and operational order files remain unknown. These are structural/planning outcomes, not autonomous engineering trials.

## Independent review

A separate read-only subagent reviewed the actual implementation, focused tests and fixture. Two actionable findings were repaired and checked again: foreign-only realizations could previously appear executable with no writable path; decision construction previously failed to compare candidate prior authority and exact frozen mandate/slot mappings. The reviewer withdrew a third suspected scope-widening finding after rereading the actual inequality and checking the adversarial case. Final bounded review reported no open actionable findings and independently ran focused Government/inventory tests plus positive/negative CLI cases.

Implementation review also found and repaired native scanner risks: intermediate symlink roots, Windows path aliases, swallowed Git classification errors, omitted ignored-admission scope in the report digest, and inconsistent-read detection. Additional planning checks distinguish absent files from unavailable descendants of an excluded ancestor. Passing checks establish the exercised contracts only; they do not demonstrate hostile-process isolation or semantic sufficiency.

## Gate evidence

Initial fixed Markitect `check` and engineering `context` ran successfully at `691ce1dee3484371c4aea725db4ed5cc3f2e3719`. The strict check passed; separately reported canonical projection state was incomplete. Focused Government, inventory, binding, decision and CLI checks and the real example's Go checks pass. All 20 prescribed Contribution schema/example/format/model/context commands, `go vet ./...`, `go build ./...`, `go mod verify`, root formatting, artifact accounting and configured module checks pass.

The first `go test ./... -count=1` run failed: the new application fixture required classification in the import gate, and the unchanged Host package hit its ten-minute package timeout (the active installation test had run for one second; queued controller scenarios had not yet started). The fixture is now categorized as application code with all product imports still forbidden, covered by a negative test. This classification does not exempt its imports. The final fixed-candidate check/impact/verify results are retained in local ignored `.artifacts/government-g1/` and reported with the exact commit in the Worker chat. The initial timeout is not a successful full-suite result. No CI, release, runtime pilot, comparison advantage or human acceptance is claimed.

## Remaining boundary and G2 handoff

G1 native capture is provisional rather than an atomic Git snapshot. Digests identify selected material, not an authenticated Owner. Processes retain caller OS rights; worktrees and path fencing are a cooperative boundary. Boundary metadata and exclusions are not semantic conformance. Declared purpose, cabinet selection, model/file relations and technical checks can all be insufficient for real business correctness.

Next implement one actual area order end to end: immutable candidate workspace/input capture; a prior-Constitution and exact cabinet/mandate freeze; configured Executor, independent Verifier and Ressort runs; actual file changes and checks; fresh complete assent to the final material/evidence; durable promotion intent and fenced compare-and-swap of a dedicated Active-Ref; bounded success and abort reports. Demonstrate refusal for absent votes/review/evidence, changed candidates and stale active base. Actual G2 execution and Host takeover, recursive execution (G3), amendment activation/conflict (G4), and queue/recovery (G5) remain unimplemented.
