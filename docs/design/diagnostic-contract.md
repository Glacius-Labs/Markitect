# Diagnostic coordinates: separate tool contract

**Superseded:** [DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation) replaced the v1 language whose coordinates this page fixes; step T1 built the new form and removed the v1 form. The [model language guide](../guides/model-language.md) describes Main and its diagnostics. This page is kept for its reasoning.

**Status:** Chosen tool contract, separate from the [normative language specification v1](language-contract.md). This profile stipulates the meaning of exact body link coordinates. It is not an additional prerequisite for language conformance or canonical bytes and not a new Core definition. Complete support by the Go product compiler is not claimed by this.

Tools may offer this profile. If they output an exact coordinate, it MUST satisfy the following rules; they must not guess positions. For the profile, MUST, MAY and MUST NOT apply as mandatory rule, permission and prohibition.

## Coordinate form

- Every mapped body link occurrence receives `startByte` and `endByte`: a zero-based, end-exclusive UTF-8 byte range `[startByte, endByte)` in the decoded original `body`. It covers the complete link notation including both double brackets. It must not be counted against rendered text or text normalized for Markdown.
- The mapping to the original YAML additionally receives `source.file`, `source.start` and `source.end`. Start and exclusive end use one-based line and column. Columns count Unicode code points, not UTF-8 bytes or screen widths. CRLF counts as one line break.
- YAML escapes, folding, quotation marks and Markdown containers must be taken into account in the mapping. A link folded across file lines may have a multi-line YAML source range even though the decoded link is single-line.
- Repeated links remain individual, ordered occurrences. Identical targets or coordinates must not trigger deduplication. Coordinates are not an additional identity and are not included in canonical model bytes.

## Missing or unverifiable coordinates

Link recognition and target resolution MUST remain independent of coordinate computation. Missing, outdated or unverifiable source mapping and pure resource aborts of the coordinate phase must neither remove known link occurrences nor replace a content judgment. In particular, they must not prevent a successfully completed canonical projection.

A tool MUST explicitly report missing profile coverage. Existing verifiable partial coordinates may be retained; missing coordinates must not be output as complete, neither with zero values nor with estimated values. The definition and the order of its recognized occurrences remain known even without exact coordinates.

If the content check itself is not completed because of parser, resolution or resource problems, the rules of the language specification remain in force: for this model scope there is no valid digest. Language errors already established remain errors. An independent coordinate result is not evidence of a complete content check.

## Report of the design probe

The [Exchange verifier](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/exchange-example/verify.py) separates these results:

| Report part | Scope |
| --- | --- |
| `body_link_validation` | Syntax and resolved ordered link occurrences, without prescribed coordinates. |
| `canonical_models` | Completely checked canonical model bytes and digests per model scope. |
| `diagnostic_coordinates` | Separate mapping according to this profile with its own occurrences and entries on missing coverage. |

`diagnostic_coordinates.status` is `passed` if all occurrences considered there are completely mapped, and otherwise `not_completed`. For content not yet recognized or not resolved, the probe likewise claims no complete coordinate coverage. The status influences neither model validity, digest nor the overall status of the content check. A separate conformance judgment on this tool profile must consider it separately.

[diagnostic_coordinates.py](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/exchange-example/diagnostic_coordinates.py) contains the source mapping; [diagnostic_probes.py](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/exchange-example/diagnostic_probes.py) checks its separation from the canonical evaluation. The existing counter-probes for exact byte/YAML ranges remain tool profile tests. Historical reports keep the structure and the limits of their statements that they had at the time.
