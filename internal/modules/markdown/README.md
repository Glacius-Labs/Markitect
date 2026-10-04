# Markdown module

This module renders a bounded Markdown projection from `core.SemanticModel`, the
host-supplied Markdown configuration, and the fixed snapshot's exact file bytes.
It does not load or reinterpret authoring YAML, perform filesystem writes, or
consume another provider's generated outputs as semantic input. `ViewPaths` and
`DomainPaths` expose the paths the host must reserve from ordinary input
classification when this projection is enabled.

Local resource views retain their source-relative navigation behavior and
custom-domain contract views serialize only the normalized Core schema and
policy results. The host remains responsible for selecting the target and
materializing returned bytes under the normal managed-output ownership rules.

`Validate` applies explicitly configured functional-claim and documentation
router checks to normalized resource data and the same fixed file map. The Host
selects those settings from the Project model and remains responsible for
snapshot acquisition and reporting returned diagnostics.
