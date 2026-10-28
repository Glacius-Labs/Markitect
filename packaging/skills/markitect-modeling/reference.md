# Standard Library reference

Contents: markitect-values 1.0.0: Package `markitect/values`, alias `Values`; markitect-data 1.0.0: Package `markitect/data`, alias `Data`; markitect-api 2.0.0: Package `markitect/api`, alias `Api`; markitect-practice 1.0.0: Package `markitect/practice`, alias `Practice`; markitect-architecture 1.1.0: Package `markitect/architecture`, alias `Architecture`; Reserved IDs.

Generated from the Library sources under `libraries/` of the Markitect repository. Each Library is its own model; a project lists the ones it uses in `model.yaml` and passes each with `--dependency`. Import a Library Package with an alias, for example `- {package: markitect/api, as: Api}`.

Core 1.2.0 (`markitect/core`) defines the meta-Kinds: `Kind` (with `properties` and `implements`) and `Property` (with `name`, `valueType`, `minCount`, `maxCount`, `values`, `target`). `valueType` is one of `string`, `integer`, `number`, `boolean`, `enum`, `object`, `reference`. `maxCount: -1` means unbounded.

## markitect-values 1.0.0: Package `markitect/values`, alias `Values`

| Kind | Implements | Fields | Meaning |
| --- | --- | --- | --- |
| `Values.Enumeration` | FieldType | `members`: objects of `memberValue`, `meaning`, 1..n | A closed set of exact string values. Each member states its wire value and, where useful, its meaning. |
| `Values.FieldType` |  | none | Any type a field, argument or result can have. It is an interface without Properties: every Kind that implements it is accepted where a FieldType is required. It has no items of its own. |
| `Values.Scalar` | FieldType | `base`: one of `string`, `integer`, `number`, `boolean`, 1..1<br>`lowest`: number, 0..1<br>`highest`: number, 0..1<br>`measuredIn`: reference to Unit, 0..1 | A simple value: a string, integer, number or boolean, optionally bounded and measured in a unit. Format rules beyond the bounds are stated in the body of each Scalar. |
| `Values.Unit` |  | `symbol`: string, 1..1 | A unit a numeric Scalar is measured in, for example cents or minutes. |

Inner Properties used inside object fields: `name` (Property `Values.fieldName`): string, 1..1; `list`: boolean, 0..1; `meaning`: string, 0..1; `value` (Property `Values.memberValue`): string, 1..1; `nullable`: boolean, 0..1; `fieldType` (Property `Values.ofType`): reference to FieldType, 1..1; `refersTo`: reference to FieldType, 0..1.

## markitect-data 1.0.0: Package `markitect/data`, alias `Data`

| Kind | Implements | Fields | Meaning |
| --- | --- | --- | --- |
| `Data.Entity` | Values.FieldType | `fields`: objects of `fieldName`, `ofType`, `list`, `nullable`, `refersTo`, 1..n<br>`key`: string, 0..1 | A record type: named fields, each with a field type. A field whose type is an Entity embeds that record; refersTo says the value identifies an existing record of that type instead. The body states what the record stands for and how its fields change. |
| `Data.Invariant` |  | `entities`: reference to Entity, 1..n | A condition that always holds for the records of the listed Entities; the body states the condition. |

## markitect-api 2.0.0: Package `markitect/api`, alias `Api`

| Kind | Implements | Fields | Meaning |
| --- | --- | --- | --- |
| `Api.Operation` |  | `wireName`: string, 1..1<br>`mutation`: boolean, 1..1<br>`arguments`: objects of `fieldName`, `ofType`, `list`, `nullable`, `refersTo`, 0..n<br>`refusals`: objects of `refusal`, `when`, 0..n, no order<br>`results`: objects of `fieldName`, `ofType`, `list`, `nullable`, `refersTo`, 0..n<br>`permissions`: objects of `forRole`, `scope`, 0..n | A request the program accepts: its wire name, arguments, the refusals it can return, results and the roles permitted to call it. The body states its effect. Project Kinds that implement Operation declare these Properties and may add their own. |
| `Api.RefusalCode` |  | `code`: string, 1..1 | A refusal the program returns instead of a result. Its code is the exact wire value; its body says what it means. |
| `Api.Role` |  | none | A role a caller can hold. Its body says who holds it. |

