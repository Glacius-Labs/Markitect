# Bibliothek: Änderungsdurchlauf für überfällige Ausleihen

Datum: 12. Oktober 2026. Status: konzeptioneller Durchlauf zum [aktuellen Modell](library-model-example.md), kein ausgeführter Impact-, Verify- oder Apply-Lauf. Die fachliche Änderung ist ein Gesprächsbeispiel und kein erteilter Implementierungsauftrag.

## Ausgangspunkt und gewünschte Änderung

Bisher fordert `eligible-member` ein existierendes Mitglied mit aktiver Ausleihberechtigung. Der Verhaltensvertrag `borrow` verlangt diese Rule. Der neue Wunsch lautet:

> Mitglieder mit überfälligen aktiven Ausleihen dürfen keine weiteren Exemplare ausleihen.

Die Änderung betrifft den geltenden Vertrag, nicht nur eine Anwendungsdatei. Vor Umsetzung wird ein fester Modell-/Repositorystand als Basis bestimmt; der Entwurf bildet einen eigenen Kandidaten.

## 1. Prosa ändern und Struktur prüfen

Ein erster Entwurf könnte die Rule-Prosa ändern, aber nur die bisherige Member-Kante behalten:

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Rule
metadata:
  namespace: library
  name: eligible-member
purpose: Definiert die Berechtigungsbedingung für das Ausleihen.
spec:
  public: true
  description: |
    Ein Mitglied darf ausleihen, wenn es existiert,
    seine Ausleihberechtigung aktiv ist und es
    keine überfällige aktive Ausleihe besitzt.
  uses:
    - kind: Concept
      namespace: library
      name: member
```

Ein Strukturvalidator kann Identität, Feldvertrag und Zieltyp prüfen. Er kann die Bedeutung „überfällig“ nicht aus Prosa formal auswerten oder allein daraus beweisen, dass die Loan-Abhängigkeit fehlt. Dass weitere Ausleihinformation im konservativen Kontext erscheint, ersetzt diese deklarierte fachliche Beziehung nicht.

## 2. Modellreview ergänzt Bedeutung und Beziehungen

Der Review erkennt den neuen Begriff und fragt nach Zeitquelle, Zeitgrenze, Zeitzone und dem Verhältnis zur Rückgabe. Die folgende Definition ist ein Kandidat, keine bereits angenommene vollständige Zeitregel:

### Neue Datei `model/library/overdue-loan.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Concept
metadata:
  namespace: library
  name: overdue-loan
purpose: Definiert den Begriff einer überfälligen Ausleihe.
spec:
  public: true
  description: |
    Eine Ausleihe ist überfällig, wenn sie aktiv ist
    und der maßgebliche Zeitpunkt nach ihrer Fälligkeit liegt.
    Zeitquelle, Zeitzone und genaue Grenzbehandlung sind
    vor Umsetzung ausdrücklich festzulegen.
  uses:
    - kind: Concept
      namespace: library
      name: loan
    - kind: Concept
      namespace: library
      name: active-loan
```

Die Rule `eligible-member` erhält zusätzlich:

```yaml
uses:
  - kind: Concept
    namespace: library
    name: member
  - kind: Concept
    namespace: library
    name: overdue-loan
```

Der unveränderte Vertrag `borrow` verlangt weiterhin `eligible-member`. Er kopiert die neue Regel nicht in seine eigene Prosa.

```text
borrow
  requires → eligible-member
                 uses → member
                 uses → overdue-loan
                            uses → loan
                            uses → active-loan
