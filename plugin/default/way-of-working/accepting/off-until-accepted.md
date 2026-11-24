# Off until accepted

Type: [Rule](../../core/rule.md)

## Applies to
[Feature](../plan/feature.md), [planning](../planning/planning.md), [the Builder](../team/builder.md), [release](release.md)

## Rule
`main` can always be released. A feature is built switched off: what its stories bring to `main` cannot be reached by the users of the product until the feature is accepted. The feature's planning says how it is switched, in the way the project's module for infrastructure names; its stories build behind that switch, and where the application is started for trying, the person who accepts sees the feature switched on. When the feature is accepted, switching it on is the last thing its acceptance does: a change like a polishing, checked by a Reviewer. A story without a feature that has something to accept is switched the same way, until it is `done`. A story with nothing to accept, such as a bug, changes what is in use already and needs no switch. That something which is switched off cannot be reached is no bug and no finding. So a release takes `main` as it is, and only what is accepted, or needs no acceptance, is in use.

## Example
The feature "Renew online" is planned with the switch `renew-online`. Its first story lands on Monday, its second on Tuesday; members see nothing of it. On Wednesday the person accepts the feature, the helper switches it on, and the next release brings it to the members.
