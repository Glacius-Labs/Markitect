# Adoption und kontrollierter Agentenarbeitsweg

Status: konkretisierter Produktentwurf für die Umsetzung auf `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`. Nutzerauftrag: diesen Entwurf nach Review implementieren. Die vorhandenen öffentlichen Befehle und Releases bleiben kompatibel; ein neuer `markitect <verb> ...`-Einstieg trägt den Modul-/Managervertrag unter `.markitect/`.

## Was „nur über Markitect“ bedeutet

Anweisungen in AGENTS.md oder CLAUDE.md machen den Arbeitsweg auffindbar. Sie erteilen keine technische Garantie. Ein Agent mit uneingeschränkter Shell und denselben Dateirechten wie der Nutzer kann solche Anweisungen, Hooks oder einen lokalen Wrapper umgehen. Die Garantie muss deshalb an einer ausdrücklich bezeichneten Grenze liegen:

| Modus | Tatsächliche Grenze |
|---|---|
| Begleiteter Editor | Agent erhält Modellkontext und benutzt Markitect-Befehle; direkte Änderungen werden beim Abgleich sichtbar. Keine Präventionsgarantie |
| Kontrollierter lokaler Lauf | Neue getrennte Agentenprozesse liefern nur gebundene Vorschläge; Markitect besitzt Kandidatenbildung, Dateiprüfung, Tests und Übernahme. Provider wird schreibgeschützt konfiguriert; dies ist keine von Markitect behauptete OS-Isolation |
| Isolierter Lauf | Prozess läuft in einem ausdrücklich gewählten, gepinnten Container/VM ohne Schreibzugriff auf das Quellrepository. Repositorybytes kommen nur als begrenzte Eingabe, Vorschläge über den Ergebniskanal. Ein getrennt berechtigter Host führt Apply aus |
| Verbindliche Annahme | CI bzw. geschützte Zielbranch überprüft den exakten Kandidaten und die zugehörigen Modell-/Prüfbindungen. Wenn starke Herkunft verlangt wird, werden Schlüssel außerhalb der Agentenumgebung gehalten; ein lokaler Digest allein authentifiziert keinen Entscheider |

Der implementierte Modus wird im Laufbericht ausgewiesen. Eine verlangte Isolation ohne verfügbare und geprüfte Ausführungsgrenze blockiert den Start; es gibt keinen stillen Rückfall auf volle Rechte. Authentifizierte Enterprise-Attestierung und Branchschutz werden nicht durch das Setzen eines lokalen YAML-Felds erzeugt. Installationsanweisungen dürfen bestehende Nutzer-/Systemkonfiguration nicht still überschreiben.

`isolated` wird ausschließlich durch einen Host-kontrollierten Launcher mit überprüfter Image-/Policyidentität, nicht schreibbarer Quelle, privatem Ergebniskanal und davon getrennt berechtigtem Applybroker begründet. Providerflags, benutzereigene Hooks oder eine behauptete Sandboxkonfiguration reichen nicht. Die Reichweite wird getrennt angegeben: lokale Repository-Schreibisolation ist keine Garantie gegen externe Schreibzugriffe. Eine weitergehende Garantie verlangt ausdrücklich geprüfte Egressbeschränkung auf den Inferenzbroker und außerhalb des Workers gehaltene Git-/Dienstcredentials. Fehlt dieser Nachweis, wird die stärkere Garantie verweigert. Der erste lokale Produktlauf kann ehrlich `controlled-local` sein; er darf sich nicht als isoliert bezeichnen.

Neue Projektaufrufe verwenden eine explizite erlaubte Prozessumgebung anstelle der vollständigen geerbten Umgebung. Notwendige Provideranmeldung wird gezielt ausgewählt; fremde Git-/Diensttokens, nutzerdefinierte Hook-/Plugin-/MCP- und Konfigurationsvariablen werden nicht ungeprüft weitergegeben. Runner-/CLIbytes, geprüfte Version, Args, effektive Konfigurationsquellen und relevante Umgebungsbindung gehören zum Runtimefingerprint; sensible Werte werden weder protokolliert noch in Projektdateien gespeichert. Nicht überprüfbare Erweiterungen blockieren einen Lauf, der ihre Abwesenheit voraussetzt. Die historische agentexec-Verwendung behält ihre kompatible Semantik; neue Aufrufe benutzen die neue explizite Begrenzung.

