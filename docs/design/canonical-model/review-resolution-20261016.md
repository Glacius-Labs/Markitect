# Resolution of the supplied Manager Apply review

Date: 16 October 2026. Design review record requested by the owner. This file records the supplied findings, their treatment and evidence limits. The [Apply concept](manager-apply-concept.md) owns the resulting proposal; the [implementation plan](manager-apply-implementation-plan.md) owns its future delivery criteria. This record does not implement or accept those future capabilities.

## Sources and compared revisions

The owner supplied an [assessment against Main](sources/concept-assessment-20261016.txt) and [five proposed refinements](sources/concept-refinements-20261016.txt) from a separate session. These copies preserve the supplied text, with line endings subject to Git's normal text rules. The following SHA-256 digests identify the original attachment bytes:

| Original attachment | SHA-256 |
| --- | --- |
| Assessment | `6e3701ae31d3673b9abf9e21c49d2495379518dfb958ff88204315c4b5592c65` |
| Refinements | `1ffd77cec4f78d71d44739ac57c1c860230526f834fcc683ef186ff89d217ed6` |

The assessment names product Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310` and design snapshot `dbbf594f6e12047f28c03ab24257f6063baadc57`. At the start of this response, fetched `origin/main` still resolved to that product SHA. The design branch was at `b6952063558633ce627ae4321008ae0ea2468d46`, with no product-source difference from Main. Its later [owner clarification](sources/manager-realization-20261016.md#two-apply-cycle-and-freedom-to-redesign) already replaced the assessed migration sequence.

The current [operation guidance](../../../src/internal/runtime/projectrun/operation.go) confirms an important assessment finding: Reconcile already covers complete responsibility against the accepted model, missing realization and reasoned no-change. Existing runtime mechanics are useful material to evaluate during architecture design. This source reading does not establish that the new language, Package organization or registry is implemented.

## Treatment of the findings

| Finding | Refinement and tradeoff | Owning section and delivery step |
| --- | --- | --- |
| Managers and reviewers can miss the same cross-area effect. | Both ask which local statements require another area's behavior. Report a suspected missing connection with its source passage, affected area when known and suspected effect through the common parent. A required unresolved assessment blocks completion. Apply cannot edit the link. This adds a concrete review duty without claiming complete semantic discovery. | [Boundary questions](manager-apply-concept.md#checking-effects-across-package-boundaries); P00, P06, P07 and P11. |
| Package structure may be a poor work organization. | Keep one Manager per local Package initially. Show local work, cross-area coordination and observed time/Agent effort using existing records. Keep local work, descendant work and waiting distinguishable. Authors can consider concrete problems; there is no automatic restructuring or threshold. | [Reports](manager-apply-concept.md#reports-and-execution-history); P08 and P10. |
| File ownership does not show that required capabilities exist. | Require a reason and observable realization or concrete use for every positive Definition disposition. The Host checks completeness, shape and bindings; reviewers judge substance. Vocabulary-only cannot remove obligations on uses. This strengthens the existing table without creating an atomic requirements database. | [Deliveries](manager-apply-concept.md#deliveries-and-evidence); P00, P06, P08 and P11. |
| Repeated reviews can add effort while sharing blind spots. | Give local, integration and final checks distinct questions, purposes and candidate bindings. Reference resolved findings; renew assessment when the subject or assumptions change. Retain fresh independent reviews and the frozen library expectations. The expectation set evaluates this experiment rather than imposing a second product specification on adopters. | [Review responsibilities](manager-apply-concept.md#distinct-review-responsibilities); P06 to P11. |
| Custom vocabulary increases author effort. | Derive one small optional ordinary software vocabulary from the library example. Provide visible editable Definitions and a context preview showing statements, inclusion reasons, responsibilities and explicit connections. This helps authors without changing Core or adding another authoring language. | [Authoring](manager-apply-concept.md#authoring-and-inspectable-preparation); P00 and P04. |

The common design test is whether an added structure exposes a concrete failure possibility or saves recurring human work. The normal user result stays a concise account of changes, model basis, evidence, limitations and required decisions. Agent coordination is inspectable detail rather than a new management task for every user.

## Recommendations superseded by later owner input

Do not restore the assessment's proposed temporary compatibility bridge. The owner permits direct deletion and replacement, and the revised plan already uses one target architecture. Preserve required properties such as bounded review, source guards and honest recovery by testing the replacement; this does not require retaining old interfaces or workflows.

Do not replace the primary demonstration with unchanged-model drift repair followed by no-change. The selected sequence is **M1 → Apply → inspect A1 → model-only M2 change → Apply to A1 → inspect A2**, using real Agents and one product build/profile. Drift repair and no-change remain useful additional cases. A product defect requiring a new build preserves failed evidence and restarts the complete comparison.

The supplied assessment's reported checks and historical provider evidence remain its own observations. They are not new tests run by this response, hosted CI results or acceptance of the redesigned product.

## Next concrete work

Start P00: write a complete readable library description, derive the smallest useful ordinary vocabulary, encode M1, specify the exact M2 change and independently review both expectation sets. Review cross-area effects, including member suspension affecting lending, before running Agents. The five-to-three admission-limit change remains the primary second-Apply witness; existing loans remain returnable.

Use that concrete example to finish P01's Go responsibility/dependency design and keep/rewrite/delete map, then P02's external Playground design. The Playground calls the actual product and evaluates it; it does not implement another Manager system. Exact record schemas, package names and transport choices belong to those designs before substantial runtime work.

The concept is sufficiently defined for these bounded design steps. It does not yet establish semantic completeness, Agent reliability, lower human effort or a general cost advantage. No additional Core concept, Agent role, semantic index or organization strategy is needed to resolve the supplied findings now.

Return to the [concept](manager-apply-concept.md), [plan](manager-apply-implementation-plan.md) or [canonical-model overview](README.md).
