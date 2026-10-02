# Parcel delivery support workspace

Canonical YAML lives under `.markitect/areas/`; the Project selects generated views under `docs/markitect/`. This synthetic project models one support task: prepare a refund response for a delayed or damaged parcel. It has two ownership areas, a shared customer-data rule, a policy text, a workflow, a decision contract, and a support agent behind a Skill entrypoint.

The fixture supports two distinct exercises. The read-only MCP/CLI parity task is in `tasks/query-only-prompt.md`, with its oracle in `tasks/query-only-expected.yaml`. A separate CLI-only policy mutation task is in `tasks/policy-edit-prompt.md`, with expected results in `tasks/expected.yaml`. The measured Windows CLI adoption run is described in `docs/onboarding.md`.

Resource identities:

- `Skill support/refund-triage`
- `Rule shared/customer-data`
- `Text support/refund-policy`
- `Workflow support/refund-review`
- `Contract support/refund-decision`
- `Agent support/refund-specialist`
