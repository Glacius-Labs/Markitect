# Roadmap planning with the owner, 10 October 2026

- **Source type:** owner statements, preserved verbatim (German). Terminal line wrapping is removed; the wording is unchanged.
- **Setting:** after pull requests #89 to #91 were merged, the owner and the integrator session planned the next phase. The [roadmap](../../implementation-plan.md) and the [backlog](../../work-items/backlog.yaml) are the result.
- **Use:** the [register](../register.md) records the decisions taken from these statements. Do not edit the quoted text.

## Priority: a clean, stable and testable main

```text
Unser Ziel ist es jetzt, nachdem der Integrator alles auf main gemerged hat, den Cleaner alles aufräumen zu lassen damit das Repo sauber, professionell, gut dokumentiert, einheitlich und stabil ist. Du solltest jetzt dafür sorgen, damit du dich abhängig vom Cleaner orchestrierst und deine Änderungen mit ihm auf main bekommst. Dadurch erhoffe ich mir erstmal einen sauberen main Stand. Von dort aus müssen wir planen wie wir weiter vorgehen.

Der Scientist hat deinen Plan bekommen und hat ihn mit seinen Vorhaben abgeglichen und implementiert bereits. Eine weitere frische Session hat deinen Prompt für die Runtime bekommen und implementiert gerade dort entsprechend. Wir müssen nicht unbedingt auf die beiden warten, sollten sie bei Gelegenheit aber integrieren und zusammenführen und anschließend dafür sorgen, dass Testläufe entsprechend zuverlässig funktionieren und tatsächlich Markitect prüfen und nicht irgendwelche Provider CLI Spezifikia, mit Berechtigungen, Zeichenformaten oder sonstigen environmentspezifischen peripheren Problemen kämpfen. Oberste Prio ist einen sauberen und stabilen und testbaren main Stand zu bekommen.
```

## Structure and conventions

```text
Sekunde, du solltst nicht unbedingt den Konventionen des Cleaners folgen. Nur wenn du meinst dass sie sinnvoll sind und keinen Grund siehst sie anzupassen kannst du das so beibehalten. Ansonsten restriktiere dich davon nicht unnötig. Du müsstest in deinem PR nur sicherstellen, das dann entsprechend umzuziehen. Mir ist nur wichtig, dass wir eine gute organisierte Struktur im Repo haben, dass alles seinen Platz hat und die Konventionen einheitlich sind
```

## Test reliability is reworked separately

```text
Merge jetzt deinen Stand und überprüfe ihn dann nochmal kurz. Dann gebe ich dir ein Liste an Themen die ich als nächstes angehen will, wir planen und erstellen eine Roadmap und starten dann entsprechend den Prozess. Ich habe dem Cleaner bereits mitgeteilt dass unsere Testsuite, Pipeline, auto Smoke Tests und so weiter noch relativ unzuverlässig laufen und demnächst separat jeweils überarbeitet und gehärtet werden. Daher soll uns das erstmal nicht unnötig aufhalten.
```

## Topic list

