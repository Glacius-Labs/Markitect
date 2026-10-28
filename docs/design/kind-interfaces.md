# Kind interfaces and the value-type vocabulary

**Redesign decided, not built:** [DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation) replaces this with Markdown Definitions without typed fields, a single Core `Kind` and Libraries of plain Kind Definitions. This page describes Main today.

**Status:** design v0.3, 26 October 2026, after an independent review (nothing blocking; all should-fix points applied). The owner confirmed the direction on 26 October 2026: Go-style interfaces instead of inheritance, and a clean type vocabulary ([DEC-075](../concepts/register.md#dec-075-kind-extension-and-one-core-120), as amended). It replaces design v0.2 (Kind extension with `extends`), which never merged. The normative text is [language contract §3.3](language-contract.md#kind-interfaces); this document keeps the reasoning and the migration for the [canonical model track](model-track-charter.md).

## Why not inheritance

v0.2 let a Kind inherit fields from base Kinds. It worked, but it brought what Go deliberately avoids:
- **hidden fields:** a Kind's full contract was spread along a chain;
- **conflicts:** name clashes between two parents, diamond inheritance, and a fragile base whose new field breaks every extender;
- **merging rules:** effective-field resolution, re-declaration errors and a cycle walk over the chain.

The two real needs behind it can be met without inheritance:

| Need | Without inheritance |
|---|---|
| **Reuse fields** of a Library Kind | Already possible: a project Kind lists the Library's Properties in its own `properties` and adds its own (language contract §9, "Use Property under another Kind") |
| **"Any of these"**: one reference that accepts several Kinds, or a rule that applies to every operation-like Kind | **Interfaces:** a Kind declares which interface Kinds it implements; a reference to an interface accepts every implementing Kind |

## Interfaces

**One new Core Property, `implements`, on Kind:** an optional list of references to interface Kinds.

```yaml
# Library interface: the fields every operation has
imports:
  - {package: markitect/core, as: Core}
id: Operation
kind: Core.Kind
body: A request the program accepts. Its effect is described in the body of each operation Kind.
spec:
  properties: [wireName, mutation, arguments, refusals, results, permissions]
```

```yaml
# Project Kind: declares all of those fields itself, plus its own
imports:
  - {package: markitect/core, as: Core}
  - {package: markitect/api, as: Api}
id: LedgerOperation
kind: Core.Kind
body: An operation that also posts its effect to the ledger.
spec:
  implements: [Api.Operation]
  properties: [Api.wireName, Api.mutation, Api.arguments, Api.refusals, Api.results, Api.permissions, ledgerAccount]
```

## Rules

1. **Conformance.** A Kind that implements I declares, in its own `properties`, every Property that I declares (same Property identity). It may declare more. Nothing is inherited: the implementer's field contract is exactly its own `properties`. A missing Property fails with `contract.implements_missing`, naming the Property's Package and ID, at the `implements` value.
2. **Visibility.** The implementer references each interface Property by its own identity. The Property's Package must therefore be visible to the implementer's file, and its model must be a direct dependency of the implementer's model (language contract §8).
3. **Acceptance.** A value reference whose Property requires the target Kind T accepts a Definition whose Kind is T, or implements T directly or through interfaces it implements. Conformance makes this safe: every implementer has at least T's Properties, transitively.
4. **Any Kind can be an interface.** There is no interface marker. A Kind that names an interface already implemented through another one is permitted.
5. **Acyclic.** The `implements` graph must not contain a cycle (`contract.implements_cycle`). Across models a cycle is impossible, because model dependencies are acyclic.
6. **Meta-Kinds stay fixed.** `Kind` and `Property` cannot be implemented (`contract.implements_meta`). A Kind names an interface at most once (`contract.implements_duplicate`). `implements` is a list (`contract.implements_list`). Acceptance applies only to value references; `kind`, `target`, `properties` and `implements` keep exact identity checks.
7. **Meaning only on meta-Kind definitions.** On a definition of the meta-Kind Kind, `implements` has this meaning. A field named `implements` on an ordinary project Kind is plain data.
8. **Field names.** Two interface Properties with the same effective field name cannot both be implemented; the existing field-name collision check reports it.
9. **Incomplete interfaces.** Conformance against an interface whose own contract failed or is incomplete is reported as not completed, not as passed.
10. **Modeled scope.** A scope selector on Kind I also selects items of Kinds that implement I, directly or transitively. Its scope Property is declared by I and therefore, by conformance, by every implementer. A marker interface without Properties cannot carry a scope field.
11. **Context.** The type-contract closure follows `implements`, so a Manager sees the interface contract of every Kind it uses.

Changing an interface's Properties breaks its implementers, as adding a method to a Go interface does. Versions are exact labels (language contract §8.1), so a project sees such a change only when it moves to a new pinned version. How Libraries version their interface Kinds is a W4 authoring policy.

## The value-type vocabulary

Today "type" means three things: a Kind is the type of a Definition, the Core Property `type` is a field's value type, and earlier Library drafts added more type names. The cleanup:

| Term | Meaning |
|---|---|
| **Kind** | The type of a Definition: what you declare, implement and reference |
| **`valueType`** (Core Property, formerly `type`) | A field's value type: string, boolean, integer, number, enum, object or reference |
| **Interface** | A Kind that other Kinds implement; no separate Core concept |

Libraries add no generic base type; "any of these" comes from an interface. A Library Kind for simple values must not be called `ValueType`. References are matched ignoring case, so it would collide with the Core Property `valueType` in every file that imports both (language contract §7.1). W4 picks a distinct name, for example `Scalar`.

## Core 1.2.0

Still one Core edition with migration (Option A of DEC-075), with ten Definitions:
- the Kinds `Kind` and `Property`;
- the Properties `name`, **`valueType`** (formerly `type`), `minCount`, `maxCount`, `values`, `target`, `properties` and **`implements`**.

Amended pinned bodies:
- `Kind`: the 1.1.0 sentences unchanged, plus "implements names interface Kinds; the type declares every Property of each in properties";
- `Property` and `values`: "For value type …";
- `target`: definitions of a Kind that implements the required Kind, directly or through other interfaces, also satisfy it;
- `implements`: the Kind must declare every Property of each interface in its own properties.

## Migration

| Area | Change |
|---|---|
| Compiler | `contracts.go` (the closed Property meta-contract reads `valueType`; `implements` list, cycle and conformance checks), `core.go` (`fixedCore`: Kind, Property, `valueType`, `values`, `target`, `implements`), `values.go` (`typedMetaSpec` emits `valueType` and `implements`; acceptance through `implements`), `canonical.go` (`implements` is an unordered declaration) |
| Organization | `context_contracts.go` (reads `valueType`; the closure follows `implements`), `context_scope.go` (reads `valueType`; lineage through `implements`) |
| Generated view | `internal/modeldocs` reads no `type` field; no change beyond the new spec fields |
| Canonical bytes | Every Kind's spec gains `"implements": []`; every Property's spec has `valueType` instead of `type`. Every digest changes, as Option A expects |
| Core fixtures | `testdata/core` 1.2.0 with `value-type.yaml` and `implements.yaml` (registry: `value-type.yaml` and `implements.yaml` entries in `.markitect/registry/examples/registry.yaml`); the embedded Core copies in `internal/model/core_fixture_test.go` and `internal/organization/core_fixture_test.go`; the Core vector in `canonical_core_vector_test.go`; the JSON golden in `canonical_integration_test.go`; the `type.yaml` paths in `contracts_test.go` |
| Inline test specs | 77 YAML `type:` occurrences in 9 Go test files (16 in the two embedded Core fixtures, the rest in contract, value and context tests); the interface tests are new |
| Library fixtures | `software-basics` 0.3.0 (new digest only). `testdata/library/m1` has two Properties with `type` (`loan-period-days.yaml`, `max-open-loans.yaml`) |
| Comparison tooling that reads m1 live | `tools/comparison/requirements.py` (`property_meaning` expects `type`) and the Phase B overlays it lays over m1 (`phase-b/m3/…/loan-period-days.yaml`, `short-loan-period-days.yaml`, `phase-b/m4/…/max-renewals.yaml`). These are migrated together; the rendered documents stay byte-identical, which `test_requirements.py` checks |
| Self-model | G1 landed first and migrated its one Property (`vocabulary/command.yaml`), the embedded Core and `docs/markitect/project.md`. Slice R3a then retired the self-model (DEC-076); the Property lives on in the bootstrap test fixture `internal/bootstrap/testdata/product/` |
| Registry | The new tests `internal/model/interfaces_test.go` and `internal/organization/context_interfaces_test.go` have exact entries in the product/model and product/organization shards |
| Frozen study material | Phase C snapshots and other records stay historical at their recorded commits, as already noted. The Phase B overlays above are the exception, because a live tool reads them |

## What changed against v0.2

- `extends` and field inheritance are gone: no effective-field merging, re-declaration rule or diamond handling.
- Conformance is a superset check on each Kind's own `properties`.
- The reference check, the cycle walk, the meta checks, the scope lineage and the context closure from v0.2 are reused with `implements` instead of `extends`.
