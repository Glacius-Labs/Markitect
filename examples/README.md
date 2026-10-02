# Examples

- [Minimal rollback review](minimal/README.md) is a runnable synthetic fixture with the explicit Markdown target enabled.
- [Repository layout](repository-layout/README.md) demonstrates canonical Areas, human documentation and native provider output with Markdown views disabled.
- [Markdown navigation](markdown-navigation/README.md) checks source-relative typed and ordinary links, images, references and unchanged code examples in relocated views.
- [Code and documentation](documentation/README.md) links ordinary Go input to a focused documentation resource and its review workflow.
- [Documentation placement](documentation-placement/README.md) is a small authoring exercise with an existing canonical owner and optional router validation.
- [Content package](content-package/README.md) supplies a small versioned set of exported resources.
- [Package consumer](package-consumer/README.md) pins that archive and composes its exports explicitly.

Examples with generated Markdown opt in through `spec.targets: [markdown]`; `package-consumer` also demonstrates provider output linking straight to canonical YAML. The minimal fixture's regression test lives in [example_test.go](example_test.go). The [documentation scenario](../internal/app/documentation_scenario_test.go) checks exact context, impact and review eligibility when its declared source input changes.
