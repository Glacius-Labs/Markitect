# Examples

- [Minimal rollback review](minimal/README.md) is a runnable synthetic fixture.
- [Code and documentation](documentation/README.md) links ordinary Go input to a focused documentation resource and its review workflow.
- [Content package](content-package/README.md) supplies a small versioned set of exported resources.
- [Package consumer](package-consumer/README.md) pins that archive and composes its exports explicitly.

The minimal fixture's regression test lives in [example_test.go](example_test.go). The [documentation scenario](../internal/app/documentation_scenario_test.go) checks exact context, impact and review eligibility when its declared source input changes.
