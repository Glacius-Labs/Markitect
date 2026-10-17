# Owner clarification: deterministic Manager process

Date: 17 October 2026. Source: owner conversation `01a148ed-f2c8-7f91-85db-76be64df6bfe`. The German passages below preserve the owner's words. The current contract belongs in the [Apply concept](../../design/canonical-model/manager-apply-concept.md#tasks-and-delegation), not in this source record. This clarification supersedes the earlier parent-authored child prompts and root-planned order; it does not claim implementation or measured benefit.

## Concern about drift

> Hier sehe ich schon einen Drift von dem was ich mir ursprünglich gedacht hatte.
> Wir haben ja klare Regeln und ein explizites Modell. Kontext können wir größtenteils deterministisch weitergeben. Genauso können wir die Delegation heruntergeben. Ich möchte eigentlich nicht, dass der Root Manager plant, welche Manager es gibt und wer welchen Prompt bekommt. Das macht den Prozess weniger zuverlässig und shifted wieder Vertrauen aus Prozess in AI. Die Idee von einer typisierten Modellsprache war es die Struktur zu nutzen um vieles einfach deterministisch und damit zuverlässig zu verteilen.

## Compiler, participation and communication

On context compilation:

> Das hier kann der Compiler zusammenstellen. Das muss keine AI machen. Die Beziehungen sind bereits explizit modelliert. Wenn nicht ist es ein Modellierungsfehler, den wir hier beim Apply nicht kompensieren wollen.

On participation and communication:

> Sicherer wäre es hier, wenn jeder Manager angeschaltet ist und entscheidet ob es seinen Bereich betrifft. Es soll nicht daran scheitern, dass der Root Manager etwas übersieht. Ausserdem wollen wir sicher sein, dass jeder Manager nur mit seinen direkten Kindern sprechen kann oder mit seinem Parent.

On the review and integration procedure:

> Das sollte fest und deterministisch sein. Das darf nicht davon abhängen, was eine AI plant.

## Proposed Apply flow

> Bei der Reihenfolge stelle ich mir das eigentlich in etwa so vor: Beispiel Apply
>
> - Apply wird in einen statischen Prompt übersetzt: "Sorge dafür dass das Artefakte mit Modell übereinstimmen" oder so ähnlich
> - Kontext wird zusammengestellt und damit das Template befüllt und dann der Prompt noch mit eingefügt.
> - Root Manager erhält den Prompt und gibt ihn an seine Kinder Manager weiter.
> - Kinder Manager gibt an Kinder Manager weiter, rekursiv alles, Kontext und drumherum wird aus Compiler, Konfigurationen etc zusammengestellt für jeden Manager separat.
> - Leaf Manager prüfen ihren Stand und melden "passt" oder passen ihren Code entsprechend an und melden das ihrem Parent Manager. Falls es Abhängikeiten gibt oder ähnliches muss es den Baum hoch und an der entsprechenden Stelle wieder runter, wieder indem es entweder deterministisch abgeleitet wird oder indem alle gefragt werden. Und so läuft dann die Kommunikation den Baum hoch und runter bis alle meinen durch zu sein und alle Manager akzeptiert haben.
> - Eventuell macht hier auch ein wenig Koordination, bestenfalls relativ deterministisch Sinn.
> - Wenn alles passt wird success gemeldet und der Command schließt ab. Jeder Agent startet immer als Session und macht erstmal nichts, bevor er keine Nachricht bekommt.
> - Jeder Manager weiß nur, das was er wissen muss und sollte nicht zu viele Details kennen und es nur delegieren.
>
> So in etwa stelle ich mir den Prozess vor und so in etwa läuft es ja auch in Unternehmen.
>
> Dabei könnten auch Metriken gesammelt werden und Hot Paths entdeckt werden etc. Ich habe hier sicher nicht alles bedacht aber so im Großen und Ganzen sollte das in etwa so laufen denke ich.

## Confirmation and documentation request

The assistant refined this into unchanged Host-guaranteed recursive propagation, every local Manager participating, local work plus upward integration, adjacent-only communication, explicit dependency waits, parent-owned prerequisite work, independent review, bounded corrections and exact final evidence/publication. The owner confirmed:

