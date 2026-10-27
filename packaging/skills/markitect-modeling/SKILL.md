---
name: markitect-modeling
description: Write or change a Markitect canonical model (the typed YAML model under .markitect/) from requirements or an owner's decision, using the standard Libraries, and check it with markitect check and the model analysis tool. Use when asked to create a model, add or change operations, records, rules or architecture in a model, or to review a model's quality.
---

# Writing a Markitect canonical model

A Markitect model is a program written in typed prose. Kinds are types, typed fields carry the shape, a Definition's body is its function body, and each Package is a module that becomes a Manager. Write it the way you would write code in a typed language: small, typed, no duplication, dependencies pointing one way.

Read [reference.md](reference.md) for the Library Kinds and [examples.md](examples.md) for copyable Definitions before writing.

## Who decides

The owner decides; you write. Everything the model states always holds, so every statement is a decision.

- Write only what the requirements or the owner state. If the requirements leave something open, leave it open. Do not invent a policy, a limit, an error case or a default.
- **Architecture is the owner's decision.** If the owner has not decided the module cut and the dependency directions, propose them (a table: Package, what it holds, what it may depend on) and stop until the owner approves. Do not write Packages, DependencyRules, Placements or Interfaces that the owner has not approved.
- A requirement that contradicts another, or that you cannot model without guessing, goes back to the owner as a question. Do not repair it in the model.

## Workflow

1. **Read the requirements completely.** List what each concern needs: project (purpose, scope, language, build and test commands, permitted dependencies), value types, records, operations, cross-cutting rules, architecture, way of working (test and documentation duties).
2. **Settle the architecture with the owner** (above). Definitions may not sit in the model root, so the project's own decisions live in a top-level project Package named after the project. Its children are the modules: usually a shared kernel (value types, roles, shared refusal codes and cross-cutting rules), an outer layer that wires requests to the areas, and one Package per area. Keep the tree flat unless the owner wants layers; an empty parent Package still becomes a Manager.
3. **Write the vocabulary first:** Scalars with their ranges and units, Enumerations with their values and meanings, Units, Roles, RefusalCodes with their wire codes. Shared ones go into the shared kernel; one area's own ones go into that area.
4. **Write the records** as `Data.Entity` with typed fields; `key` names the identifying field; `refersTo` says a field identifies an existing record. Embed a record by using its Entity as the field type.
5. **Write the operations** as `Api.Operation`: `wireName`, `mutation`, `arguments`, `refusals` (in checking order, each with `when`), `results`, `permissions` (a role without an entry has none). The body states only the effect. If the project needs more fields on every operation (for example an audit target), define a project Kind that `implements: [Api.Operation]` and declares all of its Properties plus the new one.
6. **Write the rules** as `Practice.Rule` (or one of its obligation Kinds) with `appliesTo`. A rule many Packages follow lives once in the shared kernel with `appliesTo: [<project>]`; add one modeled-scope selector to `execution.yaml` so it reaches every Manager (see examples.md).
7. **Write the architecture the owner approved:** in the project Package a `Practice.Project`, one `Architecture.DependencyRule` per Package, `Architecture.Placement` for every cross-cutting concern realized in one place; in each area an `Architecture.Interface` with the operations and types others may use.
8. **Map the code:** `execution.yaml` ownership gives each Package its folder; its checks are the build and test commands.
9. **Compile, lock and lint** (below), fix every error and every strong warning, and justify each remaining warning in one line.
10. **Check fidelity:** go through the requirements statement by statement and point at the Definition that carries each one. Then ask for an independent review against the requirements; a model is not done before that review.

## Rules that the compiler does not catch

- **Typed shapes.** Arguments, refusal codes, fields, results, permissions and scopes are typed fields, never lists in a body. A body line starting with "Arguments:", "Refusals:" or "Fields:" is a defect.
- **Small Definitions.** Over 150 words of prose (tables and code blocks excluded) is a warning; over 300 means it holds several decisions: split it.
- **Once.** Each decision is stated in exactly one Definition. Do not restate a rule in an overview, an example or another area; link to it instead.
- **Dependencies follow the rules.** A typed reference or a body link from one Package to another is a dependency. Where the dependency rules forbid it, describe the need in prose without a link, and let the outer layer supply it (dependency inversion).
- **No process.** No Manager steps, no run results, no test evidence in the model.
- **One architecture.** No component list or layer description beside the Package tree.

## Files and names

- One Definition per file: `imports`, `id`, `kind`, `body`, `spec`. File name: the ID in lower-case kebab case.
- A Package is a folder under the model root. Reference another Package's Definition through an import alias (`Members.member`); your own Package's without one.
- **IDs:** no ID may equal a Core or Library ID ignoring case (`name`, `valueType`, `members`, `Operation`, `fields`, `key`, `code`, `scope`, `when`, `types`, `concern` and the others in reference.md). A Kind and a Property in one Package may not share an ID ignoring case. If a short reference becomes ambiguous across imports, qualify it.
- **YAML:** quote strings that contain `:`, `#`, `,`, `'`, `{`, `[`, or start with a backtick, a digit or a dash. Write bodies as literal blocks (`body: |-`).
- **Body links:** `[[alias.id|text]]` or `[[id|text]]`. Every link is a dependency edge.

## Compile, lock and lint

The model compiles with every Library passed explicitly. In a project repository with the model under `.markitect/` and the Libraries vendored, for example under `vendor/markitect/`:

```text
markitect check --repo . --dependency ABS/vendor/markitect/core/model.yaml --dependency ABS/vendor/markitect/values/model.yaml ... > check.json
python -I -B tools/model-analysis/model_analysis.py lock check.json --repo . --out .markitect/model-lock.yaml
markitect check --repo . --dependency ... > check.json
python -I -B tools/model-analysis/model_analysis.py lint check.json --repo .
python -I -B tools/model-analysis/model_analysis.py graph check.json
```

Use absolute dependency paths in the platform's native spelling. `check` exits 0 only when the model compiles and the lock pins the current Library digests. Commit before checking: `check` reads the committed tree.

Treat these lint signals as errors: `unknown-package`, `dependency-rule`, `import-cycle`, a `size` over 300. Fix `isolated`, `unused-import` and `undefined-term` unless you can say in one line why they are right.

## Done means

- `markitect check` exits 0 with the lock written.
- `lint` has no strong finding; every warning is fixed or justified.
- The graph has no cycle, and its edges follow the dependency rules.
- Every requirement statement maps to a Definition, and nothing in the model lacks a source in the requirements or an owner decision.
- An independent fidelity review found nothing missing or wrong.
