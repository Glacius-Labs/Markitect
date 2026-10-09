# Umsetzungsvertrag und Teamaufteilung

Status: konkrete, koordinierte Implementierung des Nutzerauftrags auf der frisch abgefragten Hauptlinie `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`. Integration erfolgt lokal auf `codex/model-driven-delivery`. Die ursprüngliche Planung bei `d3f3b43` bleibt erhalten. Kein Push, Release oder Eingriff in andere aktive Arbeitskopien ist Teil dieses Auftrags.

## Gewählte technische Abbildung

Der bestehende Standardbibliothek-Core bleibt unverändert. Die neue Host-Frontend-Abbildung benutzt seine vollständigen Identitäten, referenztypisierten Properties und normale `Compile`-Funktion. Abgeleitete Zuordnungs-/Kontext-/Impactreports sind keine zweite kanonische IR. Historische Project-/Projectionbefehle bleiben kompatibel; neue öffentliche Befehle verwenden `project` und sprechen von Modell, Manager, Artefakt, Kandidat und Umsetzung.

Das minimale Schema trägt `project.markitect.example.org/v1alpha1` und fünf Kinds:

| Kind | Verbindlicher Inhalt |
|---|---|
| Manager | Zweck, optionaler Elternmanager, ausdrücklich verantwortete Repositorypfade bzw. Präfixe, lokale Entscheidungsvorgaben |
| Statement | Kategorie `concept`, `rule`, `use-case`, `architecture` oder `workflow`; Beschreibung; öffentliche Sichtbarkeit; ausdrückliche `uses`- und `requires`-Referenzen auf Statements |
| Artifact | Rolle, realisierte Statements, Pflicht/Anwendbarkeit und Begründung, bekannte erwartete Pfade, relevante Checks |
| Check | deklarierte literal argv, verantworteter Scope über Modellzuordnung, relevante Statements, Grenzen seiner Aussage |
| Decision | ausdrücklich referenzierter Gegenstand, Entscheidung/Begründung und vorhandene Autorisierung; kein selbstauthentifizierendes Freigabefeld |

`Statement` als gemeinsamer typisierter Ziel-Kind vermeidet ein fingiertes Union-Referenzfeature. Erweiterungen bleiben auf dieser technischen Basis zu prüfen. Die Auswahl ist ein kleiner bewusster Startwortschatz, keine Quellcodeontologie oder eingebaute Unternehmensorganisation.

Die Hostkonvention setzt Modellnamensräume aus dem Verzeichnis relativ zu `.markitect/model/`: Ordnersegmente verwenden Kleinbuchstaben, Ziffern und Bindestriche und beginnen mit einem Buchstaben; Punkte sind als Segmenttrenner reserviert. Beispielsweise wird `commerce/sales/orders` zu `commerce.sales.orders`; der Modellwurzel entspricht der leere Namespace. Authored `metadata.namespace` muss dazu passen; Änderungen werden als Identitätsänderung sichtbar. Eine Managerdefinition liegt in `manager.yaml`; andere Definitionen können beim Slice bleiben. Jeder interne Ordner gehört zum nächsten deklarierten Manager. Elternmanager delegieren über ausdrückliche Kindverträge; widersprüchliche Geschwister-Dateiansprüche schlagen fehl. Pfadselektoren beginnen bewusst mit exakten Pfaden oder Verzeichnispräfixen; beliebige Query-/Patternsprachen sind kein Bedarf dieses Durchlaufs. Ein Manager kann mit `.` den Repositoryumfang verantworten; die davon unabhängige Inhaltsauswahl im Manifest verwendet weiterhin konkrete Pfade und Präfixe.

`.markitect/project.yaml` enthält Version, Projektname, exakt ausgewählte Modelldateien und Inventarscope/Ausschlüsse. Die fachliche Corekompilierung sieht nur die ausgewählten Definitionsbytes. `.markitect/runtime.yaml` konfiguriert Runner/Provider und begrenzte Ausführung, getrennt von Fachbedeutung. `.markitect/drafts/` bewahrt Entwürfe; `views/` enthält lesbare Sichten, `runs/` gebundene operative Daten und `cache/` wiederherstellbare Ausgaben. Diese Ausgaben werden nicht rekursiv erneut als Sollquelle inventarisiert.

## Endlicher Befehlsvertrag

