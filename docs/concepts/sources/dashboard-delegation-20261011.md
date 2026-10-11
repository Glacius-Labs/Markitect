# Delegated clarification and IDEA-01 source check (11 October 2026)

This record preserves the provenance for a delegated clarification to DEC-023 and a source check prompted by IDEA-01. It keeps the current user direction, the original owner description and the independent assessment's interpretation distinct.

## DEC-023 delegated clarification

**Authority:** On 11 October 2026 the user delegated this clarification to the integrator:

> Bei geänderter Datei auch deren realisierte Statements und direkte Verbraucher change-Priorität erhalten. Die Impact-Menge/Pläne konservativ unverändert lassen.

This is the user's current delegated direction. It is not an earlier answer to the open owner question recorded in the prior handoff. The 10 October KG-02 answer selected marking impact elements as `change` or `context`, while this clarification makes the changed-file rule explicit.

**Scope:** A changed file routes through declared file mappings: Statements mapped directly to the file, or Statements realized by Artifacts mapped to the file. Those Statements and their direct consumers receive `change` priority; their declared realizations and Checks follow the rule already recorded in DEC-023. This does not add Impact elements, infer dependencies from source contents, narrow a Manager plan, or permit skipping context.

### Fixed-source verification

- **Source:** `origin/main` at `7f1fde0df4519f8205a45e5d5c917df7d58d5cbd`.
- **Temporary test:** `src/internal/modules/projectmodel/zz_dec023_temporary_test.go`, injected into a temporary export at `C:\Users\Consiliari\AppData\Local\Temp\codex-dec023-verify-ba79af34250046e782f57ef595733030\source`; SHA-256 `F0C4AB7AD0A539B7051CB28BDA093479030C1FA130A853B07E2E43D37B621D86`. The test was not committed.
- The fixture changes only the digest of declared `src/inventory/stock.go`, mapped to the Artifact realizing the inventory `stock` Statement. It checks `change` for `stock`, its direct `order` consumer and `src/orders/order.go`; `context` for the transitive `shipment` consumer and `src/shipping/shipment.go`.
- It compares Impact digest, changed definitions, affected Statements, Managers, files, Checks and unknowns before and after calling `Explain`. The four Manager routing IDs are unchanged.
- **Result:** `go test ./src/internal/modules/projectmodel -count=1` passed (`ok`, 2.506s) with the temporary test present.
- **Limit:** This is one synthetic, explicitly mapped fixture and the existing projectmodel package suite. It does not call the full `projectrun.Plan` path, cover every mapping form, or establish semantic product benefit. In source, projectrun derives change routing from `projectmodel.Impact` and its `Managers`; it does not call `Explain` (`src/internal/host/projectrun/plan.go`). No permanent test was added.

## IDEA-01 assumption source check

The independent assessment at archived commit `8469b19a2553b9520942647c0035428f9345241d`, `docs/design/independent-assessment-20261010.md`, Claim 17 states:

> Das Hauptproblem ist Vergessen (Abdeckung), nicht Fehlverständnis.

This supports the wording of ASM-006 as an **assessment-derived hypothesis**: “Forgetting (coverage), rather than misunderstanding, is the main problem.” It is untested and is not the owner's comparative claim.

The owner's [original product description](owner-product-description-20261009.md) says: „je größer und komplexer es wird desto mehr wird vergessen.“ It reports forgetting as projects grow and become complex, but it does not compare forgetting with misunderstanding. The source distinction is why ASM-006 does not attribute that comparison to the owner or treat it as established evidence.

## Research and positioning pointers

The same archived assessment retained these research leads. Its own text says the review lens had not rechecked them; the conclusions below are therefore **unverified pointers, not findings about Markitect**. Recheck primary sources before citing them as evidence.

- **Model-driven engineering:** Hutchinson and Whittle, <https://eprints.lancs.ac.uk/id/eprint/69765>.
- **Spec-driven development:** Fowler/Böckeler, <https://www.martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html>, and the Thoughtworks Radar, <https://www.thoughtworks.com/radar/techniques/spec-driven-development>.
- **Traceability upkeep:** mapping study, <https://arxiv.org/pdf/2108.02133>.
- **Architectural fitness functions:** <https://concepts.dsebastien.net/concept/architectural-fitness-functions/>; the assessment says no comparison with Markitect components had been made.
- **Multi-agent systems:** Anthropic research system, <https://www.anthropic.com/engineering/multi-agent-research-system>; Cognition's <https://cognition.ai/blog/dont-build-multi-agents> and <https://cognition.com/blog/multi-agents-working>; MAST, <https://arxiv.org/abs/2503.13657v2>.
- **LLM judges:** Goel et al. 2025, <https://arxiv.org/pdf/2502.04313>; the cited follow-up was seen only through a secondary summary, <https://huggingface.co/papers/2605.29800>.

These pointers preserve the assessment's positioning questions for later research. They do not establish that Markitect is better or worse than any alternative, nor do they change the product thesis or authorize a study.