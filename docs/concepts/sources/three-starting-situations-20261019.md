# Three starting situations — 19 October 2026

Recorder's note: direct owner discussion in the Codex chat **Markitect-Arbeitsweise durchgehen (2)**, thread `01a1542d-641c-7643-86e9-ffcf003e1420`. Owner quotations preserve their original wording. The assistant summary is labeled separately. The [vision](../../vision.md#three-starting-situations) owns the high-level workflow; [DEC-057](../register.md#dec-057-three-fundamental-starting-situations) records its acceptance. This source records documentation authority, not implementation or runtime evidence.

## Owner identifies the three situations

> Lass uns stattdessen darüber sprechen welche Ausgangssituationen wir haben und wie wir damit jeweils umgehen:
>
> - Modell existiert, Artefakte nicht
> - Modell existiert, Artefakte existieren
> - Modell existiert nicht, Artefakte existieren
>
> Das sind die möglichen Ausgangssituationen die ich sehen würde

## Owner specifies the working direction

> In der klassischen Arbeitsweise wir normalerweise Fall 2 existieren.
> Wenn wir erst modellieren und ein Projekt anfangen haben wir Fall 1.
> Wenn wir ein Projekt haben, für das wir zukünftig Markitect verwenden wollen haben wir Fall 3.
>
> Fall 1: Manager müssen das Projekt aufbauen. Dabei sollte eigenltich der Root Manager anfangen das Sekelett herzurichten und darauf aufbauend dann seine Kinder und so weiter.
>
> Fall 2: Manager reconcilen so wie geplant und gedacht
>
> Fall 3: Modell ableiten, so lange bis Verify passt, interaktiv mit dem Nutzer vor allem bei Widersprüchen oder solchen Dingen. Quasi apply nur umgekehrt.

## Assistant synthesis discussed before acceptance

Recorder's summary, not verbatim owner text:

- Build: Root prepares root-owned foundations, children build their areas recursively, and parents integrate the results upward. Construction follows dependencies while preserving each Manager's responsibility.
- Reconcile: the accepted model remains the specification for artifacts. A model change supplies new intent; a bug can require realization repair against unchanged intent. Parents investigate contradictory child claims and escalate unresolved meaning.
- Adopt: Managers examine existing artifacts, propose and integrate their model, and refine it through read-only Verify and owner clarification. Observations, proposed intent, accepted decisions and open questions remain distinct. Existing artifacts stay unchanged during derivation; known defects are retained rather than promoted into desired behavior. Artifact repair subsequently uses reconciliation.
- The situations share the Manager organization but have different change targets. Name the relevant project scope, since different areas can be in different situations. The owner works with understandable meaning rather than YAML.

The existing [Extract design](../../design/canonical-model/commands.md#extract-the-existing-stand) already distinguishes adoption readiness with recorded deviations from a conforming realization. The discussion's target of passing Verify does not erase that distinction or authorize weakening the model to hide a defect.

## Owner accepts documentation

> Halte das schonmal so fest im Repo in den docs. Diese 3 Fälle sind elementar und high level die Themen die hier relevant sind.

Recorder's interpretation: this accepts the discussed high-level account for documentation. It does not dispatch implementation, change the running Apply assignment, introduce model-writing rights into artifact Apply or authorize provider runs, studies or releases. The earlier discussion explicitly requested no implementation.
