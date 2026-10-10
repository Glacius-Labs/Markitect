# Government als Modellpflege im Betrieb

**Herkunft und Status:** Direkter Nutzerbeitrag aus der Produktdiskussion `01a121f1-b948-7050-ae5d-9921b99db9c0`, 9. Oktober 2026. Die Concepts-Sammlung vergibt die lokale Referenz `USER-20261009-02`; sie ist keine Historian-Fund-ID. Der Nutzer ordnet Government als frühere Idee ein. Eine unreife Implementierung vermutet er auf einem Branch; diese wurde für die Notiz nicht lokalisiert oder verifiziert.

## Unveränderter übergebener Wortlaut

```text
Hierzu gab es mal eine Art Government Idee. Eine relativ unreife Implementierung liegt irgendwo in einem Branch herum. Die Betrachtungsweise war hierbei folgende: Wenn ich ein Modell ändern kann und es anschließend relativ zuverlässig überprüfen und umsetzen lassen kann sodass das Repository genau dieses Modell abbildet, dann besteht meine Aufgabe in der Pflege des Modells.

Nun arbeitet man normalerweise so, dass es User Stories, Features oder kurz gesagt Work Items gibt. Diese müssten interpretiert und angemessen in das Modell eingearbeitet werden. Auch gibt es technische Themen die irgendwie berücksichtig werden müssten. Wenn Änderungen immer nur durch das Modell gehen können, müssen wir damit irgendwie umgehen. Wenn mir Markitect erlaubt den Coding Agents mehr zu vertrauen und die Sauberkeit des Repositories und damit des Projekts zu gewährleisten und jederzeit auf Knopfdruck prüfen zu lassen verschiebt sich meine Aufmerksamkeit auf jede Änderung des Modells. Dabei ist das Endziel ja möglichst viel Vertrauen in die Coding Agents und dadurch eine möglichst hohe Autonomie der AI ohne dass man befürchten muss dass das Repository am Ende komplett chaotisch und unübersichtlich wird. Es kann auch passieren, dass beim Anwenden des Modells Probleme auffallen und dann müsste ich ja am Ende mehr anleiten als vorher. Daher war die Idee auf Markitect eine Art Regierung aufzubauen. Ich bin der Präsident und bekomme meine Briefings, werde nur bei relevanten Themen gefragt und ansonsten kümmern sich "Gerichte" und "Ämter" um die Pflege und Anpassung des Modells. Dabei kann es auch Ministerien oder ähnliches geben, wie eins für Sicherheit, Architektur, Code Hygiene, Anwenderfreundlichkeit, Innovation etc. Diese schalten sich vor allem bei Modelländerungen ein und dürfen ihre EInwände geben und gehen miteinander in eine Art Gericht.
Das Apply ist sozusagen die Exekutive, und wir fügen Judikative und Legislative hinzu. Damit hätte man eine Art Organismus geschaffen mit hoher Autonomie und Einstellbarkeit. Wie gut das sich realisieren lässt und was dabei in der Praxis herauskommt muss man testen und ausprobieren. Ausserdem muss das irgendwie mit dem Management Konzept vereinbart werden und wir müssen irgendwie dafür sorgen dass das Modell durchgängig eine hohe Qualität hat und nicht verkommt.

Konzeptbezug: Delegation der Modellpflege zusätzlich zur Realisierung; Work Items und technische Bedürfnisse müssen in akzeptiertes Soll übersetzt werden. Mensch/„Präsident“ erhält Briefings und wird bei relevanten Themen gefragt. Fachministerien können Einwände einbringen, Ämter bearbeiten und Gerichte beurteilen Konflikte. Sicherheit/Architektur/Hygiene/UX/Innovation sind Beispiele. Offene Fragen ausdrücklich erhalten: Vereinbarung mit rekursivem Management, laufende Qualität des Modells, Eskalationsaufwand und tatsächliche autonome Leistungsfähigkeit. Keine Feinplanung, Tests, Studien oder neue Roadmap starten. User wartet dafür auf die laufenden Pakete, Studienzahlen und stabile Version. Keine ACK-Schleife.
```

## Konzeptstruktur

Die Nutzerbeschreibung verknüpft Modelländerungen, Interpretationen von Work Items und technische Anliegen mit der Idee, die Modellpflege an Ämter und fachliche Prüfrollen zu delegieren. Briefings und Eskalation sollen dem Menschen Sichtbarkeit über relevante Vorgänge geben, während Routinearbeit autonomer laufen kann. Der Nutzer nennt ausdrücklich das Risiko, dass Anwendung des Modells mehr Anleitung verlangt und das Modell im Betrieb an Qualität verliert.

Die ausführlichere Einordnung und die getrennte Konzepterfassung stehen als C20 bis C22 in den [übergreifenden Konzeptnotizen](canonical-model-and-delegated-development.md). Die Regierung, Präsidentenrolle, Ämter, Gerichte und Ministerien sind als Analogie und frühere Idee überliefert. Das Apply als Exekutive sowie Legislative und Judikative beschreiben das Gedankenbild, keine vorhandene oder angenommene Architektur.

## Status der möglichen Rollen

| Rolle im Gedankenbild | Vom Nutzer benannte Aufgabe | Offene Abgrenzung |
|---|---|---|
| Präsident | Briefings erhalten und bei relevanten Themen gefragt werden | Befugnisse, Relevanzschwelle und Ausnahmeverfahren |
| Ämter | Modellpflege und Anpassung übernehmen | Mandat und Abhängigkeit von Entscheidungsautorität |
| Ministerien | Bei Modelländerungen Einwände vorbringen; Beispiele sind Sicherheit, Architektur, Code Hygiene, Anwenderfreundlichkeit und Innovation | Auswahl, Dauerhaftigkeit und Konfliktlösung |
| Gerichte | Einwände miteinander beurteilen und Konflikte bearbeiten | Unabhängigkeit, Urteilskraft und Verhältnis zu Annahme/Entscheidung |
| Exekutive | Apply führt die beschlossene oder anderweitig autorisierte Änderung aus | Konkreter Übergang von Urteil oder Entscheidung zu Apply |
| Legislative und Judikative | In die bisherige Apply-Exekutive-Analogie ergänzte Bereiche | Verhältnis zu den Gerichten und zu Änderungen des geltenden Solls |

Die Tabelle ordnet allein die überlieferte Analogie. Sie weist keiner Rolle neue Befugnisse zu.

## Offene Fragen und Planungsgrenze

Der Nutzer stellt Umsetzung, praktische Wirkung, Vereinbarkeit mit rekursivem Management, Erhalt der Modellqualität und tatsächliche Autonomie ausdrücklich zur Diskussion. Eine Government-Variante ist dadurch nicht als Produktentscheidung angenommen. Der Nutzer hat Feinplanung zudem bis zum Abschluss der laufenden Arbeitspakete, dem Vorliegen der Case-Study-Zahlen, einem sauberen Main-Stand und einer verwendbaren stabilen Markitect-Version zurückgestellt. Dieser Eintrag beginnt keine Feinplanung, keine Tests oder Studien und keine Roadmap-Arbeit.