```

Die neue Definition liegt im festgelegten Modellordner und wird bei dessen nächster vollständiger Erfassung automatisch geladen; eine zusätzliche Dateiliste ist nicht zu ergänzen. Ihre Realisierung wird dem Implementierungsartefakt zugeordnet; Prüfbeiträge und Check-Scope sind bewusst zu bestätigen beziehungsweise zu ergänzen. Der neue benannte Begriff allein erzeugt noch keinen Nachweis.

Nach Modellergänzung und Klärung reservierter Entscheidungen wird erneut auf einem festen Modellstand verglichen. Ein vor Ergänzung berechneter Plan wird nicht still als Plan für den neuen Digest verwendet.

## 3. Welche Aussagen sind relevant?

Einordnung nach der Architekturergänzung vom 13. Oktober: Die folgende Tabelle erläutert die fachliche Wirkung. Das [erweiterte Bibliotheksmodell](library-model-example.md) und sein [Architektur-YAML](library-architecture-example.md) ergänzen Provider/Consumer, Modulzuordnung und getrennte Repositorybeiträge. Ein vollständiger neuer Durchlauf muss auch diese Kanten berücksichtigen; die Tabelle ist keine berechnete vollständige Impactliste.

| Aussage | Grund ihrer Relevanz |
| --- | --- |
| `eligible-member` | Direkt geänderte Rule. |
| `overdue-loan` | Neue Definition, die zur neuen Bedingung gehört. |
| `borrow` | Verlangt die geänderte Berechtigungsregel. |
| `member` | Liefert Mitgliedsidentität und Berechtigungsstatus. |
| `loan` | Liefert Mitgliedsbezug und Fälligkeit. |
| `active-loan` | Bestimmt, welche Ausleihen berücksichtigt werden. |

Dies ist die fachliche Erklärung der Änderung, keine vollständige berechnete Reportliste. Der bisher geprüfte Routingstand bildet eine konservative Hülle über `uses`/`requires` in beide Richtungen. Über Loan/Copy und weitere Verbraucher können auch Verfügbarkeit, Eindeutigkeit, Titel und Rückgabe in die Untersuchung gelangen. Schon der alte Bibliotheksgraph verbindet damit fast alle fachlichen Definitionen; der neue Begriff erweitert ihn.

Die alleinstehende Systemübersicht wird nicht aufgrund ihres Namens automatisch zum globalen Impact-Ziel. Explizite Geltungs-/Zielbeziehungen und Review können zusätzliche notwendige Wirkungen ergeben.

Die Vereinigung der Kanten aus Alt- und Neusicht wird in beiden Richtungen untersucht. Dadurch bleiben mindestens die in jedem Einzelstand betroffenen Erreichbarkeiten erhalten; kombinierte alte/neue Kanten können weitere Pfade bilden. Entfernte Kanten, Pfade oder Owners verlieren ihre bisherige Wirkung nicht einfach aus der Prüfung. Unknown führt zu konservativer Erweiterung oder ausdrücklich ungelöstem Umfang. Explain-Klassen `change`/`context` sind Priorisierung und Erklärung; sie geben keine generelle Erlaubnis zum Weglassen.

Die Regeln für die neuen nominalen Kinds Concept, Rule und UseCase müssen in der späteren Implementierung ausdrücklich dieselbe Behandlung erhalten. Das heute geplante vNext-Modell wurde nicht mit dem alten Code ausgeführt.

## 4. Von Aussagen zu Dateien und Checks

| Deklaration im Beispiel | Erwarteter Untersuchungsbereich |
| --- | --- |
| `backend-implementation` trägt Berechtigung, Ausleihe und technische Adapter. | `backend/` |
| `frontend-implementation` trägt Bedienung und Ablehnungsdarstellung. | `frontend/` |
| `database-configuration` trägt Datenhaltung und nötige Schema-/Migrationsbeiträge. | `deployment/database/`, sofern die tatsächliche Änderung diese Beiträge betrifft. |
| `library-verification` liefert Fach-/Vertrags-/Architekturprüfbeiträge. | `tests/` |
| `library-documentation` erklärt System und Ausleihvertrag. | `docs/library/`, Projektansicht und README. |
| Fach-/Architektur-Checks untersuchen die benannten Verpflichtungen. | Deklarierte Prüfverfahren; derzeit nur Platzhalter. |

Ein tatsächliches Dateiinventar liefert die konkreten Dateien innerhalb dieser Bereiche. Die Zuordnung ist breit und garantiert keine Auswahl einzelner Funktionen. Der Agent untersucht die Inhalte, bevor er entscheidet, welche Kandidatendateien anzupassen sind. Betroffen heißt nicht automatisch zu verändern.

Der Scope des vorhandenen Checks macht ihn relevant; er beweist nicht, dass neue Überfälligkeitsfälle schon enthalten sind. Der Review muss neue oder veränderte Prüffälle identifizieren und den konkreten ausführbaren Anschluss bestätigen. Konkrete Dateien und Ergebnisse werden an den tatsächlichen Stand gebunden, nicht aus diesem Gespräch erfunden.

## 5. Was bleibt für den methodischen Review?

### Vor Umsetzung zu entscheidende Fachfragen

- Gilt „überfällig“ exakt bei Fälligkeit oder erst danach?
- Welche Zeitquelle und Zeitzone gelten?
- Reicht eine einzelne überfällige aktive Ausleihe zur Sperre?
- Hebt deren Rückgabe die Sperre sofort auf, soweit keine weitere überfällige Ausleihe besteht?
- Zu welchem Zeitpunkt wird die Berechtigung während einer Vergabe beurteilt?
- Sind bestehende Ausleihen und Datenhaltung von der geänderten Auslegung betroffen?

### Unabhängiger Realisierungsreview

- Werden nur Ausleihen des richtigen Mitglieds untersucht?
- Werden zurückgegebene Ausleihen aus der aktiven Menge ausgeschlossen?
- Prüfen alle relevanten Vergabeeinstiege die neue Regel?
- Bleibt eine abgelehnte Vergabe ohne zusätzliche Ausleihe?
- Bleibt Rückgabe für gesperrte Mitglieder möglich?
- Wird der neue Ablehnungsgrund verständlich dokumentiert?
- Decken konkrete Checks die behaupteten Fälle ab, und welche Aussagen bleiben unbelegt?

### Vorgeschlagene Prüffälle

| Fall | Erwartung nach der noch zu bestätigenden Fachregel |
| --- | --- |
| Aktives Mitglied ohne überfällige aktive Ausleihe. | Diese neue Bedingung verhindert die Vergabe nicht; übrige Regeln gelten weiter. |
| Mindestens eine überfällige aktive Ausleihe. | Neue Vergabe wird abgelehnt. |
| Verspätet, aber bereits zurückgegebene Ausleihe. | Diese Ausleihe allein sperrt nicht. |
| Mehrere Ausleihen, davon eine überfällig und aktiv. | Mitglied wird nach der gewählten „eine genügt“-Regel gesperrt. |
| Genau am Fälligkeitszeitpunkt. | Erwartung folgt der ausdrücklich gewählten Zeitgrenze. |
| Rückgabe durch gesperrtes Mitglied. | Rückgabe bleibt möglich; Sperrwirkung wird anhand der verbleibenden Ausleihen beurteilt. |

## 6. Ergebnisse und Grenzen

Die vorgesehene Struktur macht Änderungsgegenstand, explizite Beziehungen, Verantwortung, Realisierungsbereiche und Check-Auswahl nachvollziehbar. Der methodische Ablauf vervollständigt Bedeutung, entdeckt fehlende Beziehungen und beurteilt Umsetzung sowie geeignete Evidenz.

Strukturell gültig, ein tatsächlich bestandener Check, unabhängiger Review und menschliche Zustimmung sind getrennte Befunde. Auch eine breite Impact-Hülle beweist keine vollständige Modellierung.

Quellenbasis der historischen Routingbeschreibung: [Code an Main-SHA 32afd1…](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/impact.go). Dieser Sourcebefund wurde im Gespräch gelesen; heute wurde weder aktuelles Main erneut abgerufen noch dieses Beispiel ausgeführt. Eingang der Dokumentationskonsolidierung: `07e03215c968302361f5acedc8e69bd573ac95be`; Kind-Aktualisierung am 12. Oktober 2026 ausgehend von `5d18e8f4438cc1d24a10882f04b10501d65db20e`, Branch `docs/product-model-concepts`, Worktree `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