```text
Hier ist meine grobe Themenliste:
- Case Studies, Fine Tuning, Playground Achitecture: Architektur für den Playground verstehen, dokumentieren, optimieren und parametrisierbar und einfach ausführbar machen und härten.
- Runtimethema implementieren und einbinden.
- Vereinfachen der CLI Verbs so wie man es auch von anderen CLIs wie kubectl, terraform etc kennt. Kein unnötiges project z.B. zwischendrin und convenience verbs.
- Knowledgegraph evaluieren, implementieren, als Variante testen: Muss über die überarbeiteten tests und runtime tests getestet werden.
- Unit Tests und generelle saubere und modulare testsuite, die jede der funktionen entsprechend automatisiert abtestet sodass die Funktionalität der CLI jederzeit gewährleistet werden kann.
- Runtime Tests, CI CD Tests etc härten, sauber und vollständig implementieren und eventuell sehr ähnlich zum Playground aufbauen.
- Appserver und MCP Path prüfen ob sinnvoll, prüfen dass einheitlich mit CLI Verbs und Funktionen und mit Test Suite absichern. Ausserdem auch über runtime tests bzw playground testbar machen.
- Bug Hunter Chat mit Subagents, die nach Bugs suchen und das Repo und die Implementierungen weiter härten und stabilisieren.
- Skills, Agents.MD, Claude.md und .codex und .claude files für den Ablauf nochmal prüfen, konsolidieren, mit den CLI Verbs und Funktionen vereinheitlichen und eventuell verbessern.
- Code und Architecture Cleanup und Dokumentation + Map für den Überblick und generell nochmal durch die Dokumentation und Architektur durchgehen und in Module, Probleme etc organisieren.
- Government, Autonomous Organism, Cockpit, Personal Assistant, Monitoring, Gerichte und so weiter.
- Lerneffekte untersuchen, die wir ergänzen könnten
- Weitere aufgenommene Ideen prüfen, evaluieren, planen, umsetzen, ggfs evaluieren über Playground und integrieren.

Bei einigen dieser Dinge müsste ich eventuell aktiv mitreden wobei bei den meisten Themen bereits ein Umsetzungsplan oder eine Umsetzung irgendwo in einem Branch rumliegt. Ich möchte soweit parallelisieren wie es sinnvoll geht und kontinuierlich entsprechend integrieren. Ausserdem wollen wir den Überblick nicht verlieren und müssten uns eine einheitliche Benamung für die Branches ausdenken, die Arbeitspakete entsprechend aufteilen und definieren und eine Art Roadmap erstellen um den Überblick nicht zu verlieren.
```

## A fresh roadmap; earlier plans are inputs

```text
Nimm keine Roadmap eines anderen Branches einfach an. Die Wahrscheinlichkeit ist hoch, dass sie auf alten Ständen beruhen. Die Ideen und Konzepte sollten wir aufnehmen und gesondert gegen den aktuellen main Stand und die aktuelle Vision prüfen und entsprechend berücksichtigen. Aber wir machen aktuell eine komplett neue frische Roadmap auf dem aktuellen Stand basierend und auf der Themenliste.
```

## Windows

```text
Beachte bei unserem Playground und tests die wir machen, dass wir Windows zunächst erstmal nicht wirklich berücksichtigen müssen falls es uns zu sehr bremst oder zu viele Probleme bereitet. Wir könnten später überlegen wie wir damit umgehen.
```

## Branch names

```text
Wir sollten ab jetzt die Branches auch mit dev/ oder sinnvollen präfixen wählen statt claude/ und codex/. Damit können wir dann auch die "alten" Branches von den aktuellen unterscheiden und nach und nach aufräumen. Die namen sollten entsprechend sinnvoll gewählt sein und nicht irgendwelche Nummern hinten haben.
```

## Push rule

Recorder's note: the integrator proposed this rule after a session waited for a non-required Windows run before pushing a review fix:

> Korrekturen auf einem PR dürfen sofort gepusht werden, auch wenn ein nicht erforderlicher Lauf (Windows) noch läuft. Erforderlich ist nur Linux (DEC-013). Die Regel „kein Push während CI“ gilt weiterhin für den erforderlichen Linux-Lauf.

The owner answered:

```text
Ja, nimm die Push-Regel so in die Roadmap auf
```

## Compatibility

```text
Kompatibilität ist auch kein Thema. Wir entwickeln noch aktiv und niemand sonst benutzt Markitect. Also Kompatibilität soll kein Driver für Entscheidungen oder Implementierung sein
```

## External libraries

Recorder's note: the integrator asked which third-party libraries Markitect's inner layers (Core, Modules, CLI) may use. It offered three options: an explicit allowlist enforced by the architecture gate, standard library only, or no rule. The owner answered in English:

```text
Markitect should use any library that helps to achieve its goal. We shouldnt be unnecessarily strict about it 
```
