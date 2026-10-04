# Engineering Operations A/B Seed

This directory is the preparation record for a paired longitudinal comparison. arm-a and arm-b start from the same application, architecture facts, operational history, hook intent, pipeline checks, fixture bytes, and command expectations. The arms differ in how project policy and generated/managed artifacts are maintained and checked.

## Arms

Arm A is a capable conventional setup. The accepted rules live in docs/engineering/agent-rules.md. AGENTS.md and .claude/CLAUDE.md are manually synchronized full-text provider instructions; each carries a digest of the shared source. scripts/check_operations.py verifies both projections, the exact hook digest, named CI scalar commands against the hook, and the exact managed path/owner inventory. Python unit tests exercise the contract. The checker is project-owned, offline and literal; it does not infer semantics or claim to execute GitHub Actions.

Arm B uses the same facts in Markitect Rule and Skill resources. Markdown, Codex and Claude outputs are generated from canonical resources. Its Project Areas explicitly include rules, docs, modules, entrypoints and exact source inputs. The configured module check validates the exact hook digest and literal pipeline scalars. markitect-artifacts.yaml declares exact managed roots, one classification per file, tooling owners, and reasoned exclusions. The same Go tests and CI behavior remain in both arms. Markitect establishes declared structure and inputs; it does not establish Go behavior or provider runtime behavior.

Both arms have substantial human-readable architecture and operations docs, root and nested Go modules, cancellation-aware child execution, focused tests, a parser fixture, pre-commit, and CI. A root go test does not traverse the nested module; both hook and CI run the nested test command. Historical OPS-142/OPS-188 facts and the undecided shared-library boundary are identical. The parser fixture is identical and must not change during task 11.

## Task and evaluator handling

tasks/task-set.yaml contains twelve ordered, identical prompts with expected classifications, escalation flags, allowed prefixes, and affected-set labels. `parallel-task-set.yaml` separately forks P01/P02 for each arm from retained trial 1 after main task 06; they map to the already planned main tasks 07/08 and do not substitute for them. Per-fork evaluation checks the shared 01–06 state plus only that fork's card. Final integration checks both parallel changes together against cumulative frozen expectations for main tasks 01–08. The harness copies only arm-a or arm-b into a fresh actor checkout; it never copies task metadata, common/, project.yaml, this comparison record, or evaluator implementation into that checkout. The external evaluator and fixed objective drift vector remain outside this seed. Tasks 09–12 are holdouts.

The command contract is documented in tasks/README.md. Candidate binaries are supplied through PATH by the harness; no Markitect source or executable is copied into either arm. Baseline project commits and candidate commits are harness evidence and must be recorded by full SHA. This preparation did not commit either arm.

## Validation boundary

Arm A's project checker and unit tests, both root Go modules, both nested module tests, Arm B's structural check, renderer check, module helper, and artifact helper are the baseline local validation gates. The checks were run against the uncommitted working-tree fixtures using the frozen v0.13.0 candidate binary where applicable. Fixed-revision verify cannot be claimed before the harness commits an arm seed; v0.13.0 correctly rejects verify without --revision. A local check is technical evidence, not human acceptance or a trial result.

## Authoring and upkeep limits

Arm A requires updating its accepted source and manually synchronizing two provider entrypoints, then updating literal checks and ownership entries as needed. Arm B requires editing canonical YAML, rendering provider/document views, and updating typed hook/pipeline or artifact declarations. Both receive the same code/tests, facts, prompts, time and feedback budgets. Count setup, rule changes, projection repair, path accounting, failed checks, and recovery as human attention. Keep missing measures unavailable; do not infer tokens or attention from file counts, output bytes, helper calls, or elapsed time.
