# Continuous change (lane 10)

This lane designs how external triggers become tasks, model changes and Applies, so that not every change starts by hand. The owner's goal, 27 October 2026: "at some point we want external triggers being converted into something like tasks, which result in model changes and applications of it. Not all of the changes should be done manually."

The lane is design only. It builds nothing until the owner says so. This page holds the lane's scope, a design sketch and its open owner questions, and links to the pages that own the details.

## Scope

- **Owns:** the loop around Apply: trigger sources, intake, triage, the queue of tasks, when an Apply starts, and the limits of that loop.
- **Uses, does not own:**
  - How a model change is written and judged belongs to the modeling-skills lane (lane 7).
  - The commands the loop calls belong to the commands lane (lane 9).
  - How a verified result lands on the target branch, and how a landing is reverted, belong to the Git variant (lane 3, [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)).
  - Where the owner sees the inbox and briefings belongs to the Cockpit lane (lane 11).
  - How the loop is measured belongs to the evaluation suite (lane 4).
- **Existing frame:** this is the "later axis" of [DEC-004](../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis): idea, work item, trigger or support ticket → Model. Its fully delegated form is Government ([IDEA-002](../concepts/register.md#idea-002-government-delegated-model-maintenance)). Nothing on this axis is adopted yet.

## Sketch

### Trigger sources

| Source | Example | Usual outcome |
|---|---|---|
| Issues and tickets | A bug report, a feature request | Repair or model change |
| Incidents and logs | An alert, an error in production logs ([IDEA-017](../concepts/register.md#idea-017-operational-check-through-the-managers)) | Mostly repair |
| Dependency updates | A new library version, a security advisory | Repair, or a model change if the model pins the version ([DEC-033](../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)) |
| Model dependencies | A new version of an imported Library | Model change |
| User requests | A chat request | Mostly model change |
| Shared Notes | A side finding from an earlier Run | Owner inbox only: a Note is "not a ticket to be executed automatically" ([§19.2](../concepts/v1-concept.md#192-no-automatic-authority)) |

A scheduled Verify of the unchanged model could be a further source; it is not a V1 command and would be lane 9's to design.

Each source is an adapter behind one small intake contract, as Git is the abstraction for remotes: no Markitect logic depends on GitHub, GitLab or Jira.

### From trigger to Apply

```text
trigger → intake → task → triage
                            ├─ no action / duplicate / out of scope → close with a reason
                            ├─ repair: a named Definition already states the expected behavior
                            │     → Apply on the unchanged model, with the task as an outside Finding
                            ├─ model change: intent is missing, wrong or silent
                            │     → proposed model change (lane 7) → acceptance → Apply
                            └─ unclear or a decision only a person can make → owner inbox
Apply result → lands on the target branch (lane 3) or ends with a model blocker
            → the task closes with its evidence; the source is updated
```

- **Intake** turns a trigger into a task: what was observed, its source, a link back, and a dedupe key. Several triggers about the same thing merge into one task. The raw trigger text stays at intake; see the safety section.
- **Triage** is an agent judgment over the current model. A task is a repair only when triage can name the Definition and the expected behavior that Definition already states, the test of [§18.3](../concepts/v1-concept.md#183-findings-as-an-actionable-handover). If the model is silent on a behavior at the project boundary, that is a model gap ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)), so the task becomes a model change.
- **Repair** needs no model edit, but it needs a concept change. Today Findings enter Implement only from a Verify Reject, and the host sets their origin from the Manager or check that produced them ([§10.4](../concepts/v1-concept.md#104-apply-flow), §18.3). With no outside Finding, [DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model) rejects any change to an input already verified under the same model, and Verify does not see the task. The repair path therefore needs an Apply input of outside Findings that Implement receives and that Verify independently confirms as fixed. The [outside Findings design](../design/outside-findings.md) proposes it; lane 9 owns the command side.
- **Model change** produces a proposed model revision, checked by the compiler and judged by its reviewers. It becomes the Apply's input only once accepted.
- **Batching:** accepted changes can be collected into one model revision and one Apply, so that a busy day does not start twenty Applies.

### Where people stay in the loop

- The owner reviews model changes, not code diffs ([DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review), [PRM-004](../concepts/register.md#promises)).
- The model stays authoritative ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)). The loop never edits the model to make an Apply pass. A model blocker ends the Apply and becomes an owner question in the inbox.
- Autonomy grows in steps, each one an owner decision:

| Step | Repairs | Model changes |
|---|---|---|
| 0, today | The owner starts `markitect apply` | The owner edits the model with a coding agent and commits it |
| 1 | Automatic: intake, triage, Apply, landing | Proposed automatically; the owner accepts each, or a batch, in the inbox |
| 2 | Automatic | Changes inside a mandate the owner grants are accepted by agents; the owner gets a briefing ([IDEA-009](../concepts/register.md#idea-009-regular-briefings)). This is Government. |

- **Protected Packages.** The owner can mark Packages, such as billing or CI configuration, whose changes always wait for acceptance, even repairs. Impact before a run comes only from declared relationships, and dependencies in code are a known gap ([PRM-001](../concepts/register.md#promises)). So the check runs after Verify: landing waits when the verified candidate changes a file that the registry assigns to a protected Package. It is deterministic and needs no risk classifier.
- **Evidence moves the line.** A step widens only when landed changes show no more reverts or incidents than before.

### Safety and limits

- **Trigger text is untrusted.** Issue bodies, logs and advisories are data, never instructions. Intake rewrites each trigger into a Finding that references the model: the Definition, what was observed, what is expected. The raw body never reaches a Manager. Anyone who can open an issue must not be able to steer agents, widen scope or reach secrets. A trigger grants no authority; only the owner's acceptance or mandate does.
- **Data in triggers.** Logs and tickets can carry secrets or personal data. Intake redacts them before anything reaches a Finding, a Note or stored evidence, and the raw trigger is kept only as long as needed.
- **Budgets.** Triage and proposals spend provider turns too, so each has its own limit, beside the Apply budget: per task, per day and per Apply. When a limit is reached, the loop stops and reports; it does not queue silently.
- **One Apply per target branch at a time.** Further tasks wait or batch. What happens when the target branch moves during a run stays with lane 3.
- **Kill switch.** The owner can pause the whole loop and stop every running Apply, which ends it incomplete with its evidence, as an interrupted `markitect apply` does today.
- **Traceability.** Every landed change names its task, its triggers, its model revision and its Apply evidence.
- **Undo.** Results land without owner review, so a wrong landing needs a fast, plain way back. For a repair, revert the landing. For a model change, revert the model revision and its realization together, or open a model-change task; reverting only the code leaves the model ahead, and the next Apply would make the change again. Projects with no way to notice a bad landing, such as monitoring or alerts, stay at step 0 or 1 with owner acceptance.
- **Circuit breaker.** When the revert or incident rate of landed changes rises, the loop drops back a step by itself, for example from automatic repairs to proposed repairs, until the owner widens it again.
- **Model quality.** Many small automatic proposals can bloat or decay the model, the risk the owner named for Government. Proposals pass the same model-quality judgment as hand-written ones (lane 7).
- **Unchanged rules.** CLI account login only, never API keys. No release or deployment. Runs stay closed: no question reaches the owner during a run.
- **No daemon in Core.** The loop is an outer operator that calls bounded Markitect commands, from CI or a scheduler, as the vision describes ([executable governance](../vision.md#what-executable-governance-means)). Core stays a set of commands.

## State

- 27 October 2026: lane started; this sketch written. Nothing built.
- 27 October 2026: on question 1 the owner answered "Yes: design outside Findings as a concept 1.0 amendment, with the command side in lane 9. Still build nothing." The [outside Findings design](../design/outside-findings.md) is that proposal; it is not adopted.
- The owner's commission of 27 October covers design only. The register records no lift of the [DEC-011](../concepts/register.md#dec-011-detailed-planning-of-future-ideas-waits) planning hold for this axis; it should record this commission as design-only.

## Open owner questions

1. **Outside Findings.** Answered for design: the [proposal](../design/outside-findings.md) is written. Adopting it is the next owner decision, together with its three open points. A report about boundary behavior the model is silent on still ends the Apply as a model gap.
2. **First slice.** After question 1, start with step 1 for repairs only, on one pilot project and one trigger source? The recommendation is yes, with GitHub issues as the source, because repairs leave the model unchanged.
3. **Automatic spending.** Today provider turns are spent only on the owner's prompts and on runs or sessions the owner starts ([DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration), [DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply)). Step 1 needs one new decision that lets the loop run triage, proposals and Apply by itself, each within its budget. Does the owner want that?
4. **Accepting model changes.** One by one as they come, or as a batch in a daily inbox? The recommendation is a daily batch.
5. **Where the loop runs.** As an outer operator in CI or a scheduler, outside Core? The recommendation is yes. Whether Apply runs in CI at all is still open in [DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution).
6. **Undo.** Is "revert the landing for a repair; revert model and realization together for a model change" the right default?
7. **Protected Packages.** Should the owner mark Packages whose changes always need acceptance, and should the circuit breaker tighten the loop by itself? The recommendation is yes to both.
8. **Government.** Step 2 needs mandates: which kinds of model change agents may accept on their own. Leave it until step 1 has evidence? The recommendation is yes.
9. **Order with other lanes.** Step 1 depends on unattended landing (lane 3) and an intake or triage command (lane 9). Should this lane wait for those, or prototype against today's Apply with a branch hand-back?