Der Host bindet Auftrag, Modell-/Artefaktsnapshot, ausführbare Runnerbytes und Providerkonfiguration. Jeder Manager wird getrennt aufgerufen und erhält seinen relevanten Kontext, öffentliche Abhängigkeiten und nötige Kindberichte. Worker erhalten nur die zu bearbeitenden Artefakte. Prüfprozesse erhalten einen frischen Kontext. Der Host akzeptiert ausschließlich korrekt gebundene strukturierte Resultate. Er begrenzt Pfade, Umfang, Dateigröße, Modi und Schreibzuständigkeit, prüft die Quelle erneut und stellt einen isolierten Kandidaten bereit. Ein Ergebnis kann Arbeit, Rückfrage, Eskalation oder begründetes No-op sein; fehlende Verifikation bleibt sichtbar.

Vor Apply werden erwartete Basis, tatsächlicher Zielbaum und der exakt geprüfte Kandidatenbaum erneut verglichen. Eine nachträgliche manuelle Änderung oder geänderte Prüfung macht den Plan ungültig. Die Übernahme bindet gemeinsam Modell, Zuordnung und Artefakte; Teilfehler werden als solche protokolliert und können nicht zu einem gültigen Abschlussreceipt führen. Ein CI-Abgleich prüft tatsächliche finale Bytes statt nur eine vom Agenten gemeldete SHA.

Codex nutzt eine ausdrücklich gepinnte schreibgeschützte Einmalinvocation; Claude eine geprüfte eingeschränkte Einmalinvocation ohne Edit-/Shell-/MCP-Werkzeuge. Konkrete Flags sind Adapterverträge mit Versionstest, keine Coresemantik. Die bestehende `internal/host/agentexec`-Grenze und der Codexrunner werden wiederverwendet, ohne ihre historischen Protokollfelder als neue öffentliche Produktsprache zu etablieren.

