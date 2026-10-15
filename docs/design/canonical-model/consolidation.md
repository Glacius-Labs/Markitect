# Gemeinsamer Designstand für Markitect

Stand: 15. Oktober 2026. Auftrag: aktuellen Main und die beiden Modellkonzepte konsolidieren und die offenen Widersprüche gemeinsam entscheiden. Diese Seite ist der Einstieg in das weitere Designgespräch. Empfehlungen sind Vorschläge; eine Übernahme von Dokumenten ist keine Entscheidung über ihre offenen Inhalte.

## Quellen und Zusammenführung

| Quelle | Fester Eingang | Beitrag |
| --- | --- | --- |
| `main` | `6472d42a93c1be7e542d1fa06ad10a308dd15b8a` | Aktuelle Produktgrundlage, Implementierung, Ursprungserklärung und bestehende Verträge. |
| `docs/product-model-concepts` | `3e5583cb67eb931e830211ce63378d4526191a3c` | [Typisierte Prosa, Sprachentwurf und Bibliotheksmodell](../product-model-refinement/README.md); darin dokumentierte Nutzerentscheidungen und offene Syntaxfragen. |
| `docs/canonical-model-vocabulary` | `3e5ccdbe217b66a5332aec8b913116f90cb2091a` | [Vierzehn Arbeitsbegriffe](README.md), [Beziehungen](relationships.md) und [Tasks als Begriffsbeispiel](task-management-example.md). |
| `docs/task-management-model` | `45c99e82a61aa5f1774b093970f2709a632424e9` | Ergänzende [eigenständige Tasks-Projektdokumentation](../task-management/README.md) und ausdrücklich [historischer Modellierungsversuch](../task-management-model/README.md). |

Die beiden eigentlichen Modellkonzepte sind `product-model-concepts` und `canonical-model-vocabulary`. Tasks ist zusätzliches Vergleichsmaterial. Seine eigenständige Dokumentation bleibt als Projektbeschreibung ohne vorausgesetzte Markitect-Taxonomie lesbar.

Die Arbeitsbranch `docs/design-consolidation` beginnt auf dem oben genannten Main. Aus den drei Dokumentationsbranches werden ausschließlich ihre neuen Designordner übernommen. Ihre Änderungen außerhalb dieser Ordner betreffen nur die gemeinsam neu aufgebaute Designnavigation. Der alte Produktcode des Bibliotheksbranches wird nicht übernommen: Sein gemeinsamer Vorfahr liegt 505 Main-Commits zurück. Das neue Vokabular und der Tasks-Branch haben diesen Main bereits als Vorfahren.

Die Quellen bleiben mit ihrer Herkunft erhalten. Diese Seite führt ihre Aussagen zusammen; neue Nutzerentscheidungen werden ausdrücklich datiert in Definitionen und betroffenen Entwürfen nachgeführt. [Der Main-Abgleich](current-main-comparison.md) nennt die heute tatsächlich untersuchten Implementierungsgrenzen. Der [Vergleich vom 11. Oktober](../product-model-refinement/canonical-model-main-comparison.md) bleibt historische Evidenz für seinen damaligen Stand.

## Gemeinsame Grundlage

Markitect beschreibt die beabsichtigte Projektwelt und lässt deren Repositoryrealisierung koordiniert ausführen. Menschen entscheiden Bedeutung und Freiräume; Strukturprüfung, nachvollziehbarer Impact, begrenzte Zuständigkeiten, unabhängige Reviews und konkrete Checks unterstützen die Realisierung. Das ist die Grundlage der [Vision](../../vision.md) und des [Konzeptregisters](../../concepts/register.md).

Beide Designlinien lassen sich darin zusammenführen: Die Begriffe erklären, **welche Art von Aussage** beschrieben wird. Ein Definitionskörper enthält die individuellen Aussagen. Explizite Beziehungen verbinden Definitionen. Daraus sollen Kontext, Realisierungsaufträge und Prüfaufträge entstehen. Wie genau diese Ableitung funktioniert, ist pro Beziehung zu bestimmen. Ein Wort, ein Link, eine Zusammensetzung und eine Realisierungspflicht sind verschiedene Dinge.

