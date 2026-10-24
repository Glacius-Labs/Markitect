# Return to one container per Turn, 21 October 2026

## Origin and status

Direct owner decision in the Markitect concept/status conversation. This supersedes the selected Manager-container reuse direction of IDEA-018. The present change records intent and assignments; Session A owns the targeted runtime/self-model migration. Adoption does not mean the rollback has already been implemented or verified.

## Owner decision

> Dann gehen wir zurück zu einem Container pro Turn und behalten die anderen Verbesserungen. Halte das im Konzept fest, passe die Aufträge an und informiere den Apply-Ablauf-Chat, damit Claude nicht weiter an der Wiederverwendung arbeitet. Der Apply Chat soll mir dann einen Prompt für Claude Session A geben, damit er das wieder zurückbaut

## Recorder notes: observation and rationale

The comparison used B0 `b9cdd6d893fa9487a09016a3520c3c3442660c90` and B1 `fd94e365f9c1c060c9419be67b70ca072b6c1128` (reuse integrated by PR #572). At inspection on 21 October:

- `SA-B0-SN-3` completed assessment, Implement and Verify with `success` and publication `no_change`, taking 731 seconds. Its external assessment script failed with a diagnostic-shape error, so this is an observed product result, not a fully passed external evaluation.
- `SA-B1-SN-1` ended `incomplete` after 371 seconds, before Verify: `capture of the stand at the end of p2 is uncertain: workspace volume remains mounted by 3 container(s)`. No publication occurred. This is a lifecycle integration failure caught by the capture guard, not evidence of an unsafe publication or a speedup.
- `SA-B0-SQV-2` and `SA-B1-SQV-1` identified the missing maximum-name-length policy during Model assessment and stopped without publication. The external scoring did not yet confirm the expected outcome; preserve that discrepancy instead of declaring the whole comparison passed.

The inspected reports/logs were under `/home/user/mk-sa/` in WSL Ubuntu. A must retain the actual evidence in its closing handoff; this source record is not a replacement for those reports. Reuse was observed and real-container checks had passed in #572, but no reliable whole-workflow benefit was established. The owner chooses lower lifecycle complexity for the current stabilization phase. These observations do not prove that reuse can never work; future reconsideration needs a measured bottleneck and a new owner decision.

## Recorder notes: scope and coordination

Use a fresh container for each Agent Turn, with observed cessation/removal before accepting stable work or capturing a candidate. Manager identity, bound context and the provider's supported conversation/home continuity remain independent of container lifetime. Keep fresh independent reviewers, separate checks/probes/helpers, enforceable rights, cancellation limits and guarded publication.

Stop further reuse development, including held-instance/capability-cache fixes and B2 optimization. Preserve unrelated improvements: direct/staged composition, mandatory Model assessment and closed Runs, context delivery/rehydration, observation kinds/finding closure, and independent safety fixes that remain relevant to per-Turn execution. Remove reuse-only machinery rather than keep a permanent switch between two container strategies. Do not reset Main or revert unrelated integrated work wholesale.

The [workflow](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-workflow.md#per-turn-container-baseline) owns the target. The [A–D handoff](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/development/claude-direct-runtime-handoff-20261021.md#container-decision-one-fresh-instance-per-turn) and [delivery plan](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/implementation-plan.md#per-turn-container-rollback) own the bounded assignment. The existing `Apply-Ablauf nachvollziehen` chat (`01a157ed-1baf-7401-a1d0-9dc760529eea`) must receive the decision and integrated sources and provide the owner a prompt for Claude Session A. This documentation starts no provider series and resets no consumed allowance.
