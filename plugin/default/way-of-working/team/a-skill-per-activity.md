# A skill per activity

Type: [Rule](../../core/rule.md)

## Applies to
Every [activity](../../core/activity.md) an agent does

## Rule
An activity an agent does is realized as a skill, as a module of the product is realized as code: the activity's document says what is done, and the skill is what the agent runs. There is one skill per activity, named `markitect-` and the file name of the activity, such as `markitect-review` for the review, and the same for every provider. The module that holds the activity names its skills under Realized in.

A skill is built like code: when an activity changes, through a finding of type Process and a planning, the story names the activity's document under Realizes, a Builder writes the skill anew from that document and what it links, and a Reviewer confirms it ([one trace](../review/one-trace.md)). Nothing generates it, and nobody changes a skill without changing its activity first. The way of working comes with its skills built, so a new project builds none before its first day ([set up the project](../set-up.md)).
