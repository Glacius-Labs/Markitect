# Handoff: kanonisches Modell und semantischer Index

Stand: 15. Oktober 2026. Repository: Glacius-Labs/Markitect. Arbeitsbranch: `docs/design-consolidation`. Bestehender Draft-PR: [#266](https://github.com/Glacius-Labs/Markitect/pull/266). Der Nutzer beauftragt, diesen Handoff und den gesamten bisherigen Arbeitsstand remote zu sichern, um das Design in einer anderen Session fortzusetzen. Es ist kein Merge oder Release beauftragt.

## Einstieg für die nächste Session

1. Die Branch `docs/design-consolidation` auschecken und den neuesten Remote-Stand laden. Die Commit-ID des Pushs steht im Git-Verlauf; dieser Handoff versucht nicht, seinen eigenen Commit vorab zu benennen.
2. Repository-[AGENTS.md](../../../AGENTS.md), insbesondere Vision, Common misreadings und Konzeptregister, beachten. Die neue Designrichtung ersetzt vorhandene Produktverträge nicht stillschweigend.
3. Den [aktuellen Core-Sprachvertrag](minimal-language/language-contract.md) lesen. Er besitzt die gewählte Sprachgrundlage. Abschnitt 11 enthält nachfolgende Verfahrensvorschläge; [Abschnitt 12.1](minimal-language/language-contract.md#121-stand-der-definition-des-kanonischen-modells) ordnet deren Reife ein.
4. Für Herkunft und weitere Produktfragen den [Konsolidierungsstand](consolidation.md) lesen. Die [lange Herleitung](minimal-language/README.md) dokumentiert frühere Varianten, auch inzwischen abgelehnte Empfehlungen; sie ist nicht neben dem neueren Vertrag maßgeblich.

## Ziel des Nutzers

Markitect soll ein kanonisches, typisiertes Modell der beabsichtigten Projektwelt ausdrücken: was sein soll und welche Festlegungen gelten. Ausgangspunkt ist eine ausführliche typisierte Projektdokumentation. Die Ausdrucksmittel müssen von Projekten selbst verwendet werden können; Bibliotheken helfen nur beim Import fertigen Vokabulars.

Die Ebenen bleiben unterschieden: kanonische Projektbeschreibung; Ableitung und Arbeitsweise von Apply; entstandene Realisierung und ausgeführte Nachweise. Agentenmandate, PR-Verfahren oder tatsächliche Dateien sind dadurch nicht automatisch notwendige Modellbegriffe. System, Container, Component, Module, Artifact, Mandate, Decision und weitere Begriffe sind Vokabularkandidaten, keine privilegierten universellen Core-Kinds. Frühere Wackelkandidaten und Gründe bleiben im Konsolidierungsstand erhalten.

## Vom Nutzer gewählte Sprachregeln

- Gemeinsame Form für alle Definitionen: erforderliche `id`, `kind`, `body`, `spec`; optionales `title`. `body` ist die gewählte Bezeichnung und verbindliche Prosa.
- Fester Anfang mit den Meta-Kinds **Kind** und **Property**. Kinds, Properties und konkrete Gegenstände sind selbst Definitionen. Der bekannte Seed umfasst zwei Kinds und acht Meta-Properties; seine Bedeutung wird vom Compiler fest kontrolliert.
- Geschlossene Propertyverträge und Werttypen string, boolean, integer, number, enum, object, reference. Anforderungen und Rollen entfallen zunächst als eigene Sprachmittel. Ihre Bedeutung kann verbindliche Prosa beschreiben.
- Kompakte Properties direkt unter `Kind.spec.properties`; sie normalisieren zu eigenen Definitionen. Ohne eigene inline id bestimmt der Feldname den Identitätsbaustein; eine explizite inline id erhält ihn bei Umbenennung.
- Logische Identität ist Paket-ID plus lokale id. Kind, Datei, title und Versionsnummer gehören nicht zu dieser Identität. Lokale Referenzen suchen nur im eigenen Paket, `alias:id` nur im ausdrücklichen Import. Keine globale Namenssuche oder stillen Redirects.
- Pro Paket eine genau gebundene Version mit Inhaltsdigest im Modell; Importkürzel sind zum deklarierenden Paket lokal. Kein latest oder Mehrversionsimport im ersten Umfang.
- Körperlinks `[[Ziel]]` beziehungsweise `[[Ziel|Anzeigetext]]` verwenden dieselbe Auflösung für Kinds, konkrete Definitionen und normalisierte Properties. Der Link identifiziert einen Bezug; die fachliche Aussage kommt aus dem Satz. Keine doppelte manuelle Referenzliste und keine automatische Geltung allein aus einem Link.

Beispiel der stabilen Property: Feldname sender wurde zu requester; `id: sender` in der kompakten Property erhält die lokale Property-ID `Exchange/sender`. `[[connector:Exchange/sender|requester]]` bleibt deshalb adressierbar. Konkrete spec-Schlüssel müssen trotzdem migriert werden.

## Ausgearbeitete Vorschläge, noch keine pauschal gewählten Verfahren

Der Nutzer hat Kontextbildung, semantischen Index, Aktualisierung und einen Datenvertrag durchspielen lassen. Nicht unterstellen, dass dadurch jedes Detail in Abschnitt 11 angenommen oder implementiert ist.

- Strukturelle Verbindungen berechnen, fachliche Geltung unabhängig beurteilen. Ein Kind-body kann neue Pflichten auf unveränderte konkrete Definitionen übertragen.
- Ein Index trennt Quellaussage, allgemeine Auslegung, konkrete Anwendung und untersuchte Abdeckung. Er ist abgeleitet und keine zweite kanonische Wahrheit.
- Wiederverwendung braucht direkte Inhalts-/Strukturabhängigkeiten, vorausgesetzte Befunde, Auswahlräume samt Nichttreffern, negative Voraussetzungen und vollständige Prüfung neuer Einflüsse im gebundenen Modell. Fehlende Links oder Suchtreffer beweisen keine Nichtbetroffenheit.
- Der Datenvertrag in Abschnitt 11.13 sieht interpretation, application, coverage und influence als Berichtstypen vor. Unveränderliche Reviews und Bindings trennen fachliche Bestätigung von Aktualität am Zielstand. Das sind keine neuen Core-Kinds.
- Es gibt keinen Vorrang „lokal/spezifisch gewinnt“. Eine bloß behauptete Ausnahme von einer unveränderten allgemeinen Pflicht ist gegebenenfalls ein Konflikt. Der Index entscheidet nicht selbst neue Architektur oder Ausnahmen.

### Letzter vollständig besprochener Fall

M3: list-documents hat requester api, check-status hat requester onprem. Beide müssen pro Versuch monoton messen und zusätzlich die Uhrquelle protokollieren. M4 ändert nur die globale Protokollierungsaussage: check-status wird ausdrücklich ausgenommen. Zeitmessung und Rollenbelegungen bleiben unverändert.

Das [Datensatzprotokoll M3 → M4](minimal-language/language-contract.md#1114-vollständiges-datensatzprotokoll-für-m3--m4) enthält im betrachteten Vordergrund:

- Übernahme von T, T1, T2 als Zeitmessungsrecords mit drei neuen reuse-Bindings.
- Neue Logauslegung L4, neue positive Anwendung L4-1 auf api und neue ausgeschlossene Anwendung L4-2 auf onprem.
- Neuen Einflussbefund I34 sowie Quellen-, Zeitmessungsfall-, Logfall- und Änderungsabdeckung CS4, CT4, CL4, CD34.
- Insgesamt acht neue Vordergrundrecords, elf neue Bindings und nach der angenommenen Beispielpolicy 27 neue Soll-Prüfberichte. Das bedeutet nicht 27 Agentenläufe.

Wichtig: Dies ist eine **symbolische Lesedarstellung mit angenommenen erfolgreichen Reviews und vorhandener Hintergrundabdeckung**. Es existiert weder eine selbständig ladbare vollständige M3/M4-Indexdatei noch ein produktiver Indexvalidator. Es wurden keine unabhängigen semantischen Reviews oder Connector-Laufzeittests ausgeführt. Nicht aus dem Katalog einen bestandenen Produktnachweis machen.

## Reifegrad und nächster sinnvoller Schritt

Die fachliche Sprachgrundlage ist weitgehend gewählt. Noch offen sind vollständige Lexik/Unicode/Escapes und YAML-Leseform, vollständiges Körperlink-Markup, Paket-/Lockformate, kanonische Kodierung, Quellenabbildung und genaue Diagnosen. Der heutige Produktcompiler implementiert diese neue Gesamtform nicht. Die Verfahren zur Zuordnung von Modell und tatsächlicher Realisierung, Prüfplanung, Integration und Apply sind noch generisch anzubinden.

Empfohlene Fortsetzung: Die bereits gewählte kleine Sprache präzise spezifizieren, mit gültigen und ungültigen Beispielen abschließen und anschließend einen durchgängigen Compilerprototyp bauen. Andere Projektbeispiele sollen Ausdruckslücken aufdecken, bevor neue Core-Konzepte hinzukommen. Der Nutzer hat diesen nächsten Implementierungsschritt noch nicht ausdrücklich beauftragt; der nächste Gesprächsschritt kann ihn zuerst auswählen. Indexmechanik nicht mit zusätzlichem kanonischem Vokabular verwechseln.

## Mitgelieferte Evidenz und Reproduktion

- [structural-probe.json](minimal-language/structural-probe.json) und [exchange-walkthrough-probe.json](minimal-language/exchange-walkthrough-probe.json): bestehender Go-Core nach ausdrücklicher Adaption von definition auf purpose; 13 konkrete Definitionen und 17 Referenzkanten. Kein neues Produktformat.
- [Bootstrap-Probe](minimal-language/bootstrap/probe.json): separater endlicher Entwurfscompiler, 51 Definitionen einschließlich Seed und 113 Kanten; 16 negative Gegenproben.
- [Kompaktprobe](minimal-language/bootstrap/compact-probe.json): frühere Kurzform normalisiert auf denselben Stand; neun negative Autorenformfälle. Ihr damaliges Verbot einer inline id ist historisch und wird vom späteren Identitätsvertrag nicht rückwirkend umgeschrieben.
- [Identitätsprobe](minimal-language/identities/probe.json): drei gebundene Stände mit 51 gleichen logischen Identitäten, Importupdate und sender/requester; neun negative Fälle. Keine Laufzeitprüfung der neuen Zeitmessungspflicht.

Die zuvor temporären Prototypen sind unter [probes](minimal-language/probes/README.md) bytegetreu archiviert. Der Replay führt auf Kopien der Eingaben neue begrenzte Strukturprüfungen aus und verändert die historischen Berichte nicht. Ihre sourceRevision bleibt die tatsächlich damalige Basis `e6b4fc67675d70e63b08bf09844d770d155874a8`. `.gitattributes` erhält die Bytes der digestgebundenen YAML-Dateien und Archive, auch wenn sie CRLF enthalten.

Der [Connector-Entwurf](connector-example.md) enthält Provenienz zu der lokal gelesenen Referenzanwendung. Er wird mit dem ausdrücklichen Auftrag zum vollständigen Remote-Handoff mitgesichert. Die Anwendung selbst, Konfiguration, temporäre Repository-Checks und lokale Laufzeiten gehören nicht zu diesem Commit.

## Praktischer Wiedereinstieg

Nach den Schritten aus AGENTS.md für den Dokumentations-Manager genügen für diese Dokumentation zunächst:

```text
python -B scripts/check-docs.py docs/design/canonical-model
python -B docs/design/canonical-model/minimal-language/probes/replay.py
```

Der optionale Replay mit `--with-go` prüft zusätzlich die beiden archivierten älteren Go-Proben über einen temporären Overlay. Benötigt werden Python mit PyYAML sowie dafür die in CONTRIBUTING verlangte Go-Version. Keine Referenzanwendung, kein Broker und kein Providerkonto wird gestartet. Vollständige Produkt- und CI-Gates gelten vor einem Merge unverändert; begrenzte Dokumentations-/Entwurfsprüfungen sind kein vollständiges Project Verify.
