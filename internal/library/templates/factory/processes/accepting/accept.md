# Accept

Type: [Step](../../../../packages/company/step.md)

## Description
The person accepts the story: it goes to `done`, and the claim is released ([claiming](../../rules/plan/claiming.md)). When it was the feature's last open story, the feature goes to `acceptance`. A feature is accepted as a whole once its stories are done, and goes to `done`. Who sets which state, the [milestones](../../rules/plan/milestones.md) say. What the person finds problematic goes into the next round of [review](review.md).

## Performed by
The person who accepts, by default the [Admin](../../roles/admin.md), with [accept](../../agents/accept.md).
