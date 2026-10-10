# Historian Eingangsregister für Markitect Konzepte

Dieses Register nimmt relevante Funde aus der Markitect-Historie auf. Der Historian darf sie an den Concepts-Chat übergeben. Jede Aufnahme bewahrt Fund-ID, Herkunft und den Aussagezustand; sie schafft weder einen Roadmap-Eintrag noch eine Produktentscheidung oder Implementierung.

## Aufnahmeregel

1. Eine Fund-ID bleibt zusammen mit ihrer konkreten Quelle stabil. Vor einer neuen Aufnahme nach derselben Kombination suchen.
2. Bei derselben Fund-ID und Quelle den vorhandenen Eintrag fortschreiben, wenn es sich um denselben Gedanken handelt. Neue Belege oder spätere Wiederholungen als Ergänzung mit Datum erfassen; nicht kopieren, umbenennen oder als neue Idee ausgeben.
3. Eine ähnliche Aussage mit anderer Quelle bleibt als eigener Fund erkennbar. Beziehungen und Abweichungen zwischen den Quellen werden verlinkt; ihre Herkunft wird nicht zusammengezogen.
4. Nutzerwortlaut möglichst wörtlich mit Fundstelle erhalten. Historian-Zusammenfassung und Concepts-Interpretation getrennt kennzeichnen. Fehlende Metadaten nicht erschließen.
5. Für jeden Fund Idee, Problem, Mechanismus, Annahmen, erhoffte Wirkung, Herkunft und stabile Fund-ID festhalten. Bei Bedarf Grenzen, Gegenbelege, offenen Fragen und Beziehungen ergänzen.
6. Eine spätere explizite Revision bleibt neben dem älteren Wortlaut auffindbar. Festhalten, wer sie vorgenommen hat, auf welche Aussage sie sich bezieht und ob sie frühere Geltung tatsächlich ändert.
7. Fundnotizen gehen in die spätere gemeinsame Auswertung ein. Sie lösen keine automatische Roadmap-, Produkt- oder Implementierungsänderung aus. Rückmeldeschleifen und Nachrichten an andere Chats sind aus einer Fundübernahme nicht abzuleiten.

## Eintragsvorlage

```text
Fund-ID:
Quelle:
Funddatum oder Zeitraum: nur falls überliefert
Wörtlicher Wortlaut oder präzise Fundstelle:
Vom Historian übermittelte Zusammenfassung:
Status der Aussage: Nutzerabsicht | Erfahrung | Hypothese | Interpretation | Entscheidung | implementierte Fähigkeit
Idee:
Problem:
Mechanismus:
Annahmen:
Erhoffte Wirkung:
Grenzen und Gegenbelege:
Offene Fragen:
Beziehungen zu vorhandenen Fund-IDs oder Konzepten:
Änderungsverlauf:
```

„Implementierte Fähigkeit“ ist nur mit einem aktuell verifizierten Quellstand einzutragen. Ein historischer Plan, eine damalige Aussage oder ein Chatbericht allein belegt keine heutige Fähigkeit. Nutzerabsicht und Projekthypothesen bleiben als solche gekennzeichnet.

## Eingangsstand

Die [Nutzerbeschreibung vom 9. Oktober 2026](user-description-20261009.md), der [Diskussionsimpuls zur Spezifikationswirksamkeit](discussion-impulse-20261009-01.md) und weitere direkte Nutzerbeiträge bleiben eigenständige quellengebundene Konzeptnotizen; sie werden nicht nachträglich als Historian-Funde etikettiert.

## Aufgenommene Historian-Funde

### HIST-MARKITECT-20261009-FORMAT-001

- **Deduplizierungsschlüssel:** Fund-ID `HIST-MARKITECT-20261009-FORMAT-001` plus die nachfolgenden Original-Chat- und Nachrichten-IDs.
- **Herkunft:** Historian-Übergabe in den Concepts-Chat; direkte Korrektur in „Markitect YAML als Ontologie“, Chat `01a11d05-adbf-7082-b693-caebbfb83a2a`, Nachricht `01a11ed2-56ee-71c1-a545-29cca0b23454`. Weitere referenzierte Quellen: Nachricht `01a11ed4-5510-7023-b01b-d4ba3ad6234d` im selben Chat sowie Design-Chat `01a11c80-37aa-7fe0-9586-35d916ce6561`, Nachricht `01a11f78-979a-7e30-9627-b31c280ef5ac`.
- **Historian-Zusammenfassung:** Der Nutzer wies „Markitect structures engineering knowledge and makes it usable as Markdown“ als falsche Beschreibung zurück. Er verband „Markdown“ im Namen mit verbreiteten Kontext- und Antwortformaten für AI-Agenten und sagte, Markdown Front Matter könne entfallen.
- **Aussageart:** Direkte Produktkorrektur, über den Historian mit konkreten Quellen übermittelt.
- **Offene technische Frage:** Der Historian kennzeichnet den Management-Branch als Referenzkandidaten; typisiertes Modell beziehungsweise YAML für knappe Spezifikationen und Markdown-Projektionen waren mögliche Richtungen. YAML gegenüber OWL oder Knowledge Graph blieb laut Übergabe offen.
- **Bezug zu Concepts:** Die ursprüngliche Nutzerbeschreibung vergleicht Ordner und Namespaces mit klassischer Markdown-Dokumentation und Code beim Gruppieren und Organisieren. Das begründet keine Wahl des Markitect-Quellformats. C01 und C03 bleiben formatneutral. Die Korrektur ändert den Wortlaut der Nutzerbeschreibung nicht.
- **Grenze:** Die hier vorliegende Zusammenfassung ersetzt nicht den vollständigen Wortlaut der verknüpften Originalnachrichten und beweist keinen aktuellen Implementierungsstand. Kein Formatentscheid wurde daraus abgeleitet.
