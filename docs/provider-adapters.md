# Provider adapters

This document covers the built-in Codex and Claude file projections, configured adapters, and the reconciliation model supported in v0.10.0. Markitect renders Codex and Claude entrypoints only when `Project.spec.targets` selects them. In the earlier v0.9.1 model, `Skill` resources created `.agents/skills/<name>/SKILL.md` and `.claude/skills/<name>/SKILL.md`; `Agent` resources created `.codex/agents/<name>.toml` and `.claude/agents/<name>.md`. The current generic model represents that bundled AI-working vocabulary as a Domain. Canonical sources supply descriptions, text, explicit dependencies, and provider settings. Generated provider entrypoints link directly to canonical sources and do not require Markdown resource views. Unsupported fields fail strict parsing. `render --write` changes owned outputs; `check` reports missing, changed, stale, retired, and unregistered outputs.

## Additional source adapters (unreleased)

The first parallel wave adds separately built [GitHub](../cmd/markitect-adapter-github/README.md) and [Azure DevOps](../cmd/markitect-adapter-azure-devops/README.md) command consumers. They compare explicit canonical repository properties with exact, sanitized JSON captures staged through the existing command protocol. They have no live API calls, credentials or Apply capability. Their operation-free Plans report observed conformance findings; Verify fails for remaining differences or stale evidence. Capture validation proves only correspondence with the supplied recording, not its authenticity, authorization or current remote state.

These are source capabilities, not new built-in adapter types or binaries included in published v0.12.0. The .NET source adapter also rejects foreign namespaces on recognized MSBuild elements and relevant attributes as incomplete literal-reference evidence. [Wave evidence](validation/parallel-development-wave-1.md) records the tests and integration boundaries. No Core/schema/protocol meaning changes.

## Configured adapters and reconciliation

The v0.10.0 contract declares adapters under Project `spec.adapters`. Each entry names the adapter, its type and version, and its explicit mapping/configuration. These mappings are canonical model inputs. Adapters consume validated normalized data; they do not interpret raw resource YAML or silently infer ownership from generated files. The built-in `markitect-render` adapter covers local managed projections and needs no `spec.adapters` entry. Other adapter types are selected explicitly by their configured entries; they may observe a target and return a named, digested observation independently of the canonical desired model, so external drift remains visible even if the model has not changed.

Reconciliation separates four operations. `observe` reads a configured target and reports its state. `plan` emits a concrete, input-bound plan without applying it. `apply` requires the selected plan and explicit write intent, validates that it is still current, and executes only its declared operations. `verify` observes again and checks the result against the plan. The plan and observation identify their source snapshot, adapter configuration, and desired/observed digests; a successful operation does not imply semantic correctness or human acceptance.

For local generated files, the current CLI exposes this flow:

```powershell
New-Item -ItemType Directory -Force .artifacts/markitect/reconcile | Out-Null
markitect reconcile --repo . --action observe --adapter markitect-render
markitect reconcile --repo . --action plan --adapter markitect-render > .artifacts/markitect/reconcile/plan.yaml
markitect reconcile --repo . --action apply --adapter markitect-render --plan .artifacts/markitect/reconcile/plan.yaml --write
markitect reconcile --repo . --action verify --adapter markitect-render --plan .artifacts/markitect/reconcile/plan.yaml
```

`observe` and `plan` print YAML to standard output and do not write. Save the plan in the reserved `.artifacts/markitect/reconcile/` evidence directory before applying. `apply` is limited to the working tree and re-plans against current inputs before writing; stale or modified plans fail. Unmanaged destination collisions appear in `conflicts` and make the plan incomplete; it offers no operation to take over those files. The plan never deletes stale generated files automatically. Review removals separately.

### Supported command adapter contract

The generic command adapter supports observe, plan, and verify; capability-gated apply is available when explicitly configured. The separately built .NET reference adapter is read-only, checks mapped literal unconditional `ProjectReference` entries, and reports unsupported MSBuild semantics as incomplete. Neither adapter establishes evaluated build-system correctness or transfers human acceptance. See the [roadmap](implementation-plan.md) for the consumer proof and evidence limits.

