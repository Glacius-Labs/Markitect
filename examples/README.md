# Current examples

The [canonical Library M1 model](../testdata/library/README.md) is the current worked example. The source smoke in `tools/smoke` makes the specified M2 loan-limit edit in a temporary repository and exercises actual read-only preparation, context, generated-view identity and refusals. It builds the packaged root-layout CLI and standalone bootstrap. It creates no Library realization or provider/Apply evidence.

The [Library oracle](../tools/library-oracle/README.md) supplies the independent oracle and bounded transport checks. [Project workflow](../docs/guides/project-workflow.md) owns current usage and [CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) owns source gates.

The earlier Shop and Project/Domain fixtures are retired from Main. Their exact source remains at [the pre-retirement revision](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/examples); the archive refs pin the historical releases (see the [archive index](../docs/archive.md)). See the [retirement record](../docs/validation/distribution-retirement-20261017.md).

## Explicit command configuration

[runtime-profile.example.yaml](runtime-profile.example.yaml) and [run.example.json](run.example.json) replace Setup with explicit static profile and invocation inputs. Copy them outside the repository and fill every placeholder as described in the [workflow](../docs/guides/project-workflow.md#explicit-runtime-and-invocation-files). The examples select normal account login and finite invocation/resource limits. The [example tests](../internal/bootstrap/configuration_examples_test.go) prepare both exact substituted files with controlled image metadata, verify native and Check settings plus their account reference, and exercise CLI Apply routing to a deliberate stop before runtime startup. Invalid preflight still starts nothing. This tests static configuration and CLI transport; native account and Apply execution remain separate evidence.
