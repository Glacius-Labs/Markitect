# Small library model M1

Date: 16 October 2026. The [M1 manifest](m1/model.yaml) and its YAML Definitions encode the [business rules confirmed by the owner](business-rules.md). M1 now owns the example's intended behavior. The readable rules remain its recorded derivation source; future changes belong in the canonical model. On 20 October 2026 M1 took over the two owner decisions that the demonstrations had bound with each run ([DEC-053](../../docs/concepts/register.md#dec-053-owner-decisions-bind-the-whole-run), [source](../../docs/concepts/sources/run-wide-owner-decisions-20261018.md)): an unknown request or argument field returns `invalidRequest` ([callable interface](m1/model/library/callable-interface.yaml)), and a borrow whose due date would fall after 9999-12-31 is refused ([borrow](m1/model/lending/borrow.yaml)). Under [DEC-063](../../docs/concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate) such boundary decisions belong in the model.

M1 describes one library's members, physical book copies, loans, returns and calculated overdue notices. It requires code, tests and usage documentation. Its [callable interface](m1/model/library/callable-interface.yaml) fixes a local Go JSON Lines program, its build and test commands and its wire format as model content, so the Managers receive it like every other obligation; internal source design remains a realization choice. The [independent evaluation](../../tools/library-oracle/evaluation-contract.md) and [Playground contract](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-playground.md) specify the two-run assessment. No implementation or Agent execution has been produced by authoring this model or its evaluation design.

## Small vocabulary

The ordinary imported [software-basics model](vocabulary/model.yaml), version 0.2.0, contains three Kinds:

| Kind | Meaning |
| --- | --- |
| [Concept](vocabulary/model/software/concept.yaml) | A meaningful domain concept, including its identity and characteristics. |
| [Rule](vocabulary/model/software/rule.yaml) | A condition that must hold in its stated scope. |
| [UseCase](vocabulary/model/software/use-case.yaml) | An operation with inputs, conditions, results and expected refusals. |

Their `spec` is empty: binding meaning and explicit relationships are expressed in `body`. These are ordinary project vocabulary, not new Core constructs. They do not choose an Agent role or output file. Live members, copies and loans are runtime records described by the model, not pre-populated YAML objects.

One local [LendingPolicy](m1/model/lending/lending-policy.yaml) Kind adds exactly two required integer Properties, [maxOpenLoans](m1/model/lending/max-open-loans.yaml) and [loanPeriodDays](m1/model/lending/loan-period-days.yaml). Their positivity is a semantic obligation; integer type and cardinality alone do not enforce a numeric range. The concrete [loan-policy](m1/model/lending/loan-policy.yaml) is the single source of the values five and fourteen:

```yaml
spec:
  maxOpenLoans: 5
  loanPeriodDays: 14
```

Its body explains the values' application without repeating the numeric literals. This keeps M2's policy change in one field.

## Feature Packages and Definitions

There are 22 local Definitions in five Packages. The model root is not a Package. `library` contains shared scope and rules; it is an ordinary Package, not the root Manager or a special language container. The imported vocabulary and Core do not add local Package Managers.

| Package | Definitions |
| --- | --- |
| `library` | [library](m1/model/library/library.yaml), [inputs](m1/model/library/inputs.yaml), [realization](m1/model/library/realization.yaml), [callable-interface](m1/model/library/callable-interface.yaml) |
| `members` | [member](m1/model/members/member.yaml), [eligibility](m1/model/members/eligibility.yaml), [register-member](m1/model/members/register-member.yaml), [set-member-status](m1/model/members/set-member-status.yaml), [inspect-member](m1/model/members/inspect-member.yaml) |
| `catalog` | [copy](m1/model/catalog/copy.yaml), [availability](m1/model/catalog/availability.yaml), [register-copy](m1/model/catalog/register-copy.yaml), [inspect-copy](m1/model/catalog/inspect-copy.yaml) |
| `lending` | [LendingPolicy](m1/model/lending/lending-policy.yaml), its two Properties, [loan-policy](m1/model/lending/loan-policy.yaml), [loan](m1/model/lending/loan.yaml), [borrow](m1/model/lending/borrow.yaml), [return](m1/model/lending/return.yaml), [inspect-loans](m1/model/lending/inspect-loans.yaml) |
| `lending/reminders` | [assess-overdue](m1/model/lending/reminders/assess-overdue.yaml) |

The default organization can derive an ordinary root Manager above these local Packages. It does not require a canonical Manager Definition. Profiles, teams, the realization registry and execution history stay outside this model.

## Explicit connections

Each source file imports the foreign Packages it uses. Its binding statements then name the relevant Definitions with the language's `[[target]]` or `[[target|label]]` body links. There is no parallel generic `uses` list. Imports provide visibility; the surrounding statement explains the relationship and scope.

