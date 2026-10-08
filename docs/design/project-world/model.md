# Zielmodell: Fachsprache, Verantwortung und Umsetzung

Status: Planungsentwurf. Die [festgehaltenen Richtungsentscheidungen](README.md) bestimmen die Richtung. Die folgenden Modelltypen sind Vorschläge, keine vorhandenen YAML-Kinds oder CLI-Befehle. Maschinenrelevante Beziehungen sollen später in kanonischem YAML stehen; Markdown erläutert sie und schafft keine impliziten Graphabhängigkeiten.

## 1. Verständlicher Produkteinstieg

Der Einstieg beginnt beim Projekt und seinem Zweck. Skills, Agents und anbieterspezifische Integrationen erscheinen bei den Werkzeugen zur Umsetzung. Das Produkt braucht fünf verbundene Sichten:

| Sicht | Gegenstände und Fragen |
|---|---|
| Projekt und Fachmodell | Ziele, Kontext, Begriffe, Konzepte, Zustände, Übergänge, Regeln, Use Cases und Architekturentscheidungen |
| Organisation | Verantwortungsbereiche, Elternbeziehungen, Verantwortliche, Delegation und Entscheidungsbefugnisse |
| Prozesse und Workflows | Wie Ideen geklärt, Modelle geändert, Arbeit ausgeführt, Ergebnisse geprüft und Änderungen übernommen werden |
| Realisierung | Welche konkreten Artefakte Modellpflichten umsetzen und welche technischen Fähigkeiten dafür benötigt werden |
| Arbeitsstand und Nachweise | Beobachteter Bestand, Aufträge, Kandidaten, Befunde, aktuelle Prüfungen und offene Lücken |

Dies sind Ansichten auf verbundene Gegenstände, keine fünf konkurrierenden Wahrheitsquellen. Eine reine Aufteilung in Dateitypen würde fachliche Zusammenhänge nicht erfassen.

## 2. Gemeinsame Fachsprache

### Definitionen und Kontexte

Ein akzeptierter fachlicher Gegenstand benötigt stabile Identität, Kontext, kurze Bedeutung, Zweck und verantworteten Bereich. Fachliche Beziehungen referenzieren Identitäten. Ein Namespace grenzt Namen ab; er ist nur dann ein fachlicher Kontext, wenn das ausdrücklich festgelegt wurde. Gleichlautende Begriffe in verschiedenen Kontexten sind zulässig, unaufgelöste Mehrdeutigkeit ist es bei verbindlichen Referenzen nicht.

Begriffe können zu reicheren Konzepten gehören: Eine Order hat etwa Zustände und Beziehungen; eine Reservierung besitzt Lebenszyklusregeln. Das Modell muss nicht jedes Substantiv zum eigenen Gegenstand machen. Separate Definitionen sind sinnvoll, wenn Bedeutung, Wiederverwendung, Verantwortlichkeit oder Änderungsfolgen relevant sind.

Synonyme dürfen auf dieselbe Identität zeigen. Identische Schreibweise erzeugt keine fachliche Gleichheit. Verbindungen zwischen Kontexten benötigen ausdrückliche Zuordnung; ein Einkaufsvorgang ist nicht allein wegen des Namens dieselbe Order wie ein Verkaufsvorgang.

### Regeln und Use Cases

Eine Regel benennt ihren Geltungsbereich, beteiligte Konzepte und das erwartete Verhalten. Ein Use Case beschreibt Ziel, Akteur, Voraussetzungen, Ergebnis, relevante Abläufe und Fehlerfälle; er referenziert die geltenden Konzepte und Regeln.

Menschenlesbare Prosa bleibt möglich. Für maschinelle Folgen wird zusätzlich ausdrücklich deklariert, welche Definition verwendet oder welche Regel konkretisiert wird. Ein Assistent kann Referenzen vorschlagen; bloße Worterkennung oder Markdown-Links aktivieren keine Abhängigkeit und keine Pflicht.

### Definition ist nicht Durchsetzung

Es gibt getrennte Prüfungen mit jeweils ausgewiesenem Umfang:

- **Struktur:** Identitäten, Referenzen, Typen und erlaubte Beziehungen sind gültig.
- **Modellkonsistenz:** Ausdrücklich formal beschriebene Bedingungen werden durch passende deklarierte Checks geprüft.
- **Umsetzung:** Tests und andere Prüfungen untersuchen konkrete Artefakte gegen die Modellpflichten.
- **Semantische Bewertung:** Unabhängige Prüfer beurteilen Bedeutungsfragen, soweit sie nicht durch deterministische Checks beantwortet werden.

