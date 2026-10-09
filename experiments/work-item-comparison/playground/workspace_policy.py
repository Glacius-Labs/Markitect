"""Mechanical rendering of the public workspace-snapshot completion policy."""

from __future__ import annotations


class PolicyRenderError(ValueError):
    """A public policy input no longer matches the reviewed rendering rules."""


# Exact old/new pairs make policy adaptation explicit and fail closed when the
# public baseline changes. Functional requirements and acceptance criteria are
# intentionally not rendered or edited here.
_REPLACEMENTS: dict[str, tuple[tuple[str, str], ...]] = {
    "AGENTS.md": (
        (
            "Implement the released work and its dependencies, check the results, merge finished work to this isolated main, then return at the station boundary. The controller retains a snapshot and releases the next fixed wave.",
            "Implement the released work and its dependencies, check the results, then return at the station boundary. The controller retains an immutable snapshot of the actual workspace and releases the next fixed wave.",
        ),
        (
            "Work on feature branches, preserve existing behavior, add meaningful risk-based tests, diagnose and repair ordinary failures, and rerun affected checks. A failed command, product error or ambiguity answerable from this project is not automatically a stop. Keep code, tests and readable documentation consistent; no predetermined architecture is required. Merge checked work to main and verify the integrated result.",
            "Choose your work organization freely, preserve existing behavior, add meaningful risk-based tests, diagnose and repair ordinary failures, and rerun affected checks. A failed command, product error or ambiguity answerable from this project is not automatically a stop. Keep code, tests and readable documentation consistent; no predetermined architecture is required. The controller snapshots the actual workspace at each station boundary; no branch or merge is required.",
        ),
        (
            "items, assumptions, branch/commits, check commands/results/logs, failures/repairs, reviews, merged/open work, model/doc upkeep if applicable and the next step",
            "items, assumptions, workspace changes, check commands/results/logs, failures/repairs, reviews, open work, model/doc upkeep if applicable and the next step",
        ),
        (
            "S3 requires at least two executing agents working on independent groups with overlapping intervals and integrated contributions.",
            "S3 requires at least two executing agents working on independent groups with overlapping intervals and both contributions present in the station workspace snapshot.",
        ),
        (
            "Choose native planning, prompts, roles and branches yourself.",
            "Choose native planning, prompts and roles yourself, and organize implementation as you choose.",
        ),
        (
            "Keep TEAMWORK.md with actual identities, requested/observed model and effort where available, intervals, contribution and merge SHAs, reviews/conflicts and native receipts.",
            "Keep TEAMWORK.md with actual identities, requested/observed model and effort where available, intervals, workspace contributions, reviews/conflicts and native receipts.",
        ),
    ),
    "QUALITY.md": (
        (
            "The independent final assessor examines the frozen main commit, checks the declared requirements and legacy behavior, executes its own additional boundary/regression cases drawn only from these public requirements, and reviews source and documentation.",
            "The independent final assessor examines the immutable final workspace snapshot, checks the declared requirements and legacy behavior, executes its own additional boundary/regression cases drawn only from these public requirements, and reviews source and documentation.",
        ),
        (
            "Critical required behavior failure or missing main integration prevents complete success.",
            "Critical required behavior failure or a missing/invalid workspace snapshot prevents complete success.",
        ),
        (
            "Record setup, model preparation/upkeep, implementation, tests, failed attempts, repairs, reviews, merge and oversight.",
            "Record setup, model preparation/upkeep, implementation, tests, failed attempts, repairs, reviews, snapshot and oversight.",
        ),
        (
            "Assess each retained station against its released requirements, then the final rename across active behavior/interfaces, implementation, tests, documentation/config",
            "Assess each retained station workspace snapshot against its released requirements, then the final workspace snapshot across active behavior/interfaces, implementation, tests, documentation/config",
        ),
        (
            "actual team overlap/starts/contributions/merges versus unknown observations",
            "actual team overlap/starts/contributions versus unknown observations",
        ),
    ),
    "AGENTS.fragment.md": (
        (
            "Integrate and check the result yourself.",
            "Check the result yourself; the controller retains the actual workspace at each station boundary.",
        ),
        (
            "Work on feature branches and merge finished checked work to this isolated main. At each fixed wave boundary report completed/open items, main and contribution commits, checks and genuine resource blockers.",
            "Organize work as you choose; branches and merges are not required. At each fixed wave boundary report completed/open items, workspace changes, checks and genuine resource blockers.",
        ),
    ),
    "task-prompt.txt": (
        (
            "Hier liegt das Backlog. Implementiere die Arbeit, halte die Projektregeln ein, prüfe die Ergebnisse und merge die fertigen Änderungen nach main.",
            "Hier liegt das Backlog. Implementiere die Arbeit, halte die Projektregeln ein und prüfe die Ergebnisse. Der Controller sichert an jeder Stationsgrenze einen unveränderlichen Snapshot des tatsächlichen Arbeitsbereichs; ein Git-Merge ist nicht erforderlich.",
        ),
    ),
}


def render(name: str, text: str) -> str:
    """Render one supported public instruction file for workspace snapshots.

    Each reviewed source clause must occur exactly once. This prevents a future
    public-policy edit from silently producing a partly converted study input.
    """
    if not isinstance(name, str) or name not in _REPLACEMENTS:
        raise PolicyRenderError(f"unsupported workspace policy input: {name!r}")
    if not isinstance(text, str):
        raise PolicyRenderError("policy text must be a string")
    result = text
    for old, new in _REPLACEMENTS[name]:
        count = result.count(old)
        if count != 1:
            raise PolicyRenderError(
                f"{name} expected exactly one reviewed source clause, found {count}: {old[:100]!r}"
            )
        result = result.replace(old, new, 1)
    return result
