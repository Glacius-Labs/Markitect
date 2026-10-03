# Measurement

This document defines measurement procedure for Markitect product work. The [vision](vision.md) owns the human-attention and delegated-work thesis; the [roadmap](implementation-plan.md) owns delivery order. Product tests and measurement fixtures use Go; any Markitect-owned persisted measurement record uses YAML.

## Human attention and delegated work

The future product question is whether people can govern more autonomous engineering work without losing architectural intent. No current value for that outcome is established. Fewer interventions count as improvement only when task quality, required escalation, authority and independently checked architecture outcomes are preserved. Silent violations or skipped review requirements are failures, not autonomy.

Use repeated realistic task/change sequences with a materially simpler baseline, such as owner guidance plus existing architecture tests. Freeze task/rubric, source and policy history, actual agent/model settings, tool versions, permissions and environment. Counterbalance run order and use fresh isolated sessions. Include setup, discovery, recovery, model authoring and recurring upkeep, not just the implementation interval. A single run or an unmatched policy history cannot establish sustained attention savings.

| Future metric | Required observation and limit |
|---|---|
| Autonomous agent-hours / human intervention-hours | Record actual execution intervals and measured human attention separately; distinguish waiting and overlapping parallel work. An open agent session is not active work. Report both quantities and unavailable or zero denominators; do not substitute elapsed workflow time for attention. |
| Completed changes / human review actions | Count independently assessed task outcomes and actual human reviews, corrections and acceptance decisions. Keep automated agent reviews separate. A green command or merged PR is not automatically an accepted change. |
| Architecture-compliant changes before human intervention | State the finite model assertions and specialist implementation checks that ran, their fixed inputs and independent review coverage. Keep passed, failed, waived, incomplete and unknown evidence distinct; do not call modeled conformance complete implementation correctness. |
| Architectural escalation and intervention | Record decisions versus routine clarifications, source/scope correction, steering, manual review, reconciliation and recovery. Score missing necessary escalations and unnecessary escalations against an owner-defined rubric. Fewer events alone is not better. |
| Human time allocation | Measure implementation, routine review/synchronization, specification, architecture decisions, initial setup and recurring model/adapter maintenance. Record total attention as well as its distribution; moving effort into YAML authoring is not automatically a saving. |
| Synchronization or reconciliation work eliminated | Identify a real prior manual step and owner, the replacement operation, outcome and subsequent maintenance. Generated file counts or several views of one owner do not prove a truth source or human step was removed. |
| Drift detection and context/impact usefulness | Predeclare expected rules, violations, inputs and affected sets where independently knowable. Record misses, unnecessary inclusions, review noise and causes. Separate direct policy subjects from conservative impact; unassessed resources are not automatically false positives. |

Record human interventions where they occur and preserve failures, exclusions, missing evidence and later corrections. Measure repeated architecture evolution as well as ordinary implementation so package, exception, mapping and projection upkeep is visible. Tokens and model/tool calls remain secondary observations only when instrumented; bytes, helper read events and agent duration are not token usage or cognition.

The [real-code pilot](validation/real-project-adoption-pilot.md), [matched comparison](validation/agents-md-vs-markitect.md) and [consumer inventory](validation/parallel-wave-konfyra.md) retain contrary evidence and missing instrumentation. They do not establish human-attention reduction. Future studies must allow the simpler baseline to win and ask whether modeling costs more than the consistency work it replaces. This section specifies evaluation, not a telemetry implementation, benchmark run or automatic acceptance policy.

## Correctness first

Use a synthetic project with explicit shared policy and two separately scoped areas. Record expected effects before running the tool. Cover local and shared Rule changes, removed dependencies, Contract-binding changes, and an unmodelled input change. Unknown inputs must remain conservative until they are declared.

Go scenario tests compare exact expected context and impact sets. Go benchmarks measure deterministic operations after fixture construction. They can report allocations, context bytes, and affected-resource count; they do not measure human authoring speed or model savings. Keep the source revision, command, platform, and raw output together in excluded development artifacts.

```powershell
go test ./internal/app -run AuthoringScenario -count=1
go test ./internal/app -run DocumentationScenario -count=1
go test ./internal/app -run '^$' -bench Authoring -benchmem -count=3
```

