# Proposal: Aufbau des kanonischen Modells

Datum: 11. Oktober 2026. Status: Proposal und Vorbereitung der gewünschten gemeinsamen Umsetzung; einzelne Formatentscheidungen noch offen, keine Implementierung erfolgt.

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

### 8.1 Begründung der Designentscheidungen

Diese Begründungen machen den Proposal für einen Implementer nachvollziehbar. Sie sind Designargumente und überprüfbare Erwartungen, kein gemessener Nachweis der erhofften Produktwirkung.

| Entscheidung | Warum sie zur Leitfrage passt | Alternative und bewusste Abwägung |
| --- | --- | --- |
| Kleiner struktureller Core, Softwarebedeutung im Projektvokabular | Dieselbe Typ-, Identitäts- und Referenzprüfung kann unterschiedliche Systeme tragen. Fachbegriffe sollen beschreibbar sein, ohne Bibliotheks- oder Architekturlogik in den generischen Compiler einzubauen. | Universeller Katalog vieler Core-Kinds würde Semantik und Compiler koppeln. Erst ein belegter struktureller Ausdrucksbedarf rechtfertigt eine Core-Erweiterung. |
| Typisierte Facetten als erster Kandidat | Ein heutiges Statement kann verständliche Prosa und präzisere Daten zusammenhalten. Zusammengehörige Festlegungen haben damit eine kanonische Quelle und bleiben unabhängig referenzierbar. | Eigene nominale Kinds bieten stärkere Zieltypprüfung, erhöhen aber Vokabular- und Referenzaufwand. Facetten benötigen ihrerseits strikte Rollenvalidierung; beliebige Zusatzdaten würden den Nutzen verlieren. |
| Inhaltliche Rollen vor endgültigen YAML-Typen | Die Sprache soll den Beschreibungsbedarf eines Systems erfüllen. Bibliothek und Kontrastbeispiel zeigen, welche Rollen eigenständig sein müssen und welche Details kohäsiv zusammengehören. | Direkt den C#-Klassenkatalog als YAML abzubilden würde illustrative Objektstrukturen ungeprüft zum Produktvertrag machen. |
| Komposition, Abhängigkeiten und Verantwortung unterscheiden | Ein Baum ordnet Zugehörigkeit; Querbeziehungen beschreiben Zusammenarbeit und Auswirkungen. Ein Graphpfad allein sagt nicht, wer entscheiden oder schreiben darf. | Jede Beziehung als allgemeine Dependency zu behandeln verwischt Kontext, Pflicht und Befugnis. Die heutige Namespace-/Managerkonvention bleibt ein ausdrücklich gewählter Ausgangspunkt. |
| Realisierungsverpflichtungen plus deklarierte Pfadbereiche | Das Modell kann beschreiben, welcher Beitrag notwendig ist, während der konkrete Dateibestand abgeleitet wird. Main kann Pfadpräfixe und Mehrfachzuordnung bereits verarbeiten. | Jede Datei manuell als eigene Fachdefinition zu pflegen erhöht Synchronisationsaufwand. Vollautomatische Sourcezuordnung wäre ein zusätzlicher, noch unbelegter Mechanismus. |
| Neue Semantik durch alle Sichten führen | Ein typisiertes Feld hilft der Agentenkoordination nur, wenn Kontext, Änderungsfolgen und Prüfpflichten es ebenfalls berücksichtigen. Main besitzt dafür bereits Analyse-, Graph- und Explain-Strukturen. | Ein reines Schema-Upgrade könnte Daten kompilieren, die der Umsetzer nicht sieht oder deren Änderung keine geeignete Prüfung auslöst. |
| Modellstruktur und realisierte Fachregel getrennt beurteilen | Die Definition einer Invariante kann syntaktisch konsistent sein, während der Anwendungscode sie verletzt. Auch ein vollständiger Dateiindex beweist keinen Verhaltensvertrag. | Eine pauschale „Verify passt, also vollständig“-Zusage würde den tatsächlichen Prüfumfang überschreiten. Jeder Nachweis muss seine Aussage und Grenze nennen. |

Die wichtigste Annahme ist, dass zusätzliche Struktur relevante Entscheidungen sichtbar macht und ihren Erhalt gezielter prüfen lässt. Offen bleibt, ob dieser Nutzen den zusätzlichen Autoren- und Prüfaufwand übersteigt. Dieser Plan misst das nicht; spätere Evaluation muss beide Seiten erfassen.

### 8.2 Umsetzungsvorbereitung für den Implementer

Vom Nutzer gewünscht: gemeinsame Umsetzung mit dem Implementer im nächsten Arbeitsschritt. Diese Übergabe bereitet sie vor; offene Formatentscheidungen werden dadurch nicht stillschweigend angenommen.

