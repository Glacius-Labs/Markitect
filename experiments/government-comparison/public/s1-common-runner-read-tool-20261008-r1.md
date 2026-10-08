# Gemeinsamer Runner: öffentlicher Leseversuch R1

**Vor dem Turn gestoppt. Echte Lesefähigkeit bleibt unbewiesen.** Der einmalige Grant `s1-common-runner-read-tool-20261008-r1` ist verbraucht und geschlossen; der Slot wurde am 8. Oktober 2026 um 06:32:45 UTC ausdrücklich freigegeben. Kein Wiederholungs- oder Diagnoseversuch folgte. Vor der Abschlussmeldung zeigte der kanonische Koordinationsstand noch die Zuweisung; deren Übernahme nach der dokumentierten Freigabe gehört dem Overseer.

Die Metadatenvorprüfung beobachtete die verlangten ausgewählten Konfigurationswerte und kompatible Anforderungen. Danach wurde `thread/start` einmal geschrieben. Beim Warten auf dessen Antwort löste stderr-Aktivität den vorab festgelegten Abbruch aus. Es liegt keine validierte Threadantwort vor; ob intern ein Thread entstand, bleibt unbekannt. Kein `turn/start`, kein Werkzeugaufruf und kein abgeschlossener Turn wurden beobachtet. Der stderr-Inhalt wurde verworfen; seine Ursache lässt sich aus dem Beleg nicht bestimmen. Ein verweigerter Lesezugriff ist damit ebenfalls nicht nachgewiesen.

| Tatsächlicher Zähler | Neuer Verbrauch |
|---|---:|
| Actorreservation | 1 |
| App-Server-Prozessbaum | 1 |
| Geschriebene Threadstart-Anfrage | 1 |
| Validierte Threadstart-Antwort | 0 |
| Geschriebene Turnstart-Anfrage | 0 |
| Beobachtete Werkzeug-Items | 0 |
| Zusätzliche CLI-/Produkt-/Studienläufe | 0 |

Die kumulative Bilanz enthält sechs App-Server-Bäume und sechs Actorreservationen; die fünf historischen Actorstarts werden erhalten. Die neue Reservation belegt keinen Modellstart. Neue Nutzung, native Provideranfragen/-Retries, Servingidentität und Gesamtverbrauch bleiben unbekannt. 53.331 bekannte historische Tokens und die native Historie 15/16/13/2250 bleiben unverändert. Alle sechs Vergleichszellen stehen weiterhin auf **NOT RUN**.

Metadatenphase 0,144 Sekunden, gesamte native Sitzung 0,521 Sekunden einschließlich 0,368 Sekunden Cleanup; Controller 4,947 Sekunden, äußerer Receipt-Readback 7,430 Sekunden. Die anwendbaren Fristen wurden eingehalten. Native Exit 0, Worker und äußerer Lauf Exit 1. Insgesamt 50.140 stdout- und 765 stderr-Bytes wurden nur im begrenzten Arbeitsspeicher verarbeitet und verworfen. Es gab keinen Interrupt ohne bekannte eigene aktive IDs und keinen Nachstart. Lokales Cleanup bestätigt keinen Provider-Billingstop.

Der Quellstand `1ed5d31dfc0a74dd13d09e74472fef31975d0839` und Freeze-Stand `38d6eeb4d42deb2778f2843a02f1c2a5d7666788` wurden vorab unabhängig geprüft. Die Offlineprüfungen bestanden mit 26/26 und anschließend 9/9 gezielten Fällen, darunter ein Erfolgsfall mit dem echten gepinnten Schema-Validator; ein anfänglich falsch erwarteter Fehlercode und dessen Korrektur sind erhalten. Das ist Harnessvalidierung, kein Capability- oder Qualitätsnachweis.

Alle 3.993 historischen Paketdateien und fünf Ledger bleiben unverändert. 27 Quellpins, 37 Freezeinputs, 26 relevante Schemamitglieder und sieben neue Original/Kopie-Paare sind geprüft. [Terminaler Handoff](../evidence/s1-common-runner-read-tool-20261008-r1/terminal-handoff.md), [Istbilanz](../evidence/s1-common-runner-read-tool-20261008-r1/terminal-summary.json), [Nachprüfung](../evidence/s1-common-runner-read-tool-20261008-r1/postreview.md) und [Belegmanifest](../evidence/s1-common-runner-read-tool-20261008-r1/manifest.json) bilden den Abschluss.

Aktive Threadberechtigungen, OS-Isolation, allgemeine S1-Fähigkeit, Schreiben/Build/Test, frühere Ablehnungsursachen, Vergleichsqualität und menschliche Akzeptanz sind weiterhin offen.
