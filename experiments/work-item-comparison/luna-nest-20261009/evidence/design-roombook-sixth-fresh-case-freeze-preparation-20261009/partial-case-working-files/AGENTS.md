# Project rules

Read README.md, BACKLOG.md, STATIONS.json and QUALITY.md. The current station is .study/station.json. All requirements are public from the outset; implement only the released station, then return so the fixed dispatcher can retain its state. Work on a feature branch, preserve existing behavior, add meaningful tests, and merge finished work into this repository's main. This main is isolated; never push or change another repository.

Decide routine implementation details yourself. You may plan, document, use ordinary tools, delegate, review and repair within the common cell time. Every executing helper and reviewer must use gpt-6-luna with high reasoning; inherit that configuration and record any departure rather than silently changing it. Use at most four simultaneous helpers and depth two.

Use only this project, its declared tool installation and its own temporary files. Do not read past study results, other cells, chats, personal memory, private evaluation materials or global process/agent inventories. Shared host rights do not make that an OS boundary. Report unexpected foreign exposure and stop. Do not inspect credentials or change global tool settings.

Keep a concise WORKLOG.md: work item, checks, failures, repairs, reviews, model/document changes if applicable, merge outcome and unresolved decisions. Keep code, tests and readable documentation consistent. No predetermined architecture is required.

At each station handoff write .study/completion.json with station equal to the released station ID, status "complete" or "blocked", completed work-item IDs, and brief remaining issues. Commit it with the work. This is completion metadata, not a quality verdict. An interrupted run resumes from this repository and its own session. Before every commit run git diff --cached --check immediately. WORKLOG.md may point to PROGRESS.md rather than duplicate it.

The third station requires actual collaboration by at least two executing agents on independent work groups, with overlapping work intervals and integrated contributions. Choose your own native planning, prompts, roles and branches. Record actual agent IDs, requested and observed model/effort where available, start/end times, contribution/merge SHAs, conflict handling and reviews in TEAMWORK.md; retain native receipts where available. A request for parallelism alone is not completion. If your installed product cannot support this, report the missing capability rather than imitate a team. The dispatcher supplies no semantic help.

<!-- BEGIN MARKITECT MODEL-FIRST -->
For a short Work Item, issue, bug, idea, or change request, use markitect project with the repository's .markitect/project.yaml selector, use the repository-local Markitect model-first skill and follow the shared [workflow](.markitect/workflows/model-first.md). Begin in ordinary conversation, preserve decisions and open questions, satisfy the repository's actual readiness review and acknowledgement policy using your own authorized actor, then use the accepted model and existing Plan/Run/Verify/Apply lifecycle. Ask only for material intent or authority outside the Work Item's delegation, or review the repository policy explicitly reserves for a human. Resume persisted work without replaying completed work.
<!-- END MARKITECT MODEL-FIRST -->

