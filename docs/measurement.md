# Measurement

This document defines measurement procedure for Markitect product work. The [roadmap](implementation-plan.md) owns delivery order. Product tests and measurement fixtures use Go; any Markitect-owned persisted measurement record uses YAML.

## Correctness first

Use a synthetic project with explicit shared policy and two separately scoped areas. Record expected effects before running the tool. Cover local and shared Rule changes, removed dependencies, Contract-binding changes, and an unmodelled input change. Unknown inputs must remain conservative until they are declared.

Go scenario tests compare exact expected context and impact sets. Go benchmarks measure deterministic operations after fixture construction. They can report allocations, context bytes, and affected-resource count; they do not measure human authoring speed or model savings. Keep the source revision, command, platform, and raw output together in excluded development artifacts.

```powershell
go test ./internal/app -run AuthoringScenario -count=1
go test ./internal/app -run '^$' -bench Authoring -benchmem -count=3
```

## Authoring exercise

Copy the complete minimal example into an isolated Git repository and commit a baseline. Give an agent a normal requirement, access to the CLI, and the bundled `authoring` context. An independent reviewer checks the changed owner, required relationships, unaffected scopes, generated outputs, and fixed impact. Retain failures and corrections. This is a product usability exercise, not an A/B benchmark or provider-runtime certification.

## Controlled model comparison

Prepare the task oracle outside the actor's context. Run baseline and Markitect variants from identical fixed snapshots with the same model, effort, permissions, and task. Use separate workspaces and fresh sessions; include discovery and recovery costs. An optional check-only variant can isolate the effect of context and impact queries.

Start with a small set of local Rule edits, shared rules, and dependency changes, then extend to renames, binding/scope changes, code-to-document inputs, provider-output drift, and stale references. Repeat each variant at least three times before describing a trend. Report successes and invalid runs alongside duration and token counts; separate correctness from speed.

Capture only observed data: model-reported token usage, calls, tool calls, searches, files read, wall time, context bytes, affected entries, review reuse, and human interventions. Missing values remain unavailable; do not estimate tokens from bytes. Context/impact precision and recall require a known expected set. Report undefined ratios with numerator and denominator rather than substituting 100%.

Review reuse should be exercised for identical inputs, relevant changes, removed edges, tool/config changes, and unknown files. Reuse makes no model call. Compare the full repeated task before claiming sustained savings.

## Optional local records

No telemetry is installed or enabled. Consider local opt-in YAML records only after a controlled exercise establishes useful fields. Counts, durations, identifiers, and hashes are enough; prompts, credentials, and source documents do not belong in generic metrics. Do not rank individual contributors. A statistics CLI, remote collection, and database are separate future decisions.