Für einen definierten Begriff ohne ausführbaren Check bleibt die Durchsetzung als solche offen. Das verhindert nicht seine Nutzung als präzise Vorgabe. Ein erfolgreicher Strukturcheck beweist weder fachliche Richtigkeit noch Einhaltung aller Regeln. Ein allgemeiner Solver oder die automatische Interpretation beliebiger Prosa gehört nicht zum ersten Umfang.

## 3. Bereiche, Namespaces und Module

| Begriff | Vorgeschlagene Bedeutung | Verantwortung |
|---|---|---|
| Bereich | Dauerhafte Verantwortung für einen Teil des Projekts; kann weitere Bereiche enthalten | Genau ein verantwortlicher Rollenbezug, eigener Zweck, klarer Scope und Integrationspflicht |
| Namespace | Stabiler Namensraum für Identitäten; optional ausdrücklich zu einem fachlichen Kontext erklärt | Genau ein verantwortlicher Rollenbezug und ein zuständiger Bereich |
| Modul | Wiederverwendbare, versionierte Einheit für Modellvokabular, Prozesse, Checks oder Integrationen | Genau ein verantwortlicher Rollenbezug für die Moduldefinition; Adoption durch ein Projekt wird separat verantwortet |
| Verantwortlicher | Dauerhafte Rolle mit auflösbarer Besetzung und definierten Befugnissen | Keine implizite Berechtigung durch Git-Identität, Installation oder Agentenstart |
| Arbeiter / ausführender Agent | Zeitlich begrenzte Instanz, die einen konkreten Auftrag bearbeitet | Handelt innerhalb des delegierten Auftrags und verändert keine eigenen Befugnisse |

Ein Bereich ist weder ein Ordner noch ein Modul oder Namespace. Explizite Zuordnungen können häufige 1:1-Fälle bequem ausdrücken, ohne diese Gleichheit universell vorauszusetzen. Ein Modul kann mehrere Namespaces liefern; ein Bereich kann mehrere Namespaces und Module verantworten oder einsetzen.

Der Rollenbezug bezeichnet den einzigen Verantwortlichen der jeweiligen Einheit. Ihre Bereichszuordnung beschreibt den organisatorischen Ort und Delegationsweg; sie schafft keinen zweiten konkurrierenden Verantwortlichen. Der Bereich verantwortet seinerseits die Integration seines gesamten Umfangs.

Um die gewünschte Namespace-Verantwortung ausdrücken zu können, schlägt dieser Plan eine explizite Namespace-Deklaration oder ein gleichwertiges kanonisches Register vor. Bestehende Namespace-Strings in Definitionen werden damit auf eine verantwortete Einheit abgebildet. Ein bloßes Namenslabel ist noch kein vollständiger Ownershipvertrag. Die genaue YAML-Form ist offen; die geforderte eindeutige Verantwortung ist es nicht.

Ein wiederverwendbares Modul hat einen Herausgeber bzw. Maintainer. Das übernehmende Projekt benennt einen lokalen Verantwortlichen für Auswahl, Version, Anpassungen und Einsatz. Das sind unterschiedliche Verantwortungsgegenstände, nicht zwei konkurrierende Eigentümer derselben Definition. Installation aktiviert keine Regeln und delegiert keine Befugnisse. Projektlokale Erweiterungen erhalten einen eindeutigen Besitzer; sie ändern das gepinnte Ursprungsmodul nicht heimlich.

„Verantwortlicher“ bezeichnet genau einen kanonischen Rollenbezug, nicht zwingend eine einzelne natürliche Person. Die Rolle kann durch eine Person oder ein Team besetzt sein; Entscheidungs- und Konfliktregeln müssen dann auflösbar bleiben. Dieselbe Rolle kann mehrere Einheiten verantworten. Das Projekt benennt seinen Root-Verantwortlichen ausdrücklich.

## 4. Rekursive Verantwortung

Alle Ebenen verwenden denselben Bereichsvertrag. Der Entwurf verlangt keine feste Anzahl von Ebenen oder Sondertypen pro Tiefe:

```text
Projekt
  Handel
    Bestellabwicklung
      Order-Lebenszyklus
        Stornierung
    Lagerbestand
      Reservierungen
```

Pro Modellrevision gelten folgende vorgeschlagene Invarianten:

