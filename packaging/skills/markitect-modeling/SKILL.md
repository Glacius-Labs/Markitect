---
name: markitect-modeling
description: Use when asked to create, extend or review a Markitect canonical model (the typed YAML under .markitect/), to turn requirements into a model, to add operations, records, rules or architecture to one, or to judge a model's quality.
---

# Writing a Markitect canonical model

A Markitect model is a program written in typed prose. Kinds are types, typed fields carry the shape, a Definition's body is its function body, and each Package is a module that becomes a Manager. Write it the way you would write code in a typed language: small, typed, every rule stated once, dependencies pointing one way.

Before writing, read [reference.md](reference.md) (the Library Kinds) and [examples.md](examples.md) (copyable Definitions that compile).

## Who decides

The owner decides; you write. Everything the model states always holds, so every statement is a decision.

- Write only what the requirements or the owner state. If the requirements leave something open, leave it open and say so where it matters. Do not invent a policy, a limit, an error case or a default.
- **Architecture is the owner's decision.** If the owner has not decided the module cut and the dependency directions, propose them and **stop**: a table (Package, what it holds, what it may depend on), the placements, the code folders, and your questions. Write no model file until the owner answers.
- A requirement that contradicts another, or that you cannot model without guessing, goes back to the owner as a question.

## Workflow

Copy this checklist into your reply and tick it off as you go.

```text
[ ] 1 Requirements read completely; open points listed
[ ] 2 Architecture approved by the owner (or proposed, and stopped)
[ ] 3 Trace table: every requirement statement -> planned Definition
[ ] 4 Vocabulary, records, operations, rules, architecture written
[ ] 5 check exit 0, lock written
[ ] 6 lint: no strong finding; every warning fixed or justified
[ ] 7 Trace table re-checked against the written model
[ ] 8 Final report written; independent review requested
```

1. **Read the requirements completely.** List what each concern needs: project (purpose, scope, language, build and test commands, permitted dependencies), value types, records, operations, cross-cutting rules, invariants, architecture, way of working (test and documentation duties).
2. **Settle the architecture with the owner** (above). Definitions may not sit in the model root, so the project's own decisions live in a top-level project Package named after the project. Its children are the modules: usually a shared kernel (value types, roles, shared refusal codes, cross-cutting rules), an outer layer that wires requests to the areas, and one Package per area. Keep the tree flat unless the owner wants layers; an empty parent Package still becomes a Manager.
3. **Write the trace table first:** each requirement statement, in order, mapped to the Definition (Package/ID and Kind) that will carry it. A statement without a target is a gap; a planned Definition without a statement is an invention. Fix both before writing YAML.
4. **Write the model** in this order:
   - **Vocabulary:** Scalars with ranges and units, Enumerations with values and meanings, Units, Roles, RefusalCodes with their wire codes. Shared ones in the shared kernel, an area's own ones in that area.
   - **Records** as `Data.Entity` with typed fields; `key` names the identifying field; `refersTo` says a field identifies an existing record; an Entity as a field type embeds the record.
   - **Invariants** as `Data.Invariant` for what must always hold for records, above all around state changes (a balance is never negative; an unpaid order is never shipped).
   - **Operations** as `Api.Operation`: `wireName`, `mutation`, `arguments`, `refusals` in checking order with `when`, `results`, `permissions` (a role without an entry has none). The body states only the effect. For extra fields on every operation (an audit target), define a project Kind that `implements: [Api.Operation]` and declares all of its Properties plus the new one.
   - **Rules** as `Practice.Rule` or an obligation Kind with `appliesTo`. A behavior rule many Packages follow (input formats, money, time, the order of checks) lives once in the shared kernel with `appliesTo: [<project>]`; way-of-working obligations (tests, documentation) and architecture live in the project Package. One modeled-scope selector in `execution.yaml` brings every rule to the Managers its `appliesTo` names (see examples.md).
   - **Refusals shared by every operation** (a missing or invalid argument, an unknown referenced record) are stated once in a rule such as the order of checks and are not listed on each operation; an operation lists only its own conditions. That the shared codes are not linked from each operation is intended: the rule is their single source. `Api.refusals` is ordered: the first refusal that applies is the one returned. If the requirements leave the precedence of simultaneous failures open, the order you list is a decision: propose it to the owner with the architecture, and do not also call it "not required" in a rule.
   - **Check order across layers:** when the outer layer checks request shape (unknown fields, wrong case) and the areas check arguments, say in the order-of-checks rule which comes first.
   - **Refusal codes the requirements leave free:** `RefusalCode.code` is meant as the wire value. If the requirements put no code on the wire (only a status and a reason), name each condition with a model-internal code and state once, in the protocol's rules, that codes are not on the wire.
   - **Architecture** the owner approved: in the project Package a `Practice.Project`, one `Architecture.DependencyRule` per child Package (the project Package itself needs none) and an `Architecture.Placement` for each concern realized in one place; in each area Package an `Architecture.Interface` (the shared kernel may have one listing its types; the outer layer needs none). Every type another Package uses in a typed field, such as a Scalar that a record field reuses, belongs in its owner's Interface `types`; lint reports a missing one as `interface`.
   - **Code mapping:** `execution.yaml` ownership gives each Package its folder; a Package without code of its own, such as the project Package, owns the module root. The checks are the build and test commands.