> Ja, so meine ich das. Vergleiche diesen Ablauf bitte mit dem aktuellen Konzept und der Umsetzung. Wo müssen wir konkret nachschärfen und was sollten wir dem Coordinator jetzt mitteilen?
>
> Ausserdem sollten wir das genauso festhalten, damit wir das nicht vergessen und das Konzept entsprechend anpassen. Der Prozess muss klar sein und gut durchdacht. Wir wollen dabei so nah an der Metapher bzw Realität bleiben wie möglich. Sobald das geklärt ist können wir auch schauen, wie man das am besten technisch realisiert und ob unser aktueller Ansatz dazu passt.

The organization metaphor describes bounded responsibility, messages, escalation and integration. It is not permission for AI Managers to select participants, rewrite binding instructions or invent the workflow. At this point in the conversation, exact queue ordering, message schemas and provider-session allocation remained implementation/design questions. The later confirmation below selects the queue policy while leaving technical representations open. The owner separately authorized direct coordination with the implementation Coordinator.

## Explore, planning and limited deliveries

Later in the same conversation, the owner asked for independent subagent analysis and proposed:

> - Erst Explore durchlauf
> - Dann Plan durchlauf
> - Dann Koordination, manche Agenten müssen jeweils warten, weil sie eventuell auf etwas aufbauen, was noch geändert wird.
> - Dann Review und Integration
>
> Dabei ist es wichtig, dass Agenten höherer Ebene einen breiteren und weniger tiefen Stand haben und mit den Aussagen arbeiten die ihre Kinder mitteilen.

The owner also asked about locks, parent hints, parallel work and a balanced deterministic process. Three subagents examined prerequisites, readiness and review invalidation. The assistant assessed their proposals and recommended this refinement:

- All Managers first report a read-only Explore assessment before artifact writing starts. Local planning follows without a global Plan barrier.
- Concrete prerequisites govern readiness. The Host chooses the oldest eligible work, with stable ties; determinism is relative to recorded events, not a promise of identical provider timing across runs.
- A limited normal delivery can become usable while its Manager stays open. Independent review, Host acceptance and composition into the consumer's exact basis precede dependent implementation. Reviews therefore happen during handoffs as well as at integration.
- Parent hints remain separate from the unchanged operation and compiled context. They cannot create binding model rules or narrow responsibilities. Parents use broad evidence-linked child summaries and request detail when needed.
- Dependency waiting is separate from resource exclusion. A waiting Manager holds no active write grant; uncertain effects are reconciled before handoff. Read-only work and reviews of immutable candidates may run concurrently; the first Apply retains one writer.
- Prior review remains current only for its unchanged complete bound subject. An AI claim of no impact is insufficient.

The owner confirmed the proposed flow and requested persistence and coordination:

> Ja, der Ablauf passt für mich so. Halte das im Konzept fest und stimme es mit dem Coordinator ab. Lass uns noch klären, wie wir das Review-Budget sinnvoll gestalten, ohne erfolgreiche Teillieferungen unnötig zu begrenzen.

This confirmation selected the flow above and initially left the review-budget replacement open: the existing three-submission limit could be consumed by successful limited deliveries. The following exchange resolved that question. This source records product intent, not implementation or execution evidence.

## Review budget confirmation

The assistant proposed a separate rejection counter per run, Manager and review purpose, with no reset through successful deliveries or new tasks. All work would still consume the finite shared execution budget. The explicit question was:

> Soll das Review-Limit künftig Zurückweisungen zählen: standardmäßig nach zwei Ablehnungen prüft der Parent den Korrekturplan, nach der dritten endet die betroffene Arbeit unvollständig; erfolgreiche Teillieferungen zählen nicht mit und setzen nichts zurück? Alle Arbeiten verbrauchen weiterhin das endliche Gesamtbudget.

The owner selected:

> Ja, Zurückweisungen separat begrenzen

The [execution contract](../../design/canonical-model/manager-apply-execution.md#admission-amendments-and-finite-correction) records this selected rule. Successful limited deliveries do not consume or replenish the rejection allowance; local and integration review remain distinct purposes. This supersedes the earlier submission-count design, not the finite assignment budget or mandatory evidence. Technical implementation and real Apply proof remain separate work.