1. Es gibt einen Projektwurzelbereich. Jeder weitere Bereich hat genau einen Elternbereich; die Elternbeziehung ist azyklisch.
2. Jeder Bereich, jedes Modul und jeder Namespace benennt genau einen gültigen Verantwortlichen. Elternverantwortung ersetzt eine fehlende Angabe nicht stillschweigend; eine ausdrücklich gewählte gemeinsame Rolle ist erlaubt.
3. Jeder akzeptierte Modellgegenstand besitzt genau einen kanonisch verantworteten Bereich. Weitere Beteiligung, Mitwirkung und Prüfung werden separat referenziert.
4. Unterbereiche übernehmen ausdrücklich delegierte Aufgaben und Befugnisse. Sie können keine zusätzliche Entscheidungsbefugnis aus ihrer Existenz ableiten.
5. Ein Elternbereich bleibt für den Gesamtauftrag, die Schnittstellen seiner Kinder und die Integration verantwortlich. Erfolgreiche Einzelprüfungen ersetzen seine eigene Integrationsprüfung nicht.
6. Fachliche Beziehungen dürfen Bereiche überqueren. Die Bereichshierarchie ersetzt den fachlichen Graphen nicht; betroffene gemeinsame Vorfahren koordinieren zusammengesetzte Änderungen.
7. Ausführung darf temporär feiner zerlegt werden, ohne für jeden Teilauftrag einen dauerhaften Bereich anzulegen.

Beliebige Modellierungstiefe bedeutet keine unbegrenzte Laufzeitrekursion. Aufträge benötigen weiterhin explizite Zeit-, Ressourcen- und Parallelitätsgrenzen, nachvollziehbare Abbrüche und Wiederaufnahme. Ein Blatt kann durch einen Menschen, einen Agenten oder ein deterministisches Werkzeug umgesetzt werden; Managementrollen müssen nicht künstlich vervielfacht werden.

## 5. Organisation, Prozesse und Workflows

Die Organisation beschreibt Zuständigkeit und Entscheidung. Ein Prozess beschreibt den wiederholbaren Weg zu einem Ergebnis einschließlich Pflichten und Entscheidungspunkten. Ein Workflow konkretisiert einen solchen Ablauf mit Schritten, Eingaben, Ausgaben, Fähigkeiten und Übergängen. Ob er bereits ausführbar ist, wird ausdrücklich angegeben; eine Checkliste ist noch keine durchgesetzte Laufzeitsteuerung.

Beispiel: Ein Prozess für Modelländerungen verlangt Vorschlag, Folgenabschätzung, befugte Entscheidung, Umsetzung und Prüfung. Die Organisation bestimmt den Entscheider. Ein Workflow legt fest, welche Schritte mit welchen Werkzeugen erfolgen. Ein konkreter Durchlauf hält Ergebnisse, Fehler und Nachweise fest.

Fachliche Abläufe wie „Order stornieren“ und Engineering-Abläufe wie „Geschäftsregel ändern“ können dasselbe allgemeine Prozessvokabular nutzen, bleiben aber durch Zweck, Kontext und Rollen unterscheidbar. Ein kaufmännischer Akteur erhält dadurch keine Repository-Schreibbefugnis.

## 6. Entstehung und Änderung des Modells

Brainstorming-Notizen, Skizzen und offene Fragen dürfen ohne formales Schema entstehen. Für einen ausgearbeiteten Vorschlag können Herkunft, Alternativen und Unsicherheit festgehalten werden. Erst eine ausdrückliche Entscheidung innerhalb geltender Befugnisse übernimmt Definitionen ins akzeptierte Sollmodell. Entwurfsstatus ist keine aktive Pflicht.

Die Minimalfolge lautet: freie Exploration → begrenzter Vorschlag → akzeptiertes Modell → Umsetzung und Nachweise. Die Benutzeroberfläche soll diese Verdichtung unterstützen, statt jede Notiz in einen Verwaltungsakt umzuwandeln. Unverbindliche Notizen können als Herkunft erhalten bleiben, ohne zu aktiven Graphabhängigkeiten zu werden.

Beobachtete Dateien können Modelllücken aufzeigen. Sie begründen einen Vorschlag, ändern aber keine geltende Definition. Bei unverändertem Soll und fehlerhafter Umsetzung wird die Umsetzung repariert. Eine echte Modelländerung wird am kanonischen Besitzer entschieden und löst neue Folgenabschätzung aus.

## 7. Artefakte, Fähigkeiten und Nachweise

Eine Modellpflicht kann durch mehrere Dateien gemeinsam realisiert werden. Eine Datei kann mehrere Pflichten unterstützen. Implementierung, Test, Dokumentation und Betriebskonfiguration haben unterschiedliche Rollen; eine erläuternde Dokumentation ist allein noch kein Nachweis des Verhaltens.