`markitect project <action>` ist der neue Einstieg. Jede schreibende Aktion verlangt ausdrücklich `--write` und bindet eine Vorschau bzw. einen angenommenen Auftrag; keine globale Providerkonfiguration wird verändert.

| Aktion | Ergebnis |
|---|---|
| `init` | read-only Vorschau bzw. geprüfte Initialisierung ausschließlich unter `.markitect/` |
| `check`, `index`, `context` | deterministischer Struktur-/Ownership-/Abdeckungsbericht, beide Zuordnungsrichtungen, relevanter Managerkontext |
| `impact` | Vergleich fester alter/neuer Quellen inklusive entfernten Bezügen, unveränderten benötigten Artefakten und unbekanntem Scope |
| `document` | lesbare Spezifikation mit Quellen-/Prüfstand; gezielte Ausgabe unter `.markitect/views/` |
| `edit` | gebundener Modelländerungsvorschlag, validierter Kandidat und befugte Annahme; kein manuelles YAML erforderlich |
| `discover`, `distill`, `adopt` | ausgewählte feste Belege, Grounding-/Klärungsvorschlag und explizit angenommene Teilübernahme |
| `plan`, `run`, `resume`, `status` | gebundener Auftragsbaum, getrennte Manageraufrufe, gezielte Wiederaufnahme und nachvollziehbarer Zustand |
| `verify`, `apply` | unabhängige deklarierte Checks am tatsächlichen integrierten Kandidaten; guarded Übernahme mit frischen Bindungen |

Die Integration liefert diese Oberfläche zusammenhängend, implementiert aber in überprüfbarer Reihenfolge: P1 trägt Init/Check/Index/Context/Impact/Edit und die ausgewählten Discovery-/Distillations-/Adoptionsprimitive; P2 Plan/Run/Resume/Status/Verify/Apply am begrenzten Shop; P3 verallgemeinert die Modulrekursion; P4 vervollständigt lesbare Sichten, Agentenführung und Installationserklärung. Frühe Scaffoldaktionen brauchen noch keine vollständige Gesprächsoberfläche. Alle Ebenen behalten ihre eigenen Abnahmekriterien.

Ein bereits erteilter Nutzerauftrag kann `edit` und `run` autorisieren. `--write` ist die technische Anzeige dieser bestehenden Absicht, keine zusätzliche menschliche Genehmigungsschleife. Externe Veröffentlichung bleibt gesondert. JSON-Dateien sind Transport für Agenten und Belege; der normale Nutzer sieht die fachliche Darstellung.

## Team und gemeinsame Schnittstellen

Alle Implementierenden verwenden Luna mit Reasoning High und isolierte Worktrees ab dem koordinierten Vertragscommit. Es gibt genau einen Writer pro Paket. Shared DTOs, Root-CLI-Wiring, Root-Dokumente, `go.mod`, CI und Integration bleiben beim Koordinator. Keine Coreänderung ist vorgesehen. Paketnamen unter `internal/modules` beschreiben Implementierungseinheiten, nicht neue installierbare Modultypen.

| Arbeit | Exklusive Pfade | Ausstieg |
|---|---|---|
| Reines Projektmodell | `internal/modules/projectmodel/` außer koordinierter Typdatei | Schema, Ownership/Index, typisierte Beziehungseffekte, Kontext und Impact mit deterministischen Negativfällen |
| Host-Frontend | `internal/host/projectwork/` | striktes Laden, Init, Modellproposal, Snapshotbindung, Sichten und sichere zugehörige Writes |
| Ausführung | `internal/host/projectrun/` | persistenter Auftragsbaum, getrennte Agentenaufrufe über agentexec, Kandidat, Prüfung, Apply und Resume |
| Brownfield | `internal/host/projectadoption/` | exakte Discovery, belegte Vorschläge/Fragen, gebundene Resolution und Modellübernahme ohne Codewrites |
| Claude-/Ausführungsgrenze | `internal/tooling/clauderunner/` und koordinierte Host-/agentexec-Anpassungen | geprüfter Vorschlagsadapter und explizite lokale Ausführungsgrenze, keine permissive Ersatzinvocation; reine Protokolltests von echten Providerläufen getrennt |
| Bedienung und Beispiele | `internal/host/projectcli/`, `examples/project-world/`, `docs/project-workflow.md` | vollständige neue Befehle und ausführbares Shopbeispiel, Installation/Adoption/Alltagsarbeit nachvollziehbar |

