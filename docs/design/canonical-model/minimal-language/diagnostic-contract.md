# Diagnosekoordinaten: separater Werkzeugvertrag

**Status:** Gewählter Werkzeugvertrag, getrennt von der [normativen Sprachspezifikation v1](language-contract.md). Dieses Profil legt die Bedeutung exakter Körperlinkkoordinaten fest. Es ist keine zusätzliche Voraussetzung für Sprachkonformität oder kanonische Bytes und keine neue Core-Definition. Eine vollständige Unterstützung durch den Go-Produktcompiler wird damit nicht behauptet.

Werkzeuge dürfen dieses Profil anbieten. Wenn sie eine genaue Koordinate ausgeben, MUSS sie die nachfolgenden Regeln erfüllen; sie dürfen keine Positionen erraten. Für das Profil gelten MUSS, DARF und DARF NICHT als verpflichtende Regel, Erlaubnis und Verbot.

## Koordinatenform

- Jedes zugeordnete Körperlinkvorkommen erhält `startByte` und `endByte`: einen nullbasierten, am Ende exklusiven UTF-8-Bytebereich `[startByte, endByte)` im dekodierten ursprünglichen `body`. Er umfasst die vollständige Linknotation einschließlich beider Doppelklammern. Er darf nicht gegen gerenderten oder für Markdown normalisierten Text gezählt werden.
- Die Zuordnung zum ursprünglichen YAML erhält zusätzlich `source.file`, `source.start` und `source.end`. Start und exklusives Ende verwenden einbasierte Zeile und Spalte. Spalten zählen Unicode-Codepoints, keine UTF-8-Bytes oder Bildschirmbreiten. CRLF zählt als ein Zeilenumbruch.
- YAML-Escapes, Faltung, Anführungszeichen und Markdown-Container müssen in der Abbildung berücksichtigt werden. Ein über Dateizeilen gefalteter Link darf einen mehrzeiligen YAML-Quellbereich besitzen, obwohl der dekodierte Link einzeilig ist.
- Wiederholte Links bleiben einzelne, geordnete Vorkommen. Gleiche Ziele oder Koordinaten dürfen keine Deduplizierung auslösen. Koordinaten sind keine zusätzliche Identität und werden nicht in kanonische Modellbytes aufgenommen.

## Fehlende oder nicht belegbare Koordinaten

Die Linkerkennung und Zielauflösung MÜSSEN von der Koordinatenberechnung unabhängig bleiben. Fehlende, veraltete oder nicht belegbare Quellenabbildung und reine Ressourcenabbrüche der Koordinatenphase dürfen weder bekannte Linkvorkommen entfernen noch ein Inhaltsurteil ersetzen. Sie dürfen insbesondere keine erfolgreich abgeschlossene kanonische Projektion verhindern.

Ein Werkzeug MUSS fehlende Profilabdeckung ausdrücklich ausweisen. Vorhandene belegbare Teilkoordinaten dürfen erhalten bleiben; fehlende Koordinaten dürfen weder mit Nullwerten noch mit geschätzten Werten als vollständig ausgegeben werden. Die Definition und die Reihenfolge ihrer erkannten Vorkommen bleiben auch ohne exakte Koordinaten bekannt.

Wenn die Inhaltsprüfung selbst wegen Parser-, Auflösungs- oder Ressourcenproblemen nicht abgeschlossen ist, bleiben die Regeln der Sprachspezifikation wirksam: Für diesen Modellumfang gibt es keinen gültigen Digest. Bereits nachgewiesene Sprachfehler bleiben Fehler. Ein unabhängiges Koordinatenergebnis ist kein Nachweis vollständiger Inhaltsprüfung.

## Bericht der Entwurfsprobe

Der [Exchange-Prüfer](exchange-example/verify.py) trennt diese Ergebnisse:

| Berichtsteil | Umfang |
| --- | --- |
| `body_link_validation` | Syntax und aufgelöste geordnete Linkvorkommen, ohne vorgeschriebene Koordinaten. |
| `canonical_models` | Vollständig geprüfte kanonische Modellbytes und Digests je Modellumfang. |
| `diagnostic_coordinates` | Getrennte Abbildung nach diesem Profil mit eigenen Vorkommen und Angaben zur fehlenden Abdeckung. |

`diagnostic_coordinates.status` ist `passed`, wenn alle dort betrachteten Vorkommen vollständig zugeordnet sind, und sonst `not_completed`. Bei noch nicht erkannten oder nicht aufgelösten Inhalten beansprucht die Probe ebenfalls keine vollständige Koordinatenabdeckung. Der Status beeinflusst weder Modellgültigkeit, Digest noch den Gesamtstatus der Inhaltsprüfung. Ein eigenes Konformitätsurteil über dieses Werkzeugprofil muss ihn gesondert betrachten.

[diagnostic_coordinates.py](exchange-example/diagnostic_coordinates.py) enthält die Quellenabbildung; [diagnostic_probes.py](exchange-example/diagnostic_probes.py) prüft ihre Trennung von der kanonischen Auswertung. Die existierenden Gegenproben für genaue Byte-/YAML-Bereiche bleiben Werkzeugprofiltests. Historische Berichte behalten ihre damalige Struktur und Aussagegrenze.
