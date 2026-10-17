# Usage

Current source implements canonical v1 read-only preparation and a guarded generated model view. Apply and execution remain unavailable in the staged [rebuild](implementation-plan.md). The [design contracts](design/canonical-model/README.md) govern later bounded packages; they do not imply executable commands.

Current source uses the model-first command surface documented in [Project operations](project-operations.md), with workflow and ownership explained in [Project workflow](project-workflow.md). Release preparation commands are documented in the repository [distribution guide](../integration/README.md).

This page formerly documented the Project/Domain CLI, its schemas and examples, and the experimental canonical Projection alpha. Those source interfaces and guides are retired. The immutable [v0.14.1 usage guide](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/usage.md) preserves the complete historical contracts and command syntax. Existing repositories are not converted implicitly; migration requires explicit project-owner decisions.

## Historical Project/Domain CLI compatibility

<a id="legacy-projectdomain-cli-compatibility"></a>

See the [v0.14.1 compatibility guide](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/usage.md#legacy-projectdomain-cli-compatibility).

## Historical initialization contract

<a id="initialize-a-project"></a>

See [v0.14.1 initialization behavior](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/usage.md#initialize-a-project). The current explicit-input boundary is documented in [Project operations](project-operations.md#cli-setup-and-current-roots).

## Historical Markitect-first release candidate

<a id="markitect-first-release-candidate"></a>

See the [v0.14.1 usage guide](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/usage.md#markitect-first-release-candidate). Current contributor guidance is in [CONTRIBUTING.md](../CONTRIBUTING.md).

## Historical canonical controller actions

<a id="canonical-controller-actions-source-only-alpha"></a>

The source-only alpha contract is preserved at the [pre-removal source commit](https://github.com/Glacius-Labs/Markitect/blob/d083c8f37b6bb1e09a9c5a7947c421651a7e755b/docs/usage.md#canonical-controller-actions-source-only-alpha).

## Historical retained-evidence refresh

<a id="retained-evidence-refresh-source-only-alpha"></a>

See the [pre-removal source contract](https://github.com/Glacius-Labs/Markitect/blob/d083c8f37b6bb1e09a9c5a7947c421651a7e755b/docs/usage.md#retained-evidence-refresh-source-only-alpha).

## Historical failed-policy analysis

<a id="read-only-analysis-of-failed-policies-unreleased-source"></a>

See the [pre-removal source contract](https://github.com/Glacius-Labs/Markitect/blob/d083c8f37b6bb1e09a9c5a7947c421651a7e755b/docs/usage.md#read-only-analysis-of-failed-policies-unreleased-source).

## Historical selective-adoption preparation

<a id="selective-adoption-preparation-and-copy-me-unreleased-source"></a>

See the [pre-removal source contract](https://github.com/Glacius-Labs/Markitect/blob/d083c8f37b6bb1e09a9c5a7947c421651a7e755b/docs/usage.md#selective-adoption-preparation-and-copy-me-unreleased-source).

## Historical declared-scope completion audit

<a id="declared-scope-completion-audit-source-only-alpha"></a>

See the [pre-removal source contract](https://github.com/Glacius-Labs/Markitect/blob/d083c8f37b6bb1e09a9c5a7947c421651a7e755b/docs/usage.md#declared-scope-completion-audit-source-only-alpha).
