# Native Codex work for project Managers

## Decision

Keep `proposal-only` as the default project runtime for compatibility. Add an explicit, opt-in `native-work` mode for Manager executors using the real Codex CLI 0.162.0 with `gpt-6-luna`, `high` reasoning, and the `luna-high` profile. Native setup requires the project's Codex onboarding to be present before setup. It uses the caller's normal profile, authentication and rules; it does not rewrite global configuration or claim OS isolation.

The outer conversational model author remains responsible for the user interaction and initiating project operations. Markitect schedules each model-declared Manager exactly once. The native Codex process is that bounded Manager worker, running in its own fresh candidate workspace; it is not an outer-author substitute and cannot schedule another Manager. Its normal file, shell, test and Git tools are available within that workspace under the caller's local permissions.

## Instruction and candidate boundary

For each native invocation, Host selects only the exact existing onboarding-owned instruction paths declared by project `ToolPaths`: root `AGENTS.md`, the generated Codex skill files and explicitly declared reference Markdown. It does not infer prose-link dependencies, read global skill/configuration directories, or include arbitrary descendants. The exact bytes and Git modes must match the fixed accepted project revision and also match absolute-path/mode/digest pins in the worker's `RuntimeFiles`; existing plan, fingerprint, resume and Apply freshness guards therefore cover them. The canonical project model remains YAML; generated provider `SKILL.md` files retain required provider-discovery frontmatter and are not an alternate canonical authoring format.

The current operation skill set is `markitect-init`, `markitect-extract`, `markitect-design`, `markitect-implement`, `markitect-cleanup`, `markitect-verify`, `markitect-apply`, `markitect-check`, `markitect-suggest` and `markitect-configure`. Their generated `SKILL.md` and declared `references/operating-guide.md` files, plus the model-first entry and recovery reference, are selected only by exact ToolPaths. The native workspace also contains the already bounded Manager inputs. The adapter validates paths, modes, digests, UTF-8, inventory size, symlink/reparse conditions, allowed ownership and control-plane exclusions before harvesting candidate files.

The adopting checkout remains untouched during execution. Host still validates candidate scope and ownership, stale fixed-source bindings, required checks and separate reviews, then uses the existing Verify and reviewed guarded Apply controls. Review outputs remain proposals and do not authenticate or record human acceptance. Deletions and binary candidate outputs are unsupported and rejected.

## Receipt and helper limit

The typed `nativeWork` receipt records workspace base/final and delta digests, changed paths, observed tool-call count, and helper starts/accounting. These workspace manifests are adapter claims bound to response stdout; the Host does not independently reconstruct the adapter's full filesystem inventory from those digests. The separately recomputed candidate delta and the existing candidate ownership/freshness/apply guards remain authoritative for what Markitect accepts. A receipt is execution provenance, not proof of semantic correctness or full OS process accounting.

Native helper agents are currently disabled. Codex CLI 0.162.0's `exec --json` stream does not carry a complete spawn/wait lifecycle record; observed function calls and returns alone do not prove helper count or resource use. Setup fixes the helper limit to zero, and any nonzero prelaunch helper limit fails. `helperStarts: 0` records this configured tool surface only; it must not be presented as a count of all child processes. A future helper mode requires a complete, trustworthy lifecycle and resource-accounting source before it can be enabled.

## Setup and evidence boundary

Initialize the model-first project and apply `project onboard` first, then preview and apply the native `project setup` edit through its exact digest. The command requires explicit caller-selected input/output price weights and a positive maximum-cost budget. Those weights support local cost estimation; they are not a provider quote, invoice, or hard provider billing cap. Normal `proposal-only` setup remains available without the native profile requirement.

The [workflow guide](../../project-workflow.md#opt-in-native-codex-work) and [adapter contract](../../../internal/tooling/codexrunner/README.md#opt-in-native-work) describe the current user-facing setup and execution boundary. The [native-work validation record](../../validation/project-native-work-2026-10-09.md) binds only the source revision, focused checks, and any bounded runtime outcome it records. Configuration and passing source checks alone are not a real native proof, semantic-quality result, human acceptance, or comparative advantage. Broader adoption and any one-shot product proof remain separately governed.
