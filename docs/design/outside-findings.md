# Outside Findings

**Status:** proposed amendment to [concept 1.0](../concepts/v1-concept.md), design only. The owner asked for this design on 27 October 2026: "Yes: design outside Findings as a concept 1.0 amendment, with the command side in lane 9. Still build nothing." It is not adopted. Adoption is a separate owner decision for the [register](../concepts/register.md); the concept's text then changes through its owner. It comes from the continuous-change lane and is now tracked on the [topics page](../workstreams/topics.md#continuous-change).

## Problem

A bug report often describes a defect against a model that already states the right behavior. The fix needs no model change, but today an Apply cannot carry it:

- Findings come only from inside the Apply: from Assessment and from a Verify Reject. The host sets their origin from the Manager or check that produced them ([§10.4](../concepts/v1-concept.md#104-apply-flow), [§18.3](../concepts/v1-concept.md#183-findings-as-an-actionable-handover)).
- On an input already verified under the same model, Verify rejects any change when it finds no defect ([DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model)). Verify never sees the bug report, so it can reject a correct fix.

The path inside already exists: an Assessment Finding that names a Definition of the checked model becomes an Initial Finding of Implement (§18.3). What is missing is a way in for reports from outside, and the rules for checking them.

## Proposal

An **outside report** is a claim to check, not an order. It becomes a Finding only once Assessment confirms that the model supports it. Verify then checks the candidate against it independently.

1. **Input.** Apply takes an optional list of outside reports as a bound input, next to the model and the target. Each names exactly one Definition, what was observed, what is expected, and a source reference such as an issue link. Nothing else: no free instructions, no raw trigger text.
2. **Initial gate.** Before any Manager starts, the host checks that each report names a Definition of the checked model and stays within size limits. Otherwise Apply refuses with diagnostics, as for an invalid model ([§5.3](../concepts/v1-concept.md#53-initial-gate)).
3. **Assessment judges the model side only,** as its mandate already is ([§11.1](../concepts/v1-concept.md#111-assessment)). For each report:
   - **Supported:** the named Definition states the expected behavior. The report becomes an Assessment Finding with its outside origin kept, and so an Initial Finding of Implement.
   - **Not supported:** the model states otherwise, or the behavior lies within realization freedom. The report goes no further and is listed in the result. The Apply continues.
   - **Model blocker:** the model is silent on a behavior that matters at the project boundary. That is a model defect under [DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate), and the Apply ends as today.

   Whether the starting candidate actually has the defect is not Assessment's question. Implement and Verify settle it on the code.
4. **Implement** works the Initial Findings as today. If the candidate already meets the obligation, it submits the unchanged candidate.
5. **Verify** receives the supported reports as Findings: Definition, observed and expected. It does not receive Assessment's or Implement's reasoning, so its independence ([§19.4](../concepts/v1-concept.md#194-independence-of-verify)) holds. For each, it judges against the model and the candidate:
   - **Met:** the candidate meets the stated behavior, whether it was fixed or never broken.
   - **Not met:** a realization Reject with a Finding, and the correction loop runs as today.
   - **Not supported:** Verify reads the Definition as not stating the behavior. The report is closed and listed, so a disagreement with Assessment does not loop until the budget runs out.
6. **Change minimality.** On an input already verified under the same model, only changes needed to meet a supported report's obligation, with their necessary consequences ([§11.2](../concepts/v1-concept.md#112-implement)), are allowed. Any other change is still a Reject under DEC-073.
7. **Scope.** The limit is the obligation, not the Package. A fix may change files of any Package when the named Definition's behavior needs it; the organization routes the work ([§3.7](../concepts/v1-concept.md#37-tasks-not-process-d13)). A report cannot change the model or override a Manager's judgment ([§19.2](../concepts/v1-concept.md#192-no-automatic-authority)).
8. **Statuses in host state, not prompts.** Root's judgment in Assessment and in Verify carries one status per report. The host refuses a judgment that leaves a report unjudged, and the single result validation ([§10.3](../concepts/v1-concept.md#103-validity-of-closed-results-d15)) checks it.
9. **Result.** The Apply result lists every report with its final status: met, not supported, model blocker, or open when the Apply ended without publication. The caller can close, reroute or retry the task from it.

## Concept passages that would change

- §4: the term "outside report", and that a Finding can carry an outside origin.
- §5.3: the gate checks outside reports.
- §10.1, §10.3 and §10.4: the input, the result validation and the per-report result.
- §11.1 and §11.3: Assessment and Verify judge outside reports.
- §13.2, and §14.1 if the judgment tool changes: per-report statuses in the judgment.
- §18.1 and §18.3: the result lists reports; a Finding's origin can be "outside".
- §19.4: Verify receives the supported reports.
- [DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model): the rule in point 6.
- [DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate) stays as it is: point 3 applies it.

## Command side

The commands topic owns how outside reports reach Apply, for example a file argument, and its limits. Continuous change needs only a bound input of this shape and the per-report status in the result.

## Open points

1. **One blocker ends the batch.** Under DEC-063, one report that hits a model gap ends an Apply that carries several. The proposal accepts this: the loop retries the other reports in a new Apply, and the gap becomes a model-change task.
2. **Reproducing test.** Should a fix have to come with a test that fails before it? The proposal leaves that to Implement's ordinary judgment and Verify's review, with no host rule.
3. **Timing.** The amendment would come after V1 acceptance, not into it, so that the host lane's acceptance package stays unchanged.