**Lesereihenfolge:** [Leitfrage und Arbeitsstand](system-description-notes.md), dieser Proposal, [fester Main-Abgleich](canonical-model-main-comparison.md). Der ältere C#-Entwurf dient als Beispiel und Gegenprobe, nicht als verbindliche Klassenliste.

Der Dokumentationsbranch `docs/product-model-concepts` stammt aus einer älteren Main-Basis. Der Implementer soll einen Implementierungsbranch vom dann frisch abgerufenen `origin/main` anlegen und die vereinbarten Dokumentationsänderungen gezielt übernehmen. Vorher festen SHA, Pfad, Branch und lokale Änderungen erfassen sowie die aktuellen Beitragshinweise und den Modellzonen-Owner prüfen. Den alten Dokumentationsbranch nicht allein als aktuelle Produktbasis verwenden. Die folgenden Quellbereiche stammen aus dem Abgleich an `32afd1a57e715130d056e98e433721c6e7a10278`; ihre Zuständigkeit beim Start erneut bestätigen.

| Reihenfolge / Paket | Konkretes Ergebnis | Voraussichtlich betroffene Bereiche | Abnahmebedingung |
| --- | --- | --- | --- |
| U0 — Ausgangspunkt und Beschreibungsbedarf | Main-Delta prüfen; Umfang, Bibliotheksausschnitt und CLI-Kontrast in gewöhnlicher Spezifikation beschreiben; benötigte Felder/Rollen einschließlich Kontext-/Impactwirkung bestimmen. | Proposal, zuständige Produkt-/Modellverträge; Koordination mit Modell-Owner. | Feldbedarf ist aus den Fällen begründet; Formatoptionen sind eingegrenzt, Freiräume und Prüfgrenzen ausdrücklich festgelegt. Keine zweite kanonische Quelle. |
| U1 — Kleinster zusammenhängender Beschreibungsfall | Bibliotheksausschnitt mit Systemgrenze, Exemplar/Ausleihe, Ausleihvertrag, Eindeutigkeitsregel sowie Code-/Prüf-/Dokumentationspflichten als gültige Modellquellen. | Neues Beispiel und versionierter Schema-Vertrag in `src/internal/modules/projectmodel/projectmodel.go` oder dessen abgestimmtem Nachfolger. | Der Fall beschreibt eine wesentliche Fähigkeit samt Daten, Regel und Realisierung; ein gültiges bisheriges Minimalprojekt bleibt unter seiner bisherigen Version gültig. |
| U2 — Strukturelle und semantische Validierung | Geschlossene Felder, Wert-/Kardinalitätsregeln, erlaubte Zielrollen und Kompositionsregeln mit deterministischen Diagnosen. | Projectmodel-Schema und Analyse/Validator; Core nur bei konkret belegter Ausdruckslücke. | Ungültige Rolle, fehlende Pflichtangabe, unzulässige Referenz und Kompositionszyklus werden am passenden Quellgegenstand gemeldet. Fachinstanzregel wird nicht als Compilerbeweis ausgegeben. |
| U3 — Lesesichten und Sichtbarkeit | Strukturierte Inhalte im Report und benötigten Managerkontext; passende Graphbeziehungen und lesbare Ausgabe. | `types.go`, `analyze.go`, `Context` in `impact.go`, `graph.go`; bestehende Host-/Ausgabeanschlüsse nach Bedarf. | Jede bindende neue Angabe ist im zuständigen Umsetzungs-/Prüfkontext verfügbar; private Inhalte gelangen nicht in fremde Ansichten. Graph bleibt abgeleitet. |
| U4 — Änderungsfolgen | Jedes neue Feld und jede Relation explizit für Digest, Delta, Kontext, Impact und Erklärung behandeln; alte und neue Endpunkte berücksichtigen. | `impact.go`, `explain.go`, Snapshot-/Hostbindungen soweit deren Vertrag betroffen ist. | Semantische Änderungen erreichen betroffene Realisierungen und Checks; Entfernung einer Relation verliert deren alte Wirkung nicht. Schreibänderung und unbekannter Umfang behalten die dokumentierte Behandlung. |
| U5 — Realisierung und Verify-Anschluss | Vorhandene Artifact-/Check-/Coverage-Strukturen mit vereinbarten Beitragstypen verbinden; Prefix-Zuordnung und eindeutige Schreibzuständigkeit nutzen. | Projectmodel, `src/internal/host/projectcoverage`, `projectwork` und bei tatsächlich geänderter Bindung `projectrun`. | Mehrere Aussagen können dieselbe Datei betreffen; jede konkrete Ausgabe hat einen Owner. Unklassifizierte Dateien bleiben Befunde. Bericht trennt Struktur, Coverage, ausgeführte Checks und unbelegte Aussagen. |
| U6 — Kontrast, Dokumentation und Integration | Kleines CLI-Gegenbeispiel, gezielte Negativfälle, aktuelle Nutzer-/Formatdokumentation und normale Integrationsprüfung. | Beispiele, zuständige Dokumente, erforderliche Tests und Beitragseinträge. | Sprache trägt beide Beispiele; Aufwand und fehlende Semantik sind sichtbar. Projektgates laufen am festen Implementierungskandidaten; Dokumentation behauptet nur belegte Fähigkeiten. |