Inner Properties used inside object fields: `role` (Property `Api.forRole`): reference to Role, 1..1; `refusal`: reference to RefusalCode, 1..1; `scope`: one of `own`, `all`, 1..1; `when`: string, 1..1.

## markitect-practice 1.0.0: Package `markitect/practice`, alias `Practice`

| Kind | Implements | Fields | Meaning |
| --- | --- | --- | --- |
| `Practice.ConventionObligation` | Rule | `appliesTo`: string, 0..n | A duty about conventions: naming, formatting, structure or style the code follows. |
| `Practice.DocumentationObligation` | Rule | `appliesTo`: string, 0..n | A duty about documentation: what must be documented, where and for whom. |
| `Practice.Project` |  | `language`: string, 1..1<br>`moduleRoot`: string, 1..1<br>`permittedDependencies`: string, 0..n<br>`buildCommand`: string, 0..1<br>`testCommand`: string, 0..1 | The anchor of a project: its language, module root, permitted third-party dependencies and the commands that build and test it. The required checks in the project's execution.yaml are authoritative; these fields describe them for readers and agents. |
| `Practice.Rule` |  | `appliesTo`: string, 0..n | A binding statement; the body is the rule. Every Kind that implements Rule is a rule too and declares appliesTo. |
| `Practice.TestObligation` | Rule | `appliesTo`: string, 0..n | A duty about tests: what must be tested and how, for the Packages it applies to. |

## markitect-architecture 1.1.0: Package `markitect/architecture`, alias `Architecture`

| Kind | Implements | Fields | Meaning |
| --- | --- | --- | --- |
| `Architecture.DependencyRule` | Practice.Rule | `appliesTo`: as in Practice<br>`mayDependOn`: string, 0..n | An allowlist between the project's own Packages: each Package in appliesTo, with the Packages below it, may depend only on itself and on the Packages in mayDependOn and below them. A dependency is any typed reference or body link in the model and any import in the code that realizes it. Libraries, Core and third-party code are governed by the permitted dependencies of the Project. |
| `Architecture.Interface` | Practice.Rule | `appliesTo`: as in Practice<br>`operations`: reference to Api.Operation, 0..n<br>`types`: reference to Values.FieldType, 0..n<br>`refusalCodes`: reference to Api.RefusalCode, 0..n | The public surface of the Package it is defined in: the operations, types and refusal codes that other Packages may use. Packages outside that Package and the Packages below it use only what the Interface exposes. appliesTo names the Packages bound by it, usually the root. |
| `Architecture.Placement` | Practice.Rule | `appliesTo`: as in Practice<br>`concern`: reference to Practice.Rule, 1..1<br>`realizedIn`: string, 1..1 | Where a cross-cutting rule is realized: the Package in realizedIn realizes the concern for the Packages in appliesTo, which do not realize it themselves. appliesTo usually equals the appliesTo of the concern. |

## Reserved IDs

No project ID may equal one of these ignoring case: `appliesTo`, `arguments`, `base`, `buildCommand`, `code`, `concern`, `ConventionObligation`, `DependencyRule`, `DocumentationObligation`, `entities`, `Entity`, `Enumeration`, `fieldName`, `fields`, `FieldType`, `forRole`, `highest`, `implements`, `Interface`, `Invariant`, `key`, `Kind`, `language`, `list`, `lowest`, `maxCount`, `mayDependOn`, `meaning`, `measuredIn`, `members`, `memberValue`, `minCount`, `moduleRoot`, `mutation`, `name`, `nullable`, `ofType`, `Operation`, `operations`, `permissions`, `permittedDependencies`, `Placement`, `Project`, `properties`, `Property`, `realizedIn`, `refersTo`, `refusal`, `RefusalCode`, `refusalCodes`, `refusals`, `results`, `Role`, `Rule`, `Scalar`, `scope`, `symbol`, `target`, `testCommand`, `TestObligation`, `types`, `Unit`, `values`, `valueType`, `when`, `wireName`.
