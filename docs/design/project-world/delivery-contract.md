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

Der zunächst erwogene Containerlauncher wurde in diesem Kandidaten nicht umgesetzt. `isolated` blockiert deshalb ausdrücklich; es gibt keinen Rückfall auf lokale volle Rechte. Ein gepinnter und geprüfter Container-/VM-Launcher mit getrennt berechtigtem Applybroker, geschützten Credentials und nachgewiesener Egressgrenze bleibt eine eigene offene Implementierung. Ebenso fehlen Live-Codex-/Claude-Läufe und eine autonome Gesprächsoberfläche im Markitect-CLI. Der [Prüfbericht](../../validation/project-world-delivery.md) trennt die tatsächlich ausgeführten Sourcegates von diesen offenen Nachweisen.