Die gemeinsame Entscheidung über Facetten versus nominale Kinds und Versionierung folgt aus den Beschreibungsfällen von U0 und fällt vor dem Schema-Vertrag in U1. U1 bis U5 bilden gemeinsam den ersten funktionsfähigen Durchstich. Ein nur erlaubtes Schemafeld ist kein fertiges Feature. U2 und U3 folgen dem festgelegten Feldvertrag; U4 benötigt deren gemeinsame Darstellung. U5 nutzt die bestehende Ausführungslinie und erweitert sie nur für konkret veränderte Verpflichtungen. Der Modellvertrag hat einen koordinierten Owner; andere Arbeiten können danach klar getrennte Dateien oder Beispiele übernehmen. Die Paketnamen hier sind Vorschläge und keine vergebenen Backlog-IDs.

### 8.3 Prüfplan für die spätere Implementierung

Heute werden diese Prüfungen nur geplant. Bei der Umsetzung gelten die dann aktuellen Checks aus `CONTRIBUTING.md` zusätzlich zu den folgenden konkreten Risiken:

| Risiko | Gezielter Nachweis am Implementierungskandidaten |
| --- | --- |
| Fachrolle ist strukturell gültig, aber semantisch falsch | Ein Vorbedingungsverweis auf einen unpassenden Gegenstand und ein ungültiger Fachbeziehungsendpunkt werden abgewiesen. |
| Neue Bedeutung fehlt im Agentenkontext | Ein Manager bekommt den benötigten Verhaltensvertrag samt Regel; ein fremder privater Inhalt bleibt verborgen. |
| Änderung wird im Impact übersehen | Regeländerung, geänderte Eigenschaft und entfernte Beziehung erreichen erwartete Owner, Artefakte und Checks in alter/neuer Betrachtung. Kombinierte Änderungen verkleinern den erforderlichen Umfang nicht. |
| Erklärung und Impact widersprechen sich | Explain bindet den passenden Digest und erklärt die betroffenen Listen, ohne sie durch Change-/Context-Priorisierung zu verkürzen. |
| Realisierungsindex wird zweite manuelle Wahrheit | Verzeichnispräfix und tatsächlicher Dateizugang erzeugen aktuelle Coverage; Mehrfachzuordnung bleibt möglich, widersprüchliche Schreibowner bleiben Befunde. |
| Sprache wird unnötig schwer oder bibliotheksspezifisch | Das CLI-Gegenbeispiel kommt ohne Ausleihe, Datenbank, Zustandsautomat oder zusätzliche Managementebenen aus. |
| Versionierung verändert alte Bedeutung still | Bisherige Modelle behalten ihren Versionsvertrag; neue Syntax/Regeln besitzen eine ausdrückliche Versionierungs- beziehungsweise Migrationsentscheidung. |

Zur Übergabe gehören feste Code-/Modell-SHAs, lokaler Zustand, tatsächlich ausgeführte Prüfungen samt Ergebnis, nicht ausgeführte Prüfungen und verbliebene offene Entscheidungen. Repo-Extraktionsstudien, allgemeine Ausdrucksevaluator-, Government- oder Cockpit-Arbeit gehören nicht zum ersten Durchstich. Die endgültige Implementierungsfreigabe bezieht sich auf den gemeinsam gewählten Umfang.

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

- Quelle: Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`, Planungs- und Vergleichsauftrag sowie anschließender Wunsch nach Begründung und Umsetzungsplan für die gemeinsame Arbeit mit dem Implementer vom 11. Oktober 2026.
- Dokumentationsworktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Eingang der Gesprächsnotizen: `357a726347676ca8589a173b88aa8cb34d241c3d`; Worktree beim Beginn sauber.
- Ursprüngliche Branchbasis: `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.
- Aktueller Main-Abgleich: separat an `32afd1a57e715130d056e98e433721c6e7a10278`; über Git-Objekte gelesen, nicht mit dem älteren Working Tree gleichgesetzt.
- Lokale Arbeit: dieser Proposal, Vergleichsdokument und Navigations-/Statushinweise im Entwurfsordner. Keine Produktimplementierung, Tests, Studien oder Agentenausführung am Beispiel.

Die lokale Commit-SHA des fertigen Dokumentationsstands wird bei der Übergabe berichtet. Der Proposal ersetzt keine akzeptierte Main-Entscheidung und keine kanonische Produktdokumentation.
