# Accept

Type: [Step](../../../../packages/company/step.md)

## Description
The QA accepts the story: it goes to `done`, and the claim is released ([claiming](../../rules/plan/claiming.md)). When it was the feature's last open story, the feature goes to `acceptance`. A feature is accepted as a whole once its stories are done, and goes to `done`; a feature without an interface is accepted by whoever worked it out. Who sets which state, the [milestones](../../rules/plan/milestones.md) say. What the QA finds problematic instead, it drops into the context; it releases the claim and leaves the state as it is.

## Performed by
The [QA](../../roles/qa.md), with [accept](../../agents/accept.md); for a feature without an interface, its [Developer](../../roles/developer.md).