Quellen für diese Grenze: [OpenAI Sandboxing](https://learn.chatgpt.com/docs/sandboxing), [OpenAI AGENTS.md](https://learn.chatgpt.com/docs/agent-configuration/agents-md), [Claude CLI](https://code.claude.com/docs/en/cli-reference), [Claude Permissions](https://code.claude.com/docs/en/permissions). Providerverhalten wird zusätzlich gegen die konkret konfigurierte CLI geprüft. Unbekannte Versionen oder nicht unterstützte Restriktionen werden ausdrücklich gemeldet.

## Brownfield-Distillation

Die Übernahme eines bestehenden Repositories ist kein automatisch akzeptiertes Reverse Engineering. Beobachteter Code, dokumentierte Absicht und künftig gewünschtes Verhalten können auseinanderliegen. Der Ablauf hält sie getrennt:

1. **Discovery:** Projektbesitzer wählt eine feste volle Commitrevision, exakt zu lesende reguläre Dateipfade und begründete Ausschlüsse. Eine optionale Verzeichnisinventur liefert zunächst nur Metadaten; die daraus gewählte konkrete Dateiliste wird vor dem Bloblesen ausdrücklich Teil des Captureauftrags. Vorschau zeigt Auswahl und Umfang; Binärdateien und nicht ausgewählte Inhalte werden nicht als gelesene Fachquelle ausgegeben. Arbeitsbaumänderungen oder ein verschobenes HEAD ersetzen niemals die ausgewählte Revision. Laufende Implementierung und bestehende Agentenkonfiguration bleiben maßgeblich.
2. **Begriffsfindung:** Aus den ausgewählten Bytes entstehen belegte Kandidaten für Begriffe, Zustände, Regeln, Abläufe, Architektur und Arbeitsweise. Lexikalische Treffer sind Fundstellen, keine Bedeutungsdefinitionen. Jede Aussage verweist auf Commit, Pfad, Byte-Digest und gegebenenfalls Zeilen. Ein Agent darf Vorschläge liefern; der Compiler erfindet keine Semantik aus Dateinamen.
3. **Klärung:** Gleichlautende Begriffe, Synonyme, unterschiedliche Kontexte, Code-/Doku-Widersprüche und fehlende Vorgaben werden als Fragen mit Alternativen sichtbar. Hypothese, dokumentierte Aussage, Beobachtung und entschiedene Vorgabe sind unterscheidbar. Ungeklärte Fragen dürfen bewusst vertagt werden, der davon abhängige Scope bleibt unübernommen.
4. **Organisation:** Zusammengehörige Konzepte und Verhalten werden als vertikale Slices vorgeschlagen. Managerverantwortung, öffentliche Verträge, Dateizuordnung, Projektarchitektur, Workflow und benötigte Artefakte entstehen gemeinsam. Eigentum folgt einer angenommenen Entscheidung, nicht Häufigkeit, AI-Konfidenz oder einem Ordnernamen.
5. **Übernahme:** Eine explizite Resolution bindet Auswahl, Vorschlag, Antworten/Vertagungen und Zielstand. Markitect validiert einen überprüfbaren Modellkandidaten und zeigt dessen Dateizuordnung und Lücken. Erst befugte Annahme aktiviert den gewählten Umfang. Der Übernahmevorgang verändert nur Markitect-eigene Modell-/Arbeitsdateien; Code und Produktdokumentation werden nicht nebenbei repariert.
6. **Abgleich:** Bestehende Artefakte werden gegen angenommene Erwartungen geprüft. Konformität, erforderliche Reparatur, zusätzliche Fachentscheidung und fehlende Evidenz bleiben getrennt. Unbekannter Scope wird nicht als erfolgreich migriert gezählt.

Adoption erfolgt inkrementell, beispielsweise zunächst Orders und Inventory. Für jeden übernommenen Scope gibt es genau einen gültigen Modellbesitzer; alte Dokumente bleiben Herkunft bzw. erklären ausdrücklich ihren weitergeltenden Scope. Ein veralteter Import ist keine zweite Wahrheit. Ein Rollback stellt den vorherigen akzeptierten Modell-/Zuordnungsstand wieder her; tatsächliche Codeänderungen werden nur mit ihrem eigenen Kandidaten reversiert. Das alte `prepare`/`copy-me`-Verfahren bleibt als explizite Belegaufnahme kompatibel und wird nicht still als vollständige Migration umbenannt.

Der geschlossene Transportvertrag besteht aus `Discovery` (Commit, Auswahl, Ausschlüsse, Evidenzpfade/-digests), `Distillation` (Discoverydigest, belegte Aussagen, Beobachtungsmethode, Begriffe, Widerspruchspaare, Fragen, Vorschläge und unklarer Scope) und `Resolution` (exakter Report-/Vorschlagsdigest, benannte übernommene Scopes, Antworten/Vertagungen, Zielbasis, Schema-/Buildbindung und vorhandener Entscheidungsauftrag). Statische Codelektüre wird nicht als beobachtetes Laufzeitverhalten ausgegeben. KI-erzeugte Reports binden zusätzlich Runner-/Konfigurationsidentität. `discover` erzeugt die feste Aufnahme; `distill` validiert einen gelieferten Report oder fordert ihn über einen expliziten Runner an; `adopt` nimmt eine Resolution entgegen und zeigt/übernimmt den daraus validierten Modellkandidaten ausschließlich unter `.markitect/model/` und dem zugehörigen Projektmanifest. Ein geänderter Report, Vorschlag, Quell- oder Zielstand invalidiert die Vorschau. Orders angenommen und Inventory vertagt ergibt Teilübernahme mit ausdrücklich offenem Inventory, kein vollständiges Projekt-PASS.

Die Reihenfolge bleibt ausdrücklich: P1 validiert ausschließlich gelieferte gebundene Reports und Resolutionen, ohne selbst einen Provider zu starten. Generierung über einen Runner wird erst mit P2 angebunden und in P4 in die durchgehende Benutzerführung aufgenommen. Ein bereits vorhandener Gesprächsagent darf den gelieferten Report erstellen; dadurch erhält die deterministische P1-Prüfung keinen eigenen Agentenruntime.

## Installation und tägliche Arbeit

Die globale CLI-Installation und der optionale projektlokale Toolpin sind getrennte Vorgänge. Die bestehenden verifizierten Releasewege installieren die veröffentlichte Version; neue Sourcebefehle benötigen den geprüften Kandidatenbuild, bis eine eigene Veröffentlichung beauftragt ist. Bestehendes `init` bleibt kompatibel; der neue Einstieg lautet `markitect init` und erzeugt ausschließlich Markitect-Dateien unter `.markitect/`.

Bis zu einer beauftragten Veröffentlichung wird der Kandidat über `go run ./src/cmd/markitect <verb> ...` aus dem Wurzelverzeichnis der exakt bezeichneten Sourcearbeitskopie oder deren gebaute Binary ausgeführt. Dies aktualisiert weder die globale Installation noch einen Projektpin. Der bestehende `install`-Befehl akzeptiert weiterhin ausschließlich das verifizierte versionierte Releasebundle. Gesprächsbedienung setzt einen bereits verfügbaren unterstützten Agentenhost bzw. eine explizit konfigurierte Runnerinvocation voraus; die deterministische CLI selbst enthält kein Sprachmodell und verlangt keine heimliche Accountkonfiguration.

Für ein neues Projekt beschreibt der Nutzer Ziel, Begriffe und erste Abläufe im Gespräch. Der Agent liest `context`, reicht Modelländerungen als gebundene Vorschläge bei Markitect ein und zeigt eine lesbare fachliche Differenz. `check`, `model`, `impact` und erzeugte Sichten liefern die Fakten. Unverbindliche Ideen bleiben Drafts. Ein bereits erteilter Auftrag zu Modellpflege und Umsetzung braucht keine zweite pauschale Freigabe.

Für ein bestehendes Projekt beginnt derselbe Weg mit `adopt start` und einem Distillations-/Klärungsvorschlag. Planung allein startet keine Implementierung. Erst ein beauftragter `run` delegiert aus dem angenommenen Modell. Markitect erstellt einen Kandidaten, integriert Kinder, führt deklarierte Prüfungen aus und zeigt fehlende Umsetzung bzw. unbelegte Kriterien. Apply übernimmt ausschließlich den exakt gebundenen Kandidaten innerhalb der bestehenden Autorisierung. Ein späterer reparierender Lauf darf dasselbe Soll verwenden.

Der Unterschied zum üblichen Agentic Coding liegt im dauerhaften Ergebnis: Fachentscheidungen, Verantwortung und Realisierungsbezüge überleben den Chat. Der nächste Agent rekonstruiert sie aus dem Modell statt aus der Erinnerung seines Vorgängers. Der Nutzer beurteilt fachliche Änderungen und verbleibende Entscheidungen; Agenten entscheiden gewöhnliche Implementierungsdetails. Dokumentation ist eine lesbare Sicht auf denselben Stand. Dies ist ein zu prüfender Arbeitsweg, noch kein gemessener Produktivitätsvorteil.

## Zusätzliche Abnahme

| Fall | Erwartung |
|---|---|
| Agentenhinweis oder Hook wird umgangen | Begleitmodus behauptet keine Prävention; finale Zuordnungs-/Prüfgrenze erkennt ungedeckte Änderung |
| Angeforderte Isolation ist nicht verfügbar | Start blockiert; kein permissiver Fallback |
| Provider liefert fremde Pfade, veraltete Bindung, falsche Rolle oder übergroße Ausgabe | Kein Kandidat bzw. kein Apply |
| Repository oder Runnerkonfiguration ändern sich während des Laufs | Ergebnis wird veraltet; gezielter neuer Abgleich |
| Discovery findet widersprüchliche Doku und Code | Beide Aussagen und Klärungsfrage bleiben vorhanden; keine automatische Wahl der Absicht |
| Ein Scope wird vertagt | Er bleibt explizit unübernommen; kein behauptetes Gesamtprojekt-PASS |
| Resolution oder Quellbytes ändern sich nach Vorschau | Übernahmeplan wird abgelehnt bzw. neu erzeugt |
| Neues Projekt ohne händisches YAML | Gespräch/Agent reicht Vorschläge über Markitect ein; lesbarer Diff und Dokumentation sind vorhanden |
| Installation/Upgrade auf bereits eingerichteter Maschine | Bestehende Providerkonfiguration und Legacyprojekte bleiben erhalten |

Die vollständige technische Prüfung umfasst mindestens ein neues Projekt, einen begrenzten Brownfieldfall und den Shop-Stornierungsablauf einschließlich negativer Pfad-/Bindungs-/Prüffälle. Tatsächliche Provideraufrufe und OS-Isolation werden getrennt von Protokolltests ausgewiesen.