Der Modulvertrag benutzt `core.Model` als Eingabe. Der gemeinsame abgeleitete Report enthält Manager, Statements, Artefakterwartungen, Checks, konkrete Dateieinträge, Befunde und Digests. Öffentliche reine Funktionen sind `Schema() core.Schema`, `Analyze(core.Model, []File) Report`, `Impact(Report, Report) ChangeImpact` und `Context(Report, managerID) (ManagerContext, error)`. IDs bleiben vollständige `core.DefinitionIdentity.Key()`-Werte; lesbare Namen sind Darstellung, keine zweite Identität.

Host stellt `Load(root, revision) (*Project, error)` mit Model, Report, unveränderlichem Snapshot, Config und Digest bereit. Modellwrites erfolgen über `PlanEdit`/`ApplyEdit` mit erwarteter Basis und konkreten Dateioperationen. Init und Dokumentation haben eigene bounded APIs. Runtime und Adoption konsumieren diesen Hostvertrag, nicht umgekehrt. CLI komponiert sie und übernimmt keine Compilersemantik. Runner nutzen `agentexec.Config` und die vorhandenen gebundenen Request/Response/Receipt-Verträge; historische interne Feldnamen dürfen im neuen Benutzerfluss verborgen bleiben.

## Integrationsgates

1. Kleines neues Projekt samt Modell, Index und lesbarer Sicht ohne manuelles YAML.
2. Brownfieldfixture mit dokumentiertem Widerspruch, begründeter Klärung und unübernommenem Restscope.
3. Shop-Stornierung über getrennte Orders-/Inventory-Manager, tatsächlichen gemeinsamen Kandidaten, lokale und Integrationsprüfung sowie kontrollierten Abbruch/Resume.
4. Negative Pfad-, Ownership-, Selbstautorisierungs-, stale-plan-, fehlende-Check- und Pflichtlöschfälle; keine umetikettierte Teilabdeckung als Ganzprojekt-PASS.
5. Fokussierte Paketsuites, unabhängiger Gesamtdiffreview und die normalen Contribution-Gates am integrierten Kandidaten. OS-Isolation und tatsächliche Codex-/Claude-Modellläufe erhalten eigene Evidenzzeilen.

Ein bestandener Protokolltest ist keine Produktivitätsmessung, OS-Sicherheitszertifizierung oder menschliche Annahme. Während der Integration werden Befunde behoben; das Team beendet seine Arbeit nicht bei API-Stubs oder bloßen Plänen.

## Tatsächlicher Stand der Ausführungsgrenze

Dieser Sourcekandidat implementiert den im [Agentenarbeitsweg](adoption-and-agent-boundary.md) vorgesehenen ersten Modus `controlled-local`. Er kontrolliert ausgewählte Eingaben, begrenzte Vorschläge, Managerzuordnung, Prüfungen und den Apply-Weg. Repositoryanweisungen und Providerflags können einem Prozess mit gewöhnlichen Nutzerrechten keine wirksame OS-Schreibgrenze geben.

Der zunächst erwogene Containerlauncher wurde in diesem Kandidaten nicht umgesetzt. `isolated` blockiert deshalb ausdrücklich; es gibt keinen Rückfall auf lokale volle Rechte. Ein gepinnter und geprüfter Container-/VM-Launcher mit getrennt berechtigtem Applybroker, geschützten Credentials und nachgewiesener Egressgrenze bleibt eine eigene offene Implementierung. Eine autonome Gesprächsoberfläche im Markitect-CLI und Live-Claude-Nachweise bleiben offen. Die Fortführung führt echte Codex-Aufrufe am Shop aus; der [Prüfbericht](../../validation/project-world-delivery.md) bindet deren Ergebnisse an die tatsächlich getesteten Sourcekandidaten.

## Fortführung zur tatsächlichen Benutzung

Der Folgeauftrag verlangt einen echten Shop-Durchlauf und das Schließen der dabei sichtbar werdenden Bedienlücken. Dafür gelten diese ergänzenden Verträge:

- `project setup` entdeckt einen ausdrücklich gewählten nativen Provider, Python und die Adapter im bezeichneten Toolroot. Es zeigt eine normale Runtime-Mutation mit exakten Dateibindungen und endlichen Grenzen; nur der passende Editdigest erlaubt die Übernahme. Setup startet keinen Agenten, verändert keine globale Konfiguration und liest keine Anmeldegeheimnisse. Tokenraten sind ausdrücklich vom Aufrufer gesetzte Budgetgewichte, keine behaupteten Providerpreise. Eine lesende Diagnose darf ungeprüfte Anmeldung nicht als gültig ausgeben.
- `project plan --since COMMIT` vergleicht die alte feste Modellbasis mit dem aktuellen akzeptierten Commit. Die Ausführungsbasis bleibt aktuell und muss den ausgewählten Arbeitsbytes entsprechen. Impactpflichten werden mit ausdrücklich gewählten Managern vereinigt; eine Auswahl kann nötige Arbeit nicht wegfiltern. Ohne Modelländerung oder ausdrückliche Eingrenzung bleibt die Planung konservativ. Jeder Manager kennt den Unterschied zwischen vollständiger eigener Arbeit samt Delegation und späterer Integration des Gesamtauftrags.
- `project distill --generate --write` darf genau einen konfigurierten, begrenzten Analyseaufruf starten. Der Host bindet die tatsächliche Runnerreceipt und prüft den gelieferten Entwurf gegen dieselben festen Discoverybelege. Bericht und Receipt bleiben Vorschläge; Klärung und Scopeannahme erfolgen gesondert. Ein Kostenlimit begrenzt die Annahme anhand gemeldeter Nutzung, nicht die Rechnung vor einem Provideraufruf.
- Der echte Shopversuch ändert zunächst das Modell: Stornierung ist zusätzlich im Zustand `packing` erlaubt. Anschließend müssen die betroffenen Manager Code, Dokumentation und Regressionstest anpassen und integrieren. Die vorhandenen Regeln für versendete Orders, atomare Reservierungsfreigabe und Idempotenz gelten weiter. Ein unabhängiger Verhaltenstest prüft anschließend die tatsächlich übernommenen Bytes; Protokoll-PASS allein schließt den Versuch nicht.

Providerfehler und Versuchsgrenzen bleiben Bestandteil des Ergebnisses. Nicht verfügbare Modelle werden nicht still ersetzt. Der lokale Lauf erhält zusätzliche deklarierte Werkzeugbeschränkungen, ohne daraus eine nicht geprüfte OS-Isolation abzuleiten.

Für die tatsächliche CLI-Bedienung baut `resolve` einen gebundenen Resolution-Transport aus ausdrücklich gelieferten Fragen-/Scopeentscheidungen. Host ergänzt die aktuelle Tool-/Schema-/Zielbindung; weder Agent noch Nutzer müssen interne Buildidentitäten nachkonstruieren. Das erzeugt keine Annahmeentscheidung und verändert kein Modell. Generierte Brownfieldvorschläge erhalten den ausdrücklich ausgewählten öffentlichen Zielkontext zusätzlich zu den getrennt ausgewählten Quellbelegen. Dateirealisierung bleibt ein Lesebezug: Ein Artefakt kann Dateien mehrerer Verantwortlicher nennen, ohne deren Schreibrechte zu übernehmen; der ausführende Agent erhält die konkret aufgelöste Schreibzuständigkeit.

Bekannte ungültige Antwort- oder Dateivorschläge dürfen innerhalb des ausdrücklich gesetzten `maxRetries` erneut beim selben Manager angefordert werden. Der verworfene Vorschlag wird nicht angewendet; jede tatsächlich erhaltene Invocation und ihre Kosten bleiben dauerhaft gezählt. Die Reparatur erhält denselben begrenzten Kontext samt konkreter Compilerdiagnose. Transportfehler mit unbekanntem Ausgang, fachliche Blockaden, Quellenänderungen und fehlgeschlagene Prüfungen sind keine automatische Wiederholung. Aktive Zuständigkeiten dürfen als öffentliche Routingmetadaten sichtbar sein, ohne private Unterkontexte offenzulegen.

Der echte Shopversuch hat außerdem eine konkrete Integrationslücke gezeigt: Ein Kandidat kann strukturell gültig sein und erst in den unabhängig ausgeführten Checks scheitern. `project repair --run RUN_ID --write` führt deshalb ausdrücklich bekannte fehlgeschlagene Pflichtchecks an die Manager zurück. Dieser Reparaturschritt behält Laufidentität, ursprünglichen Zeitbeginn, Starts, Nutzungs-/Kostenledger und die vorherigen Prüfbelege. Er verändert weder das akzeptierte Modell noch Prüfpflichten oder Dateiverantwortung. Der begrenzte Reparaturrundenvertrag ist von einer Wiederholung mit unbekanntem Providerergebnis getrennt. Ein reparierter Kandidat braucht eine neue unabhängige Prüfung; ein unverändert fehlgeschlagener Kandidat kann dadurch keine gültige Applyfreigabe erhalten. Dieser Vertrag ist eine Implementierungsanforderung; der Prüfbericht muss den tatsächlichen Nachweis separat ausweisen.


