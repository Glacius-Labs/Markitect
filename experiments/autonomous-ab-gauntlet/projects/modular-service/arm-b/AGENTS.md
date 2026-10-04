# Modular service engineering guidance

Start architecture work from the canonical Markitect project and selected Skill. Use the generated Skill projection when following implementation steps; its owner is `.markitect/areas/constitution/implement-change.skill.yaml`.

## Architecture

- `internal/core` owns stable primitives shared by modules.
- `internal/contracts` owns consumer-facing cross-module interfaces and DTOs.
- Each package under `internal/modules/<name>` owns its data and use cases.
- A module may import `core` and `contracts`, but must not import another module. Coordinate through a contract and keep composition in `internal/app`.
- Preserve compatibility of exported contract types unless the task requests a contract change. Update every consumer and test when a contract changes.
- The typed Architecture resources under `resources/` own Product, Module, Interface, Feature, Handler, and UseCase intent. `module-boundaries.rule.yaml` owns the normative boundary.

## Checks and evidence

Run `go test ./internal/architecture` and `go test ./...`. For a Markitect intent change, update canonical YAML first, then run `markitect check`, selected `context`, and `impact` against fixed revisions. Run `markitect render` and configured `verify` for the candidate. The fixed artifact helper is `markitect-check-artifacts --repo . --config .markitect/coverage.yaml`. Generated Markdown and Codex files are projections; edit their YAML owner and regenerate.

## Scope and decisions

Do not bypass contracts by importing a sibling module or reaching into its storage. Keep changes inside the requested module and its declared contract consumers. If a request requires a new cross-module dependency, changes who owns business policy, conflicts with canonical intent, or leaves the owner unclear, stop and ask the service owner before changing code. Do not create a waiver or exception to continue implementation. A green check is technical evidence, not owner acceptance.
