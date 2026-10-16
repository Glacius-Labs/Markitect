# Vollständiges vereinfachtes Exchange-Beispiel

Stand: 16. Oktober 2026. **Aktueller Status: wegen Command/command unter den neu gewählten Schreibregeln nicht konform.** Die fachliche Modellierung dieser beiden Definitionen ist auf Nutzerwunsch vor einer Umbenennung oder Zusammenfassung zu klären. Der aktuelle Prüfer muss den Konflikt melden und darf ihn nicht selbst korrigieren.

**Autorenorganisation:** Die gewählte Konvention ist package by feature / vertical slices: zusammengehörige Kinds, Properties und konkrete Gegenstände werden fachlich gemeinsam abgelegt, weiterhin mit einer Datei pro Definition. Der ältere Aufbau dieses Durchstichs trennt noch vocabulary/ und project/, um Imports und Referenzformen zu erproben. Er ist keine empfohlene Vorlage für die Package-Aufteilung. [Abschnitt 7.3](../language-contract.md#packages) zeigt die fachliche Gruppierung; der Strukturprüfer entscheidet nicht anhand von Ordnernamen über deren fachliche Angemessenheit.

Dieses eigenständige Designbeispiel verwendet die [abgeschlossene v1-Designbaseline](../language-contract.md), vom Nutzer am 16. Oktober 2026 bestätigt: eine Datei pro Definition, dateilokale Imports, direkt verfügbare importierte Namen, kurze Property-IDs, deren vollständige ID als Standardfeldname und den Core mit neun Definitionen. Die bekannte Command/command-Kollision betrifft das Beispielvokabular und ist keine offene Sprachentscheidung. Das Beispiel ist kein Eingabeformat des heutigen Produktcompilers und kein ausgeführter Connector. Markitect ist in Go implementiert; der Python-Prüfer ist eine separate Entwurfsprobe und keine Entscheidung für Python oder PyYAML im Produkt.

Der [Bericht vor dem Designabschluss](verification-before-v1-design-baseline.json) bleibt erhalten; [verification.json](verification.json) bindet die aktualisierten Statusangaben. Der Abschluss ändert keine Sprachregel, Core-Definition, kanonischen Core-Bytes oder externe Lockbindung. Produktive Go-Implementierung und ein breiter Konformitätskorpus bleiben gesonderte Arbeit.

## Modelle, Packages und Dateien

| Modell | Manifest | Fassung | Inhalt |
| --- | --- | --- | --- |
| connector-example | [model.yaml](model.yaml) | Lokal, ohne Veröffentlichungsnummer | Vokabular und Projekt: 24 Definitionen. |
| markitect-core | [dependencies/core/model.yaml](dependencies/core/model.yaml) | Synthetische 1.0.0 | Core: neun Definitionen. |

```text
model.yaml                              # connector-example; root: model
model/
  example/
    connector/
      vocabulary/                       # example/connector/vocabulary
        exchange.yaml
        ...
      project/                          # example/connector/project
        api.yaml
        ...
dependencies/
  core/                                 # explizit bereitgestellter lokaler Ablageort
    model.yaml                          # markitect-core; version: "1.0.0"; root: model
    model/
      markitect/
        core/                           # markitect/core
          kind.yaml
          ...
```

Insgesamt sind es **33 Definitionen in drei belegten Packages**, verteilt auf zwei Modelle. Auch die drei Zwischenordner unter den jeweiligen Modellwurzeln bilden Packages. Modell-ID, Installationsordner und root werden nicht automatisch zu einem Package-Präfix. Die Modellwurzel selbst ist kein Package und darf keine Definitionsdateien unmittelbar enthalten. Jede Definition liegt in einem benannten Unterordner; dessen nicht leerer relativer Pfad zur jeweiligen root ist ihr Package-Name. Elternpackages werden nicht geerbt. Ein anderer Speicherort der Modelle erhält diese Namen.

Der [Manifestvertrag aus Abschnitt 8.1](../language-contract.md#models) verwendet genau model.yaml mit formatVersion, id, optionaler lokaler version, erforderlicher relativer root und erforderlichem dependencies-Mapping. connector-example bindet markitect-core genau auf "1.0.0". Seine eigene version ist absichtlich ausgelassen; als veröffentlichtes oder konsumiertes Abhängigkeitsmodell wäre eine genaue Fassung erforderlich. Die Core-Fassung ist synthetisch und behauptet keine veröffentlichte Bibliothek.

Der Prüfer erhält das Einstiegsmanifest und das Core-Manifest ausdrücklich als lokale Eingaben. dependencies/core/ ist der gewählte Ablageort dieses Durchstichs, keine allgemeine Installations- oder Lockkonvention. Der Prüfer verfolgt die direkten Modellanforderungen, kontrolliert genaue Identität/Version und lädt die dadurch erreichbaren Modellwurzeln. Es gibt keine Netzwerkauflösung, Versionssuche oder automatische Installation. Das [gewählte Lockformat](../language-contract.md#lock) ist inzwischen als [model-lock.yaml](model-lock.yaml) gespeichert. Struktur, externe Modellmenge, Versionen und echte kanonische Inhaltsdigests werden geprüft. [Abschnitt 8.3](../language-contract.md#canonical-bytes) bestimmt model-digest/1; [canonical/core.json](canonical/core.json) speichert die exakten 3769 Core-Bytes.

Ein mit Definitionen belegtes Package gehört genau einem Modell; auch unterschiedliche lokale IDs erlauben keine Aufteilung dieses Packages. Gemeinsame Zwischenordner sind möglich: Ein zusätzliches Modell mit markitect/http könnte denselben leeren Zwischenordner markitect besitzen. Modellübergreifend binden ausschließlich belegte Pfade und ihre ableitbaren Zwischenpfade Namen; rein leere Teilbäume sind dafür ohne Wirkung. Die Gegenproben prüfen diese Zusammenführung sowie Kollisionen gebundener Namen einschließlich straße/strasse. Die drei Definitionsgruppen stehen jeweils unter ihrer eigenen root:

| Package | Definitionsordner |
| --- | --- |
| markitect/core | [dependencies/core/model/markitect/core/](dependencies/core/model/markitect/core/) |
| example/connector/vocabulary | [model/example/connector/vocabulary/](model/example/connector/vocabulary/) |
| example/connector/project | [model/example/connector/project/](model/example/connector/project/) |

Der Prüfer erfasst sämtliche YAML-Definitionsdateien unter jeder Modellwurzel. Das ausdrücklich gebundene model.yaml und seine benachbarte model-lock.yaml sind Metadaten und werden auch bei root: . nicht als Definitionen erfasst; eine Gegenprobe prüft diese Nachbarschaft und die Ablehnung einer zusätzlichen Wurzeldefinition. Die früheren package.yaml-Dateien samt definitionFiles und Versionen pro Package entfallen. Es gibt keinen zusätzlichen definitions/-Pflichtordner. Die lokale id bleibt ausdrücklich in jeder Definition und folgt nicht dem Dateinamen. Umbenennen einer Datei im selben Packageordner erhält die Identität; ein anderer Packageordner ändert sie. Package-Namen und volle Qualifizierung verwenden /, kurze Aliasqualifizierung weiterhin Punkte. Lokale IDs und Feldnamen enthalten keine Punkte; `Connector.Exchange` ist eine Aliasreferenz auf die lokale ID `Exchange`.

Erst imports in einer Definitionsdatei machen fremde Packages sichtbar. Diese müssen aus dem eigenen Modell oder einer direkten Modellabhängigkeit stammen. Auch Packages desselben Modells sind untereinander nicht automatisch sichtbar; Eltern- und Kindordner öffnen keine Namen. Transitive Modelle stehen zur Vertragsprüfung bereit; ein direkter Zugriff auf ihre Packages braucht eine eigene direkte Modellabhängigkeit und einen dateilokalen Import. Aliase ändern die verfügbare Definitionenmenge nicht.

Die API verwendet kind: Connector.Container, der konkrete Command kind: example/connector/vocabulary/Command und list-documents weiterhin kind: Exchange. Alle Formen lösen dieselben kanonischen Package-/ID-Paare auf.

Der Bericht prüft Manifeste, Versionsgleichheit und Package-Zugehörigkeit getrennt vom weiterhin ungültigen gesamten Definitionsgraphen. Die Lockprüfung besteht jetzt tatsächlich gegen den kanonischen Core-Inhalt. Sein SHA-256-Digest ist 562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1. Der Core wird dafür als eigenes Modell samt seiner Abhängigkeitshülle vollständig validiert. Für connector-example wird wegen Command/command kein Digest ausgegeben; die bestandene externe Lockbindung erklärt den gesamten Definitionsgraphen nicht für gültig. Künstliche Vergleichsdigests bleiben auf ausdrücklich benannte Gegenproben beschränkt.

Der Bericht bindet zusätzlich die gelesenen Dateibytes über SHA-256. Diese Dateihashes bleiben Byteprovenienz und sind von den separat berechneten kanonischen Modelldigests zu unterscheiden. Leere Ordner stehen nur im erfassten Ordnerinventar. Die gemeinsame Modell-ID-/Versionslabelgrammatik ist gewählt: kleingeschriebene NFC-Unicode-Abschnitte nach XID mit Unicode-Dezimalziffern auch am Anfang, getrennt durch . oder -. Die Probe kontrolliert diese Form in Manifesten, Anforderungen und Lock, dazu die Geltungsbereiche von Kollisionen und exakte Versionsgleichheit. Definitionen unmittelbar in root werden als Sprachfehler abgewiesen, ohne Dateien automatisch zu verschieben. Die Gegenproben kontrollieren den Ausschluss der Wurzel aus dem Package-Inventar, ihre fehlende Package-Identität und die unveränderte Quelldatei bei Ablehnung. Die physischen Wurzelpfadregeln sind ebenfalls gewählt: . oder ein exakt geschriebener relativer /-Pfad mit vorhandenen Ordnern nach Packageordnergrammatik. Ungültige Pfadformen, falsche tatsächliche Schreibweisen, Umleitungen und überlappende gebundene Modellwurzeln sind Sprachfehler. Beide gespeicherten root: model entsprechen bereits dieser Form.

**Core:** [Kind](dependencies/core/model/markitect/core/kind.yaml), [Property](dependencies/core/model/markitect/core/property.yaml), [name](dependencies/core/model/markitect/core/name.yaml), [type](dependencies/core/model/markitect/core/type.yaml), [minCount](dependencies/core/model/markitect/core/min-count.yaml), [maxCount](dependencies/core/model/markitect/core/max-count.yaml), [values](dependencies/core/model/markitect/core/values.yaml), [target](dependencies/core/model/markitect/core/target.yaml) und die gemeinsame Property [properties](dependencies/core/model/markitect/core/properties.yaml).

**Exchange-Vertrag:** [Exchange](model/example/connector/vocabulary/exchange.yaml), [requester](model/example/connector/vocabulary/requester.yaml), [receiver](model/example/connector/vocabulary/receiver.yaml), [command](model/example/connector/vocabulary/command-property.yaml), [requestChannel](model/example/connector/vocabulary/request-channel.yaml), [replyChannel](model/example/connector/vocabulary/reply-channel.yaml), [timeoutSeconds](model/example/connector/vocabulary/timeout-seconds.yaml) und [maxAttempts](model/example/connector/vocabulary/max-attempts.yaml).

**Ergänzendes Vokabular:** [Container](model/example/connector/vocabulary/container.yaml) mit [location](model/example/connector/vocabulary/location.yaml), [Command](model/example/connector/vocabulary/command-kind.yaml) mit [operationId](model/example/connector/vocabulary/operation-id.yaml), [Channel](model/example/connector/vocabulary/channel.yaml) mit [topicPattern](model/example/connector/vocabulary/topic-pattern.yaml), [delivery](model/example/connector/vocabulary/delivery.yaml) und [broker](model/example/connector/vocabulary/broker.yaml), sowie [MqttService](model/example/connector/vocabulary/mqtt-service.yaml).

**Projektgegenstände:** [list-documents](model/example/connector/project/list-documents.yaml), [api](model/example/connector/project/api.yaml), [onprem](model/example/connector/project/onprem.yaml), [list-documents-command](model/example/connector/project/list-documents-command.yaml), [requests](model/example/connector/project/requests.yaml), [replies](model/example/connector/project/replies.yaml) und [messaging](model/example/connector/project/messaging.yaml).

Die IDs Command und command sind unter der neuen kollisionsunabhängigen Groß-/Kleinschreibregel derselbe Name im Paket und deshalb ein Fehler. Die unterschiedlichen Dateien command-kind.yaml und command-property.yaml lösen diesen Definitionskonflikt nicht. Die übrigen Dateinamen wurden ausdrücklich auf kleingeschriebenes kebab-case umgestellt; die offene fachliche Entscheidung wurde dabei nicht durch eine automatische Umbenennung ersetzt. Eine tatsächliche Umbenennung ist von der strengen Prüfung zu unterscheiden: Der Prüfer selbst verändert niemals die Definitionen.

## Einordnung nach der v1-Konsolidierung

Die [Sprachspezifikation v1](../language-contract.md) besitzt die Regeln. [Historische Herleitungen](../language-history.md) und [nicht gewählte Verfahrensvorschläge](../procedure-proposals.md) sind davon getrennt. Die neun Core-Definitionen bleiben normativer Bestandteil der Spezifikation; die Definitions-, Manifest- und Lockausschnitte unten sind informative Ansichten der Beispieldateien.

Der Prüfer liest Abschnitte über stabile Anker statt über Titel oder Nummern. Er vergleicht die normative Core-Selbstbeschreibung und die informativen Ausschnitte mit ihren jeweiligen Quellen. Diese README ist deshalb jetzt selbst eine hashgebundene Prüfeingabe. Der [Bericht vor der redaktionellen Konsolidierung](verification-before-v1-consolidation.json) bleibt als historischer Befund erhalten; [verification.json](verification.json) bindet den neu geprüften Stand. Die Konsolidierung ändert weder die neun Core-YAML-Inhalte noch die kanonische Projektion oder den Core-Digest.

## Gewählte Korrekturen nach dem v1-Review

Die [Referenzauflösung](../language-contract.md#names) erlaubt vollständige Qualifizierung auch im eigenen Package ohne Import. Gleichnamige sichtbare Definitionen sind zulässig; jede verwendete Referenz muss eindeutig sein. Doppelte Importaliase oder doppelte Packageimports bleiben Deklarationsfehler.

Nur Packages mit eigenen Definitionen sind importierbar. Leere Ordner und Zwischenpackages bleiben Bestandteil der Ordnerstruktur, öffnen aber keine Definitionsnamen. Sie bleiben aus dem Digest ausgeschlossen; Zwischenpfade belegter Packages nehmen weiterhin an der modellübergreifenden Kollisionsprüfung teil. [package_probes.py](package_probes.py) prüft eigene und importierte Ziele, mehrdeutige Kurz-/Aliasreferenzen, strenge Schreibweise, leere Packages und die [abschließende Dateierfassung](../language-contract.md#packages). Jede `.yaml` wird als Definition gelesen; nur die ausdrücklich gebundenen Manifest-/Lockpfade sind Metadaten. `.yml` und Endungsvarianten sind Fehler, andere reguläre Dateien werden ignoriert. Die Gegenproben laufen auch auf tatsächlichen temporären Dateibäumen und prüfen unveränderte Quelldateien.

[canonical_probes.py](canonical_probes.py) bestätigt gleiche Bytes für kurze und vollständig qualifizierte eigene Strukturreferenzen sowie die Ablehnung leerer Packageimports vor der Projektion. Der [Bericht vor diesen Korrekturen](verification-before-v1-package-fixes.json) bleibt historisch erhalten; [verification.json](verification.json) wird ausdrücklich für den neuen Stand erzeugt. Die neun Core-Definitionen und ihre gespeicherten Bytes bleiben unverändert. Diese drei Korrekturen ändern keine der kleineren body-Präzisierungen; die anschließend ausgewählte Namensvereinfachung steht im folgenden Abschnitt.

## Namensbindung leerer Ordner

Die [modellübergreifende Namensbindung](../language-contract.md#models) folgt denselben belegten Packagepfaden, die im kanonischen Inhalt stehen, einschließlich ihrer ableitbaren Zwischenpfade. Rein leere Teilbäume binden über Modellgrenzen hinweg keine Namen. Lokal werden weiterhin alle Ordner einschließlich leerer Teilbäume streng geprüft; sie bleiben Packages und werden nicht importierbar.

[package_probes.py](package_probes.py) lädt tatsächliche getrennte Modellwurzeln mit zwei Peerbibliotheken: Belegtes `straße/feature` und leeres `strasse/empty` in verschiedenen Modellen müssen zusammen gültig sein. Hinzufügen und Entfernen des leeren Teilbaums erhalten alle kanonischen Bytes, echte Digests und die bestehende Lockbindung. Sobald `strasse/empty` eine Definition enthält, kollidieren die abgeleiteten Zwischenpfade. Die Gegenproben prüfen außerdem die erneute logische Validierung, eigenständige Modellprüfung, Kandidatenreihenfolge, lokale Schreib-/Kollisionsfehler und unveränderte Quellen. Der [Bericht vor dieser Bereinigung](verification-before-empty-package-bindings.json) bleibt erhalten; [verification.json](verification.json) bindet den aktualisierten Stand. Core-Bytes und externe Lockdatei bleiben unverändert.

## Nichtleerer body mit Unicode White_Space

Die [body-Regel](../language-contract.md#body) verlangt mindestens ein Zeichen außerhalb von Unicode White_Space 15.1.0. Der Prüfer und die Link-Anzeigetextprüfung verwenden dieselbe explizite Zeichenmenge aus [name_profile.py](name_profile.py). Python-`strip()` bestimmt die Sprachregel nicht mehr. Leere Strings und reine White_Space-Texte werden abgewiesen; vorhandener Leerraum wird nicht entfernt oder umgeschrieben. Gewöhnliche Stringwerte und das optionale title behalten ihre bisherigen Regeln.

Die Gegenproben prüfen alle 25 White_Space-Zeichen einzeln, gemischten Leerraum und Zeichen außerhalb dieser Menge, insbesondere die von Python zusätzlich als Leerraum behandelten U+001C bis U+001F. Deren strukturelle Zulässigkeit hebt die dokumentierte Fähigkeitsgrenze der Markdownprobe nicht auf. [canonical_probes.py](canonical_probes.py) bestätigt unveränderte führende und abschließende Leerraumzeichen im body und unterschiedliche Digests gegenüber dem Text ohne diesen Leerraum. Ein reiner White_Space-body wird vor der Projektion abgewiesen. Der [Bericht vor dieser Präzisierung](verification-before-body-whitespace.json) bleibt erhalten; [verification.json](verification.json) bindet den neuen Stand. Die gespeicherten Definitionen, Core-Bytes und die externe Lockdatei ändern sich nicht. Die anschließend gewählte Erhaltung von NUL im Link-Anzeigetext steht im folgenden Abschnitt.

## Literales NUL im Link-Anzeigetext

Das [Körperlinkprofil](../language-contract.md#body-links) erhält nach YAML dekodiertes U+0000 im Anzeigetext unverändert. Ein tatsächliches U+FFFD bleibt davon verschieden. [body_links.py](body_links.py) verwendet weiterhin CommonMark-normalisierten Text zur Erkennung geschützter Kontexte, liest aktive Körperlinks aber aus dem zugehörigen literalen Inline-Text. Diese Wiederherstellung gehört zur Inhaltsauswertung und verwendet keine optionalen Diagnosekoordinaten. Kann die begrenzte Probe den literalen Inline-Text nicht wiederherstellen, meldet sie eine Fähigkeitsgrenze statt einen ersetzten oder erfundenen Anzeigetext auszugeben.

Die Gegenproben prüfen NUL allein und zusammen mit U+FFFD, wiederholte Links, Zitate, Überschriften, Listen, Körperlink-Escapes und die YAML-NUL-Escapes einschließlich ihrer Quellbereiche. Code, Autolinks, HTML-Rohblöcke und gewöhnliche Referenzlinks bleiben geschützt; auch CommonMarks Gleichbehandlung von NUL und U+FFFD bei gewöhnlichen Referenznamen bleibt wirksam. [canonical_probes.py](canonical_probes.py) bestätigt unveränderte body-Werte und unterschiedliche Digests für NUL und U+FFFD. [diagnostic_probes.py](diagnostic_probes.py) prüft die Erhaltung bei fehlenden YAML-Metadaten und erzwungenem Ausfall der optionalen Koordinatenabbildung. Der [Bericht vor dieser Korrektur](verification-before-literal-nul.json) bleibt erhalten; [verification.json](verification.json) bindet den neuen Stand. Core-Bytes und Lock bleiben unverändert. Die anschließend korrigierte Core-Gleichwertigkeit fehlender Imports und `imports: []` steht im folgenden Abschnitt.

## Fehlende und leere Imports im Core-Vergleich

Der Vergleich mit den [neun festen Core-Definitionen](../language-contract.md#core-definitions) berücksichtigt die allgemeine Gleichwertigkeit fehlender Imports mit `imports: []`. Die Autorenquellen bleiben unverändert. Nichtleere Imports werden vollständig geprüft; die folgende Erweiterung vergleicht anschließend die aufgelösten Inhalte.

[canonical_probes.py](canonical_probes.py) prüft jede der neun Definitionen einzeln mit leerer Importliste und anschließend alle gemeinsam, auch zusammen mit ausdrücklich gesetzten ID-Standardnamen. Die kanonischen Bytes und der feste Core-Digest müssen identisch bleiben. Negative Gegenproben verhindern, dass null, andere Werttypen oder ungültige nichtleere Importlisten durch den Vergleich verschwinden. Der [Bericht vor dieser Korrektur](verification-before-core-empty-imports.json) bleibt erhalten; [verification.json](verification.json) bindet den neuen Stand. Die gespeicherten Core-Dateien, ihre Bytevorlage und die externe Lockdatei bleiben unverändert.

## Core-Vergleich nach aufgelöstem Inhalt

Der anschließende kritische Abgleich hatte gezeigt, dass vollständig qualifizierte eigene Core-Referenzen trotz identischem Ziel abgewiesen wurden. Diese Korrektur ist jetzt umgesetzt: Der Vergleich verwendet die gewöhnliche Namensauflösung für kind, target und properties und vergleicht wirksame Standardwerte. Ungeordnete Deklarationen werden sortiert, aber nicht dedupliziert. Imports werden vor dem Vergleich vollständig geprüft; nach erfolgreicher Auflösung beeinflussen ihre Schreibform und gültige unbenutzte Imports den Inhaltsvergleich nicht. Es gibt keine zweite Auswahlregel für den Core und keine Ausnahme von strenger Schreibweise oder Mehrdeutigkeitsprüfung.

[core_probes.py](core_probes.py) prüft jede der neun Definitionen mit vollständig qualifiziertem kind, die Feld-/Zielreferenzen, sämtliche Aliasreferenzen, umbenannte Aliase, gültige unbenutzte Imports und Kombinationen mit expliziten Standardnamen sowie veränderter Deklarationsreihenfolge. Diese Varianten durchlaufen die vollständige Inhaltsprüfung und müssen dieselben Core-Bytes ergeben; kein Prüfschritt wird dafür ausgesetzt. Gegenproben prüfen echte Vertrags-, Ziel-, Kardinalitäts- und Textänderungen, verbotene Felder, fehlende Definitionen, fehlerhafte Imports, Schreibfehler, Mehrdeutigkeiten und Duplikate. Ein gleichnamiges Ziel in einem anderen Package bleibt verschieden. Alle Vergleiche lassen die Autorenquellen unverändert.

Der [Bericht vor dieser Korrektur](verification-before-semantic-core.json) bleibt erhalten; [verification.json](verification.json) bindet den neuen Stand einschließlich der neuen Gegenproben. Die gespeicherten Core-Definitionen, die kanonische Bytevorlage und die externe Lockdatei bleiben unverändert.

## Lokale IDs und Feldnamen ohne Punkte

Die [Namensregeln der v1](../language-contract.md#spelling) verbieten Punkte in allen lokalen Definitions-IDs und effektiven Feldnamen, einschließlich `spec.name` und geschachtelten Objektverträgen. Die gespeicherten Exchange-Dateien verwenden bereits kurze IDs und Feldnamen ohne Punkte und benötigen keine Umbenennung. `Core.Kind` beziehungsweise `Connector.Exchange` sind weiterhin Aliasreferenzen; Package-Namen und volle Qualifizierung behalten `/`.

| Gegenstand | Autorenform |
| --- | --- |
| Lokale Kind-ID | `Exchange` |
| Lokale Property-ID und Standardfeldname | `requester` |
| Direkte Referenz auf diesen Kind | `Exchange` |
| Referenz mit dem deklarierten Alias Connector | `Connector.Exchange` |
| Vollständige Referenz auf diesen Kind | `example/connector/vocabulary/Exchange` |

Gegliederte IDs und Feldnamen werden abgewiesen, auch wenn sie unbenutzt sind oder eine Property einen ausdrücklich abweichenden Feldnamen setzt. Es gibt keine automatische Migration. Die Namens-, Package- und kanonischen Gegenproben prüfen diese Ablehnung und weiterhin Unicode, Aliasauflösung, Dezimalzahlen, exakte Versionslabels wie `"1.0"` sowie unveränderte Punkte in Textwerten. Der [Bericht vor der Namensvereinfachung](verification-before-dotless-names.json) bleibt erhalten; [verification.json](verification.json) bindet den neuen Stand. Die neun Core-Definitionen, ihre kanonischen Bytes und die externe Lockbindung bleiben unverändert. Exakte Diagnosekoordinaten sind inzwischen im folgenden separaten Werkzeugvertrag geregelt. Die anschließend gewählte Behandlung von `x` als gewöhnlichem Versionszeichen steht im folgenden Abschnitt.

## Diagnosekoordinaten als unabhängiges Werkzeugprofil

Die [Sprachspezifikation](../language-contract.md#body-links) verlangt vollständige Linkerkennung, Zielauflösung und Vorkommensreihenfolge. Der separate [Diagnosevertrag](../diagnostic-contract.md) legt genaue UTF-8-/YAML-Koordinaten fest. [body_links.py](body_links.py) liest standardmäßig nur Inhalte; [diagnostic_coordinates.py](diagnostic_coordinates.py) ergänzt optional belegbare Positionen. Die kanonische Projektion ruft keine Koordinatenabbildung auf.

Der Bericht trennt `body_link_validation` und `canonical_models` von `diagnostic_coordinates`. Fehlende oder veraltete YAML-Metadaten, nicht belegbare Markdownabbildung oder reine Koordinatenabbrüche ergeben dort `not_completed`. Bereits belegbare Teilkoordinaten bleiben erhalten; fehlende werden nicht erfunden. Dieser separate Status verändert weder ein gültiges Inhaltsurteil noch dessen Digest. Syntax-/Zielfehler und unvollständige Inhaltsprüfung blockieren weiterhin die kanonische Ausgabe.

[diagnostic_probes.py](diagnostic_probes.py) prüft diese Trennung mit fehlender und veralteter Herkunft, erzwungenen Inline-/YAML-Abbildungsfehlern und Speicherabbrüchen ausschließlich der Koordinatenphase. Dieselben geprüften Inhalte müssen dabei dieselben kanonischen Bytes ergeben; Reihenfolge und Wiederholungen bleiben erhalten. Die bisherigen exakten Quellenbereichsproben werden als Werkzeugprofiltests fortgeführt. Der [Bericht vor der Trennung](verification-before-diagnostic-separation.json) bleibt erhalten; [verification.json](verification.json) bindet den neu geprüften Stand einschließlich Diagnosevertrag.

## Versionslabels mit gewöhnlichem x

Die [v1-Labelregeln](../language-contract.md#models) reservieren `x` nicht mehr. `"x"`, `"1.x"` und `"release.x"` werden in Manifest, Abhängigkeitsanforderung, Lock und kanonischen Bytes exakt als Text gebunden. `"1.x"` wählt keine numerischen Fassungen wie `"1.7"` aus. `latest`, Wildcards und Bereichsoperatoren bleiben verboten; Unicode-, NFC-, Schreib- und Kollisionsregeln gelten unverändert.

Die Manifest-, Lock- und kanonischen Gegenproben prüfen erfolgreiche genaue Bindung, unpassende numerische Fassungen und unveränderte Labeltexte in den kanonischen Bytes. Der [Bericht vor dieser Auswahl](verification-before-literal-x.json) bleibt historisch erhalten; [verification.json](verification.json) bindet den neu geprüften Stand. Die gespeicherten Modellfassungen und der Core-Digest ändern sich dadurch nicht.

## Prüfung und Aussagegrenze

Aus dem Repositorywurzelverzeichnis mit Python 3.13 (Unicode 15.1.0) in einer isolierten Prüfumgebung. Die [Prüfabhängigkeiten](requirements.txt) betreffen ausschließlich diese Entwurfsprobe. Unter PowerShell einmalig:

```powershell
$probeEnvironment = Join-Path ([System.IO.Path]::GetTempPath()) 'markitect-canonical-model-probe-py313'
python -m venv $probeEnvironment
$probePython = Join-Path $probeEnvironment 'Scripts/python.exe'
& $probePython -m pip install -r docs/design/canonical-model/minimal-language/exchange-example/requirements.txt
```

Die Unicode-Tabellenversion und die CommonMark-Parserfassung sind gebunden. Die Umgebung liegt außerhalb des Checkouts, damit ihre Bibliotheksdateien nicht zur Projektabdeckung gehören. Unter Linux/macOS wird entsprechend eine Umgebung außerhalb des Checkouts angelegt und deren bin/python verwendet. Die folgenden PowerShell-Befehle verwenden $probePython aus dem Setupblock. Der reguläre Prüflauf unter PowerShell:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py
```

Der normale Lauf ist wegen des offenen Definitionskonflikts nicht erfolgreich. Core-Digest und Lockprüfung bestehen separat. Ein erfolgreiches --self-test bestätigt die gekennzeichneten Gegenproben und die gespeicherten kanonischen Core-Bytes, erklärt das fehlerhafte Connector-Modell aber nicht für konform. Ein Bericht über den aktuellen Quellstand kann ausdrücklich geschrieben werden; ein Fehler bleibt ein Fehler und führt zu einem Exitcode ungleich null:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py --report docs/design/canonical-model/minimal-language/exchange-example/verification.json
```

Die Gegenproben des Prüfers lassen sich getrennt vom noch fehlerhaften Beispiel ausführen:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py --self-test
```

Dieser Testmodus verwendet ausdrücklich synthetische gültige und ungültige Eingaben. Sein Erfolg erklärt das gespeicherte Beispiel nicht für konform und stellt keine Modellentscheidung über Command dar. Die Quellen werden in keinem Modus korrigiert. Die Körperlinkprüfung wird als eigener Berichtsteil ausgeführt; sie kann für die vorhandenen Links bestehen, während das gesamte Modell wegen des Definitionskonflikts abgewiesen wird. Die Zahl resolution_calls im Bericht zählt Auflösungsaufrufe des Prüfers, keine eindeutigen Graphkanten.

Die Prüfung umfasst model-lock.yaml, vollständige direkte/transitive Modellmenge, Versionsgleichheit, synthetische Digestabweichungen und unveränderte Quellen sowie model.yaml, direkte/transitive Versionsbindungen, Zugriff über direkte Modellabhängigkeiten, gemeinsame Zwischenordner und eindeutige Zugehörigkeit belegter Packages sowie die /-Form für Packages und volle Qualifizierung, Ablehnung alter Punktformen und pfadartiger Ausweichschreibweisen, Übereinstimmung des normativen Core mit den Quellen und der informativen YAML-Auszüge dieser Beispielseite mit ihren Dateien sowie ordnerbasierte Package-Erfassung, Verschieben/Umbenennen, fehlende implizite Elternimports, Unicode-Ordnerkollisionen sowie dateilokale Imports und Aliase, exakte kanonische Schreibweise, Kollisionen unabhängig von Groß-/Kleinschreibung, Kind-/Property-/Wertreferenzen, geschlossene Feldverträge, Werttypen, Kardinalitäten und effektive Feldnamen. Für Namen gilt das Unicode-15.1.0-Profil aus Abschnitt 7.2 des Sprachvertrags: XID-Bezeichner, strenge NFC-Autorenform und vollständiger kulturunabhängiger Namensvergleich. straße und strasse kollidieren in allen Namensbereichen; abweichende Referenzen werden weiterhin abgewiesen. Normale Textwerte bleiben unverändert. Die Gegenproben prüfen auch Namen ohne Groß-/Kleinschreibung, kombinierende Marken, Unicode-Importaliase und Unicode-Dateinamen. PyYAML zerlegt nur die YAML-Syntax. Eigene Skalarregeln nach Abschnitt 5.1 lesen ausschließlich true/false als Booleans, erhalten Datumsangaben als Text und verwenden beliebig genaue Integer sowie exakte endliche Dezimalzahlen ohne Typkonvertierung. Anchors, Aliase, Merge-Keys, explizite Tags, Dokumentfolgen, null und nichttextuelle Schlüssel werden abgewiesen. Ressourcenlimits der Zahlenprobe sind ausdrücklich im Bericht genannt; sie sind keine sprachlichen Zahlenbereiche.

Die Listen-Gegenproben vergleichen gewöhnliche Wertlisten unter Erhalt ihrer Reihenfolge und Wiederholungen, Core-Deklarationen und Imports dagegen ohne Reihenfolge. Fehlende und leere optionale Listen ergeben denselben Vergleichswert, auch innerhalb von Objekten. Gleichnamige gewöhnliche Datenfelder erhalten keine Meta-Bedeutung. Duplikat- und Kardinalitätsfehler werden vor dem Vergleich abgewiesen. Die frühere Listenvergleichsprobe bleibt als eigene Gegenprobe erhalten. Die vollständige kanonische Kodierung model-digest/1 besitzt nun zusätzlich feste Bytevektoren und Gleichheits-/Unterschiedsproben.

Die beiden aktiven Körperlinks in list-documents referenzieren api und onprem. Der [Körperlinkparser](body_links.py) nutzt CommonMark 0.31.2 mit der gewählten Linkerweiterung: Code, gewöhnliche Links/Bilder und HTML-Rohbereiche bleiben geschützt; aktive Links werden strikt aufgelöst. Die Inhaltsprüfung erhält alle geordneten Linkvorkommen ohne Koordinatenpflicht. Der gesonderte Diagnosebericht ergänzt belegbare UTF-8-Bytebereiche und YAML-Quellpositionen nach dem [Werkzeugvertrag](../diagnostic-contract.md). Die Gegenproben prüfen auch YAML-Escapes, Faltung, Unicode, fehlerhafte Syntax und wiederholte Links. Die Grenzen der Probe sind 2000000 UTF-8-Bytes je body und 64 Verschachtelungsebenen. Eine bekannte Leerraumabweichung der gebundenen Parserfassung betrifft U+001C bis U+001F sowie U+0085; die Probe meldet eine solche ansonsten mögliche Auswertung ausdrücklich als nicht abgeschlossen. Das ist keine zusätzliche Sprachbeschränkung. Ein Abbruch ist ausdrücklich keine erfolgreiche oder vollständige Modellprüfung.

Der [Prüfer](verify.py) ist eine begrenzte Strukturprobe für diesen expliziten Eingabesatz. Der [aktuelle Bericht](verification.json) nennt seine Eingaben, Dateihashes, Fehler und Grenzen. Der [vorherige bestandene Bericht](verification-before-strict-names.json) bleibt unverändert als historischer Befund vor der neuen Namensregel erhalten; er belegt nicht den heutigen Quellstand. Ein späterer Lauf verändert einen alten Bericht nur mit dem ausdrücklichen --report-Aufruf; dessen neue Eingabehashes sind dann neu zu beurteilen. Der Prüfer ersetzt weder den Produktcompiler noch die vollständige Sprachspezifikation.

Die Auswahl ist auf das Exchange-Beispiel begrenzt. Container besitzt hier nur den Standort; zusätzliche Komponenten- und Schnittstellenmodelle sind nicht vorausgesetzt. Channel ist das MQTT-Vokabular dieses Projekts und kein universeller Core-Kind. Die Inhalte der Commands, Topicparameter, Autorisierung, Laufzeitfehler und gewünschte Zustellwirkungen stehen teilweise als verbindliche Prosa im body. Eine erfolgreiche Strukturprüfung bestätigt ihre Umsetzung oder Widerspruchsfreiheit nicht.

Ein allgemeiner YAML-/Markdown-Konformitätsnachweis, eine produktive Go-Implementierung der neuen kanonischen Kodierung, semantischer Index, Umsetzung, unabhängige semantische Prüfung und Broker-/Providerlaufzeiten sind kein Ergebnis dieser Probe. Es werden weder Netzwerkdienste noch Modellprovider aufgerufen.

**Versionslabels:** Versionen werden als exakter Text gebunden, ohne numerischen Vergleich, Auffüllen von Komponenten, Reihenfolge oder Kompatibilitätsannahmen. Die Gegenproben prüfen eine passende zweigliedrige Version sowie die Abweisung von "1.0" beziehungsweise "1.0.1" bei einer Anforderung auf "1.0.0", auch im Lock. Die kanonischen Bytes unterscheiden "1.0" und "1.0.0" ebenfalls. Versionsbereiche und latest bleiben verboten. Die gemeinsame Zeichengrammatik aus Abschnitt 8.1 ist gewählt; die Gegenproben prüfen auch Ziffernstarts, Punkte, verschiedene Schriften, kombinierende Zeichen, NFC, ungültige Trenner und Versionskollisionen wie straße/strasse innerhalb desselben Modells.

**Physische Wurzelpfade:** [root_paths.py](root_paths.py) prüft die Autorenform vor jeder Pfadinterpretation, tatsächlich vorhandene Namen und Verzeichnisse, verbotene Umleitungen und überschneidungsfreie geladene Wurzeln. [root_path_probes.py](root_path_probes.py) prüft das auf dem Dateisystem: verschachtelte und Unicode-Pfade, unveränderte Quelldateien, falsche Groß-/Kleinschreibung und NFC, straße/strasse-Kollisionen, fehlende Ordner, Dateien statt Ordnern, unzulässige Pfadnormalisierung und verschachtelte Modelle. Eine echte Junction unter Windows beziehungsweise ein Symlink auf anderen unterstützten Hosts wird sowohl als Wurzelpfad als auch im Modellbaum abgewiesen. Die absoluten Manifestablageorte bleiben frei; die relative root-Schreibweise folgt der Packageordnergrammatik und erweitert sie nicht um die Zeichen der Modell-ID-/Versionslabelgrammatik.

**Objektzyklen und Ressourcen:** [Abschnitt 5.3](../language-contract.md#cycles-resources) erlaubt rekursive Objektverträge, sofern der Graph verpflichtender Objektfelder azyklisch ist. Der Werteprüfer folgt tatsächlich vorhandenen endlichen Objekten und verändert keine Definitionen. [object_resource_probes.py](object_resource_probes.py) prüft abbrechbare Selbst-/Mehrknotenzyklen, unmögliche verpflichtende Zyklen, endliche Werte und leere Listen, geschlossene Felder, Referenzzyklen und einen langen Vertragsgraphen ohne rekursive Vorabentfaltung.

[resources.py](resources.py) trennt Werkzeugabbrüche von Sprachfehlern. --max-object-depth konfiguriert die maximale tatsächlich besuchte Objekttiefe dieser Probe (Standard 128); die Werteprüfung beginnt außerhalb eines Objektwerts bei Tiefe 0. Der Bericht nennt außerdem die Python-Rekursionsgrenze sowie die bestehenden Körperlink- und Zahlenbudgets. Ein Abbruch liefert not_completed und für den betroffenen Umfang weder Bytes noch Digest; vorhandene Exportdateien bleiben erhalten. Erfolgreiche Läufe mit ausreichend großen Budgets ergeben dieselben kanonischen Bytes. Die normalen CLI-Exitcodes sind 0 für Erfolg, 1 für nachgewiesene Fehler und 2 für nicht abgeschlossene Prüfung; --self-test bewertet gesondert nur die gekennzeichneten Gegenproben.

Der [Abschlussabgleich](../language-contract.md#conformance) enthält nun auch die gewählte Azyklizitätsregel für Modellabhängigkeiten. Selbst-, wechselseitige und längere Modellzyklen sind Sprachfehler (failed) und erhalten keinen Digest; die frühere vorläufige Einstufung als not_completed ist abgelöst. Gültige Ketten und gemeinsam verwendete Bibliotheken werden separat geprüft, ebenso die Ablehnung beim Laden physischer Manifeste und vor der kanonischen Projektion. Im gewählten minimalen Umfang ist keine weitere offene Sprachentscheidung identifiziert. Go-Implementierung, allgemeine Parserkonformität und semantische Verfahren sind davon getrennt.

## Kanonische Bytes und echte Inhaltsbindung

[canonical_bytes.py](canonical_bytes.py) implementiert die in Abschnitt 8.3 gewählte Projektion: eigene Modellmetadaten ohne root, eigene belegte Packages/Definitionen, aufgelöste Referenzen und Körperlinkziele, wirksame Defaults und exakt erhaltene Textwerte. Sie verwendet fest sortiertes UTF-8-JSON ohne BOM oder finalen Zeilenumbruch und typisierte exakte Zahlen. Leere Packages, Autorenformatierung, Quellenpositionen, Lockdateien und fremde Inhalte werden nicht kodiert. Jede Modellfassung und ihre Abhängigkeitshülle werden vor der Projektion geprüft; Ladefehler bleiben pro Modell erhalten.

[canonical_probes.py](canonical_probes.py) prüft Gleichheit bei Quote-/Kommentar-/Schlüsselreihenfolge, gültigen strukturellen Referenzformen, Defaults und Deklarationsreihenfolgen sowie Unterschiede bei Text, Körperlinkzielen, gewöhnlicher Listenreihenfolge, Typen und exakten Zahlen. Der [Core-Bytevektor](canonical/core.json) ist als festes Artefakt gespeichert; ein unabhängiger Standard-JSON-Encoder wird gegen den projizierten Baum verglichen. Ungültige Eingaben erhalten keinen Digest. Keine dieser Proben ersetzt den Go-Produktcompiler oder einen allgemeinen Parser-Konformitätsnachweis.

Die explizite Ausgabe einer gültigen einzelnen Modellfassung ist zusätzlich möglich:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py --canonical-model markitect-core --canonical-output docs/design/canonical-model/minimal-language/exchange-example/canonical/core.json
```

Dieser Export prüft nur das gewählte Modell und dessen Abhängigkeitshülle. Er aktualisiert das Lock nicht automatisch und bestätigt nicht das andere fehlerhafte Connector-Modell. Normale Prüfläufe schreiben weder Quellen, kanonische Artefakte noch Locks.

## Ist die kanonische Form damit vollständig?

Vor der neuen Namensregel war der Definitions- und Referenzgraph dieses Beispiels strukturell geschlossen. Der jetzige Command/command-Konflikt wird ausdrücklich abgewiesen. Diese Frage ist im Projektvokabular zu lösen; beide gewählten Modellierungsrichtungen sind grundsätzlich mit dem Core ausdrückbar. Das Beispiel belegt keine Vollständigkeit der Sprache für jedes Projekt. Die [v1-Spezifikation und ihre Aussagegrenzen](../language-contract.md#conformance) definieren den gewählten minimalen Sprachumfang; produktive Compiler-/Berichtsverträge und Belege bleiben getrennte Arbeit. Die generische Anbindung an Realisierung und Apply ist eine weitere Verfahrensaufgabe.


<a id="definition-excerpts"></a>

## Informative Definitionsausschnitte

Diese Ausschnitte spiegeln die gleichnamigen Dateien des ausführbaren Beispiels wider. Maßgeblich für Sprachregeln ist ausschließlich die [v1-Spezifikation](../language-contract.md). Der letzte requester-Ausschnitt zeigt denselben ID-Standard ohne zusätzliche Namensangabe; er ist keine zweite Definition. Der Prüfer vergleicht die Ausschnitte mit den Dateien.

**exchange.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: Exchange
kind: Kind
body: >-
  Beschreibt einen Request-/Reply-Vertrag zwischen requester und receiver. Antworten behalten
  die MessageId des Requests. timeoutSeconds gilt pro Versuch; maxAttempts zählt den ersten
  Versuch mit.
spec:
  properties:
  - requester
  - receiver
  - command
  - requestChannel
  - replyChannel
  - timeoutSeconds
  - maxAttempts
```

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Anwendung in der Auftraggeberrolle dieses Austauschs.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**receiver.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: receiver
kind: Property
body: >-
  Anwendung in der Bearbeiterrolle dieses Austauschs.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**command-property.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: command
kind: Property
body: Vertrag der angeforderten Operation.
spec:
  type: reference
  target: Command
  minCount: 1
  maxCount: 1
```

**request-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requestChannel
kind: Property
body: >-
  Kommunikationsweg für Aufträge.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**reply-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: replyChannel
kind: Property
body: >-
  Kommunikationsweg für Antworten an die ausstellende Instanz.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**timeout-seconds.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: timeoutSeconds
kind: Property
body: >-
  Positive maximale Wartefrist pro Versuch in Sekunden.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**max-attempts.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: maxAttempts
kind: Property
body: >-
  Positives Versuchsbudget einschließlich des ersten Versuchs.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Anwendung in der Auftraggeberrolle dieses Austauschs.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

<a id="manifest-excerpts"></a>

## Informative Manifest- und Lockausschnitte

Die Version des Core-Beispiels ist synthetisch; der Digest bindet seine tatsächlich geprüften kanonischen Bytes.

**model.yaml**

```yaml
formatVersion: 1
id: connector-example
root: model
dependencies:
  markitect-core: "1.0.0"
```

**dependencies/core/model.yaml**

```yaml
formatVersion: 1
id: markitect-core
version: "1.0.0"
root: model
dependencies: {}
```

**model-lock.yaml**

```yaml
formatVersion: 1
models:
  markitect-core:
    version: "1.0.0"
    digest: "sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1"
```
