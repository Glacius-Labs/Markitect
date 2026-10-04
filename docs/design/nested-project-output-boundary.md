# Nested Project output boundary

`check` evaluates one root Project, while its snapshot can contain several independently runnable examples and benchmark fixtures. The root renderer knows only its own outputs, so a repository-wide stale-marker scan must recognize those child Project trees without composing them into the root model.

## Boundary rule

A nested directory establishes a separate stale-output inventory only when it contains an exact `markitect.yaml` path that `format.Parse` accepts as a closed, supported Core `Project` resource. The manifest must be below the repository root. Its directory must be disjoint from every Area configured by the root Project, every output path expected from the root renderer, every resolved parent `spec.files` input, and every selected local Domain input.

The scan skips only generated-marker stale candidates inside an accepted nested Project directory. The root `Parse`, graph, context, impact, renderer, expected-output checks, and file-input ownership remain unchanged. The nested Project and its resources do not enter the root graph or context. Run `check` from the nested Project root to validate that Project independently.

## Rejected boundaries

Malformed YAML, a `Domain` envelope, an unsupported API version or kind, and a Project resource with unknown fields do not establish a boundary. A nested Project inside a root Area remains an ordinary extra resource and the existing Project-count diagnostic makes the root check fail. A nested Project whose directory overlaps a root-generated output path or an explicitly declared parent file input is not a boundary; stale markers there remain visible to the root check.

This rule is limited to stale-marker inventory. It does not make a nested Project an exclusion mechanism for managed-artifact coverage, authorize changes to child outputs, or infer ownership from directory names.
