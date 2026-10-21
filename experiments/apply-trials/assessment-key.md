# Model assessment controls: answer key

OPERATOR MATERIAL for Session C's model assessment trial of the [direct-runtime handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization). Never copy it, [cases.py](cases.py) or [answer-key.md](answer-key.md) into an assessment packet, mandate or mount, and do not tune mandates against it and then present the same inputs as independent validation.

## Inputs

Each control is the model-only packet of a trial case: `.markitect/` with the stockroom model, the shared Core and Software dependencies and the execution settings, no realization and no registry. `python -B freeze.py out DIR` writes them under `DIR/assessment/`; [scenarios.lock.json](scenarios.lock.json) (`assessmentControls`) binds each control's version, expected conclusion, budget, commit, tree and file digests. Every packet carries the budget of A4-gap (30 minutes, 20-minute turns, 400 agent turns, 12 Passes), the `direct` strategy and the execution profiles `execute: project-agent` and `verify: project-agent`. The controls differ only in the stockroom Definitions their cases edit; the exact edits are in [answer-key.md](answer-key.md).

A changed control is a new version, and an attempt keeps the version it ran on. MA-COMPLETE and MA-FREE are at version 2 (PR #561), which tests the same materiality rule on a model that does not spell out every freedom: version 1 used S-G and S-FREE and keeps its expectations.

The conclusions follow the [current policy](../../docs/development/claude-direct-runtime-handoff-20261021.md#current-policy-supplement-mandatory-assessment-and-closed-runs) (PR #560): `accepted`, `model_gap` (a missing material decision) or `model_error` (a sourced contradiction); a Root-confirmed `model_gap` or `model_error` ends the Run Failed without publication, and nobody answers from outside it. Execution failure, an exhausted limit or missing evidence is `Incomplete`, never a detected gap.

| Control | Version | Case | Expected conclusion | Required finding | Not a finding |
| --- | --- | --- | --- | --- | --- |
| MA-COMPLETE | 2 | S-GO | `accepted` | none | anything about the name maximum (fixed at 60); a gap or demand for an explicit statement about error wording, the data file layout, the internal organization or a SKU example, which the model leaves unstated and therefore free |
| MA-GAP | 1 | S-QV | `model_gap` | the maximum name length is referred to (`stockroom/inputs`) but never given; names the missing decision | a contradiction; any other material gap |
| MA-FREE | 2 | S-FREEU | `accepted` | none | a gap about the name maximum: the model delegates it (40 to 120, stated in the README); an error, gap or correction about names holding tabs, line breaks and control characters, a deliberate and consistent choice |
| MA-CONTRA | 1 | S-XG | `model_error` | `stockroom/item` makes SKUs case-sensitive while `stockroom/inputs` normalizes them to uppercase and identifies items by the normalized SKU; the same edit also conflicts with `stockroom/add-item`'s refusal of a SKU that exists after normalization | a gap instead of the error |
| MA-NEAR | 1 | S-XNG | `accepted` | none | a contradiction about SKU case: `stockroom/item` says that differently cased SKUs name the same item after the normalization of `stockroom/inputs` |

MA-COMPLETE, MA-GAP, MA-FREE and MA-CONTRA are the four controls of C's grant. MA-NEAR is the contradiction's counter-control, prepared beside MA-FREE as the gap's; running it is outside that grant.

A finding counts as the required one when it names the source Definitions, both competing readings or the missing decision, and the consequence. Matching terms (an aid, not a verdict): gap — `name` and one of `length`, `maximum`, `limit`, `characters`; contradiction — `sku` and one of `case-sensitive`, `case sensitive`, `uppercase`, `upper case`, `lowercase`, `lower case`.

## Known minor ambiguities

The control models share the interface, input and storage texts apart from their cases' edits. The independent blind reviews (below) found these residual ambiguities in each of them. A finding that raises one of them as a minor, non-blocking observation is neither required nor a false alarm. Raising one as a material issue that blocks acceptance, or turning it into a required behavior, counts as a false alarm.

| Id | Where | Open behavior |
| --- | --- | --- |
| M1 | `stockroom/interface` | Command-line edge cases: a repeated option, the `--sku=VALUE` form, abbreviated options, `--db` after the command, extra positional arguments. |
| M2 | `stockroom/inputs` | Whether trimming and uppercasing follow Unicode or ASCII rules (for example a dotless `ı`, a leading no-break space). |
| M3 | `stockroom/storage` | Which existing files count as malformed: a 0-byte file, or valid JSON that breaks an invariant such as an on-hand quantity above 9999. |
| M4 | `stockroom/storage`, `stockroom/interface` | The exit code of a storage failure (malformed data, a missing directory), which the interface does not name a refusal. |
| M5 | `stockroom/interface` | Output details beyond the one JSON object: stderr, key order, indentation, escaping of non-ASCII text, a final newline. |

## Invented requirements

Each of the following is free under the models and counts as an invented requirement when an assessment demands it, and in MA-COMPLETE version 2, where the sentences that say so are omitted, so does demanding that the model state it: exact error wording (`stockroom/interface`: "Exact error wording is free"); the layout of the JSON data file (`stockroom/storage`: "an implementation choice"); a module, test or package organization (`stockroom/realization`: "Internal organization is free"); a particular location or form of the limit values; any behavior inferred from an existing application, since the controls have none.

## Independent validation

On 21 October 2026 a reviewer that had neither this key, cases.py nor the case names received the four control models together with the hall model under neutral names and classified each strictly on its text (as accepted, needs clarification or model error, the vocabulary before #560). It returned a material gap for S-QV (only the maximum name length), `accepted` for S-G and S-FREE (the delegation recognized as explicit freedom), and `model_error` for S-XG (item against inputs and against add-item). It reported M1 to M3 as minor for all of them. A second blind review of the same kind classified the S-XNG model alone as `accepted`: it found no contradiction and no material gap, and reported M1 to M5 as minor. For version 2 a third blind reviewer of the same kind received S-FREEU, S-QV and S-GO under neutral names with the rule that an unusual but clear and consistent choice is no error and a missing statement that something is free is no gap. It accepted S-FREEU (the delegation explicit, the name rule raised by no objection) and S-GO, and returned `model_gap` for S-QV with exactly the missing name maximum. Its minor observations were M1 to M5 and, for S-FREEU, that a black-box test must read the chosen maximum from the README. The s1 reference realizes both version 2 models: it passes the profiles `s1` and `free` and returns a name with a tab, a line break and a control character unchanged. This shows that the controls carry the intended conclusions for one careful reader; it does not establish how an Agent team assesses them.

## Recording

Record per attempt: the conclusion, every finding with its sources, whether the required finding is present, false alarms (material findings outside the required one, delegated or free choices, invented requirements, a contradiction on MA-NEAR), minor observations, disagreement within the team and owner effort outside the Run. Execution failure, timeout or missing evidence is `Incomplete`, never acceptance or rejection of the model.
