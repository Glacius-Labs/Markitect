# Validation records

Dated evidence that [concept 4.0](../concepts/v4-concept.md) and its guides still rely on. Each report preserves the exact scope, inputs, source references and outcome of its run; its date or a `pending` statement does not describe today's source. The [delivery plan](../implementation-plan.md) owns the current steps.

Records of earlier versions, the V1 qualification and comparison, the concept 2.0 build steps, Markitect Desktop, the evaluation suite and the arms pilots, are archived ([archive index](../archive.md#consolidated-on-8-november-2026)).

## The current product

- [Trials of the lean flow — 7 November 2026](lean-trials-20261107.md): three Makerspace changes with agents started by the owner's own session, on 0.19.0. The builder started the reviewer as its own subagent, the hooks fired inside subagents, the session left the work to the agents, and both a planted bug and a planted contradiction were caught.
- [How well the Makerspace specs are designed — 7 November 2026](spec-design-measures-20261107.md): coupling, stability, abstractness, cohesion, fan-in, reach, volatility and unlinked mentions on Makerspace and the 5× project, and the split of `values` into one Definition per format; the basis of `vet`'s thresholds and of [what makes a good spec](../guides/good-spec.md).
- [Relationship types on the Makerspace specs — 8 November 2026](spec-relations-20261108.md): every link of Makerspace classified by the relation it expresses. 110 of 360 links name a record by its identifier or list an Interface's parts; written as identity Definitions, visibility and Package links, the eight change steps put 113 items on the list instead of 133, and the 5× project's 354 instead of 468, with no needed item lost; the basis of [how links are written](../guides/good-spec.md#every-link-is-a-relation).
- [How far a change reaches — 3 November 2026](impact-reach-20261103.md): on Makerspace's change steps, one step of links reached 33 of the 36 existing files that truly changed, and deeper reach 35; the basis of `status` following one step.
- [A blind checker against planted bugs — 3 November 2026](k-checker-test-20261103.md): a fresh checker caught 17 of 18 planted bugs blind and 18 of 18 with the maker's claims, with no false alarm.
- [The first build at 5× — 4 November 2026](default-arm-20261104.md): one Sonnet launch built the 5× project from its typed spec and passed all 327 hidden cases, in an earlier, container-based version.

## Release evidence

- [Distribution retirement — 17 October 2026](distribution-retirement-20261017.md) with the machine-readable [retired release mapping](retired-releases-20261017.json), which lists the 18 retired release versions and the commits behind their former tags. The commit behind each retired tag is recorded in the mapping; the [archive index](../archive.md) lists the pinned pre-reset commits.
