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

## Owner selects Apply and Extract

Recorder's note: subsequent discussion in the same thread, now titled **Workflow**, on 19 October 2026. The earlier source text above retains its original wording and scope.

> Noch besser wäre es natürlich wenn wir Aufbau und Abgleich nicht unterscheiden müssten und einfach nur Apply hätten als: Model => Artifacts. Entsprechend dann den Command zurück: Artifacts => Model (Name fehlt noch)

Recorder's summary of the assistant proposal: use Apply for both initial construction and ongoing reconciliation, and Extract for the reverse direction. Apply makes artifacts agree with accepted intent. Extract derives and reviews proposed intent from artifacts, clarifies contradictions with the owner and retains known realization defects. This reuses the name already present in the command design; it does not propose replacing the existing command surface with only two verbs.

> Dann halten wir Apply und Extract so fest. Aktualisiere die Docs entsprechend: drei Ausgangssituationen, aber nur zwei Prozesse.

Recorder's interpretation: this accepts the names and mapping. Cases 1 and 2 are starting states of Apply; case 3 uses Extract. The earlier Build, Reconcile and Adopt descriptions identify situations and purposes rather than three separate process names. This documentation-only clarification preserves the existing direction, ownership, review, Verify and write-authority boundaries and adds no implementation assignment.

## Owner confirms the Extract walkthrough

Recorder's note: subsequent case-3 discussion in the same thread **Workflow** on 19 October 2026. The owner requested a walkthrough of starting without a model, forming the Manager structure, and refining the model with the user and Verify, with no implementation.

Recorder's summary of the assistant walkthrough, not verbatim owner text:

- Fix the existing repository stand and selected scope, with existing authoritative requirements and decisions. Root is the entry responsibility for that source scope and proposes the first Package/responsibility outline from the artifacts; folder names are evidence, not automatically the right model structure.
- Form the initial Manager tree from the provisional candidate structure. Parents supply assigned sources, provisional model context, known shared meanings and open questions before children investigate their areas. Develop source-supported model proposals recursively and integrate them upward. Review and correct wrong boundaries or newly discovered responsibilities visibly; exact provisional-tree lifecycle contracts remain future work.
- Parents reconcile duplicate terms, inconsistent statements and cross-area contracts. Consequential unsettled meaning goes through Root to the owner in understandable terms, without requiring YAML review. Keep observations, interpretations, proposals, accepted decisions and unresolved questions distinct; do not ask for already clear authoritative intent again.
- Independently review the model proposal, then assess the unchanged artifacts against the exact model candidate with read-only Verify. Use findings and owner clarification to revise, review, integrate and Verify again against the same source basis. Model quality and artifact conformity remain distinct judgments.
- The owner adopts the chosen model and proposed accounting through the normal project decision boundary. Known supported realization deviations can remain for later authorized Apply; do not weaken intent or alter the source artifacts to manufacture a passing Verify result. Unresolved material questions or evidence gaps remain incomplete.

The owner accepted the walkthrough and requested documentation:

> Ja, halte den Ablauf für Fall 3 mit Extract so in den Docs fest: erste Struktur aus den Artefakten ableiten, Manager daraus bilden und das Modell interaktiv mit dem Nutzer und Verify verfeinern. Weiterhin keine Implementierung.

Recorder's interpretation: this clarifies DEC-057's case 3 and the existing DEC-044 Extract design. It authorizes documentation only, not an Extract command/runtime, a provisional Manager-rebuild protocol, new model language, fixture or application changes, provider runs, demonstrations or releases. Earlier owner quotations and evidence retain their original scope.

## Owner confirms structure changes between Extract rounds

Recorder's note: subsequent discussion in the same thread **Workflow** on 19 October 2026. The owner asked how the provisional Manager structure changes when new responsibilities or incorrect boundaries emerge, still without implementation.

Recorder's summary of the assistant proposal, not verbatim owner text:

- Keep the provisional Manager structure stable within one Extract round. Record evidence-backed proposals for new responsibilities or changed boundaries; parents coordinate cross-area changes instead of Managers taking over each other's work.
- Review and integrate structural corrections between rounds, then derive the next Manager tree and contexts from the complete revised candidate. This candidate transition is distinct from final adoption of the model. Meaning-preserving corrections within the grant do not require separate owner approval each time; unresolved intended meaning or authority goes through Root to the owner.
- Carry model knowledge, sources, reference and identity mappings, proposed accounting, known deviations, findings and open questions fully into the new responsibilities, with their original status and bindings. Preserve historical conversations and assessments; an earlier positive result does not automatically approve a changed candidate. Keep the assignment's existing authority boundaries.
- A separately describable Reminders responsibility can be proposed beneath Lending for the next round. A mistaken placement beneath Catalog can be corrected through Root. Model responsibility does not have to mirror code folders, and correcting it does not require moving application files.
- Review and assess the complete revised candidate against the unchanged artifact basis. Exact technical transition, staging, interruption, recovery and Manager-rebuild protocols remain future design.

The owner accepted the rule and requested documentation:

> Ja, halte das so in den Docs fest: Managerstruktur innerhalb einer Extract-Runde stabil halten, Strukturänderungen zwischen Runden übernehmen und Erkenntnisse, Referenzen sowie offene Fragen vollständig mitführen. Weiterhin keine Implementierung.

Recorder's interpretation: this refines the accepted Extract workflow and DEC-057 without authorizing implementation, a new runtime state machine, canonical-model changes, artifact repair or provider runs. It does not require final human adoption of the model at every provisional round transition. Earlier source statements retain their original scope.
