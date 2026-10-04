# Agent Rules Module

This Module projects normalized `Agent`, `Skill`, and `Rule` resources into explicitly selected Codex and Claude entrypoints. It does not emit the opt-in Markdown resource views owned by the Markdown Module.

`Generate` accepts a `core.SemanticModel`, a `Config` built by Host, and a fixed repository-relative byte map. It returns repository-relative output bytes plus the Core identity keys that own each output. It performs no filesystem reads or writes. `Validate` evaluates the same explicit model, configuration, and bytes, including provider adapter input existence, generated-input exclusion, strict provider settings, Rule coverage, retired names, and stale provider inventory.

Host resolves public configuration before calling this Module. In particular, `RuleAdapter.RuleKeys` contains resolved Core identity keys; `AgentSettings` is Host's normalized projection of canonical Agent provider settings; `PackageVersions` contains exact selected package pins; and `ProviderAdapters` contains provider-owned settings and explicit source paths. This Module does not parse Project or resource YAML, resolve names from authoring syntax, inspect the filesystem, infer dependencies from prose, or treat one provider's output as another provider's source.

Canonical resource prose and navigation remain owned by their declared resource source. Unless `InlineAgentText` is enabled, provider entrypoints point to the canonical Agent or Skill resource. When inline text is selected, relative Markdown destinations are rebased against those canonical sources using only the supplied byte inventory; link text and unrelated prose remain unchanged. These outputs are mechanical projections, not evidence of policy compliance, provider behavior, or human acceptance.

The private `links` package exists solely to support this Module's provider projections. It is not a shared Core facility.
