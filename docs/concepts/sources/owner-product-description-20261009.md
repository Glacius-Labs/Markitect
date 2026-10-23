# Owner product description, 9 October 2026

- **Source type:** owner statement, preserved verbatim (German).
- **Recorded:** 9 October 2026 by the Concepts chat as part of its setup assignment; copied unchanged into this record on 10 October 2026. The date is the date of capture, not a claim about when the text was first written.
- **Use:** this is the origin of the [concept record](../README.md). The [register](../register.md) and the [vision](../../vision.md) interpret it; they do not replace it. Do not edit the quoted text below. Record a later change of meaning as a new or superseding register entry.

The two sections below keep the Concepts chat's original German headings and its provenance note for the clarification.

## Separat übergebene vorangegangene Klarstellung

Der Einrichtungsauftrag übergibt folgende Klarstellung. Sie ist getrennt von der unten ausdrücklich als wörtlich gekennzeichneten Produktbeschreibung erhalten; ein ursprüngliches Gesprächsprotokoll liegt dieser Aufnahme nicht zugrunde.

> In der Markitect-Welt pflegt man nicht viele Stellen gleichzeitig und hält sie manuell synchron. Man pflegt eine maßgebliche Stelle, das kanonische Modell, und lässt die Änderung anwenden. Der Mensch bestimmt das Soll; abgeleitete Realisierungen werden durch delegierte Arbeit angepasst und geprüft.

## Wörtliche Produktbeschreibung

```text
Korrekt. Meine Erfahrung zeigt, dass es AI Agenten sehr schwer fällt an alle relevanten Dateien zu denken und diese Dauerhaft konsistent zu halten. Bei kleinen Projekten geht das noch, aber je größer und komplexer es wird desto mehr wird vergessen. Selbst explizite Konsistenzruns finden nicht alles, brauchen exponentiell länger und haben Probleme Widersprüche aufzulösen. Wenn wir jetzt wollen dass Software Systeme Simpel im Sinne von verständlich sind und man über es Schlussfolgern kann brauchen wir klare Fakten darüber was es tut. Ausserdem steht dann noch die Frage im Raum, ist das was es tut denn auch das was es tun soll?

Markitect versucht das anzugehen durch ein kanonisches Modell, eine Wahrheit, ein Weltbild. Dieses gilt und die Realität bzw das Projekt bzw das Repository soll es widerspiegeln. Woher weiß ich jetzt wenn ich etwas an der Doku anpasse und einen Agenten losschicke, dass weiterhin alle anderen Regeln noch gelten und nicht kaputt gegangen sind durch die Änderungen und dass nichts vergessen wurde? Im Prinzip weiß ich es nicht ausser ich schaue mir mühsam genau an was er getan hat. Damit bremse ich die AI aber massiv aus. Daher versucht Markitect diese Probleme durch Struktur, Ordnung und Methodik zu lösen.

Es gibt ein kanonisches Modell mit einem "Typsystem", ein Compiler welcher prüfen kann ob dieses Modell "kompilierbar" ist, sodass man nicht erst "zur Laufzeit" bemerkt dass etwas nicht passt und klare Beziehungen zwischen den Dateien im Projekt und was sie eigentlich realisieren oder kurz gesagt: "Was gehört wozu?". Dadurch lässt sich auch sagen, welche Änderungen des Modells, welche Dateien betreffen und der "Blast Radius" grenzt sich ein wodurch es schwieriger wird etwas zu übersehen.

Da die Struktur im Modell ähnlich wie bei einer klassischen Markdown Dokumentation, klassischem Code und ähnlichem durch Ordner bzw Namespaces entsprechend gruppiert und organisiert ist lassen sich dadurch Verantwortungsbereiche, Abstraktionsebenen und eine Art Management Hierarchie ableiten. Diese Management Hierarchie besteht aus vielen Agenten mit klaren und begrenzten Zielen und Interessen. Das soll es zum einen ermöglichen deutlich günstigere, wobei mittlerweile ähnlich fähige Modelle zu benutzen statt die teuersten High End Modelle draufzuwerfen. Dank der Struktur und der Diff Analyse ist klar was alles geprüft werden muss nach einem Change und durch die vielen Subagenten wird nichts bei der Untersuchung vergessen und man hat Quasi das 4 Augenprinzip. Das soll minimieren dass etwas vergessen wird oder eine relevante Stelle nicht untersucht wird und etwas inkonsistent wird. Gleichzeitig ist das Modell klar und dadurch lässt sich auch ein Gesamtcheck starten, welcher Konsistenzprüfungen macht und alle Bereiche mit der kanonischen Wahrheit abgleicht. Auch Aufräumarbeiten wie Cleanups und Refactorings lassen sich so zuverslässiger umsetzen. So zumindest die Hoffnung.

Natürlich können wir nicht verhindern dass schlecht spezifiziert, modelliert oder organisiert wird, oder dass die Arbeitesweise nicht eingehalten wird. Aber das können Code und andere Tools auch nicht verhindern. Markitect soll für saubere und einheitliche Repositories, mehr Qualität bei der Arbeit und damit größeres Vertrauen in Coding Agents durch Systematik und Struktur und einen sauberen Spec Driven AI First Development Ansatz anbieten der Struktur, Methodik und Ordentliche Definition belohnend macht und nicht zur Aufräumhölle.
```
