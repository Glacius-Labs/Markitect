package projectonboarding

import "fmt"

type capability struct {
	name        string
	description string
	body        string
	reference   string
}

// capabilitySkills is the single maintained source for both native provider
// trees. Rendering these definitions keeps names, discovery metadata, bodies,
// and stage references in parity across Codex and Claude.
func capabilitySkills() []capability {
	return []capability{
		{
			name:        "markitect-init",
			description: "Initialize a new Markitect project or complete supported setup and onboarding in an existing project; bind the actual executable/runtime and preserve caller budgets and provider settings.",
			body:        "Use for a new project, installation setup, runtime binding, or repository-local guidance. Read [the init guide](references/operating-guide.md) for the relevant steps. Inspect existing state before any change; never reset an existing model, invent rates, or alter global provider settings.",
			reference:   "# Initialize and onboard\n\nUse `model` and `check` to inspect an existing project. For a new project, call `init` and review the returned file plan; write only through the explicit write form provided by its schema. Install repository-local guidance with `onboard`, preview first, and write only with its returned digest. Configure the local MCP client using the generated entrypoint's provider instructions.\n\nUse `config` and `doctor` to select and inspect the runtime: the Codex App Server by default, or a profile per role that may use a process executor. Setup does not change global client settings. Preserve existing task budgets and provider settings, review the exact runtime preview, and never invent rates or silently reset limits.",
		},
		{
			name:        "markitect-extract",
			description: "Extract a proposed Markitect model from existing code through bounded Brownfield discovery and adoption, preserving fixed source scope, evidence, and uncertainty.",
			body:        "Use when existing code must be inspected and progressively represented in the model. Read [the extract guide](references/operating-guide.md) for source selection, evidence ownership, and model-only adoption. Observed code is evidence about implementation, not accepted intent.",
			reference:   "# Extract a model from existing code\n\nSelect explicit source and target roots, fixed revisions, and bounded paths before discovery. Inspect actual files, configuration, history, and observed checks that support each proposal. Preserve evidence ownership and delegation bounds across Managers; metadata or a path name does not establish behavior. Record uncertainty and counterexamples instead of presenting inference as accepted intent.\n\nUse `adopt` for typed discovery and session stages, and its `run` stage for digest-bound Manager proposals and integrations. Review returned proposals and bindings. Keep initial adoption model-only; cleanup and source implementation are separate operations.",
		},
		{
			name:        "markitect-design",
			description: "Design project intent from a Work Item, bug, concept, rule, or rename; edit canonical YAML, assess impact, and establish scoped readiness without implementing the proposal.",
			body:        "Use when the requested work is specifically to clarify or change project intent, ownership, artifacts, checks, or model structure. Read [the design guide](references/operating-guide.md). Keep accepted decisions separate from assumptions and proposals; repair actionable diagnostics within delegated authority and recalculate impact and readiness.",
			reference:   "# Design model intent\n\nBegin with the desired outcome in ordinary language. Keep accepted intent, assumptions, alternatives, and unresolved decisions distinct. Inspect `model` and the closed `edit` input schema before editing canonical YAML. Preview the guarded mutation, inspect its structural report and `impact`, and write only with the returned digest. Re-run `check` and `ready` against the fixed project state. A draft is not accepted intent or implementation authorization.\n\nRepair concrete schema, reference, or coverage diagnostics within the task's authority. Ask only when unresolved intent, authority, or repository policy changes the outcome.",
		},
		{
			name:        "markitect-suggest",
			description: "Recommend possible improvements to a Markitect model from observed evidence while leaving canonical model files unchanged until a separate authorized design decision.",
			body:        "Use when asked for ideas, recommendations, or a model-improvement proposal rather than an accepted model edit. Read [the suggest guide](references/operating-guide.md). Keep the result proposal-only: do not edit or apply canonical model changes.",
			reference:   "# Suggest model improvements\n\nInspect the selected model and relevant repository evidence, then identify concrete gaps, alternatives, and uncertainty. Distinguish observed implementation from intended behavior; present suggested ownership, artifacts, rules, or checks as proposals with their evidence and tradeoffs. Do not write, apply, or accept canonical model changes in a suggestion-only task.\n\nIf the contributor selects a recommendation as intended behavior, use the design workflow under that authority; keep the original proposal and acceptance distinction visible.",
		},
		{
			name:        "markitect-configure",
			description: "Change supported configuration for an existing Markitect project, runtime, executable pin, or checks while preserving current budgets, provider settings, and unrelated configuration.",
			body:        "Use when an existing project's supported runtime, executable, budget, or check configuration must change. Read [the configure guide](references/operating-guide.md). Inspect current values first, preview exact changes, and preserve unrelated pins and settings; do not invent rates or silently reset limits.",
			reference:   "# Configure an existing project\n\nInspect the selected model, runtime, executable pin, budgets, provider settings, and declared checks first. Use `config` to preview the runtime (the Codex App Server by default, or per-role profiles) and `doctor` to inspect local prerequisites; write only with the exact preview digest. Run `check` afterward.\n\nPreserve caller-authorized budgets and unrelated configuration. Do not change global provider settings, invent cost rates, or replace an executable/profile merely because another option is available. New-project initialization and native skill installation belong to init.",
		},
		{
			name:        "markitect-implement",
			description: "Implement an ordinary Markitect Work Item end to end through design, checks, bounded execution, review, verification, and guarded Apply; repair compiler/test failures and resume after interruption.",
			body:        "Use as the default for an ordinary request to implement a bug, feature, or other authorized change. Read [the implementation guide](references/operating-guide.md), then chain the necessary operations autonomously to finish the request. Do not stop because design, check, verify, or apply has its own skill. Preserve the original authority, required reviews, and budgets.",
			reference:   "# End-to-end implementation and worker boundary\n\nCarry an authorized Work Item through only the intent/design, structural check, readiness, implementation, review, verification, and guarded Apply steps it needs. Use Markitect MCP from the outer Codex or Claude Code agent; `deliver` advances a ready scope through the durable workflow. Do not turn separate operation skills into extra prompts or approval rounds.\n\nMarkitect schedules each model-declared Manager once. Its default inner Manager and reviewer runtime is " + Pins().innerRuntime() + "; the project runtime may select a process executor per role. Native workers get fresh owned candidate workspaces and ordinary standard file, shell, and test tools under the caller's configured permissions. The Host exposes bounded native helper starts according to the selected runtime policy and records observed helper lifecycle; helpers are not additional model-declared Managers. Inner workers must not edit the canonical control plane or schedule duplicate Managers.\n\nRepair compiler or required-check diagnostics within the assigned mandate and budget, then continue the same run. Required verification is part of delivery. Review the exact candidate and use guarded Apply; do not skip checks, replay unknown outcomes, or treat uncommitted work as accepted. See the shared [recovery reference](recovery.md).",
		},
		{
			name:        "markitect-cleanup",
			description: "Refactor an existing realization while preserving accepted behavior, public contracts, ownership, and required checks; use the supported cleanup operation and verify the candidate.",
			body:        "Use for refactoring, maintenance, or quality improvements that preserve accepted intent. Read [the cleanup guide](references/operating-guide.md). A requirement or public behavior change is design work; implementation and required verification remain part of the cleanup request.",
			reference:   "# Cleanup while preserving accepted behavior\n\nInspect the accepted model and fixed snapshot first. Use `plan` with the cleanup operation and bounded goal, then review ownership and affected paths. Continue the same run with `run`, `verify`, and guarded `apply` (preview, then write), or use `deliver` for an acknowledged scope.\n\nPreserve accepted behavior, public contracts, owner boundaries, and declared checks. If cleanup reveals a behavior requirement change, route that part through design and readiness before implementation. Resume persisted work using [the shared recovery reference](../../markitect-implement/references/recovery.md). A cleanup plan alone does not prove implementation correctness.",
		},
		{
			name:        "markitect-verify",
			description: "Assess a candidate against accepted Markitect intent, contracts, and configured checks; collect real evidence and report drift or missing evidence without silently repairing semantics.",
			body:        "Use when asked to verify realization or audit drift. Read [the verify guide](references/operating-guide.md) for supported fixed-snapshot checks and evidence limits. Report observed results, deterministic diagnostics, and evidence that was not run separately.",
			reference:   "# Verify realization and report drift\n\nBind the assessment to the accepted model and explicit candidate or repository revision. Use `verify` with a run for a durable run, and `verify` with a revision, `check`, or `check` with coverage for the corresponding selected-snapshot assessment. Compare declared artifacts, ownership, rules, contracts, and checks with actual source, tests, and documentation; include exact paths and results.\n\nA passing compiler establishes only that compiler check. Structural diagnostics and fixed checks do not alone establish semantic truth, full compliance, or human acceptance. When assessment only was requested, report concrete gaps and missing evidence without silently changing semantics. For interrupted verification, consult [the shared recovery reference](../../markitect-implement/references/recovery.md).",
		},
		{
			name:        "markitect-apply",
			description: "Apply an already verified Markitect candidate using real preflight, freshness, branch/source/candidate/verification bindings, and required reviews.",
			body:        "Use for an explicit request to apply an already verified candidate. Read [the apply guide](references/operating-guide.md) and confirm the exact plan, run, candidate, source, branch, and verification bindings before guarded Apply. Apply does not invent implementation or authorize merge, deployment, or release.",
			reference:   "# Apply a verified candidate\n\nInspect the persisted plan, run, candidate, source revision, branch/worktree state, and verification result. Call `apply` without write to obtain the exact apply fields, then `apply` with write and those bindings. Do not reconstruct or substitute identifiers. Apply only the verified candidate after required reviews.\n\nA task that says “apply” may instead mean an ordinary product operation: inspect the request and project semantics to disambiguate it. Guarded candidate Apply does not imply merge, deployment, release, or other publication authority. If execution was interrupted, inspect the same run with `status` and consult [the shared recovery reference](../../markitect-implement/references/recovery.md) before retrying.",
		},
		{
			name:        "markitect-check",
			description: "Run structural Markitect model/compiler checks for schema, references, ownership, and inventory or coverage; interpret and repair actionable diagnostics within authority, then recheck.",
			body:        "Use when asked to check the model or diagnose a structural/model compiler error. Read [the check guide](references/operating-guide.md). A successful structural check does not prove semantic implementation correctness; use verify for realization evidence.",
			reference:   "# Structural model checks\n\nRun `check` against the intended fixed snapshot. Inspect schema, references, ownership, inventory, and coverage diagnostics. Repair only actionable causes within delegated authority, then run the check again. Use the tool's current closed input schema and `model` rather than guessing a resource shape.\n\nDistinguish a structural/model compiler error from an implementation compiler or test failure. This operation checks model structure and repository coverage; it does not prove that code realizes accepted behavior. Use `verify` with a run or a revision when realization evidence is requested.",
		},
	}
}

func recoveryReference() string {
	return `# Recover persisted work

Inspect persisted status and exact receipts before taking action. Distinguish interrupted work, a known failed check, stale state, and an invocation whose outcome is unknown. Resume the existing run where the current lifecycle supports it; repair a known failure in the same run, rerun required checks, and continue without replaying completed Manager work.

Preserve already authorized scope, time/start/retry limits, and cost budget. If a stale digest or unknown invocation prevents safe continuation, inspect persisted state and candidate bytes before preparing a fresh preview. Do not automatically replay an operation with unknown external outcome, reset limits, skip review checks, or treat a failed or uncommitted candidate as accepted.` + "\n"
}

func capabilityGuide(cap capability) string {
	return fmt.Sprintf("# %s operating guide\n\n%s", cap.name, cap.reference)
}
