# Model assessment controls: answer key

OPERATOR MATERIAL for Session C's model assessment trial of the [direct-runtime handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization). Never copy it, [cases.py](cases.py) or [answer-key.md](answer-key.md) into an assessment packet, mandate or mount, and do not tune mandates against it and then present the same inputs as independent validation.

## Inputs

Each control is the model-only packet of a trial case: `.markitect/` with the stockroom model, the shared Core and Software dependencies and the execution settings, no realization and no registry. Build it with `python -B materialize.py CASE TARGET`; [scenarios.lock.json](scenarios.lock.json) (`assessmentControls`) binds its commit, tree and file digests. The four controls differ only in `stockroom/inputs.yaml` and, for the contradiction, `stockroom/item.yaml`; the exact edits are in [answer-key.md](answer-key.md).

| Control | Case | Expected conclusion | Required finding | Not a finding |
| --- | --- | --- | --- | --- |
| MA-COMPLETE | S-G (s1) | `accepted` | none | anything about the name maximum (fixed at 60) |
| MA-GAP | S-QV | `needs_clarification` | the maximum name length is referred to (`stockroom/inputs`) but never given; names the missing owner decision | a contradiction; any other material gap |
| MA-FREE | S-FREE | `accepted` | none | asking for the name maximum: the model delegates it (40 to 120, stated in the README) |
| MA-CONTRA | S-XG | `model_error` | `stockroom/item` makes SKUs case-sensitive while `stockroom/inputs` normalizes them to uppercase and identifies items by the normalized SKU; the same edit also conflicts with `stockroom/add-item`'s refusal of a SKU that exists after normalization | a clarification request instead of the error |

A finding counts as the required one when it names the source Definitions, both competing readings or the missing decision, and the consequence. Matching terms (an aid, not a verdict): gap — `name` and one of `length`, `maximum`, `limit`, `characters`; contradiction — `sku` and one of `case-sensitive`, `case sensitive`, `uppercase`, `upper case`, `lowercase`, `lower case`.

## Known minor ambiguities

All four models share the interface, input and storage texts. An independent blind review (below) found these residual ambiguities in each of them. A finding that raises one of them as a minor, non-blocking observation is neither required nor a false alarm. Raising one as a material issue that blocks acceptance, or turning it into a required behavior, counts as a false alarm.

| Id | Where | Open behavior |
| --- | --- | --- |
| M1 | `stockroom/interface` | Command-line edge cases: a repeated option, the `--sku=VALUE` form, abbreviated options, `--db` after the command, extra positional arguments. |
| M2 | `stockroom/inputs` | Whether trimming and uppercasing follow Unicode or ASCII rules (for example a dotless `ı`, a leading no-break space). |
| M3 | `stockroom/storage` | Which existing files count as malformed: a 0-byte file, or valid JSON that breaks an invariant such as an on-hand quantity above 9999. |

## Invented requirements

Each of the following is free under the models and counts as an invented requirement when an assessment demands it: exact error wording (`stockroom/interface`: "Exact error wording is free"); the layout of the JSON data file (`stockroom/storage`: "an implementation choice"); a module, test or package organization (`stockroom/realization`: "Internal organization is free"); a particular location or form of the limit values; any behavior inferred from an existing application, since the controls have none.

## Independent validation

On 21 October 2026 a reviewer that had neither this key, cases.py nor the case names received the four control models together with the hall model under neutral names and classified each strictly on its text. It returned `needs_clarification` for S-QV (only the maximum name length, material), `accepted` for S-G and S-FREE (the delegation recognized as explicit freedom), and `model_error` for S-XG (item against inputs and against add-item). It reported M1 to M3 as minor for all of them. This shows that the controls carry the intended conclusions for one careful reader; it does not establish how an Agent team assesses them.

## Recording

Record per attempt: the conclusion, every finding with its sources, whether the required finding is present, false alarms (material findings outside the required one, delegated or free choices, invented requirements), minor observations, disagreement within the team and owner effort. Execution failure, timeout or missing evidence is `Incomplete`, never acceptance or rejection of the model.