The [code and documentation example](documentation.md) exercises exact source-file inputs with a predeclared affected set and an unrelated area. Its source-only change intentionally leaves stale prose structurally valid: the scenario verifies routing to review, not semantic correction by the compiler.

CI executes one iteration of each authoring benchmark to keep the measurement fixtures valid as the resource model evolves. This is a correctness smoke check, with no timing threshold. For timing reports, run the repeated command above on a fixed source commit and record the host; the measured section excludes Git reads, parsing, agent interaction, and provider calls.

After each immutable published release at v0.9.0 or later, `.github/workflows/release-benchmark.yaml` runs the current and immediately previous attested platform binaries on Windows and Linux. Current releases use `benchmark/fixtures/v2`; the previous binary uses v1 when its release predates 0.9.0 and v2 otherwise. The immutable v1 fixture preserves the pre-0.9 layout, while v2 exercises the selected Markdown output with canonical resources beneath `.markitect/areas/sample/` and an ordinary input beneath `docs/`. The harness gives both binaries one shared prepared repository when fixture version and bytes match. Across a version transition it prepares each compatible fixture independently, records the version, root, digest, base commit, and candidate commit for each release, and suppresses performance-change percentages because workloads differ. See the [benchmark contract](../benchmark/README.md) for invocation and metric definitions.

Each command (`check`, `render`, `context`, `impact`, and `verify`) runs once as the first fresh process and three more times as fresh processes, with release order alternating. Records include wall time, sampled peak working set, exit status, status text, and bounded output. JSON and Markdown summaries are workflow artifacts; no asset is added to the immutable release. A maintainer can dispatch the workflow with an exact published stable tag to repeat a failed measurement; it verifies the same immutable release and assets without changing them. The first run does not clear operating-system caches, and a short process can exit before memory sampling, which is recorded as null. There is no performance threshold or release-blocking benchmark gate.

## Authoring exercise

Copy the complete minimal example into an isolated Git repository and commit a baseline. Give an agent a normal requirement, access to the CLI, and the bundled `authoring` context. An independent reviewer checks the changed owner, required relationships, unaffected scopes, generated outputs, and fixed impact. Retain failures and corrections. This is a product usability exercise, not an A/B benchmark or provider-runtime certification.

## Controlled model comparison

Prepare the task oracle outside the actor's context. Run baseline and Markitect variants from identical fixed snapshots with the same model, effort, permissions, and task. Use separate workspaces and fresh sessions; include discovery and recovery costs. An optional check-only variant can isolate the effect of context and impact queries.

Start with a small set of local Rule edits, shared rules, and dependency changes, then extend to renames, binding/scope changes, code-to-document inputs, provider-output drift, and stale references. Repeat each variant at least three times before describing a trend. Report successes and invalid runs alongside duration and token counts; separate correctness from speed.

Capture only observed data: model-reported token usage, calls, tool calls, searches, files read, wall time, context bytes, affected entries, review reuse, and human interventions. Missing values remain unavailable; do not estimate tokens from bytes. Context/impact precision and recall require a known expected set. Report undefined ratios with numerator and denominator rather than substituting 100%.

Review reuse should be exercised for identical inputs, relevant changes, removed edges, tool/config changes, and unknown files. Reuse makes no model call. Compare the full repeated task before claiming sustained savings.

The [2026-10-01 documentation pilot](authoring-pilot.md) records an incomplete first exercise and a six-run repeat with the mutation boundary stated in the task card. All six repeated candidates passed fixed structural and output checks and the four-path boundary. The blind semantic review found a further wording difference between task and rubric, while token and cost measures remained unavailable. State every scored requirement in both task card and review rubric before running another comparison; keep structural validity, requested semantics, and a narrow mutation boundary as separate results.

## Optional local records

No telemetry is installed or enabled. Consider local opt-in YAML records only after a controlled exercise establishes useful fields. Counts, durations, identifiers, and hashes are enough; prompts, credentials, and source documents do not belong in generic metrics. Do not rank individual contributors. A statistics CLI, remote collection, and database are separate future decisions.
