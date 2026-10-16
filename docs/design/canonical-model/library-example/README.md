# Small library example

Date: 16 October 2026. The [business rules](business-rules.md) are the readable domain draft for the first library Apply. They describe the intended behavior before its translation into canonical model M1. This step covers domain meaning; vocabulary, YAML sources, the evaluation interface and independent executable expectations follow from it.

## Read the domain first

The example manages one library's members, physical book copies and loans. It supports registration, suspension, borrowing, returns, state inspection and a calculated list of overdue loans. The initial policy permits five open loans per active member.

To keep the example small, the draft proposes a fixed 14-calendar-day loan period, availability derived from open loans, harmless repeated returns and overdue notices that are calculated without being sent. These are example choices, not Markitect language rules. The earlier [library and architecture sketch](../../product-model-refinement/library-model-example.md) is historical input; its Web frontend, database and candidate language are not requirements of this example.

## Derive M1 from the rules

Use the [business rules](business-rules.md) to choose only the Kinds, Properties and concrete Definitions needed to express this domain. Keep type meaning in the vocabulary and project decisions in their concrete Definitions. The [completed v1 language](../minimal-language/language-contract.md) owns syntax and binding. No canonical Manager, registry record or expected output file needs to be invented as part of the domain vocabulary.

The [cross-area effects](business-rules.md#cross-area-effects) identify connections that M1 must express explicitly. The [worked cases](business-rules.md#worked-cases) provide concrete inputs for the later independent evaluation. Neither list proves that Agents will understand every obligation.

M1 must also describe its required realization as code, tests and documentation. The callable evaluation contract and implementation choices belong to the next design work, rather than being inferred from an old example. This readable draft is a derivation source; after encoding, the canonical model owns intended behavior, without a second independently maintained specification.

## Keep the second Apply focused

The planned M2 change lowers the admission limit from five to three. Existing loans remain returnable and are not removed. A member with three open loans can borrow one more under M1 and must be refused under M2, with unchanged state on refusal. All other domain rules remain the same.

The [implementation plan](../manager-apply-implementation-plan.md) owns the complete sequence: author M1, perform Apply, inspect A1, change only the model to M2, Apply to A1 and inspect A2. This domain draft is the first part of P00, not completion of the model or of that demonstration.

Return to the [canonical-model overview](../README.md).
