# The factory

**Markitect works as a factory with a quality management loop, without approvals** ([DEC-126](../decisions.md#dec-126-conversations-with-the-owner-jobs-for-everything-else)). The owner drops in ideas and designs at the start; from there the realization runs by itself, signals from outside are sorted, and the owner is called only for the decisions that are theirs.

| Factory | Markitect |
|---|---|
| The plan | The spec |
| The management | The owner, in conversations only |
| Production | Build jobs |
| The testing laboratory | Reviewer, counter-checks and control samples ([§7](trust.md)) |
| Maintenance shifts | Jobs on a schedule |
| Complaints, hints and proposals | Reports ([§9](reports-and-actions.md)) |
| A corrective action | A change on a branch |
| Goods out | The landing rule |

For developers the technical form is CI/CD: each action is like a pull request that lands when everything is green, and routines work like a dependency bot, on a schedule and on a branch of their own.

## Two kinds of work

- **Conversations** happen only with the owner: the first design, designing together when the owner wants, and choosing between proposals when a new decision is needed. A conversation never runs as a job. What it changes lands on a branch with the role of the owner ([§12](jobs-and-runners.md#rights-come-from-the-steps)).
- **Jobs** are everything else. A job starts from a commit, needs no open session, asks no one during the run, and ends with a branch, rows for the state and reports. What stays open becomes a report. It runs by hand, on a schedule or on an event, in the user's session, as a scheduled task, as a cloud task or in CI ([§12](jobs-and-runners.md#runners)).

## The jobs

| Job | Started by | Does |
|---|---|---|
| **Intake** | A signal from outside | Reads it isolated and writes a report of fixed form ([§9](reports-and-actions.md#intake)) |
| **Triage** | A new report | Sorts and rates it, sets the switch, escalates ([§9](reports-and-actions.md#triage)) |
| **Architect** | A report whose answer follows from what was decided, or an idea of the owner | Changes the spec on the action's branch |
| **Build** | A spec change on a branch, a `fail`, or a `status` that is not empty | Counter-checker, then builder, then reviewer, until the list is empty |
| **Review** | A schedule | A spot check of confirmed documents; rarely all of them ([§7](trust.md#spot-checks-and-full-reviews)) |
| **Control sample** | A schedule | Planted deviations in a throwaway copy, checked blind ([§7](trust.md#control-samples)) |
| **Bug hunt** | A schedule | Looks beyond the list; findings become reports |
| **Cleanup** | A schedule | Changes no behaviour: structure, doubled code, dead code, names |
| **Improve** | A schedule | Changes behaviour only where the spec leaves it free, and adds nothing to the spec; what should be fixed becomes a report |
| **Read and question** | A schedule, and after larger spec changes | Checks the spec itself ([§5](is-the-spec-good.md#the-spec-itself)); findings become reports |

**The build job.** For each changed document the counter-checker first writes its counter-checks from the spec alone. Then the builder realizes what `status` lists, and the reviewer confirms with the counter-checks as evidence. Suspect documents go straight to the reviewer. The job runs `status` again and repeats until the list is empty. It stops when three rounds in a row leave the list no shorter, so a list that swings down and up again ends too; what is left becomes a report, which escalates where needed. The job holds this loop: each role is its own step, and no agent drives another ([DEC-141](../decisions.md#dec-141-agents-get-tasks-not-process)).

**Slices.** A long list is cut into slices by folder, with no order, and each settled slice is merged into the job's branch, not into the main branch. What a later slice builds may make an earlier document suspect, and the loop clears it.

**Cadence.** How often each job runs is a default of the runner, not a rule of the tool, and the settings may change it. To start with: triage and build on their event; a spot check of a few documents on every review run; bug hunt, cleanup, improve and reading weekly; full review and control sample more rarely.

## The landing rule

One rule decides for every job whether its branch lands, with no question to the owner ([DEC-127](../decisions.md#dec-127-one-landing-rule-and-a-spec-change-lands-with-its-realization)). A branch lands when:

- `status` after it shows nothing that the target branch does not show too, apart from documents waiting on the owner;
- the tests and the counter-checks pass;
- the job wrote only its write area ([§12](jobs-and-runners.md#rights-come-from-the-steps)).

Otherwise the branch stays where it is, and there is a report.

**A spec change lands together with its realization.** Its realization must be finished and confirmed on the same branch, or `status` would show something new. So the main branch is always built and confirmed, apart from documents waiting on the owner: when a spec change reaches a document with an escalated report, the branch lands with everything else, and that one document stands on the main branch unbuilt and visibly waiting until the owner has chosen. `status` on the main branch is then a check for drift ([§6](the-state.md#what-status-lists)), and in CI it also starts a build job.

How a job ran is in its branch and its reports. The history holds only facts.

## Starting situations

- **A new project** starts with the first design: one long conversation that walks the areas of the template's area types with the owner, depth first, with the questions of their catalogs ([§5](is-the-spec-good.md#the-spec-itself)). For each area the architect proposes, with why and what else would do, asks only what the owner alone can decide, and writes the area at once. The owner sees the spec grow and has to be explicit, so nothing is passed through unread. The conversation ends with the spec on a branch and the build job starting; what is open becomes reports.
- **An ongoing project** runs the loop of reports and actions ([§9](reports-and-actions.md)).
- **An existing project without a spec** gets a first design in which the project answers instead of the owner. An architect job states what is as what should be, from every artifact: code, tests, documentation, scripts, pipelines and configuration. The question at each artifact is what it holds that should also be in the spec, so the spec is no transcription. Where a statement does not hold, either the spec misread the project and is corrected, or the project contradicts itself and the owner says which side holds, or the project does something nothing in it states and that looks like an accident, which the spec leaves unstated and reports. The owner settles these in a conversation, and a full review then fills the state. From there a confirmed defect stays a defect against the spec, and a build job repairs it.
