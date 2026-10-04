# Vertical slices gauntlet: seed parity and limits

This directory contains two independently runnable seeds for a controlled comparison. Both arms use the same fictional Commerce application, identical 13 tracked C# project/source/check files at their starting snapshots, the same task cards, project facts, acceptance checks, and toolchain. No adopting-project or customer source was copied.

- **A** uses explicit `AGENTS.md`, architecture documentation, an executable offline behavior/source-boundary runner, and a CI workflow.
- **B** uses the same app and runner. Its canonical Software Architecture Domain/resources encode the same module facts and v1 policy; `Skill.spec.files` declares application inputs; fixed-snapshot context and impact use the pinned model; Codex skill/Markdown projections are generated; and artifact coverage accounts for canonical resources, package archives, ordinary project inputs, outputs and tooling. The v2 package archive is included by exact digest and remains inactive until the policy-migration task.

The app is a .NET 8 console-hosted modular monolith with Orders, Inventory, and Billing. Its deterministic console runner checks in-memory command behavior and selected source-boundary rules. It is not xUnit/NUnit and has no database, ASP.NET host, EF Core, external NuGet packages, concurrency, security, persistence, or deployment proof. `NuGet.Config` clears all package sources. The local .NET 8 SDK and target packs allowed restore/build/run without network access. The frozen CI file uses setup-dotnet 8.0.418 but hosted CI was not run.

`task-set.yaml` is the sequential 01–12 oracle; tasks 09–12 are evaluator holdouts and must not be sent to actors or used to coach earlier tasks. Every task starts from the accepted state produced by the prior task in the same longitudinal chain.

## Policy evolution

B starts at exact-pinned Software Architecture package 1.0.0. The `validation: required` cohort must contain Commands; Validators are optional. Exact package 2.0.0 preserves that cohort and adds a Validator requirement. The archive and digest are present in the seed, but the active `markitect.yaml` pins only v1. Task 11 records v1 and v2 findings before edits and adds Validators only if the unchanged selected cohort lacks them. The successful task-06 state already has one, so a clean task-11 migration can correctly require no implementation change. A carries the same v1/v2 facts in project documentation/checks and task history; it does not need a fake Markitect model.

The selected cohort is exactly `engineering/UseCase:create-order` in both A's documentation and B's canonical resource. The policy change applies to that selected command, never by changing cohort labels or adding an exception. This is a predeclared distinction: model coverage is checked separately from correct task classification and project behavior.

## Fixed source and tool evidence

- Markitect candidate source: `adef79d935399f8ac63ad874dbdeab8d15c418a1`; binary version 0.13.0 and both binary and reported tool digests are in `evidence/candidate-evidence.yaml`.
- A starts at `cd58b3786ce14ea4550a2a7a3a32e23b77de5903`; B starts at `7f44f450dc69294e1c0a9c4e44dd38c2507c9114`.
- B fixed-snapshot `check`, `render` check, `verify`, ordinary compiled Context, and artifact-coverage outputs are preserved under `evidence/`. Frozen v1 and v2 archive SHA-256 values are recorded there. The fixed drift vector changes only the declared Order.cs input and correctly impacts the bounded implement-commerce Skill closure; the exact before/after commit and result are in `evidence/b-impact-drift.yaml`. A v2 throwaway probe from the untouched baseline seed demonstrates the initial missing-Validator policy finding; this is package-availability preparation evidence, not the expected task-11 result after task 06.
- A and B each pass `dotnet run --project checks/Commerce.Checks.csproj` with six baseline assertions. The runner has no benchmark or latency threshold.

Use fresh isolated copies, identical model/settings, permissions, and tool versions. Count setup, discovery, implementation, recovery, review and recurring upkeep. Capture actual human interventions separately from agent time, waiting, model claims, and automated checks. Independently evaluate each outcome against the predeclared task oracle. Do not infer attention savings from elapsed time, generated file count, context bytes, check status, or lack of questions.