| Connection | Where its meaning is stated |
| --- | --- |
| Member status controls new loans and preserves return rights. | [members/eligibility](m1/model/members/eligibility.yaml) points to borrowing, loans and returns; [lending/borrow](m1/model/lending/borrow.yaml) and [lending/return](m1/model/lending/return.yaml) explicitly refer back to that rule. |
| An open loan makes its physical copy unavailable; a valid return releases it. | [catalog/availability](m1/model/catalog/availability.yaml) names the lending Definitions; borrowing and return explicitly apply that rule. |
| Current loan state and due dates determine overdue notices. | [lending/reminders/assess-overdue](m1/model/lending/reminders/assess-overdue.yaml) directly imports and names the loan, policy, member and shared inputs. No parent-Package inheritance is assumed. |
| The callable interface names the use case behind every operation and the records it returns. | [library/callable-interface](m1/model/library/callable-interface.yaml) links each operation to its use case in members, catalog, lending and reminders, so every area sees the interface through incoming references. |
| Shared scope, input handling and delivery requirements apply across the library. | [library/library](m1/model/library/library.yaml) connects the areas and states the scope of [inputs](m1/model/library/inputs.yaml) and [realization](m1/model/library/realization.yaml). Operations explicitly reference the inputs rule. Realization directly names each area's concept or behavior, making it visible through incoming references without expanding a transitive context closure. |

Bidirectional references between Packages inside this one model are allowed. They are not cycles in the separate manifest dependency graph. A resolved link proves an explicit reference; it does not prove semantic completeness or globally activate every statement at its target.

## Derivation from the business rules

| Confirmed subject | Canonical owner |
| --- | --- |
| Scope, empty initial state, sequential work and omitted features | `library/library` |
| Member identity, registration, status and borrowing eligibility | `members/member`, `members/register-member`, `members/set-member-status`, `members/eligibility` |
| Physical copies, registration and derived availability | `catalog/copy`, `catalog/register-copy`, `catalog/availability` |
| Loans, dates, admission limit and due-date calculation | `lending/loan`, `lending/loan-policy` and its Kind/Property contracts |
| Borrowing, refusal without changes and repeated requests | `lending/borrow`, `library/inputs` |
| Returns, repeated old returns and preservation of a later loan | `lending/return`, `catalog/availability` |
| Read-only current-state inspection | `members/inspect-member`, `catalog/inspect-copy`, `lending/inspect-loans` |
| Read-only overdue notices and assessment-date boundaries | `lending/reminders/assess-overdue` |
| Identifiers, text inputs, expected errors and unchanged state | `library/inputs` |
| Implementation, tests and usage documentation | `library/realization` |
| Program boundary, language, build and test commands, wire format and operations | `library/callable-interface` |

The original [worked cases](business-rules.md#worked-cases) remain concrete inputs for later independent evaluation. This mapping is a readable review aid, not another requirements database or an executable behavior check.

## Binding and bounded validation

[model-lock.yaml](m1/model-lock.yaml) binds the exact external vocabulary and Core contents. The Core is [markitect-core 1.1.0](../core/model.yaml), the English wording of the 1.0.0 snapshot used by the archived language design probes. Only that dependency is supplied; the intentionally invalid Connector model is not loaded. Physical source paths are explicit probe inputs, not extra manifest or lock fields.

Go tests read this fixture byte for byte: `internal/bootstrap` checks that the M1 lock still binds the `testdata/core` digest, and `internal/project`, `internal/adapters/gitrepo` and the source smoke copy M1, the vocabulary and the Core into fresh repositories. Keep the bytes stable: locks pin them, and `.gitattributes` keeps the Core files free of line-ending conversion.

The earlier bounded Python language probe (`verify-model.py`, its pinned requirements and the recorded `verification.json`: 22 local Definitions, 34 including dependencies, 94 body-link occurrences, 60 distinct references between local Packages) is archived at [`archive/main-pre-r2-20261024`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/library-example); see the [archive index](../../docs/archive.md). The counts describe connections, not proof that every required connection exists. This fixture is not evidence that generated code meets the model; executable evaluation uses the [independent Library oracle](../../tools/library-oracle/README.md).

## Keep the second Apply focused

For the planned M2 change, edit only `spec.maxOpenLoans` in [loan-policy.yaml](m1/model/lending/loan-policy.yaml) from five to three. Its body already explains how to use the field and preserves existing loans and return rights. The local content digest changes; the external lock stays unchanged because the imported models have not changed.

A member with three open loans can borrow one more under M1 and must be refused under M2, with unchanged state on refusal. All other rules remain the same. The [archived implementation plan](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-implementation-plan.md) owns the two real Applies and independent inspection of A1 and A2. A2 inherits A1's repository realization, not a running process's records; [evaluation](../../tools/library-oracle/evaluation-contract.md#existing-loans-and-the-process-boundary) states the limit on runtime upgrade evidence. Authoring these contracts does not complete their executable gates or demonstrate Apply.

Return to the [documentation map](../../docs/README.md).