## Implementer-/Reviewer-Schleife und gezielte Nacharbeit

Der Nutzerauftrag ergänzt den vorhandenen Managerbaum um getrennte ausführende und prüfende Agentenrollen. Die fachliche Verantwortung bleibt beim bestehenden Manager; Rollen erfordern keine neuen Namespaces, Core-Kinds oder Dateieigentümer. Standard-Setup aktiviert das Review. Ältere explizite Runtimekonfigurationen ohne Reviewblock behalten ihren bisherigen Vertrag und erhalten dadurch keinen rückwirkenden Reviewnachweis.

Der Ablauf lautet: begrenzter Managerauftrag → Implementer-Kandidat → unabhängige Reviewer-Invocation → konkrete Befunde → überarbeiteter Kandidat → erneutes Review → Ergebnisbericht an den Manager → Integration. Ein Manager darf seinerseits gezielte Nacharbeit an aktive direkte Kinder zurückgeben. Der betroffene Teilbaum durchläuft Umsetzung und Review erneut; betroffene Vorfahren integrieren das Ergebnis frisch. Unveränderte unabhängige Geschwister behalten ihre gültigen Ergebnisse.

Ein rein delegierender Manager ohne ausgewählte Implementierungsdateien, deklarierte Artefakte und eigene aufgezeichnete Änderungen benötigt kein lokales Implementierungsreview. Der Host hält dies als `not-required` fest und erzeugt kein künstliches PASS. Delegation und Integration sowie die Reviews seiner implementierenden Kinder bleiben erforderlich. Fehlende deklarierte Artefakte und Löschungen sind keine Ausnahme von der Reviewpflicht.

Reviewer erhalten den Originalauftrag, akzeptierte relevante Modellinhalte und tatsächlich ausgewählte Kandidatenbytes. Sie erhalten kein Implementertranskript und keine Schreibbefugnis. Ein Befund bezeichnet einen konkreten betroffenen Pfad, eine nachvollziehbare Modell-/Quellengrundlage und die erwartete Korrektur. Ein Reviewer darf weder Prüfpflichten lockern noch Geschäftsregeln oder Zuständigkeiten verändern. Noch nicht ausgeführte bereichsübergreifende Checks bleiben ausdrücklich ausstehend; der Host führt sie am integrierten Kandidaten unabhängig aus.

Prüfauftrag sind der eigene Managerauftrag, die aktuelle Phase und die zugeordneten Verträge. Der globale Originalauftrag dient der Einordnung. Öffentliche Verträge anderer Bereiche erlauben die Prüfung der eigenen Schnittstellennutzung; fehlende fremde Implementierungsbytes sind kein lokaler Mangel und kein Grund für `incomplete`. Fehlende Informationen, die für die Prüfung der eigenen Pflichten erforderlich sind, bleiben dagegen ein Grund für einen unvollständigen oder eskalierten Befund.

Die Beschreibung beabsichtigten Verhaltens oder statisch erkennbarer Testabdeckung in einem Artefakt behauptet für sich genommen keine Testausführung. Reviewer prüfen solche Aussagen gegen die gelieferten Modell-/Artefaktgrundlagen und unterscheiden sie von Behauptungen tatsächlich ausgeführter oder bestandener Checks. Integrationsberichte unterscheiden eigene Änderungen, durch gelieferte Evidenz belegte Kindergebnisse und verbleibende Unsicherheit. Fehlende Implementierungsdetails in einer kompakten Zusammenfassung allein belegen keinen Implementierungsfehler; eine Nacharbeitsanfrage benötigt einen konkreten Widerspruch zur aktuellen Evidenz oder einen konkret belegten unerfüllten Vertrag.

Lokale Reviewbytes folgen der Dateiverantwortung und aufgezeichneten Integrationsänderungen des Managers. Bereichsübergreifende Artefaktzuordnungen liefern Vertragskontext; jede beteiligte Implementierungsdatei erhält ihr Review beim zuständigen Manager. Die Integration prüft das Zusammenspiel der Bereiche. So gehen lokale Korrekturen an Implementierende mit der passenden Schreibzuständigkeit.

