# Accepting

Type: [Activity](../../core/activity.md)

## Trigger
A story has landed on `main` and stands in `acceptance`; or nothing new is there, and whole features wait to be looked at. [The Assistant](../team/assistant.md) starts the Acceptance helper for the person who accepts, in a session of its own.

## Steps
1. **Take.** The person who accepts works through the list of stories in `acceptance`, oldest first. The list is shared; whoever takes a story claims it as usual: the helper enters the person under Claimed by ([claiming](../plan/claiming.md)). When nothing new is there, they look at whole features; a feature has nothing to claim, and its owner stays.
2. **Show.** The helper starts what was built, the way the project's model says the application is started for trying, and tells the person where to look and what to try: what the story names under Acceptance. The person does not have to start anything themselves.
3. **Polish.** The person polishes the interface and its use freely within the model, on the story's branch ([names in the work](../work/names.md)); the helper writes the code, the person says what they want. Details are tried and explored, not planned. The person may also bring in ideas.
4. **Review** what was polished, in rounds. When nothing was polished, there is nothing to review and nothing to merge. Otherwise, before each round the helper runs `markitect check` and looks at what changed on the story's branch; then a fresh [Reviewer](../team/reviewer.md), started by the helper, checks the result as [review](../review/review.md) says and confirms each document whose files changed ([one trace](../review/one-trace.md)). On agreement it goes to `main` as one merge whose commit names the story: the result, not every single detail. Right before that merge, the helper fetches `main`; when the story was given up meanwhile, it stops as for a lost claim ([claiming](../plan/claiming.md)). When helper and Reviewer do not agree within the rounds a review allows, the polishing is left off `main`, and what the person wanted is dropped into the context.
5. **Say what does not fit.** An acceptance does not fail, and nothing is taken back from `main`. What the person does not like and polishing cannot settle, because it would go against the model or is more than a detail, they say in a sentence; the helper drops it into the context as decided ([capture](../observing/capture.md)), and it becomes a finding that is decided ([what is no finding](../triage/not-a-finding.md)): attached to the story's feature, where the feature's next planning takes it up into a further story, or a story of its own where there is no feature. The helper recognizes on its own when a wish would go against the model, and says so instead of building around it.
6. **Close the acceptance.** The person says they have seen the story: it goes to `done`, and the claim is released. A story without a feature is switched on with that, checked by a Reviewer like a polishing ([off until accepted](off-until-accepted.md)). That says that the person has seen it and said what remains, not that they like it or that nothing remains. When it was the feature's last open story and no finding is attached to the feature any more, the feature goes to `acceptance`.
7. **Accept the feature.** A feature is accepted as a whole once none of its stories is open any more, no finding is attached to it any more, and the person is satisfied with what they see: it goes to `done`. Accepting a whole feature polishes nothing: it looks and accepts, and what it shows is dropped into the context and becomes a further story as in step 5; then the feature goes back to `in progress`. When the feature is accepted, the helper switches it on, checked by a Reviewer like a polishing ([off until accepted](off-until-accepted.md)).

## Result
The story is `done`. What the person did not like stands in the context and becomes further stories; nothing was taken back. A feature none of whose stories is open any more, and with which the person is satisfied, is accepted as a whole, and only then may it be [released](release.md).

## Participants
The person who accepts, in the role whose document says so; by default the [Admin](../team/admin.md).
The [Acceptance helper](../team/acceptance-helper.md) works with that person; a fresh [Reviewer](../team/reviewer.md) checks the result of the polishing in each round.

## Notes
The gate before production is the acceptance of the feature, not of the single story: stories that follow start as soon as their predecessors have landed ([when a story starts](../plan/when-a-story-starts.md)), and what a person changes or wants while accepting reaches them through `main` and through further stories. The helper notices deviations by the person's role document and asks for the reason ([the course of a deviation](../deviations/course.md)). At the end it leaves its [report](../observing/report.md) in the context, with any friction it recognized.
