# Agent workspaces and reviewed handoffs

Direct owner statements in the Apply workflow conversation, 22 October 2026. Quotations are verbatim. The context and disposition are recorder notes, not additional owner quotations.

## Focused mandates

> Wir wollen für jeden Durchlauf einen klaren und präzisen Auftrag haben. Das Implement soll beispielsweise nicht gleichzeitig nach Verbesserungsmöglichkeiten suchen, Modellplausibilität prüfen und Implementierung durchführen. Wir wollen liebere mehrere spezifische Runs.

> Ausserdem gibt es Dinge wie Belege und Turns und hier vermute ich dass wir zu eng koordinieren und damit immer wieder Koordinationsfehler bekommen

## Simple collaboration model

> Ich denke selbst das ist hier zu kompliziert. Die Grundidee war eigentlich 1 Container pro Manager bzw pro Agent. Jeder arbeitet auf seinem Stand und übergibt dann an seinen Manager. Der Manager integriert und prüft. So geht es bis nach oben in etwa. Ausserdem dürfen Manager ihren Eltern fragen stellen oder ihren Kindern Aufgaben geben. Also delegieren und eskalieren. Viel mehr sollte hier eigentlich gar nicht passieren.

## Concept commission

> Das geht in die richtige Richtung. Arbeite daraus ein konkretes Zielkonzept aus: ein Container und eigener Stand pro Agent, keine fachlichen Turns, Delegation, Rückfragen und geprüfte Übergaben nach oben. Noch nichts implementieren.

## Proportionate technical enforcement

> Wir müssen übrigens auch nicht unbedingt technisch für Schreibschutz und solche Dinge sorgen. Das kommt mir auch zu pingelig vor. Wir sollten es uns nicht zu kompliziert machen.

## Adoption and documentation scope

> Das passt. Übernimm das Zielkonzept in die bestehenden Konzeptdokumente und leite daraus einen konkreten Umbauplan für Apply ab. Noch nichts implementieren.

> Ausserdem beachte, dass wir nicht unbedingt kleinkariert für technischen Schreibschutz an jeder Stelle sorgen müssen. Wir sollten das nicht zu kompliziert gestalten.

## Recorder disposition

[DEC-067](../register.md#dec-067-agent-workspaces-and-reviewed-handoffs) records the adoption. The existing [Apply workflow](../../design/canonical-model/manager-apply-workflow.md) owns the accepted target, including the accepted draft's Git handoffs, parent review/integration, fresh final Verify, retained observations and ordinary Git transfer. The [delivery plan](../../implementation-plan.md#apply-rebuild-plan) owns the derived implementation sequence, not execution authority.

This replaces conflicting shared-tree, Pass/Turn and per-write-enforcement choices for the target. The 21 October rollback decision and failed reuse attempt keep their historical scope; the new design is not a restart of that experiment. This task updates documentation and registry accounting only. It starts no implementation, provider run, container experiment, deployment or new grant, and does not cancel or enlarge another session's existing finite assignment. Current code and canonical self-model remain the implemented baseline pending an explicitly commissioned migration.
