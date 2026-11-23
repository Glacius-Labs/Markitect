# A finished planning reopens only after a finding

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Planning](../../processes/planning/README.md)

## Rule
A finished planning is reopened only after a finding: then it has failed while building. An aborted build or a failed acceptance is such a case; a person reopens the story with the Architect. When nobody wants the story any more, that is decided on its finding like anything else: the Recorder sets the story `done` as given up ([milestones](../plan/milestones.md)), and what it already brought to `main` and is not wanted goes again through a story of its own.
