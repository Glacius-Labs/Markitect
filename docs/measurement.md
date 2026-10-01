# Measurement

This document defines measurement procedure for Markitect product work. The [roadmap](implementation-plan.md) owns delivery order. Product tests and measurement fixtures use Go; any Markitect-owned persisted measurement record uses YAML.

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

After each immutable published release, `.github/workflows/release-benchmark.yaml` runs the published current and immediately previous attested platform binaries against `benchmark/fixtures/v1` on Windows and Linux. It records the first and three subsequent fresh-process runs of `check`, `render`, `context`, `impact`, and `verify`, with wall time, observed peak working set, exit status, and bounded output. Its JSON and short Markdown summary are workflow artifacts; no asset is added to the immutable release. If the automatic run fails, a maintainer can dispatch the repaired workflow with the exact published tag; the workflow checks out that tag and verifies the same immutable release and assets without changing them. First-run does not clear operating-system caches; a very short process can finish before memory sampling and records null for that sample. Compare distributions and failures, not one percentage; there is no performance threshold or release-blocking benchmark gate.

## Authoring exercise

Copy the complete minimal example into an isolated Git repository and commit a baseline. Give an agent a normal requirement, access to the CLI, and the bundled `authoring` context. An independent reviewer checks the changed owner, required relationships, unaffected scopes, generated outputs, and fixed impact. Retain failures and corrections. This is a product usability exercise, not an A/B benchmark or provider-runtime certification.

## Controlled model comparison

Prepare the task oracle outside the actor's context. Run baseline and Markitect variants from identical fixed snapshots with the same model, effort, permissions, and task. Use separate workspaces and fresh sessions; include discovery and recovery costs. An optional check-only variant can isolate the effect of context and impact queries.

Start with a small set of local Rule edits, shared rules, and dependency changes, then extend to renames, binding/scope changes, code-to-document inputs, provider-output drift, and stale references. Repeat each variant at least three times before describing a trend. Report successes and invalid runs alongside duration and token counts; separate correctness from speed.

Capture only observed data: model-reported token usage, calls, tool calls, searches, files read, wall time, context bytes, affected entries, review reuse, and human interventions. Missing values remain unavailable; do not estimate tokens from bytes. Context/impact precision and recall require a known expected set. Report undefined ratios with numerator and denominator rather than substituting 100%.

Review reuse should be exercised for identical inputs, relevant changes, removed edges, tool/config changes, and unknown files. Reuse makes no model call. Compare the full repeated task before claiming sustained savings.

The [2026-10-01 documentation pilot](authoring-pilot.md) stopped after two completed runs because of a provider usage limit. It records the incomplete result and a mismatch between the task's wording and a stricter mutation oracle. State every scored mutation restriction in the task card before running another comparison; keep structural validity, requested semantics, and a narrow mutation boundary as separate results.

## Optional local records

No telemetry is installed or enabled. Consider local opt-in YAML records only after a controlled exercise establishes useful fields. Counts, durations, identifiers, and hashes are enough; prompts, credentials, and source documents do not belong in generic metrics. Do not rank individual contributors. A statistics CLI, remote collection, and database are separate future decisions.