5. **Compile, lock and lint** (below) until clean.
6. **Re-check the trace table** against the written files, then write the final report.

## Scope and placement

- `appliesTo` names Packages by path; a named Package includes every Package below it. `appliesTo: [<project>]` reaches every Manager; `appliesTo: [<project>/billing]` reaches billing and its children only. A rule without `appliesTo` governs only its own Package.
- A Placement's `appliesTo` names the Packages that rely on it, usually the same as its concern's `appliesTo`; `realizedIn` names the one Package that realizes the concern for them.

## Statements

- **Strong statements.** A rule is short, free of exceptions and true everywhere it applies. A rule full of "except when" and "unless" is several rules: split it into atomic ones, or make the exception its own Definition.
- **Typed shapes.** Arguments, refusal codes, fields, results, permissions and scopes are typed fields, never lists in a body. A body line starting with "Arguments:", "Refusals:" or "Fields:" is a defect.
- **Small.** Over 150 words of prose (tables and code excluded) or over 10 decisions is a warning; over 300 words or 15 decisions holds several decisions: split it. Decisions count each typed refusal plus each `when`, `if`, `unless`, `otherwise` or `except` in the prose outside tables.
- **Formats are bodies.** A format-only Scalar (`base: string`, no bounds, the format in its body) is normal; bounds and units are typed wherever they exist.
- **Wire shapes.** No Library Kind models a request envelope or response statuses yet. State them as rules in the outer layer with a short example; type every operation's own arguments and results as usual.
- **Once.** Each decision is stated in exactly one Definition. Do not restate a rule in an overview, an example or another area; link to it.
- **Dependencies follow the rules.** A typed reference or a body link into another Package is a dependency. Where the rules forbid it, describe the need in prose without a link and let the outer layer supply it.
- **No process, no evidence, no second architecture.** No Manager steps, no run or test results, no component list beside the Package tree.

## Red flags: stop when you think

| Thought | Instead |
|---|---|
| "I'll pick a sensible architecture myself." | Propose it and stop for the owner. |
| "The requirements don't say, I'll assume the usual behavior." | Leave it open, or ask. |
| "No Library Kind fits, I'll describe it in the body." | Look in reference.md; if nothing fits, define a project Kind. |
| "I'll list the arguments in the body for now." | Type them now. |
| "This rule needs a few exceptions." | Split it into atomic rules. |
| "The lint warning is cosmetic." | Fix it, or justify it in one line in the report. |
| "It compiled earlier, it still compiles." | Run `check` again on the committed tree. |
| "The model is done." | Only with the evidence below. |