Die semantische Verantwortung ist von konkreter Schreibkoordination getrennt: Pro Änderungskandidat hat jeder veränderbare Pfad genau einen Writer. Mehrere Bereiche dürfen dieselbe Datei benötigen; ein Integrator sequenziert oder bündelt ihre Änderungen. Die Bereichshierarchie schafft keine automatische Ordnerzuständigkeit.

Technologien werden als Fähigkeiten zur Umsetzung und Prüfung eingebunden. Der Bereich Bestellabwicklung kann .NET, SQL, Markdown und ein CI-System einsetzen. Eine neue Technologie schafft nicht automatisch einen neuen Managementbereich.

Das Repositoryinventar unterscheidet kanonische Modelleingaben, verwaltete Realisierungen, generierte Ausgaben, fremdverwaltete Artefakte, begründete Ausschlüsse und ungeklärte Dateien. Quellen im Repository generieren sich nicht selbst. Die Klassifikation und aktuelle Dateibeobachtung sind von akzeptierter Absicht unterscheidbar.

Prüfungen binden Modellversion, Kandidatenrevision, relevante Eingaben und Prüfkonfiguration. Umbenennung von Begriffen darf diese Bindungen nicht umgehen oder alte Ergebnisse nachträglich aufwerten. Die Organisation bestimmt den Annahmeweg; ein technisches PASS übernimmt keine Änderung von selbst.

## 8. Durchgehendes Beispiel

Der folgende Fall ist eine hypothetische Modellierungsübung, keine eingeführte Geschäftsregel:

1. In einer Notiz wird vorgeschlagen, dass eine Stornierung reservierten Bestand freigibt. Noch offen: Sind bereits versandte Orders stornierbar? Erfolgt die Freigabe atomar oder später?
2. Der Namespace `sales` erhält definierte Konzepte für Order und ihren Lebenszyklus. `inventory` definiert Bestand, Reservierung und Freigabe. Beide Namespaces haben ausdrücklich benannte Verantwortliche.
3. Der Use Case `sales/cancel-order` referenziert die Konzepte und eine akzeptierte Freigaberegel. Voraussetzungen und zeitliche Garantien werden entschieden; ihre genaue Ausprägung bleibt in diesem Plan offen.
4. Der Bereich Bestellabwicklung verantwortet den Use Case; Lagerbestand verantwortet die Reservierungsregel. Der gemeinsame Elternbereich Handel übernimmt die Integrationspflicht. Die Kontexte müssen nicht exakt der Managementhierarchie entsprechen.
5. Eine Änderung an der Bedeutung von Reservierung betrifft die ausdrücklich verwendenden Regeln, Use Cases und Realisierungen. Eine bisherige Prüfung wird nur bei unveränderten gebundenen Eingaben wiederverwendbar; bloße Begriffsnähe reicht weder für eine Abhängigkeit noch für einen Ausschluss aus dem Impact.
6. Ein Auftrag bindet das akzeptierte Soll, verteilt konkrete Arbeit und erzeugt einen Kandidaten mit Implementierung, Tests und Dokumentation. Technische Werkzeuge werden nach Bedarf verwendet.
7. Getrennte lokale Prüfungen und eine eigene Prüfung des vollständigen Stornierungsablaufs untersuchen denselben gebundenen Kandidaten. Ein fehlender Freigabenachweis bleibt sichtbar.
8. Eine fachlich korrekte bestehende Implementierung kann erhalten bleiben. Bei Drift werden ihre Artefakte repariert; der Auftrag erfindet keine Modelländerung, um eine fehlerhafte Implementierung zu legitimieren.

## 9. Öffentliche Sprache

Der neue Einstieg spricht von Projektmodell, Fachsprache, Bereichen, Verantwortlichen, Prozessen, Arbeitsaufträgen, Umsetzung und Nachweisen. Modul und Namespace erscheinen mit ihrer eigenen Bedeutung. Anbieterintegrationen sind optionale Werkzeuge.

Für die Arbeit gelten die Verben: modellieren, prüfen, Folgen bestimmen, planen, delegieren, umsetzen, integrieren, verifizieren und übernehmen. Diese Vorschläge sind keine implementierten CLI-Kommandos. Die Managementanalogie verlangt keine Ministerien, Gerichte oder zusätzlichen Institutionen. Solche Mechanismen brauchen einen unabhängig begründeten Bedarf.
