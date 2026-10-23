# Current examples

The [canonical Library M1 model](../docs/design/canonical-model/library-example/README.md) is the current worked example. The source smoke in `tools/smoke` makes the specified M2 loan-limit edit in a temporary repository and exercises actual read-only preparation, context, generated-view identity and refusals. It builds the packaged root-layout CLI and standalone bootstrap. It creates no Library realization or provider/Apply evidence.

The [Library Playground](../experiments/library-playground/README.md) supplies the independent oracle and bounded transport checks. [Project workflow](../docs/project-workflow.md) owns current usage and [CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) owns source gates.

The earlier Shop and Project/Domain fixtures are retired from Main. Their exact source remains at [the pre-retirement revision](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/examples); historical version tags remain available. See the [retirement record](../docs/history/distribution-retirement-20261017.md).

## Explicit command configuration

[runtime-profile.example.yaml](runtime-profile.example.yaml) and [run.example.json](run.example.json) replace Setup for the retained S1 command path. Copy them outside the repository and fill every placeholder as described in the [workflow](../docs/project-workflow.md#explicit-runtime-and-invocation-files). The examples select normal account login and finite invocation/resource limits. Their [decoder/preparation/routing test](../internal/bootstrap/configuration_examples_test.go) uses controlled image observations and does not execute a provider or Apply.
