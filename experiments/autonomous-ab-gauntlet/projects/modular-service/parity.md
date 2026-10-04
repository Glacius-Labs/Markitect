# Pairing and protocol record

Both arms contain the same application source, Go module manifest, focused package tests, architecture boundary test, CI workflow, and service architecture document at initialization. Arm A supplies concise `AGENTS.md` and architecture guidance. Arm B preserves that guidance and adds a Markitect Project, an exact local architecture package pin, Product/Core/Common/Module/Interface/Feature/Handler/UseCase resources, a normative Rule, a task Workflow, an implementation Skill, source-file ownership through Rule inputs, an artifact-coverage config, and Markdown/Codex projections.

The initial service has three independently owned modules: Orders, Inventory, and Billing. Each depends only on Core/Common, and Orders consumes Inventory and Billing through contracts. Both arms run the same Go architecture test and `go test ./...`; no arm receives a weaker source-level check. Markitect supplies policy/context and artifact-input coverage in B, but it does not analyze Go behavior. Product checks remain the implementation oracle.

`task-set.yaml` contains exactly 12 sequential cards. Each card starts from the result of its predecessor, including 07 then 08 and 09 after 08. Cards 09–12 are holdouts. Card 10 requires an owner decision; no tool result, exception, or guessed owner can satisfy that requirement. Actor prompts are stored once in the shared task set so the exact prompt bytes are identical between arms.

The evaluator lives under `oracle/` and must not be copied into actor workspaces. It runs outside each actor repository with the frozen task ID and base revision, enforces allowed paths, executes hidden behavior vectors/source assertions, runs the same ordinary Go checks, and records fixed drift checks. B's project checks use the fixed CLI and artifact helper on PATH; public full acceptance also runs the Go vet gate in both arms. It does not score prose quality or agent self-report. Task 10 produces an escalation-pending result until the human decision is separately recorded.

## Limits to preserve in the study report

This is a preparation artifact, not an A/B result. The seed adds Markitect authoring and projection upkeep up front, so report preparation time and repeated maintenance separately from implementation. The 12 cards exercise one synthetic service, not a representative sample of all repositories. Three trials per arm are the initial plan, not proof of generality. Record model/version, reasoning effort, actor settings, fresh-session state, task order, actual active intervals, human attention, interventions, failed runs, and missing measures. Preserve source revision, frozen binary digest, task/evaluator digest, per-task base, and all raw outputs. Report architecture conformance separately from runtime correctness and human acceptance.

## Preparation cost and constraints

Preparation labor and active minutes were not instrumented, so authoring cost is unavailable; the agent wall time in the transcript must not be treated as human-attention time. The B treatment adds three constitution resources, nineteen architecture resources, one Project/package pin, and generated Markdown/Codex projections. This artifact records the fixed source revision and CLI, but not a human authoring-time baseline. Keep that missing measure explicit in the study results.

The parallel overlay contains two additional cards P01/P02 that both fork from the fixed post-06 state. They do not replace or reorder the 12 longitudinal cards. The main sequence remains 01 through 12 with one task state feeding the next.