A configured command adapter has `type: command` and exact `inputs`, `observe`, `plan`, and `verify` argv. It may declare `apply` together with `allowApply: true`. Each argv starts with a bare executable name resolved through `PATH`; Markitect passes literal arguments without a shell. The command runs in a temporary directory containing only the declared snapshot inputs, receives a `markitect.example.org/adapter-request/v1alpha1` YAML request on standard input, and must return exactly one `markitect.example.org/adapter-result/v1alpha1` YAML result document in its bounded combined output. The model DTO, selected source files, adapter mapping, action, and any preceding observation or plan are included in the request as appropriate. The saved plan uses `markitect.example.org/adapter-plan/v1alpha1`.

`target` is a non-secret identity for the external destination and is included in the adapter request and saved plan; it is required when `apply` is enabled. Use `parameters` for plugin-owned, reviewable non-secret mappings that shape how canonical resources correspond to that target. Secrets do not belong in canonical mapping or saved plan data; adapters should obtain credentials through their own secure external lookup. An adapter must not use mutable ambient environment state to select a different destination from the identity declared in the plan.

```yaml
spec:
  adapters:
    - name: delivery-state
      type: command
      version: v1alpha1
      config:
        target: stable-non-secret-target-id
        inputs: [domains/delivery.yaml]
        parameters:
          policyResource: engineering/delivery.example.org/v1/ReleasePolicy/main
        observe: [markitect-delivery]
        plan: [markitect-delivery]
        verify: [markitect-delivery]
        # Add apply and allowApply: true only when this adapter is authorized to write.
```

Commands return an adapter result with the supported protocol version, adapter name, action, model digest, status, and optional target, observations, findings, or operations. When a target is configured, the adapter echoes that exact target; Markitect compares it with the requested target and with the saved plan. `incomplete` and `failed` remain distinct from `complete`; only complete observation and plan results can proceed through planning and apply. Completeness describes evidence coverage: error findings still produce a nonzero CLI exit code, including when printing a complete plan.

Plans retain the captured observation and bind the source, semantic model, Project/adapter configuration, target, adapter executable digest, CLI identity, and observation digest. A command's planning stage must deterministically derive its result from the supplied model, captured files, configuration, and observation. Apply rechecks these values and observes again to reject external drift since planning. Verify first validates the retained observation and reproduces the plan from those captured inputs, rejecting altered results before invoking the verification command. It then returns a fresh verification result. These checks preserve the declared evidence basis; they do not authenticate a human review or make a configured executable trustworthy.

Execution has a timeout and combined stdout/stderr size bound. Defaults are two minutes and 1 MiB; configured limits are capped at ten minutes and 10 MiB. Adapter commands run with the caller's local authority; the temporary working directory and staged inputs are not an operating-system security sandbox. Do not enable apply unless the adapter's effects and credential access are appropriate for that execution context. Apply is not a multi-system transaction: after an interrupted or failed apply, observe the target before deciding what remains. `apply` requires an explicit plan and `--write` on the working tree.

`spec.providerAdapters` optionally configures shared entrypoints and a strict inventory:

```yaml
spec:
    targets: [codex, claude]
    providerAdapters:
        inlineAgentText: true
        strictInventory: true
        agentContract: docs/agents/README.md
        roleRegister: docs/roles.md
        ruleSources:
            change-review:
                - docs/rules/change-review.md
                - docs/workflows/review.md
        retiredSkills: [old-review]
        retiredAgents: [old-reviewer]
```

`ruleSources` maps each Claude rule entrypoint to at least one ordered repository-relative Markdown source. These paths must exist; strict inventory requires every local canonical `Rule` to have a mapping when Claude is selected. The Project owns each aggregate rule output; individual `Agent` and `Skill` outputs have one resource owner. `agentContract` and `roleRegister` are explicit shared paths; role pointers are emitted only for selected targets. An Agent entrypoint links to `agentContract` whenever it is declared. `inlineAgentText` copies canonical Agent text into selected provider outputs and rebases Markdown navigation from the canonical YAML directory; typed links remain aimed at canonical YAML and ordinary links at their real source paths; without it, adapters link to canonical Agent YAML. Strict inventory requires explicit Agent settings for each selected target and rejects provider files outside the generated set in the owned skills, agents, and rule directories. Retired names must be valid, unique, and absent from active resources; old files fail the inventory check. Remove those files deliberately in the same reviewed migration; `render` does not delete them automatically.

Markitect validates its output paths, source paths, declared settings, collisions, and drift. It does not infer mappings from prose links. A repository still owns its root `AGENTS.md` and `CLAUDE.md`, documentation navigation, hook policy, and other domain-specific checks. Generated provider prose is routing, not a second source of policy. A package's resources cannot generate host provider entrypoints directly; use a local wrapper.
