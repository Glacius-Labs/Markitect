# Measurement

This protocol refines the supplied benchmark and metrics guide. It owns measurement procedure; [the implementation plan](implementation-plan.md) owns delivery order. Markitect tooling and tests use Go. Any future persisted Markitect measurement records use YAML, consistent with other product data.

## First prove correctness

Use a synthetic consumer with explicit shared policy and two separate customer areas. The expected resources are defined before inspecting results. Check local Rule changes, shared Rule changes, removed dependencies, Contract binding changes and unmodelled input changes. Keep the last case conservative: an undeclared file is not proof of unrelated meaning.

Go scenario tests compare exact expected impact/context sets. Go benchmarks measure the deterministic operations after fixture construction. They can report allocations, context bytes and affected-resource count. They measure this process and fixture, not human authoring speed or model savings. Keep a benchmark run's source commit, command, platform and raw output together in the excluded `.artifacts/` directory.

Run the bounded fixture from the source repository with:

```powershell
go test ./internal/app -run AuthoringScenario -count=1
go test ./internal/app -run '^$' -bench Authoring -benchmem -count=3
```

## Then exercise the authoring workflow

Copy the complete minimal example to an isolated Git repository. Record a baseline commit and give an agent a normal requirement, access to the CLI and the bundled `authoring` context. An independent reviewer checks the changed owner, required references, unaffected customer scopes, generated outputs and fixed impact. Retain failures and refinements. This is a usability exercise, not an A/B benchmark or provider-runtime certification.

## Controlled model comparison

For a later comparison, prepare the task oracle outside the actor's context before starting. Run baseline and Markitect variants from identical fixed snapshots with the same model, effort, permissions and task. Use separate workspaces and fresh conversations; include discovery and recovery costs. An optional check-only variant can isolate the effect of context and impact queries.

Start with a small pilot across local Rule edits, shared rules and dependency changes, then extend to renames, binding/scope changes, code-to-document inputs, provider drift and stale references. Repeat each variant at least three times before describing a trend. Report success and invalid runs alongside duration and token counts; separate correctness from speed.

Capture only data actually available: model-reported token usage, model calls, tool calls, searches, files read, wall time, context bytes, affected entries, review reuse and human interventions. Missing values are unavailable, never zero. Do not estimate tokens from byte count. Context recall/impact recall require a known expected set; precision requires a nonempty returned set. Report undefined ratios explicitly, including their numerator and denominator, rather than silently using 100%.

Review reuse must be exercised for identical inputs, relevant changes, removed edges, tool/config changes and unknown files. A reuse decision makes no model call. Compare the full repeated task before claiming sustained savings.

## Optional continuous measurement

No telemetry is installed or enabled by this slice. Introduce local opt-in YAML records only after the controlled pilot establishes useful fields. Counts, durations, identifiers and hashes should suffice; raw prompts, secrets and customer documents do not belong in generic metrics. Keep observations grouped by comparable tasks and scope, not employee rankings. A statistics CLI, remote collection and database are future decisions, not current commands.
