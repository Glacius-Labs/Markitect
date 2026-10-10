# Engineering constitution: executable vertical-slice policy

This consumer fixture treats architecture as a versioned, project-selected contract. The exact-pinned offline package owns one composed Software Domain; its structured rules define the resource vocabulary, relation effects, and the policy that the CLI checks. It does not add a `Pattern` or `Trait` kind to Markitect's core.

The package deliberately presents one combined Domain rather than pretending independently packaged traits can currently stack on the same resources. Markitect validates custom constraints against resources registered in that Domain API version; custom Domain references stay within one API version. A label is only a selector input, not verified `conformsTo` evidence. A future composition model would need a separate, explicit design.

## What the model says

- Each `UseCase` has exactly one owning `Module` and exactly one `Handler`. The `minTargets`/`maxTargets` relation bounds are structural: a missing, duplicate, or wrong-kind reference is invalid and cannot be waived.
- `Command` and `Query` are distinct values in the UseCase schema. Approved extension kinds are explicit. The model stays technology-neutral; folder and source-code layouts belong to a declared adapter.
- `Module.dependsOn` can refer to `Core` or another `Module` at the type level, while the canonical `module-depends-on-core-only` constraint permits only `Core`. `Core` has no outgoing dependency relation. The assertion, not an agent prompt or renderer constant, is the policy source.
- Package v2 adds a per-UseCase policy requiring a `Validator`. The policy result includes the affected subject. A dated, source-bound exception can temporarily waive that policy result; it cannot waive broken types, references, structural cardinality, or cycles.

The base Project pins package v1 and has no automatic way to assume v2. The migration test switches to the exact v2 version and digest, activates its Domain, confirms that old UseCases fail the new check, adds a bound exception for one named subject, and checks the resulting `waived` status and agent context. Adding the Validator and removing the exception returns the subject to `passed`. This demonstrates detection and governed intent; it does not generate or perform a code migration.

## Run the fixture

From the Markitect checkout:

```powershell
go run ./src/cmd/markitect-legacy format --repo examples/engineering-constitution
go run ./src/cmd/markitect-legacy check --repo examples/engineering-constitution
go run ./src/cmd/markitect-legacy model --repo examples/engineering-constitution
go run ./src/cmd/markitect-legacy context --repo examples/engineering-constitution --namespace engineering --kind Skill --name add-order
go test ./src/harness/examples -run EngineeringConstitution -count=1
```

The package archive is a deterministic fixture built from `constitution-package-v1` with Markitect's package builder. Its `source` pin is explicitly a local fixture coordinate, not a claimed upstream Git commit. To rebuild it, run the fixture's `build-package.ps1`; the script prints the resulting SHA-256 and refuses to overwrite an existing archive. Change the exact package version, archive, digest, and selected Domain together only as part of an intentional migration.