Wir führen drei Status getrennt:

- **Main-Vertrag:** Verhalten und Format, das der feste aktuelle Produktstand tatsächlich unterstützt.
- **Im Bibliotheksgespräch vereinbarte Designrichtung:** unter anderem `Mandate`, nominale Fach-Kinds, eigenständige Definitionskörper, explizite Körperlinks und die dort beschriebene `contains`-/`realizes`-Richtung. Die Quelle dokumentiert diese Entscheidungen; diese Konsolidierung implementiert sie nicht.
- **Arbeitsdefinition oder Vorschlag:** die vierzehn Begriffe und ihre Verbindungen, konkrete Felder, zusätzliche Kinds, Syntax und die nachfolgenden Empfehlungen. Neuere Commitzeit allein macht daraus keinen Ersatz für eine frühere Nutzerentscheidung.

## Widersprüche und Entscheidungen

### 1. Component: allgemeiner Systemteil oder Laufzeitbaustein?

**Bisherige Bibliotheksfassung:** Ein `Component` war auf Laufzeit-/Bereitstellungsbausteine begrenzt. Ein `System` enthielt Components; ein Component enthielt logische Modules. Der erste Beispielvertrag erlaubte keine direkte Component-zu-Component-Zusammensetzung. Siehe [Architekturbegriffe im Proposal](../product-model-refinement/canonical-model-proposal.md#4-beziehungen-gliederung-und-verantwortung).

**Neues Vokabular:** [Component](component.md) bedeutet allgemein einen Systemteil. Die [Beziehungen](relationships.md#structure-and-files) erlauben verschachtelte Components. Task Logic, Application, Command Line und Local Storage sind dabei logische Verantwortungsbereiche, keine zugesagten getrennten Laufzeiteinheiten.

**Nutzerentscheidung vom 15. Oktober 2026:** „Allgemeiner Systembaustein, auch logisch und verschachtelbar.“ Die [Component-Definition](component.md) gilt damit auch für Task Logic und für ineinander gegliederte logische Teile. Ein eigener Prozess oder eine eigene Bereitstellung ist keine Voraussetzung. Die frühere enge Bedeutung ist abgelöst.

**Folgen:** Die Bibliotheksbausteine Frontend, Backend und Datenbank bleiben gültige Beispiele für Components. Sie begrenzen den Begriff nicht mehr auf Laufzeitarchitektur. Component-zu-Component-Zusammensetzung wird zulässig; die bereits vereinbarte Ableitung des `realizes`-Umfangs folgt auch diesen `contains`-Nachfahren. Eigenständiger Nutzen und Abgrenzung von System und Module sowie vollständige Parent- und Zielmengenregeln bleiben zu entscheiden. Die beiden Kinds werden nicht allein durch diese Component-Entscheidung entfernt.

### 2. Interface: Actor-Zugang oder technischer Vertrag?

**Bibliothek:** Interfaces verbinden auch interne Modules, etwa über Service- und Repositoryverträge; `provides` und `consumes` unterscheiden Anbieter und Verbraucher. **Neues Vokabular:** Ein [Interface](interface.md) ist der Zugang eines Actors zu UseCases.

**Entscheidung:** Gilt Interface ausschließlich für Außeninteraktion, oder auch für technische Zusammenarbeit innerhalb der Software? Ein interner Aufrufer darf nicht allein deshalb zu einem neuen Actor werden, damit ein bestehender Vertrag in die Definition passt.

**Empfehlung:** Interface als vereinbarten Interaktionsvertrag fassen; Actor-Zugang ist ein konkreter Einsatz. Anbieter, Verbraucher, angebotene Operationen und Fehler müssen im jeweiligen Beispiel nachvollziehbar bleiben. Das ist eine vorgeschlagene Erweiterung der kurzen Arbeitsdefinition.

### 3. Vierzehn Begriffe sind noch keine vollständige Kind-Liste

Das neue Vokabular ergänzt Actor, Scenario, Workflow, Procedure und Condition. Es definiert Mandate, System und Module nicht. Ihr Fehlen widerruft die älteren Entscheidungen nicht. Umgekehrt begründet eine verständliche Begriffsdefinition noch keinen eigenen maschinenlesbaren Kind.

**Entscheidung:** Für jeden Begriff prüfen wir getrennt, ob er eigene Identität und Referenzierbarkeit benötigt, welche individuellen Aussagen er trägt und was sein Vorhandensein für Realisierung oder Prüfung bewirkt. Eine feste Zahl von Kinds ist kein Ziel.

**Besonderer Namenskonflikt:** `Decision.actor` referenziert im Bibliotheksentwurf ein Mandate, während Actor im neuen Vokabular eine beteiligte Rolle beschreibt. Rolle, Auftrag, ausführender Agent und dokumentierte Entscheidungsherkunft sind einzeln zu klären. Keines dieser Felder authentifiziert eine Freigabe.

### 4. Allgemeine Verbindung oder berechenbare Relation?

Das Vokabular erklärt Verbindungen wie „verarbeitet“, „dokumentiert“ und „prüft“ in Alltagssprache. Der Bibliotheksentwurf unterscheidet strukturierte Beziehungen und wählt für `realizes` einen automatischen Umfang über die `contains`-Unterstruktur.

**Entscheidung:** Die alltagssprachlichen Beziehungen werden erst nach ihrer Einzelprüfung auf einen Sprachvertrag abgebildet. Ein Körperlink erklärt eine explizite Verwendung, aber nicht automatisch Enthaltenheit, Regelgeltung, Realisierung oder Schreibzuständigkeit. Es gibt keine pauschale Umsetzung aller Verbindungen als `uses`.

**Erhaltene Designrichtung:** Wo `realizes` verwendet wird, umfasst es nach der dokumentierten Entscheidung Ziel und transitive `contains`-Nachfahren entsprechend der Beitragsrolle. Andere Beziehungen vergrößern diese Zielmenge nicht automatisch. Ändern wir die erlaubten `contains`-Kinder, müssen wir deshalb auch diese Folgewirkung prüfen.

### 5. Check: Prüfauftrag oder ausschließlich ausführbarer Befehl?

Die neue [Check-Definition](check.md) beschreibt Methode und Erfolgskriterium und kann auch einen Dokumentationsvergleich oder ein Review tragen. Main und der konkrete Bibliotheks-Feldkandidat verlangen dagegen einen nicht leeren Befehl als literale Argumentliste.

**Entscheidung:** Soll Check der allgemeine Prüfauftrag sein, dessen Methode ein Befehl oder ein begründetes Review sein kann? Oder bleiben manuelle beziehungsweise agentische Bewertungen ein gesonderter Mechanismus?

**Empfehlung:** Zuerst Erwartung, Methode und Erfolgskriterium fachlich beschreiben. Danach entscheiden, wie die Durchführung gebunden wird. Deklaration, Ausführung und aufgezeichnetes Ergebnis bleiben verschieden. Ein Reviewauftrag darf nicht als bereits ausgeführter Command-Check erscheinen.

### 6. Coverage entfernen: Konfiguration oder Produktpflicht?

Der Bibliotheksentwurf entfernt Coverage und seine Moduskonfiguration aus diesem Konzeptschritt und spezifiziert keine zusätzliche vollständige Repositoryabdeckung. Main und [DEC-006](../../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored) verlangen weiterhin, dass jede Datei zugeordnet oder ausdrücklich ausgenommen ist.

**Entscheidung:** Das Entfernen eines Begriffs oder Schalters beantwortet nicht, ob unbekannte Dateien die Realisierung weiterhin blockieren. Soll diese Pflicht erhalten bleiben, braucht auch die neue Sprache eine nachvollziehbare Dateierfassung und Zuständigkeitsprüfung. Soll sie entfallen, wäre das eine ausdrückliche Änderung der bestehenden Produktentscheidung.

**Empfehlung:** Vollständige Dateirechenschaft erhalten; Modellvereinfachung und Konfigurationsort separat entscheiden. Hier wird weder ein alter Schalter wieder eingeführt noch DEC-006 geändert.

### 7. Rule, Condition, Procedure und Workflow

Hier liegt zunächst eine Präzisierung statt eines belegten Widerspruchs vor: Rule verpflichtet; Condition beschreibt einen Wahrheitswert; Check untersucht eine Erwartung. Procedure erklärt eine Tätigkeit; Workflow koordiniert Tätigkeiten und Beteiligte. Scenario beschreibt einen konkreten Fall. Die [Beziehungsseite](relationships.md) liefert die Arbeitsabgrenzung.

**Offen:** Ob Bedingungen eigene Definitionen brauchen, wie sie ausgewertet werden und welche Workflow-Übergänge maschinell erfolgen. Die Beschreibungen begründen noch keinen Ausdrucksevaluator und keine ausführbare Workflowengine. Auch ein imperativ formulierter Procedure-Text erteilt einem Agenten nicht selbst die Befugnis, die Schritte auszuführen.

### 8. Zwei Tasks-Darstellungen mit unterschiedlicher Genauigkeit

Das [Begriffsbeispiel](task-management-example.md) verwendet Title/Identifier und beispielhafte `src/...`-Pfade. Die [eigenständige Projektdokumentation](../task-management/README.md) verwendet Text/Reference, Python 3.13, `python -m tasks` und einen genaueren Speichervertrag. Ein fehlendes Bestätigungssignal nach dem Commit beweist dort ausdrücklich keinen fehlgeschlagenen Speichervorgang.

**Konkrete Abweichung:** Das Begriffsbeispiel sagt allgemein, ein fehlgeschlagenes Speichern bewahre die bisherigen Daten. Der [präzisere Speichervertrag](../task-management/docs/storage.md#save-procedure-and-commit-point) trennt Fehler vor dem Ersetzen von Fehlern bei der anschließenden Bestätigung. Diese Grenze muss eine spätere gemeinsame Modellierung erhalten.

**Vorgeschlagene Verwendung:** Die eigenständigen Dokumente sind die inhaltliche Gegenprobe für das Tasks-Projekt; das Begriffsbeispiel bleibt ein vereinfachter Entwurf für Begriffe. Wir gleichen ausgewählte Aussagen explizit ab und geben die verschiedenen Pfade und Namen nicht als denselben fertigen Modellstand aus. Der historische sieben-Kind-Versuch bleibt historisch.

## Nächster Designschritt

Der Nutzer hat am 15. Oktober als nächsten Schwerpunkt die Entscheidung der Widersprüche gewählt und **Component** als allgemeinen, verschachtelbaren Baustein entschieden. Als Nächstes prüfen wir die verbleibende System-/Module-Abgrenzung und Interface, danach Verantwortung/Actor und die Prüfbindung. Coverage wird vor einem ausführbaren Sprachvertrag ausdrücklich geschlossen.

Für jede Entscheidung halten wir die gewählte Aussage, ein kleines Tasks-Beispiel, eine Gegenprobe an der Bibliothek, Auswirkungen auf bestehende Entscheidungen und verbleibende Fragen fest. Erst nach der Einigung ändern wir die betroffenen Begriffsdefinitionen und Beispiele. Syntax und Implementierung folgen den geklärten Bedeutungen.

Diese Konsolidierung enthält keine neue Sprachimplementierung und keinen Anwendungs- oder Wirksamkeitsnachweis. Aktuelle Produktverträge bleiben bei [Vision](../../vision.md), [Architektur](../../architecture.md), [Konzeptregister](../../concepts/register.md) und den bestehenden Operationsguides; Umsetzungsstatus bleibt bei [Roadmap](../../implementation-plan.md) und Backlog.
