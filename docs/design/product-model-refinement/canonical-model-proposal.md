# Proposal: Aufbau des kanonischen Modells

Datum: 11. Oktober 2026. Status: zur Diskussion; beauftragt, aber nicht angenommen, implementiert oder eingeplant.

## 1. Auftrag und Maßstab

Die [Leitfrage](system-description-notes.md#leitfrage) lautet: **Wie beschreibt man ein Repository für ein Softwaresystem?** Die Beschreibung soll ausreichen, daraus ein passendes Repository aufzubauen, ohne jede Datei einzeln vorzugeben.

Der [abgeleitete Arbeitsstand](system-description-notes.md#abgeleiteter-arbeitsstand) liefert den Maßstab: Ein Umsetzer soll keine wesentlichen fachlichen oder projektrelevanten Entscheidungen unbemerkt selbst erfinden müssen. Er muss erkennen, was festgelegt ist, was er entscheiden darf und woran eine passende Realisierung erkannt wird. Vollständigkeit ist auf einen erklärten Umfang und Detaillierungsgrad bezogen; sie wird durch diesen Maßstab nicht automatisch bewiesen.

Dieser Proposal plant die Beschreibungssprache und ihre Verwendung. Der [separate Main-Abgleich](canonical-model-main-comparison.md) beschreibt dokumentierte Verträge, Code und gelesene Testquellen an einem festen Stand. Der ältere [Sprachentwurf](canonical-model-draft.md) und die [C#-Skizze](csharp-model-design.md) sind Vorarbeiten, keine übernommenen Formatverträge.

## 2. Empfohlener Aufbau

**Empfehlung:** Ein kleiner struktureller Kern, ein versioniertes Vokabular für die Beschreibung von Softwareprojekten und mehrere abgeleitete Sichten auf dieselben Definitionen.

| Bestandteil | Zuständigkeit |
| --- | --- |
| Struktureller Kern | Definitionen, Identitäten, Schemas, Werttypen, Kardinalitäten und typisierte Referenzen; deterministische Normalisierung und Diagnostik. |
| Beschreibendes Projektvokabular | Bedeutung von Systemen, Fachbegriffen, Verhalten, Regeln, Architektur, Betrieb, Verantwortung und Realisierung; endliche semantische Strukturregeln für diese Begriffe. |
| Projektdefinitionen | Das konkrete Wissen und die geltenden Festlegungen der Bibliothek oder eines anderen Projekts. |
| Abgeleitete Sichten | Systemübersicht, Architektur, Verantwortungsstruktur, Artefakterwartungen, konkrete Dateizuordnung und Änderungsfolgen. |
| Ausführung und Evidenz | Agentenbindung, Kandidaten, Prüfungen, Berichte und Apply; an feste Modell- und Repositorystände gebunden. |

Das Vokabular soll wiederverwendbar sein, ohne jede mögliche Softwarearchitektur vorzuschreiben. Die sechs bisherigen Beschreibungsaspekte bleiben Fragen zur Abdeckung und verständliche Lesesichten. Sie werden nicht automatisch sechs Pflichttypen oder sechs Ordner.

`SoftwareSystem`, `Repository` und `SoftwareProject` aus der C#-Skizze sind dabei zusammengesetzte Lesesichten. Sie müssen nicht jeweils ein großes kanonisches Wurzelobjekt werden, das sämtliche Inhalte nochmals enthält.

### 2.1 Eine maßgebliche Quelle pro Aussage

Eine Definition trägt ihre geltende Aussage genau einmal. Andere Definitionen referenzieren sie. Generierte Dokumentation, Managerkontext und Realisierungsindex übernehmen diese Bedeutung als Sichten; sie besitzen keine eigene konkurrierende Spezifikation.

Lesbare Bedeutung und maschinenrelevante Struktur ergänzen sich. Begründungen, Annahmen und ausdrücklich freie Entscheidungen werden von Verpflichtungen unterscheidbar. Ein nicht beschriebener Punkt ist nicht automatisch freigegeben.

## 3. Was beschrieben werden können muss

Die folgende Tabelle definiert **inhaltliche Rollen**, zunächst keine endgültigen Kind-Namen.

| Rolle | Benötigte Inhalte und Beziehungen | Bibliotheksbeispiel |
| --- | --- | --- |
| System und Umfang | Zweck, Problem, Nutzen, Akteure, Grenzen, externe Beteiligte, Annahmen und Ausschlüsse. | Bücher zeitweilig nutzen; Ausleihe und Rückgabe; physische Beschaffung außerhalb. |
| Ziel und Qualität | Angestrebtes Ergebnis, Priorität bei Bedarf, messbares Kriterium oder begründete Bewertung, geltender Umfang. | Verfügbarkeit nachvollziehen; eindeutige Vergabe trotz Konkurrenz. |
| Fachbegriff / Datentyp | Bedeutung, Identität von Instanzen, Eigenschaften, Wertebereiche, Einheiten, zulässige Zustände. | Titel, Exemplar, Mitglied, Ausleihe; Frist und Rückgabezeitpunkt. |
| Fachbeziehung | Rollen der Endpunkte, Richtung, Kardinalität und gegebenenfalls inverse Bindung. | Ein Exemplar gehört zu einem Titel; Ausleihe verbindet Mitglied und Exemplar. |
| Fähigkeit und Verhaltensvertrag | Beteiligte, Eingaben, Auslöser, Voraussetzungen, Erfolg, Zustandswirkungen, Fehler und erhaltene Regeln. | Ausleihen, Rückgabe erfassen, Ablehnung ohne zusätzliche Ausleihe. |
| Regel / Invariante | Aussage, Gegenstand, Geltungsbereich, Bedingungen, Begründung, ausdrückliche Ausnahmen und Nachweispflicht. | Höchstens eine aktive Ausleihe pro Exemplar. |
| Architekturbaustein und Schnittstelle | Zweck, Inhalt, Teilbausteine, angebotene/benötigte Verträge, Grenzen und erlaubte Abhängigkeiten. | Bestand, Mitgliedschaft, Ausleihe; Ausleihe benötigt Berechtigungsinformation. |
| Betrieb | Ausführbare Einheiten, Umgebung, Datenhaltung, Konfiguration, externe Ressourcen und Betriebsverfahren. | Persistenter Speicher, positive Ausleihdauer, Sicherung und Wiederherstellung. |
| Verantwortung und Verfahren | Bereich, Mandat, Delegation, reservierte Entscheidungen, Reviews und Arbeitsabläufe. | Verantwortlicher Bereich für Ausleihregeln und deren Realisierung. |
| Realisierungsverpflichtung | Beitrag zur Definition, Artefaktrolle, Erforderlichkeit, technische Vorgaben, Zuordnung und Schreibzuständigkeit. | Code implementiert Ausleihe; Tests prüfen Eindeutigkeit; Dokumentation erklärt Ablehnungen. |
| Prüfverpflichtung | Zu prüfende Aussage, Verfahren, erwartete Evidenz und begrenzter Umfang. | Zwei konkurrierende Anfragen erzeugen genau eine aktive Ausleihe. |

Eine Rolle braucht erst dann eine eigene adressierbare Definition, wenn sie unabhängig referenziert, geändert, betroffen oder geprüft werden soll. Zusammengehörige Details dürfen innerhalb einer Definition bleiben. Ein einfaches Projekt muss kein Zustandsmodell, mehrere Dienste oder eine umfangreiche Organisation erfinden.

Externe Verträge, fremde Vorgaben und Betriebsbeobachtungen behalten ihre Quelle, ihren Stand und gegebenenfalls eine Aktualitätsverantwortung. Das Modell beschreibt die gewählte Bindung an diese Eingaben und die daraus geltenden Projektpflichten. Ein beobachteter Wert oder ein fremdes Dokument wird dadurch nicht automatisch zur kanonischen Projektabsicht; eine geänderte externe Grundlage kann eine erneute Bewertung auslösen.

Kurzes Begriffsgerüst: Eine **Definition** ist eine benannte Modellbeschreibung. Eine **Fachinstanz** ist beispielsweise das konkrete ausgeliehene Buch in der Anwendung. Eine **Facette** ist ein typisierter Teil der Beschreibung einer Definition. Der **Realisierungsindex** ist die abgeleitete Zuordnung konkreter Artefakte an einem festen Stand.

### 3.1 Fachtypen und Instanzen

Die Definition `Exemplar` beschreibt eine Art fachlicher Gegenstände. Sie ist nicht selbst ein ausgeliehenes Buch. Das kanonische Projektmodell enthält gewöhnlich keine Liste aller aktuellen Bücher und Ausleihen; diese gehören zur Anwendung und gegebenenfalls zu beobachteten Betriebsdaten.

Deshalb sind zwei Kardinalitäten verschieden: Eine Core-Eigenschaft kann genau einen Verweis auf die Modelldefinition `Titel` verlangen. Die Fachbeschreibung kann zusätzlich verlangen, dass jedes Exemplar in der Anwendung genau einen Titel besitzt. „Höchstens eine aktive Ausleihe“ ist wiederum eine gefilterte Regel über Fachinstanzen und Zustände. Der Compiler des Definitionsgraphen kann diese Laufzeitregel nicht allein durch Zählen seiner eigenen Kanten beweisen.

### 3.2 Entscheidungen und Freiräume

Verbindliche Technologie, Schnittstellen oder Algorithmen dürfen beschrieben werden, wenn sie zur Projektdefinition gehören. Andere Umsetzungsentscheidungen bleiben ausdrücklich beim zuständigen Bereich. Beispielsweise kann atomare Vergabe verbindlich sein, während ihre Datenbanktechnik frei bleibt.

Entscheidungen referenzieren den betroffenen Gegenstand und bewahren ihre Begründung. Eine aufgezeichnete Entscheidung ist keine automatische Bestätigung ihrer Autorisierung. Historie, Work Items und geplante Ideen bleiben vom geltenden Systemstand unterscheidbar; Git und Work Items gehören weiterhin nicht zum Gegenstand dieser Spezifikation.

## 4. Beziehungen, Gliederung und Management

Jede maschinenrelevante Beziehung braucht eine definierte Bedeutung, Richtung, zulässige Endpunkte und Wirkung auf Kontext beziehungsweise Impact.

| Beziehung | Bedeutung | Wirkung / Grenze |
| --- | --- | --- |
| enthält | Fachliche oder technische Zusammensetzung. | Im gewählten Baum eindeutiger Parent und keine Enthaltenseinszyklen. |
| verwendet | Benötigt Kontext oder Information. | Relevanz für Bewertung; keine automatische Schreibzuständigkeit. |
| benötigt / muss erhalten | Verbindliche Voraussetzung oder Verpflichtung. | Verpflichtung muss im zuständigen Realisierungs- und Prüfkontext erscheinen. |
| bietet / verlangt | Schnittstellen- oder Verhaltensvertrag. | Vertrag und kompatible Bereitstellung sind ausdrücklich prüfbar. |
| betrifft | Expliziter Geltungsbereich einer Regel. | Reichweite folgt der deklarierten Auswahl, nicht einer beliebigen Graphtransitivität. |
| realisiert | Erwarteter Beitrag eines Artefakts zu einer Definition. | Beitrag als implementiert, geprüft, erklärt, konfiguriert oder bereitgestellt unterscheiden. |
| prüft | Benanntes Verfahren untersucht eine Aussage. | Keine Zusage über ungeprüfte Aussagen. |
| verantwortet / delegiert | Organisatorisches Mandat. | Hierarchie und Befugnis; nicht gleichbedeutend mit fachlicher Abhängigkeit. |

Namespaceraum, Bausteinbaum und Managerbaum sind verwandte, aber unterscheidbare Strukturen. Eine ausdrücklich gewählte Projektkonvention kann die Namespacegliederung auf Managerzuständigkeit abbilden. Der aktuelle Main verwendet eine solche strukturelle Zuordnung; Ordner- oder Modulzugehörigkeit allein begründet im vorgeschlagenen Vertrag keine Befugnis. Sobald ein Projekt davon abweichen soll, braucht es eine ausdrückliche Abbildung; diese Freiheit wird nicht als schon vorhandene Fähigkeit behauptet. Ein übergreifendes Mandat kann mehrere Bereiche betreffen, ohne deren Elternschaft zu ändern.

Eine Änderung an einem Bereich kann andere Bereiche betreffen, ohne dass sich ihre Elternschaft ändert. Übergreifende Regeln erhalten explizite Ziele. Zyklen werden pro Beziehungstyp beurteilt; ein allgemeines Verbot sämtlicher Graphzyklen wäre zu grob.

Die Identität muss für einen Stand eindeutig und referenzierbar sein. Ein bloßer Wechsel des Quellpfads soll keine fachliche Bedeutungsänderung bewirken. Ein Namespace- oder Namenswechsel kann hingegen die vollständige Identität ändern und braucht dann einen ausdrücklichen Umgang mit Referenzen und Evidenz. Ein universeller UUID-Zwang wird nicht vorgeschlagen.

## 5. Realisierung ohne Aufzählung jeder Datei

Das Modell beschreibt **was** realisiert werden muss und welche ausgewählten technischen Bedingungen gelten. Ablagekonventionen können exakte Pfade oder Verzeichnisbereiche verwenden. Es soll nicht für jede private Hilfsdatei eine neue fachliche Definition verlangen.

Für den Aufbau eines Repositorys sind zusätzlich folgende Artefaktfamilien auf Anwendbarkeit zu prüfen. Keine davon ist allein wegen ihrer Nennung für jedes Projekt Pflicht:

| Artefaktfamilie | Zu beschreibende Entscheidung oder Pflicht |
| --- | --- |
| Quellcode und Entwicklungsumgebung | Sprache, Toolchain, Plattform und technische Konventionen, soweit verbindlich. |
| Externe Abhängigkeiten | Benötigte Fähigkeiten, Bibliotheken/Services, Versionierung und Verfahren zur Abhängigkeitsverwaltung. |
| Datenstrukturen und Migrationen | Speicherung, Schemata, zulässige Datenänderungen und Erhaltungs-/Wiederherstellungsbedingungen. |
| Tests und Analysewerkzeuge | Prüfgegenstände, Verfahren, Referenzfälle und Grenzen der Evidenz. |
| Dokumentation und generierte Sichten | Informationspflichten, Leser, Inhalte und maßgebliche Quellen. |
| Build, Paketierung und CI/CD | Wie aus Quellen ein verwendbares Ergebnis entsteht und welche Prüf-/Bereitstellungsverfahren gelten. |
| Infrastruktur und Konfiguration | Umgebungen, Ressourcen, zulässige Parameter und Umgang mit projektrelevanten externen Inputs. |
| Beobachtung und Dashboards | Relevante Messgrößen, Logs, Betriebszustände und deren Darstellung. |
| Skills, Workflows und Richtlinien | Wiederholbare Arbeit, Befugnisse, Reviews und Eskalationsverfahren. |

Damit kann etwa die Wahl „Webanwendung“ präzisiert werden: Welche Schnittstelle, Laufzeitbedingungen und Bau-/Bereitstellungsform sind verbindlich, und welche technische Auswahl darf der zuständige Bereich treffen?

| Kanonische Erklärung | Abgeleitete, standgebundene Beobachtung |
| --- | --- |
| Ausleihvertrag muss in der Anwendung realisiert werden. | Welche konkreten Quelldateien diesen Vertrag realisieren. |
| Seine Invarianten brauchen geeignete Prüfungen. | Welche Tests oder Analyseartefakte diesen Prüfauftrag erfüllen und welche Ergebnisse vorliegen. |
| Die Bedienung und Fehlerfälle müssen erklärt werden. | Welche Dokumentationsdateien diese Pflicht realisieren. |
| Ein Bereich trägt Schreibverantwortung für die gewählten Artefakte. | Welcher Verantwortliche nach der Zuordnung für eine konkrete Datei zuständig ist. |

Der daraus gebildete Realisierungsindex ist erneuerbar. Viele Definitionen dürfen dieselbe Datei betreffen; eine Definition darf viele Dateien benötigen. Eine konkrete Änderungsausgabe besitzt trotzdem eine eindeutige Schreib- und Integrationszuständigkeit.

Ein vollständig erfasster Pfadbestand wird mit diesen Erwartungen abgeglichen. Jede Datei im erklärten Umfang ist zugeordnet, als Modelldatei/Werkzeugartefakt klassifiziert oder ausdrücklich ausgenommen. Eine Übergangsausnahme bleibt als nicht konform sichtbar. Zuordnung und Inhaltskonformität werden getrennt beurteilt.

Die erste Ausprägung darf deklarierte Pfadbereiche verwenden. Eine spätere semantische Zuordnung durch Analysewerkzeuge kann ergänzen oder konservativ erweitern. Unsichere Zuordnung wird als unbekannt oder mehrdeutig gemeldet. Dateinamen, Ordner oder Prosalinks allein sind keine Implementierungsevidenz.

„Ohne jede Datei zu nennen“ bedeutet somit: **Erwartungen und Regeln pflegen, konkrete Dateien im Index erfassen.** Es bedeutet nicht, die konkrete Dateiabdeckung aufzugeben.

## 6. Autorenformat und Sprachform

### 6.1 Verständliche Quellen

Maschinenrelevante Daten werden ausdrücklich strukturiert. Lesbare Beschreibungen können zunächst direkt als mehrzeilige YAML-Texte im Modell stehen; daraus entstehen Dokumentationssichten. Falls Markdown als kanonischer Inhalt angebunden wird, braucht es einen eindeutigen Eigentümer, feste Eingabe und Digest. Dieselbe geltende Aussage wird nicht zusätzlich manuell in beiden Formaten gepflegt.

Ordner und Namespaces bündeln kohäsive Inhalte. Beispiel einer möglichen Gliederung, kein vorgeschriebenes Projektlayout:

```text
.markitect/model/
  library/
    overview.yaml
    inventory/
    membership/
    circulation/
      concepts.yaml
      contracts.yaml
      rules.yaml
      realization.yaml
  engineering/
    responsibilities.yaml
    checks.yaml
    workflows.yaml
  operations/
    environment.yaml
    configuration.yaml
```

Die Gliederung folgt Aufgaben und Gegenständen. Sie muss nicht alle sechs Betrachtungsperspektiven in getrennte Teilbäume aufteilen.

### 6.2 Empfohlene erste Sprachform: strukturierte Facetten

Für die erste Konkretisierung empfehle ich **benannte Aussagen mit versionierten, typisierten Beschreibungsfacetten**. Eine Aussage über einen Fachbegriff erhält beispielsweise strukturierte Eigenschaften und Beziehungen; eine über einen Anwendungsfall erhält einen Verhaltensvertrag. Lesbare Bedeutung und strukturierte Festlegungen gehören zu derselben kanonischen Definition.

Diese Wahl vermeidet, schon aus jeder Beschreibungskategorie einen neuen universellen Core-Kind zu machen. Sie lässt einfache Aussagen weiterhin einfach sein. Die Facetten sind verbindlich validierte Teile eines Vokabulars, keine beliebigen unkontrollierten Zusatzdaten.

Erste Facettenverträge zur Ausarbeitung:

| Facette | Strukturierte Inhalte |
| --- | --- |
| Systemkontext | Grenze, Akteure, Problem-/Zielbezüge, externe Gegenstände. |
| Fachstruktur | Instanzidentität, Eigenschaften, Datentypen, Wertebereiche, Zustände und Fachbeziehungen. |
| Verhalten | Ein-/Ausgaben, Vor-/Nachbedingungen, erhaltene Invarianten, Fehler, Zustandswirkungen, freie Entscheidungen. |
| Regel | Gegenstände, Geltungsbereich, Kategorie, Nachweispflicht, gegebenenfalls begrenzter Bedingungsausdruck. |
| Architektur | Teilbausteine, angebotene/benötigte Verträge und getrennte Abhängigkeitsarten. |
| Betrieb / Verfahren | Umgebungen, Ressourcen, Konfiguration, Schritte, Zuständigkeit und Randbedingungen. |
| Realisierung / Prüfung | Semantischer Beitrag, Erforderlichkeit, Vorgaben, Zuordnung, Nachweisverfahren und Grenzen. |

Ein Vokabularvalidator muss etwa prüfen, dass ein Eigenschaftsziel einen Fachbegriff bezeichnet und ein Vorbedingungsverweis auf eine passende Regel zeigt. Der generische Core prüft den strukturellen Typ; der Vokabularvalidator prüft die darüber hinausgehende Rollenbedeutung.

**Noch zu entscheiden:** Ob Facetten als strukturierte Eigenschaften des bestehenden `Statement`-Kinds oder als eigene nominale Kinds serialisiert werden. Als ersten Kandidaten empfehle ich die Erweiterung des Statement-Vertrags mit klar validierten Facetten. Eigenständige Kinds sind dort sinnvoll, wo eigene Identität, Referenzsignatur oder Lebensdauer erforderlich wird. Die Entscheidung wird an den konkreten Beispielen getroffen; keine aktuelle YAML-Version wird still umdefiniert.

Die Wahl ist an konkreten Fällen zu prüfen:

| Beispiel | Facetten-/Definitionskandidat | Anlass, einen eigenen Kind zu erwägen |
| --- | --- | --- |
| Private Fristberechnung innerhalb von Ausleihen | Beschriebener Verhaltensdetail im Ausleihvertrag. | Erst bei unabhängigem Vertrag oder eigener Referenzsignatur; nicht allein wegen eines weiteren Satzes. |
| Öffentlicher Ausleihvertrag | Eigene Statement-Definition mit Verhaltensfacette und eigener Identität. | Wenn Referenzen nominal ausschließlich Verhaltensverträge treffen müssen oder ein eigener geschlossener Feldvertrag nötig ist. |
| Architekturbaustein Ausleihe | Eigene Statement-Definition mit Architekturfacette. | Wenn Kompositions-/Portverträge einen nominalen Komponenten-Kind sinnvoll erfordern. |

Eigene Definitionsidentität und eigener Kind sind verschiedene Entscheidungen. Auch eine Statement-Definition mit Facette ist unabhängig referenzierbar; eine neue Identität allein erzwingt keinen neuen Kind.

Der C#-Katalog dient dabei als Lesemodell. Eine zusätzliche Kopie sämtlicher kanonischer Inhalte als separat gepflegte Objektstruktur ist nicht vorgesehen.

### 6.3 Begrenzte Formalisierung

Die Beschreibungssprache braucht gut definierte Werte, Kombination, Namen und Abstraktion. Turing-Vollständigkeit ist kein Ziel. Für wiederkehrende Regeln kommen endliche Ausdrücke wie Pflichtwert, Wertebereich, Eindeutigkeit, Vergleich oder gefilterte Kardinalität in Betracht.

Ein Ausdruck wie „Anzahl aktiver Ausleihen je Exemplar höchstens eins“ braucht definierte Mengen-, Zustands- und Zeitsemantik. Seine syntaktische Prüfung kann eine formale Modellprüfung leisten. Ob die Anwendung ihn erhält, verlangt einen geeigneten Nachweis am realisierten Stand. Der Vorschlag behauptet keinen vorhandenen Ausdrucksevaluator.

## 7. Beispiel: Ausleihen vollständig genug beschreiben

| Aussage | Strukturierter Beitrag |
| --- | --- |
| Ein Exemplar ist ein identifizierbares physisches Buch eines Titels. | Fachbegriff, Instanzidentität, Titelbeziehung. |
| Eine Ausleihe verbindet Mitglied, Exemplar, Beginn, Frist und optionalen Rückgabezeitpunkt. | Fachstruktur und Datentypen. |
| Aktiv bedeutet: Rückgabezeitpunkt fehlt. | Benannter Zustandsbegriff mit definierter Bedingung. |
| Ein Exemplar hat höchstens eine aktive Ausleihe. | Invariante über Fachinstanzen. |
| Nur ein berechtigtes Mitglied darf ein verfügbares Exemplar ausleihen. | Akteur, Vorbedingungen, Berechtigungsvertrag. |
| Erfolg erzeugt eine gespeicherte aktive Ausleihe; Ablehnung erzeugt keine zusätzliche. | Erfolgs-/Fehlervertrag und Zustandswirkung. |
| Zwei gleichzeitige zulässige Anfragen ergeben einen Erfolg und eine Ablehnung. | Konkurrenzbedingung und Prüfverpflichtung. |
| Die Vergabetechnik darf der zuständige Bereich wählen. | Umsetzungsspielraum unter den Invarianten. |
| Code, Prüfung und Bedienungsdokumentation sind erforderlich. | Unterschiedliche Realisierungsbeiträge mit Zuständigkeit und Zuordnung. |

Die Beschreibung muss außerdem erklären, ob Ausleihdauer, Identitätsvergabe, Autorisierung, Wiederholung, Persistenzfehler und Wiederherstellung festgelegt oder außerhalb des gewählten Umfangs sind. Die bisherige C#-Factory beantwortet nicht alle diese Fragen. Sie ist kein Vollständigkeitsnachweis.

Ein zweites Beispiel, etwa ein kleines zustandsloses CLI-Werkzeug, soll prüfen, ob die Sprache unnötig bibliotheksspezifische oder organisationsspezifische Voraussetzungen enthält. Diese Prüfung ist hier nur geplant.

## 8. Geplanter Weg zur belastbaren Spezifikation

Die Schritte sind ein Vorschlag zur weiteren Ausarbeitung. Es werden dadurch keine Backlogeinträge aktiviert, Implementierungsaufträge verteilt oder Studien gestartet.

| Schritt | Ergebnis | Abschlusskriterium |
| --- | --- | --- |
| 1. Beschreibungsauftrag fixieren | System-/Repositoryumfang und Fragenkatalog. | Nutzen, Grenzen und zugelassene Freiräume sind nachvollziehbar. |
| 2. Bibliothek als gewöhnliche Spezifikation schreiben | Zusammenhängende Beschreibung ohne voreiliges Schema. | Wesentliche Entscheidungen sind festgelegt, delegiert oder ausdrücklich offen. |
| 3. Facetten und Beziehungen spezifizieren | Versionierte Feld-, Rollen-, Geltungs- und Beziehungstabellen. | Jede Ergänzung beantwortet einen konkreten Beschreibungsbedarf. |
| 4. Realisierung und Nachweise zuordnen | Artefakterwartungen, Coverage, Prüfpflichten und Eigentümer. | Anforderungen lassen sich bis zu benötigten Beiträgen verfolgen, ohne jede Datei vorzugeben. |
| 5. Kontrastbeispiel und Gegenbeispiele verwenden | Ausdruckslücken und unnötige Voraussetzungen. | Die Sprache beschreibt unterschiedliche Systeme mit derselben Grundlage. |
| 6. Mit dem festen Main-Vertrag zusammenführen | Kleine, koordinierbare Änderungsvorschläge für Vokabular, Sichten, Impact und Host. | Vorhandene Fähigkeiten werden genutzt; dokumentierte und tatsächliche Lücken sind unterschieden. |
| 7. Nach gesondertem Auftrag evaluieren | Fest gebundene Referenzfälle oder Repo-Extraktion mit begrenzten Turns. | Sprache, Extraktionsfähigkeit, Prüfumfang und Nutzerwert werden getrennt bewertet. |

Die neue Vokabularsemantik braucht vor einer späteren Implementierung einen koordinierten Modellvertrag. Die bestehende Main-Entscheidung zu einem aktiven Owner der Modellzone bleibt maßgeblich; dieser Proposal ist kein paralleler Schemaumbau.

Für jede ergänzte Modellrelation muss ihre Wirkung auf Digest, Kontext, Impact, Coverage und Prüfevidenz festgelegt werden. Nicht in Lesesichten übernommene semantische Daten dürfen bei Änderungen nicht verschwinden. Unbekannter Impact bleibt konservativ; eine engere Auswahl wird begründet, nicht angenommen.

## 9. Zusagen und offene Entscheidungen

Der vorgeschlagene Vertrag soll ausdrücken können, was gilt, wozu etwas gehört, was eine Umsetzung realisieren muss, was frei bleibt und welche Evidenz verlangt wird. Strukturprüfung, vollständige Pfadzuordnung, fachliche Konformität und ausreichende Systembeschreibung bleiben unterschiedliche Urteile.

Offen sind besonders:

- Welche Facetten brauchen eigenständige Kinds, welche bleiben strukturierte Aussageinhalte?
- Welcher Formalisierungsgrad hilft, ohne die Beschreibung unnötig schwer zu pflegen?
- Wie werden bewusst delegierte Entscheidungen von unvollständiger Beschreibung unterschieden?
- Wie weit darf eine Zuordnung automatisch ermittelt werden, und welche Unsicherheiten blockieren?
- Wie werden mehrere fachliche Beiträge zu einer Datei mit eindeutiger Schreibzuständigkeit verbunden?
- Wie werden Vokabularversionen und Identitätsänderungen ohne zweite kanonische Wahrheit weiterentwickelt?

## 10. Herkunft und Status

- Quelle: Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`, zuletzt ausdrücklicher Planungs- und Vergleichsauftrag vom 11. Oktober 2026.
- Dokumentationsworktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Eingang der Gesprächsnotizen: `357a726347676ca8589a173b88aa8cb34d241c3d`; Worktree beim Beginn sauber.
- Ursprüngliche Branchbasis: `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.
- Aktueller Main-Abgleich: separat an `95251f1b76b9d7cb1c9c7321a7751146147a0b78`; über Git-Objekte gelesen, nicht mit dem älteren Working Tree gleichgesetzt.
- Lokale Arbeit: dieser Proposal, Vergleichsdokument und Navigations-/Statushinweise im Entwurfsordner. Keine Produktimplementierung, Tests, Studien oder Agentenausführung am Beispiel.

Die lokale Commit-SHA des fertigen Dokumentationsstands wird bei der Übergabe berichtet. Der Proposal ersetzt keine akzeptierte Main-Entscheidung und keine kanonische Produktdokumentation.
