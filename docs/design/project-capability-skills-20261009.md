# Capability skills and native project entrypoints

Direct user request, 9 October 2026: improve Markitect Skills, CLAUDE.md, AGENTS.md and native installation guidance, using several skills for meaningful capabilities/work steps. This extends only the active Designer native-working package. Other owners, studies, automation, Main merge and releases remain stopped. The existing one native product-proof allowance is unchanged; no extra run or budget is implied.

## Source audit before changes

Audited source: Design checkout b61eb445fdceb851640d1fd8c7084a8b7b664403, internal/host/projectonboarding/render.go and onboarding.go, plus docs/project-workflow.md. The actual product checkout has no CLAUDE.md; this audit evaluates the generated adopting-project CLAUDE.md template and its shared workflow, not a nonexistent installed file. Product-repository authoring/engineering-change skills serve another purpose and should not be replaced indiscriminately.

The generated Claude and Codex entries are identical brief paragraphs. Each installs one markitect-model-first skill which directs the agent to load the entire mixed setup/exploration/Brownfield/readiness/delivery/recovery workflow. Commands and guarded lifecycle/authority distinctions are mostly concrete. The main weaknesses are coarse discovery, loading irrelevant stages, terse catchall metadata, and the need to align instructions with the actual forthcoming native worker mode. Existing preview/digest binding and marked-block merging already preserve unmanaged entrypoint prose and reject stale writes.

Editorial quality assessment of the generated entry + referenced workflow (not a runtime score):

| Criterion | Assessment | Reason |
| --- | --- | --- |
| Commands/workflows | 15/20 | Numerous source-backed commands, but no capability-local operational entry |
| Architecture clarity | 15/20 | Model/Manager/lifecycle boundaries explained; outer project agent vs native worker routing needs explicit treatment |
| Non-obvious patterns | 10/15 | Acceptance, exact digests and durable recovery covered; no focused troubleshooting/selection |
| Conciseness | 5/15 | Short entry delegates to a large workflow covering every stage |
| Currency | 15/15 | Audited against the current source; future native-working capability is not claimed |
| Actionability | 10/15 | Concrete command forms, but broad skill and multiple placeholder records increase navigation burden |
| Total | 70/100, B | Editorial assessment only; no agent behavior or installed release validated |

## Proposed product structure

Use six functional skills, grouped by distinct user intent rather than a skill per command or Manager role. Preserve markitect-model-first as a very short compatibility router pointing to the appropriate capability; it must not require loading all skills or the full manual. Root entrypoints should briefly identify the selected project model, route to skills, and state essential shared boundaries. They should not repeat the manuals.

| Skill | Trigger and outcome |
| --- | --- |
| markitect-setup | Initialize/adopt installation scaffolding; select and bind the real executable/runtime; onboard native guidance and diagnose setup. Do not silently reinitialize an existing model, invent rates, or change global provider settings. |
| markitect-model | Translate ordinary Work Items, bugs, conceptual changes or renames into intent/decisions, canonical YAML edits, impact and scoped readiness. Repair concrete schema/reference/coverage diagnostics within task authority, recheck, and retain accepted vs proposed distinctions. |
| markitect-brownfield | Understand and progressively model existing code from fixed actual inputs; select source/target/scope, delegate supported inference and apply model-only adoption. Record uncertainty instead of presenting inferred intent as acceptance. |
| markitect-deliver | Implement accepted ready work through the real Plan/Run/Review/Verify/Apply or composed deliver lifecycle, including dependent/parallel work through supported ownership scheduling. Use actual candidate workspaces and source receipts. Required verification is part of delivery, not optional loading of a separate audit skill. |
| markitect-audit | On request or a material drift finding, compare intended model/rules/contracts with actual code, tests and docs; use supported verification/reconcile mechanisms and report concrete gaps. Separate deterministic diagnostics, observed semantic evidence and missing evidence. Do not claim complete compliance from a green compiler. |
| markitect-recover | Inspect persisted status after interruption/failure; distinguish interrupted, known failed check, stale state and unknown invocation; resume or repair the same run when supported. Repair bounded known causes, reverify and continue; do not recreate completed work or raise budgets silently. |

Descriptions must discriminate these intents and enable normal implicit discovery. Keep bodies concise with concrete local references, not generic coding advice. Use supporting references for detailed JSON shapes and stage-specific commands. Avoid copied provider-independent manuals in every skill or accidental cycles between skills. The canonical maintained source owns each shared fact once; provider outputs must be generated deterministically from that source.

Suggested native entry content, refined to the actual implementation:

> This repository uses Markitect's selected .markitect/project.yaml model. For ordinary work, select the relevant local Markitect capability skill: setup, model, brownfield, deliver, audit or recover. Persist intent and canonical model changes before delivering implementation. Repair actionable model/check diagnostics within your delegated task and continue the existing workflow. Ask only for material unresolved intent or authority. Use the actual installed command/runtime, real checks and guarded Apply; recover existing work instead of replaying it.

This is a content proposal, not a fixed wording test. A setup entry must also explain how to bootstrap when the selector or installed skills do not yet exist.

## Native integration and migration

Generate Codex skills in the supported repository-local .agents/skills layout and Claude skills under .claude/skills. AGENTS.md and CLAUDE.md are concise provider-native entrypoints. Add .codex files only when the actual supported Codex configuration requires them, not to create a decorative duplicate or change personal account/security preferences. Keep ordinary project build/test/domain instructions owned by the adopting repository, rather than inventing generic commands.

Outer project agents may make delegated canonical changes through the guarded model-edit workflow. Inner Manager workers operate on their declared candidate responsibility; native helpers are ordinary scoped helpers, not duplicate Host-scheduled Managers. Do not teach implementation workers to bypass ownership, mutate the control plane or write adopting sources before guarded Apply. Instructions must match the real working mode and describe unsupported behavior honestly.

Preserve user-written entrypoint prose and unrelated skill files. Onboarding updates must remain previewed, digest-bound, idempotent and stale-safe. Migrate known Markitect-owned skill metadata along with bodies so outdated catchall descriptions do not remain. Treat conflicting unmanaged same-name skills or ambiguous ownership explicitly; do not overwrite them based only on the name. Keep the old router usable without two competing full workflows. References must resolve from the actual installed skill directory, including on Windows/CRLF.

## Verification and handoff

Implement through Designer's canonical product sources and regenerate outputs. Include focused validation of provider parity, native skill discovery metadata, valid frontmatter, relative reference resolution, new and existing-project onboarding, preservation of custom entrypoint prose/skills, metadata migration, repeated installation, target collisions and changed-source rejection. Test meaningful observable invariants, not copies of prose or exact headings. Include a small source-backed intent-routing check and walkthrough for a normal Work Item, a compiler diagnostic and an interrupted run. Do not use private study fixtures or claim simulated walkthroughs as real agent performance.

Existing native working proof can use the resulting installed guidance if implementation order permits; do not restart a consumed proof just for this instruction change. Return exact generated paths, source/check outcomes and capability limits. No automatic Main merge or release. The user has already requested these improvements; proceed within this scope without adding a general approval round for each document.