Zum relevanten Modell gehören auch die Schnittstellen der Pflichtartefakte aktiver direkter Kinder und Artefaktverträge, die den ausgewählten Dateien ausdrücklich zugeordnet sind. Eine Datei kann einen Vertrag eines anderen Verantwortlichen realisieren. Dessen öffentliche Fachverträge müssen dann prüfbar sein; private Untermodelle und zusätzliche Implementierungsdateien werden dadurch nicht sichtbar. Diese Lesebeziehung erweitert keine Schreibbefugnis. Befunde bleiben an tatsächlich gelieferte Pfade und explizit zugelassene Vertragsidentitäten gebunden.

`runtime.review` trägt `agents` je Manager sowie positive endliche `maxRounds` und `maxManagerRounds`. Das Standard-Setup verwendet zunächst drei kumulative Reviews je Manager und Phase sowie zwei Manager-Nacharbeitsrunden für den gesamten Lauf. Manager-Nacharbeit und spätere Checkreparatur setzen diese Zähler nicht zurück. Modell, Provider, Reasoning, Toolpins, Umgebung und Preisgewichte bleiben ausdrücklich konfiguriert. Die bestehenden globalen Start-, Zeit- und Kostenlimits gelten unverändert und zählen Implementer, Reviewer, Management und Checks zusammen. Unbekannte Invocationausgänge werden nicht wiederholt. Protokollkorrekturen, fachliche Review-Nacharbeit, Manager-Rückgabe und spätere Reparatur fehlgeschlagener Pflichtchecks bleiben getrennt nachweisbar.

Ein Review bindet die konkrete Kandidatenidentität und die tatsächlich geprüften Eingaben. Die Gültigkeit beim Zusammenführen wird aus unveränderten geprüften Bytes, Modellvorgaben und Auftragsgrundlagen hergeleitet; ein neuer globaler Kandidatenname allein entwertet keine unveränderte unabhängige Prüfung. Änderungen an geprüften Eingaben verlangen frische Evidenz. Ein Manager darf mit eigenen Integrationsänderungen die Reviewpflicht nicht umgehen. Frühere Befunde, Kandidaten, Aufrufe, Kosten und Freigaben bleiben in der Laufhistorie erhalten.

Blockierende fachliche Uneinigkeit und erschöpfte Runden bleiben als blockierter Lauf mit einem Befund zum zuständigen Manager sichtbar. Eine weitere Entscheidung benötigt die befugte Ebene. Budgeterschöpfung, unklare Ausführung oder fehlende aktuelle Nachweise bleiben unvollständig. Technisches Review-PASS ist keine menschliche Annahme. Fresh Verify und guarded Apply behalten ihre bisherigen Bindungen und verlangen bei aktiviertem Review zusätzlich gültige aktuelle Reviewnachweise.

Manageraufträge tragen den expliziten Kontexttyp `projectrun-task/v1`. Integration darf geerbte offene Fragen und Risiken nur mit aktueller Evidenz und ihrem exakten Wortlaut auflösen. Verbleibende Fragen und Risiken außerhalb gezielter lokaler Nacharbeit werden mit `status: partial`, einer konkreten Frage oder einem Risiko und dem exakten `escalationTarget` an die nächste befugte Instanz weitergegeben; der äußere Ausgang ist `escalated`. Das gilt auch für benötigte Arbeit in einem Geschwisterbereich, die der eigene Kontext nicht abschließend beurteilen kann. Noch ausstehende Hostchecks sind kein offener Implementierungsmangel. `complete` darf keine offene Frage und kein offenes Risiko übergehen; Weglassen schließt keinen Befund. Eine gültige Nacharbeitsanfrage an ein direktes Kind kann ohne zusätzliche offene Frage einen vollständigen Integrationsbericht begleiten. Der Host führt diese Nacharbeit samt frischem Review und Integration vor dem abschließenden Laufschluss aus.

Die Abnahme umfasst einen behobenen Reviewerbefund, Manager-Rückgabe nach erstem PASS mit erneutem Review, unveränderte Geschwister ohne Neuausführung, abgelehnte Reviewer-Schreibvorschläge und fremde Nacharbeitsziele, veraltete Reviewevidenz, erschöpfte Runden/Budgets und unklare Aufrufe. Ein echter Luna-High-Shopdurchlauf ergänzt diese mechanischen Tests; beide Evidenzarten werden separat berichtet.