## Files and names

- One Definition per file: `imports`, `id`, `kind`, `body`, `spec`; file name is the ID in lower-case kebab case.
- A Package is a folder under the model root. Reference another Package's Definition through an import alias (`Members.member`); your own Package's without one.
- **IDs:** no ID may equal a Core or Library ID ignoring case (the reserved list is at the end of reference.md). A Kind and a Property in one Package may not share an ID ignoring case. Qualify a short reference that becomes ambiguous across imports.
- **YAML:** quote strings that contain `:`, `#`, `,`, `'`, `{`, `[`, or start with a backtick, a digit or a dash. Write bodies as literal blocks (`body: |-`).
- **Body links:** `[[Alias.id|text]]` or `[[id|text]]`. Every link is a dependency edge, and its alias needs an import in the same file, like a typed reference.

## Project layout

```text
<repository>/
  .markitect/model.yaml          manifest: id, root: model, exact Library versions (examples.md)
  .markitect/model-lock.yaml     Library digests, written by the lock command below
  .markitect/execution.yaml      ownership, the scope selector, build and test checks (examples.md)
  .markitect/ignore.yaml         apiVersion: markitect.ignore/v1 / files: []
  .markitect/registry/<project>/registry.yaml   apiVersion: markitect.registry/v1 / artifacts: []
  .markitect/model/<project>/...                the Definitions, one folder per Package
  vendor/markitect/<library>/    vendored Libraries (model.yaml, model/), committed with the model
```

Run `git config core.autocrlf false` in the repository before the first commit, so the vendored Libraries keep their bytes and digests.

## Compile, lock and lint

Run these exactly; only the paths change. Pass every Library with an absolute `--dependency` path in the platform's native spelling: on Windows `C:\...\vendor\markitect\core\model.yaml`, with backslashes; a forward-slash path is refused as "cannot require normalization". `check` reads the committed tree: commit first.

```text
markitect check --repo . --dependency ABS/vendor/markitect/core/model.yaml --dependency ABS/vendor/markitect/values/model.yaml ... > check.json
python -I -B tools/model-analysis/model_analysis.py lock check.json --repo . --out .markitect/model-lock.yaml
markitect check --repo . --dependency ... > check.json
python -I -B tools/model-analysis/model_analysis.py lint check.json --repo .
python -I -B tools/model-analysis/model_analysis.py graph check.json
```

The first `check`, before a lock exists, fails with `lock.missing`; that is expected, and the lock command reads the digests from it. After that, `check` exits 0 only when the model compiles and the lock pins the current Library digests. When `check` fails, its own output names only the first problem: run `python -I -B tools/model-analysis/model_analysis.py diagnostics check.json` to see every finding, root causes first. Treat `unknown-package`, `dependency-rule`, `interface`, `import-cycle` and every `strong` lint finding as errors.

**Known limits of the Libraries:** `Architecture.Interface` lists operations and types only, so a refusal code another Package uses cannot be listed there; `Architecture.Placement` places a `Practice.Rule`, so an invariant cannot be placed. Say so in the report instead of working around it.

## Done: evidence, not belief

Claim "done" only in a final report in this form, with the outputs pasted from runs made after your last edit and commit. `python -I -B tools/model-analysis/model_analysis.py summary check.json` prints the status, the model digest and the Definitions per Package:

```text
Check:   exit <code>, model digest <sha256>       (pasted)
Lint:    <counts incl. dependency-rule and interface>; each warning fixed or justified (pasted)
Graph:   <cycles>                                  (pasted)
Trace:   <n> of <n> requirement statements mapped; inventions: <none or list>
Open:    <points left open by the requirements>
Status:  READY FOR REVIEW | NOT READY: <numbered fixes>
```

Then ask for an independent review by someone who sees only the requirements and the model. The model is done when that review finds nothing missing or wrong.
