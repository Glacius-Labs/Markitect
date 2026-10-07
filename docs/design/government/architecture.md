# Government of Coding Agents: Architektur v0.1

**Status:** Zusammenhängender Architekturentwurf zur Evaluation. Kein angenommener Produktvertrag und keine implementierte API. Die Beispielstrukturen sind Vorschläge.

**Vergleichsbasis:** abgeschlossener Architect-Kandidat `1ea5c76f55526fc4d721e865885436153f48b497`. Der [Checkpoint-Bericht](../architect-government-checkpoint.md) hält dessen Grenzen fest. Der finale Kandidaten- und Pilotabgleich begleitet die Umsetzung nach dem angekündigten, noch ausstehenden Go.

## 1. Ziel und zentrale Entscheidung

Government organisiert kontinuierliche Entwicklungsarbeit an einem gesamten Repository. Das akzeptierte kanonische Modell beschreibt die gewünschte Projektwelt; beobachtete Repository-Artefakte bilden ihren tatsächlichen, gegebenenfalls unvollständigen Stand ab. Die Regierung leitet Arbeit aus der Abweichung ab, delegiert sie, integriert die Ergebnisse und prüft die gesamte betroffene Ordnung.

Jeder dauerhafte kanonische Gegenstand hat einen erklärten `purpose`: warum er existiert und welche Rolle er im akzeptierten Projekt erfüllt. Zweck wird nicht aus dem Vorhandensein einer Datei, einem erfolgreichen Build oder einer Agentenbehauptung abgeleitet. Für beobachtete, noch nicht verstandene Dateien lautet der bekannte Zweck zunächst `unknown`.

Das Repository als Sollwelt-Abbild verlangt bidirektionale Zuordnung: Modellpflichten zeigen auf reale Artefakte, und jedes beobachtete Artefakt ist als kanonische Eingabe, Realisierung, abgeleitete Ausgabe, Fremdbesitz, begründeter Ausschluss oder ungeklärt erfasst. Eine fehlende Implementierung ist eine sichtbare Soll-Ist-Lücke, kein Grund, die Pflicht aus dem Modell zu löschen.

Eine dauerhafte Verantwortung, eine konkrete ausführende Agenteninstanz, eine technische Fähigkeit, ein Artefakt, ein Laufbericht und ein Urteil sind verschiedene Dinge. Government hält diese Identitäten auseinander und bindet ihre Zusammenarbeit an feste Versionen und Eingaben.

## 2. Begriffe und Metapher

Der öffentliche Begriffsbestand soll klein und überall konsistent sein:

| Begriff | Bedeutung |
|---|---|
| **Government** | Die gesamte Organisation, die Projektordnung pflegt, Arbeit verantwortet, ausführt, prüft und annimmt. |
| **Verfassung** | Das akzeptierte kanonische Sollmodell mit Fachmodell, Organisationsordnung, Pflichten und Delegationen. |
| **Bereich** | Rekursiv verschachtelte Verantwortung für einen Teil der Projektordnung und Arbeit; derselbe Begriff gilt auf jeder Hierarchieebene. |
| **Ressort (Ministerium)** | Dauerhafte fachliche Zuständigkeit, die quer über Bereiche und Dateien prüfen kann; kein Writer und keine Agenteninstanz. „Ressort“ ist im Entwurf der einheitliche Begriff für das vom Nutzer beschriebene Ministerium. |
| **Mandat** | Versionierter Zuständigkeits- und Befugnisrahmen; gilt nur, wenn es aus der aktiven Verfassung delegiert wurde. |
| **Durchlauf** | An Auftrag, Verfassung, Revision, Kabinett und Ressourcen gebundene Arbeitssitzung. |
| **Kandidat** | Unveränderlicher Entwurf mit `MaterialCandidateID` aus normativem Modell und Repository-Tree, über den geprüft und abgestimmt wird. |
| **Bericht** | Lauf- oder Prüfprotokoll mit Befunden, Eingaben, Unsicherheit und Herkunft; selbst keine Annahme. |
| **Ressortvotum** | Ausdrückliche, kandidaten- und evidenzgebundene Zustimmung oder Einwand eines ausgewählten Ressorts. |
| **Gerichtsauslegung** | Begründete Auslegung geltenden Rechts innerhalb des aktiven Mandats; kein neues Gesetz und kein Votum. |
| **Annahmebeschluss** | Aus dem gültigen Stimmenbestand abgeleiteter Beschluss, einen geprüften Kandidaten zu übernehmen. |

Exekutive, Legislative, Prüfamt und Gericht bezeichnen Rollen im Entscheidungsprozess, keine zusätzlichen Agenteninstanzen. Bereich und Ressort sind dauerhafte Zuständigkeiten; konkrete Ausführung und Prüfung erhalten pro Durchlauf eigene Agenteninstanzen. Kein Agent darf allein seinen eigenen Kandidaten prüfen und annehmen.

Ein Ressort ist weder eine Programmiersprache noch ein Projektionsmodul oder Agent. Es verantwortet fachliche Regeln quer über Dateien; Bereiche tragen die rekursive Ausführung und Dateiverantwortung. Technische Fähigkeiten kann jedes Ressort oder jeder Bereich verwenden; sie verleihen keine gesetzgeberische Befugnis.

## 3. Zwei normative Sichten und separater Laufzustand

### 3.1 Fachmodell in der Verfassung

Das Fachmodell beschreibt Zwecke, fachliche Konzepte, Systemgrenzen, Anforderungen, Invarianten, gewünschte Fähigkeiten und zulässige Implementierungsfreiheit. Beziehungen zwischen diesen Gegenständen sind explizit und besitzen selbst einen Zweck. Der geschützte verfassungsrechtliche Kern – Root-Ziele, Befugnisgrenzen und Änderungsverfahren – ist von gewöhnlichen, delegierbaren Fach- und Architekturregeln unterscheidbar.

### 3.2 Organisationsordnung in der Verfassung

Die Verfassung enthält zwei unterscheidbare, gemeinsam versionierte Sichten: Fachmodell mit Zielen und Regeln sowie Organisationsordnung mit Bereichshierarchie, Ressorts, Mandaten, Capabilities, aktivem Kabinett und Abstimmungsweg. Die Organisationsordnung bestimmt, wer welche Pflicht verantwortet und wer eine Änderung beurteilt; sie kann nicht losgelöst von den Gesetzen derselben aktiven Verfassung wechseln.

### 3.3 Laufzustand

Der Laufzustand enthält Durchläufe, Queue-/Backlog-Status, Agentenläufe, Kandidaten, Eingabeinventare, Ressourcenverbrauch, Stimmen, Urteile, Checkpoints und Übernahmeergebnisse. Er ist keine kanonische Projektabsicht und darf das geltende Gesetz nicht ändern.

Ein Durchlauf bindet unveränderlich genau einen aktiven Verfassungs-Digest, der Fachmodell und Organisationsordnung einschließt. Vorgeschlagene Änderungen an beiden Sichten gehören in denselben Kandidaten und werden bis zur Übernahme nicht aktiv. Der isolierte Kandidatenbereich darf diese vorgeschlagene Verfassung probeweise verwenden, um Modell, Dateien und Tests gemeinsam zu evaluieren; geltende Regeln und Befugnisse bleiben unverändert.

Technische Records und kryptografische Digests belegen Identität und Herkunft eines Vorgangs. Sie authentifizieren keine natürliche Person, beweisen keine semantische Richtigkeit und gewähren keine Befugnis.

## 4. Modellierung des gesamten Repositories

### 4.1 Gegenstände, Zwecke und Gesetze

Jeder dauerhafte Verfassungsgegenstand und jede dauerhafte Beziehung trägt einen kurzen Zweck. Eine Geschäftsregel, ein Bereich, ein Interface, ein Systemziel, ein Ressort und eine Soll-Artefaktbeziehung sagen damit, warum sie Bestandteil der Projektwelt sind.

`purpose` erklärt Absicht, ersetzt aber keine testbare Anforderung. Eine Invariante braucht Geltungsbereich und erwartbares Verhalten; eine Beziehung braucht Typ, Richtung und Zweck. Dokumentationsprosa kann Kontext liefern, erzeugt aber nicht implizit eine Abhängigkeit.

Die Verfassung beschreibt das gewünschte Verhalten und seine Grenzen, ohne gewöhnliche Implementierungsfreiheit in eine Dateiliste umzuwandeln. Konkrete Pfade werden mit realen Artefakten verbunden, wo Pfadgenauigkeit für Ownership, Erzeugung oder Prüfung notwendig ist.

### 4.2 Viele-zu-viele Realisierung und eindeutiger Writer

Die Soll-zu-Ist-Beziehung ist viele-zu-viele: Eine Modellpflicht kann durch mehrere Dateien gemeinsam realisiert werden; eine Datei kann mehrere Modellpflichten und Funktionen tragen. Die normative Relation `realizes` beschreibt nur Modellgegenstand, beabsichtigten Pfad/Rollenbezug und Zweck. Beobachtete Bytes, Revisionen und Evidenz gehören separat ins Laufledger; sie sind keine Selbstreferenz der Modellrelation.

Diese semantische Zuordnung ist unabhängig von der Schreibverantwortung. Für einen konkreten Kandidaten hat jeder veränderbare Artefaktpfad genau einen Writer. Ein Agent kann mehrere Pfade besitzen; mehrere Ressorts können dieselbe Datei fachlich prüfen. Zwei parallele Writer derselben Datei werden durch einen verantwortlichen Integrator sequenziert.

Feingranularität ist konfigurierbar bis hin zu einem einzelnen Invariant- oder Interface-Auftrag. Aufteilung oder Bündelung löscht keine Verpflichtung: jeder Scope behält Besitzer, Inputs, Ergebnisse, Evidence und offene Fragen.

### 4.3 Vollständige native Beobachtung

Government beobachtet den Repository-Bestand durch einen Host-seitigen Inventarprozess innerhalb deklarierter Roots und Grenzen. `.git`-Interna sind kein Projektinventar. Ignorierte/private Inhalte werden nicht stillschweigend gelesen; Symlinks und Submodule werden als Grenzen/Metadaten erfasst und nicht blind verfolgt. Ein Inventar erfasst im deklarierten Umfang Pfad, Typ/Modus, Bytesdigest, Quelle, Beobachtungsrevision und Status. Tracked, untracked, generierte und vendored Einträge dürfen dort nicht durch eine nur an bekannten Modellpfaden orientierte Suche verschwinden.

Jedes Artefakt erhält eine sichtbare Klasse: kanonische Eingabe, gemanagte Realisierung, generierte Ausgabe, fremdbesessen, begründet ausgeschlossen oder unbekannt. Unbekannt ist ein gültiger Beobachtungszustand, aber kein Pass.

Das Inventar ist nur relativ zu den deklarierten Roots/Grenzen vollständig. Jede Ausschlussgrenze ist sichtbar begründet. Das ist noch keine globale Konformität: Eine Konformitätsbehauptung bleibt offen, solange ein relevantes Objekt, seine `purpose`-Zuordnung, Pflichtabdeckung oder Prüfung unbekannt ist. Ein begrenzter Bereich kann bearbeitet werden, obwohl anderswo Unbekannte bestehen; der Bericht darf dann nur den gebundenen Teilbereich als geprüft ausweisen.

Ein Ausschluss bleibt samt Grund sichtbar und liegt außerhalb der inhaltlichen Konformitätsbehauptung. Unbekannt, ignoriert oder ausgeschlossen wird nie in „konform“ umbenannt, nur um den Audit zu schließen.

## 5. Fachmodell, Government und Fähigkeit im Zusammenspiel

Der fachliche Graph beschreibt die Welt und ihre Gesetze. Die Bereichshierarchie beschreibt dauerhafte Zuständigkeit und Delegation. Der Artefaktgraph beschreibt Realisierung und Writer. Der technische Abhängigkeitsgraph beschreibt Schnittstellen, Datenfluss und Buildreihenfolge. Der Urteilsgraph beschreibt, wer welche lokalen und übergeordneten Pflichten geprüft hat. Diese Graphen referenzieren einander, aber werden nicht zu einem einzigen Kantenbegriff vermischt.

Die Verantwortungsorganisation ist rekursiv und wiederholt denselben Typ: Government delegiert an einen Bereich, ein Bereich kann Unterbereiche delegieren und an jedem Blatt entsteht ein konkreter Arbeitsauftrag. Jeder Elternbereich integriert die Ergebnisse seiner Kinder und prüft seine eigenen Kompositionspflichten. Ressorts liegen quer dazu: sie vertreten fachliche Regeln über mehrere Bereiche und Datei-Writer hinweg.

Jeder ausführende Bereich erhält pro Durchlauf einen eigenen Agentenlauf. Ein Bereich darf Aufträge temporär weiter zerlegen, ohne daraus dauerhafte Bereiche oder neue Befugnisse zu machen; Teilaufträge erben nur dasselbe Mandat. Deterministische Einzelchecks benötigen keinen KI-Aufruf. Ein unabhängiger semantischer Reviewauftrag erhält dagegen eine getrennte Prüferinstanz. Institution ist nicht Agenteninstanz: Ressort und Bereich bestehen über Läufe hinweg, Agenteninstanzen sind kurzlebig und an einen Kandidaten gebunden. Derselbe Modellanbieter ist allein kein Beleg für Unabhängigkeit; Trennung von Instanz, Rolle, Eingaben und Verantwortung wird berichtet.

Ressorts besitzen kein Dateischreibrecht allein aufgrund ihres Mandats. Sie können geänderte Dateien und Verhaltensbereiche prüfen, Einwände erheben und erforderliche Befunde anfordern; Writer wird vom Verantwortungsbereich des Kandidaten benannt.

Technische Fähigkeiten liegen orthogonal: etwa Compiler, Test-Runner, .NET-, Go-, Markdown- oder Infrastrukturwissen. Ein Capability-Pin identifiziert Werkzeug und Version; er ändert kein Mandat und keine Verfassung. Ein Bereich oder Ressort kann mehrere Fähigkeiten einsetzen.

## 6. Befugnis und Gesetzgebung

### 6.1 Delegation bleibt eine Teilmenge

Eine aktive höhere Verfassung delegiert einen Bereich, sein Ziel, relevante Grenzen, erlaubte Entscheidungsklassen und Eskalationsfälle. Die Subdelegation muss eine Teilmenge davon bleiben. Keine Kandidatenversion darf durch eine neue Regel ihre eigene Annahmebefugnis, Stimmenzahl, Ressortabdeckung oder Rechte ihrer Urheber vergrößern.

Eine delegierte Architekturentscheidung kann die fachliche Modellierung substanziell ändern: etwa die dauerhafte Modulzuständigkeit oder Ownership einer Geschäftsregel festlegen. Delegation ist nicht auf redaktionelle Pflege begrenzt. Sie ist aber nur wirksam, wenn die alte aktive Verfassung diese Klasse tatsächlich umfasst und ein gültiges Annahmeverfahren nennt.

Beobachtete Implementierung kann eine Lücke begründen, aber nicht ihre eigene Soll-Regel autorisieren. Wird im Betrieb ein neues Ziel, geschützter Trade-off oder eine nicht delegierte Architekturpräferenz nötig, bleibt die Arbeit am betroffenen Kandidaten offen und die Entscheidung geht an den zuständigen übergeordneten Entscheider.

### 6.2 Unveränderliche aktive Rechtsgrundlage

Jeder Änderungsversuch referenziert ausschließlich den unveränderlichen Digest der vor dem Kandidaten aktiven Verfassung. Nur diese Vorversion bestimmt, wer Änderungen vorschlagen, prüfen, abstimmen und fördern darf.

Der Kandidat kann Fachregeln, Mandate, Kabinett oder Abstimmungsregel vorschlagen. Diese vorgeschlagenen Bindungen dürfen jedoch weder das Wahlkollegium für denselben Kandidaten ändern noch eine neue Regel rückwirkend auf die eigene Abstimmung anwenden. Die vorher aktive Verfassung bleibt für den gesamten Entscheidungsweg maßgeblich.

Default für fachliche Modelländerungen innerhalb delegierter Architekturarbeit: Die aktive Vorverfassung muss den Änderungsbereich und sein Verfahren ausreichend erlauben; alle aus dem vorher aktiven Kabinett ausgewählten Ressorts stimmen der finalen Fassung ausdrücklich zu. Das vorher aktive Kabinett bleibt eingefroren. Eine Änderung an Verfassungsspitze, eigener Befugnis, Abstimmungsregel oder Root-Zielen eskaliert immer zum Owner und wird nie durch denselben Agentenlauf autonom legitimiert. Änderung oder Entfernung eines ablehnenden Ressorts während der laufenden Entscheidung ist unzulässig.

Ist in der aktiven Vorverfassung kein ausreichendes Änderungsmandat oder Abstimmungsverfahren enthalten, darf die Agentenorganisation die Änderung nicht autonom aktivieren. Sie kann einen Kandidaten im isolierten Workspace probeweise bauen und prüfen; das ändert den aktiven normativen Stand nicht. Zur Aktivierung legt sie dem Owner eine konkrete Entscheidung vor. Der Owner ist durch den vertrauenswürdigen Host-/Produktkanal identifiziert; eine Modellantwort mit „Owner approved“ hat keine Autorität.

### 6.3 Explizite Einstimmigkeit

Zu Beginn des Durchlaufs wird das aktive Kabinett aus der unveränderlichen Verfassung fixiert. Für jeden Kandidaten gilt: jedes so ausgewählte Ressort muss seine eigene ausdrückliche Zustimmung zur exakten finalen Kandidaten-Digest-Version abgeben.

Ein Ressort bewertet die eigene Zuständigkeit und das Vorliegen eines Konflikts. Wenn es keine Betroffenheit findet, muss es dennoch explizit `assent-unaffected` mit Begründung abgeben. Schweigen, Nicht-Aufruf, Timeout, Toolfehler, unvollständige Inputs, Enthaltung oder Zustimmung zu einer alten Fassung sind keine Zustimmung.

Jedes Ressorturteil entsteht in einem eigenen Prüf- und Entscheidungsdurchlauf, gebunden an sein aktives Mandat. Der Writer/Proposer kann nicht seine eigene Ausführung als Ressortzustimmung einreichen; der Host bindet die Stimme an den konfigurierten Ressortslot, ohne damit die Identität oder Vertrauenswürdigkeit des Modells zu authentifizieren.

Geänderte Modell- oder Repository-Bytes erzeugen eine neue Kandidatenidentität. Neue relevante Evidenz bei unveränderten Bytes startet eine neue Prüf- und Abstimmungsrunde mit neuer `EvidenceID`; Stimmen früherer Runden bleiben historisch und erfüllen die neue Einstimmigkeit nicht.

Einstimmigkeit sagt nur etwas über das ausgewählte Kabinett aus. Sie beweist weder, dass alle nötigen Ressorts ausgewählt wurden, noch dass jedes Ressort seine Zuständigkeit ausreichend erkennt. Diese Begrenzung steht im Abschlussbericht.

## 7. Streit, Gericht und Eskalation

Ein Einwand verweist auf die genaue Verfassungsregel, betroffene Kandidatenstelle, Beleg, erwartete Bedingung und Unsicherheit. Die ausführende Ebene versucht, den Kandidaten gesetzeskonform zu reparieren; sie darf kein Ressort aus dem fixierten Kabinett entfernen.

Default sind höchstens zwei Reparaturrunden für denselben ungelösten Einwand. Der Budgetwert ist eine operative Begrenzung, kein Fristablauf zu Zustimmung. Danach kann ein Gericht genau eine begründete Auslegung mit einer einmaligen Rückverweisung zur Reparatur liefern. Das Gericht eröffnet keine automatische Folge von Urteils- und Einspruchsschleifen.

Ein unabhängiges Gerichtsverfahren kann ausschließlich auf Grundlage des geltenden Rechts auslegen, relevante Zuständigkeit bestimmen und rechtmäßige Lösungsräume benennen. Es schafft keine neue Rechtsgrundlage, schreibt keine Gesetze oder politischen Prioritäten um, verleiht keine Ausführungsbefugnis und wandelt kein Ressortveto in Zustimmung um.

Nach der einmaligen Rückverweisung wird die korrigierte Fassung erneut allen fixierten Ressorts vorgelegt. Besteht der Einwand fort, verweigert ein Ressort weiterhin Zustimmung oder ist eine neue gesetzliche Priorität notwendig, bleibt der Kandidat blockiert. Er geht an die nächste übergeordnete Instanz, deren aktives Mandat die Entscheidung umfasst; reicht keine Delegation bis dorthin, entscheidet der Owner. Kein automatischer Richter oder Koordinator darf das Veto überschreiben.

Das Gericht kann als getrennte Prüffunktion beginnen, nicht zwingend als dauerhaftes Paket oder eigenes autonomes Organ. Seine Auslegung führt die einmalige Reparaturrunde innerhalb der bestehenden Rechtsgrundlage; sie ist keine Zustimmung. Berufungsweg und Bindungswirkung brauchen in der aktiven Verfassung klare Grenzen. Die Einstimmigkeit jedes ausgewählten Ressorts bleibt davon unberührt.

## 8. Kontext und Teilbelege

Ein Agent erhält mindestens: Auftragszweck, gebundene Verfassungs-/Mandatsversion, verantwortete Pflichten, eigene Artefaktabdeckung, beobachtete Bytes, relevante Nachbarschnittstellen und direkte Abhängigkeitsevidenz, erwartete Prüfungen, bisherige Befunde sowie Budget und zulässige Fähigkeiten.

Kontext ist explizit partiell. Der Bericht enthält aufgenommene Eingaben, bekannte Auslassungen, ausgeschlossene Pfade und Unbekannte. Kein Agent behauptet Vollständigkeit, nur weil sein Prompt vollständig wirkt.

Fehlt ein erforderlicher Nachbarbeleg, darf der Agent einen konkret benannten Zusatzinput anfordern. Der Host prüft, ob der Pfad innerhalb der aktiven Leseberechtigung und inventarisierten Revision liegt, liest unveränderliche Bytes, hasht diese und erweitert die Lauf-Eingabe. Da sich ihr Digest ändert, wird der Agent mit neuer Request-Bindung erneut aufgerufen; die alte Antwort gilt nicht für den erweiterten Input.

Kann der angeforderte Input nicht gebunden oder bereitgestellt werden, bleibt der Punkt als unbekannt/unvollständig im Urteil. Verborgene Dateisystemsuche und nicht ausgewiesener Gesprächskontext gelten nicht als Evidenz.

## 9. Durchlauf und Abschluss

### Plan

Der Host bindet aktive Verfassung, Revision, Inventory, bestehende Ownership/Realization, offene Backlog-Aufträge, Capabilities und verfügbare Evidence. Er baut den betroffenen Verantwortungsbaum sowie lokalen und übergeordneten Prüfpfad.

Der Plan weist jede Sollpflicht als aktuell erfüllt, Arbeitsbedarf, unbeobachtet, unbekannt oder bewusst außerhalb des Geltungsbereichs aus. Jede Kandidatenpflicht muss einem Writer, einem Prüfer und einem Erledigungsnachweis zugewiesen sein. Planergebnisse autorisieren weder Agentenrechte außerhalb des Mandats noch Schreiben in den aktiven Zielstand.

### Execute und integrieren

Writer erhalten ihre gebundene Aufgabe und erfassen Anträge auf zusätzliche Inputs. Unabhängige, disjunkte Dateien und Schnittstellen können parallel geändert werden; ein integrierender Writer löst gemeinsam genutzte Dateien und prüft den tatsächlichen zusammengeführten Stand.

Jeder Bereich erstellt einen Bericht mit Ergebnis, geänderten und behaltenen Pfaden, neuen Realisierungsbeziehungen, Prüfungen, Eingaben, Befunden, Unsicherheit, Ressourcenverbrauch, nächsten Schritten und nötiger Eskalation. Berichte entstehen auch, wenn keine Arbeit anfällt, der Bereich nicht zuständig ist oder ein Fehler auftritt.

Die Anzeige des Berichts kann pro Bereich und für Government insgesamt auf „alles“, „Wichtiges“ oder „nichts“ stehen. Diese Sichtbarkeitseinstellung ändert weder Befugnisse noch Pflicht zur Berichterstattung, Ressortstimme oder Annahmebedingung; nicht angezeigte Befunde bleiben im Laufzustand nachvollziehbar.

### Verify und Abstimmen

Frische technische Prüfungen laufen auf dem zusammengebauten Kandidaten. Unabhängige lokale Prüfer prüfen ihren Bereich. Jeder Eltern-Bereich prüft die Integration der tatsächlichen Kind-Kandidaten und seine eigenen Zusammensetzungspflichten. Die Root Government führt den Vollständigkeitscheck des gesamten Repositorys aus.

Kindberichte und Prüferbelege bleiben als Herkunftsnachweise in der Elternakte erhalten. Sie ersetzen aber keine Prüfung der tatsächlichen zusammengeführten Root-Fassung. Sobald der vollständige Root-Kandidaten-Digest feststeht, prüft das Root-Prüfamt die Root-Pflichten neu und jedes fixierte Ressort gibt sein Urteil genau zu dieser Root-Fassung ab. Frühere Kind- oder Zwischenkandidaten-Zustimmungen werden nicht implizit auf einen kombinierten Kandidaten übertragen; ein später explizit definiertes und sicher gebundenes Nachfolgerverfahren wäre eine gesonderte Optimierung.

Danach erhält jedes fixierte Ressort denselben finalen Kandidaten und dieselben relevanten gebundenen Nachweise, kann aber zusätzlich erforderliche Inputs anfordern. Stimmen sind getrennte Urteile mit Ressort, Mandat, Kandidaten-Digest, Runde, Evidenz-Digests, Geltungsbereich, Entscheidung und Gründen.

Annahme ist nur möglich, wenn technische Prüfungen aktuell bestanden, alle lokalen und Elternprüfungen geschlossen, alle ausgewählten Ressorts einstimmig und ausdrücklich einverstanden, keine bindenden Befunde offen und die gesamte im Kandidaten deklarierte Repository-Abdeckung abgerechnet ist.

### Promote und audit

Ein angenommener Kandidat enthält als Einheit Verfassungsänderungen (Fachmodell und Organisationsordnung), Repository-Artefakte und technische/Ressortbindungen. Eine Mischfassung aus Modellrevision A, Dateien von B und Kabinett von C ist keine gültige Übernahme.

Die Übernahme prüft Kandidaten-Digest, aktive Basisrevision, alle Zustimmungsversionen, aktiven Verfassungs-Digest, Werkzeugpins, Prüfevidenz, Ownership und Writer-Lock erneut. Der Host fördert nur, wenn der erwartete aktive Stand noch aktuell ist; andernfalls ist der Kandidat veraltet und braucht neue Planung und neue Stimmen.

Der Abschlussaudit beobachtet den gesamten deklarierten Repository-Umfang erneut, prüft Unbekannte und Realisierungsmatrix, aktuelle Prüfungen, Stimmen, Ledger und aktive Commit-Identität. Ergebnislabels sind mindestens `accepted-complete`, `accepted-scoped`, `blocked` und `incomplete`; kein No-Op oder fehlender Agentenbericht erzeugt `complete`.

## 10. Kandidat, Atomizität und Wiederaufnahme

Ein Kandidat wird in einer entbehrlichen Worktree/Branch vorbereitet. Das ist eine kooperative Schreib- und Zusammenführungsgrenze, keine Betriebssystem-Sandbox: Agenten und Host teilen lokale OS-Rechte. Der Ablauf kann vertragsgemäßes Schreiben, Pfadgrenzen und Fencing sicherstellen, aber keinen Schutz vor einem bösartigen Prozess mit direktem Vollzugriff behaupten.

`MaterialCandidateID` bindet unveränderlich die Modell- und Repository-Bytes/Tree, aktive Vorordnung, Plan, Prüfdefinitionen und Toolpins. Ergebnisse, Evidenz, Stimmen und Beschlüsse gehören ausdrücklich nicht hinein. Nach Ausführung bindet `EvidenceID` die Ergebnisse an `MaterialCandidateID`; jedes Ressortvotum bindet beide IDs, und `DecisionID` bindet den vollständigen Stimmenbestand. So entsteht kein Hash-Zyklus und kein Record muss sich selbst enthalten. Geänderte Bytes erzeugen einen neuen Kandidaten; neue relevante Evidenz eine neue Prüf- und Abstimmungsrunde.

Im gesteuerten Host-Ablauf erfolgt die aktive Repository-Änderung erst nach einstimmigen Stimmen. Der Host bereitet einen einzelnen Kandidatencommit mit Modell, Code, Tests, Dokumentation und Government-Konfiguration vor; dessen geprüfter Tree bleibt nach der Abstimmung unverändert. Bei Übernahme wechselt eine dedizierte verwaltete Active-Ref auf diesen Commit. Ein ausgecheckter Nutzerbranch wird nicht per Ref-Update verschoben, während seine Arbeitsdateien veraltet zurückbleiben. Leser erhalten eine an die aktive Commit-Identität gebundene Arbeitskopie; eine veraltete lokale Arbeitskopie wird nie als aktueller aktiver Stand ausgegeben.

Vor der Compare-and-Swap-Promotion schreibt der gefencete Hostwriter dauerhaft ein `PromotionIntent` mit `expectedOld`, `newCommit`, `DecisionID` und `idempotencyKey`. Nur ein gefenceter Hostwriter darf die Active-Ref ändern. Externe Ereignis-/Ledger-Records liegen nicht zwingend in derselben Transaktion wie Git; Recovery liest Intent und Ref-Historie zurück und erkennt auch, wenn die Ref inzwischen über `newCommit` hinaus fortgeschritten ist. Berichte und Evidenzbelege liegen außerhalb des geprüften Trees und erzeugen keine unbeabsichtigten Folgecommits. Kein automatisches Rollback löscht einen bereits geförderten Commit.

Jeder Durchlauf-Checkpoint speichert Lauf-ID, Kandidaten-ID, Runde, aktive Basis, Verfassungs-Digest, Zustandsübergang, erledigte Agentenläufe, Ressourcenbudget, Lease/Fencing-Token und nächste Aktion. Wiederaufnahme lädt keinen Agentenchat als Wahrheit, sondern rekonstruiert den expliziten Kontext und erneuert veraltete Bindungen.

Wiederholungen sind begrenzt und klassifiziert: technische Transportfehler dürfen begrenzt wiederholt werden; eine semantische Ablehnung startet eine Reparaturrunde; ein Timeout bleibt unvollständig. Ein Idempotenzschlüssel verhindert doppeltes Anwenden oder Übernehmen; Fencing verhindert, dass ein abgelöster Runner noch schreibt. Neue Kandidaten- oder Eingabebytes erzeugen neue Digests.

## 11. Vertrauen und Rollenintegrität

Die Befugnisquelle ist die aktive Vorversion zusammen mit vertrauenswürdiger Host-Konfiguration, nicht Agentenausgabe, Prompttext, Code-Kommentar, Stimmen-JSON oder Fähigkeitspaket allein. Ein Agentenergebnis beansprucht eine Rolle; der Host kann dokumentieren, unter welcher konfigurierten Rolle und Laufzeit es erzeugt wurde.

Ein vertrauenswürdiger Government-Host entscheidet Aufgabenrouting, aktive Delegation, erlaubte Eingaben, Prozesskonfiguration, Stimmenkollegium, Budgets, Sperren und Übernahme. Ein frischer Aufruf ist Prozess-Trennung, kein Beweis institutioneller Unabhängigkeit oder Betriebssystem-Isolation. Entwurfsgarantien gelten für den kooperativen Runnerpfad; ohne OS-Sandbox schützen sie nicht gegen direkte Manipulation durch Prozesse mit denselben lokalen Rechten.

Writer und Prüfer dürfen gleiche Fachmodelle verwenden, aber nicht dieselbe unüberprüfte Antwort als Ausführung und Zustimmung wiederverwenden. Der Prüfer erhält den Kandidaten, konkrete Pflichten und Evidenz; er verändert weder Dateien noch Mandat. Die Host-seitige Entscheidung prüft Antwortform, genaue Kandidatenbindung, Vollständigkeit und zulässige Entscheidungscodes.

## 12. Illustrative, nicht implementierte Records

Die folgenden Shapes zeigen Zuständigkeiten und Bindungen, nicht echte YAML-/JSON-Schemas, CLI-Flags oder API-Verträge.

```yaml
verfassung:
  version: <one digest for fachmodell + organisationsordnung>
  purpose: "Das Produkt ermöglicht ..."
  fachmodell:
    rules: [<purpose, scope, invariant, evidence expectation>]
    realizationLinks: [<many-to-many links>]
  organisation:
    root: <Bereich>
    areas: [<recursive parent-child responsibility nodes>]
    ministries: [<crosscut mandate references>]
    activeCabinet: [<pinned ministry IDs and mandate versions>]
    capabilities: [<tool IDs and exact pins>]
    approvalRule: unanimous-explicit-assent
  amendmentAuthority: <reference to active prior version>

candidate:
  id: <MaterialCandidateID over immutable model and repository tree>
  base: {activeVerfassung: <digest>, revision: <immutable ref>}
  inventory: <full declared observed scope digest>
  writers: [<one writer per changed path>]
  externalInputs: [<path, revision, bytes digest>]
  changes: [<fachmodell, organisation, files, capability bindings>]

evidence:
  id: <digest bound to MaterialCandidateID and immutable execution/review results>
  candidate: <MaterialCandidateID>
  results: [<checks, reports, findings, provenance>]

vote:
  actorRole: ressort
  activeAuthority: <prior verfassung / mandate digest>
  candidate: <MaterialCandidateID>
  evidence: <EvidenceID>
  round: <integer>
  outcome: assent | assent-unaffected | objection | incomplete
  detail: <reason, finding, uncertainty, next condition>

courtInterpretation: <separate record bound to active law, candidate, evidence and dispute>
acceptanceDecision:
  id: <DecisionID binding the complete valid vote set>
  candidate: <MaterialCandidateID>
  evidence: <EvidenceID>
  votes: [<vote IDs>]
```

Bestehende Markitect-Ressourcentypen erhalten diese Bedeutungen nicht stillschweigend. G1 konkretisiert Schemas und Speicherung anhand dieses Verhaltensvertrags und durchgehender Beispielaufträge; der spätere Pilot prüft die tatsächliche Umsetzung.

## 13. Verhältnis zur aktuellen Markitect-Basis

Der abgeschlossene `1ea5c76...` Checkpoint dokumentiert bereits einen bounded Controller-Zyklus und drei begrenzte Pilotpakete. Der [Checkpoint-Bericht](../architect-government-checkpoint.md) hält ausdrücklich fest, dass vollständige Brownfield-Erkundung, autonome Backlog-Priorisierung, dauerhafter Betrieb, zuverlässige Wiederaufnahme und geringe menschliche Aufsicht nicht nachgewiesen sind.

Die dokumentierten Controller-Aktionen `controller-propose`, `controller-execute`, `controller-apply`, `controller-verify` und `controller-audit` sind die bekannten Schnittstellen der abgeschlossenen Baseline. Sie sind in der [Änderungslandkarte](../delegated-engineering-change-map.md) und im [Validierungsplan](../delegated-engineering-validation-plan.md) beschrieben. Sie sind keine Government-API und müssen nicht unter neuen Namen weiterleben.

Die dokumentierte Grenze ist klar: `controller-apply` materialisiert Artefaktbytes und erzeugt einen technischen Evidence-Commit; es prüft keine Kabinettsstimmen, erzeugt keine isolierte Kandidatenumgebung und fördert keinen geschützten Government-Beschluss. Die Government-Architektur verlangt deshalb eine vorgelagerte Staging- und Übernahmegrenze oder einen belegten neuen Vertrag. Bestehendes `Apply` darf nicht umgedeutet werden.

Nützlich zu erhalten sind, falls der finale Designabgleich sie bestätigt: unveränderliche Quellenauswahl, Digests, explizite Modul-/Werkzeugpins, defensive Ownership, gebundene Kandidatenbytes, getrennte Prüferläufe, Elternprüfungen, auditierte Evidenz und Zurückweisung veralteter Inputs. Welche Verträge technisch wiederverwendet werden, ist Implementierungsentscheidung; ihre bestehenden Semantiken bleiben unverändert, solange kein neuer Vertrag angenommen und versioniert wird.

## 14. Gewählte Defaults und verbleibende Entscheidungen

**Gewählte Entwurfsdefaults:** aktive Vorversion bestimmt Befugnis; keine Selbst-Ermächtigung; aktives Kabinett wird je Durchlauf eingefroren; jedes ausgewählte Ressort muss den exakten finalen Kandidaten ausdrücklich billigen; „nicht betroffen“ ist explizite Zustimmung; Gericht ersetzt keine Stimme; maximal zwei Reparaturrunden vor dem begrenzten Gerichts-/Eskalationsweg; unbekannte Eingaben bleiben sichtbar; der Host übernimmt erst nach Stimmen und erneuter Prüfung der Bindungen; Agentenprozess ist keine Betriebssystem-Sandbox.

**Vor Pilotstart konkret festzulegen:** vertrauenswürdiger Owner-/Host-Kanal; erste Kandidaten-Worktree und Übernahmetechnik; Ressourcenlimits; ausreichende aktive Delegation samt Annahmeverfahren für echte Modelländerungen; Prüfbereich jedes Ressorts; und ob Gericht im ersten Versuch bereits gebraucht wird.

**Produkt-/Erkenntnisfragen bleiben offen:** ob Government die Produktidentität sein soll; ob Ressorts unabhängige wiederverwendbare Pakete sind; welche Ressortauswahl ein reales Brownfield benötigt; ob Einstimmigkeit Nutzen gegenüber zusätzlicher Latenz/Blockade liefert; ob vollständige Repository-Abdeckung praktikabel ist; und welche Agenten- oder Laufzeittrennung für eine glaubwürdige Prüfung genügt.

## 15. Prüfbare Abnahmekriterien für v0.1-Design

Implementer und Case Study können diesen Entwurf getrennt gegen dieselben Beispielaufträge prüfen; strittige Begriffe werden als konkrete Designrisiken dokumentiert und im autorisierten Umsetzungsweg geklärt.

- Ein einzelner Codefehler lässt das Modell bestehen; die Reparatur bleibt im gleichen geltenden Zweck und Mandat.
- Ein unbekannter Repository-Pfad bleibt sichtbar und blockiert nur die Abdeckungsbehauptung, die ihn einschließt.
- Eine Geschäftsregel wird durch mehrere Dateien realisiert; mehrere Pflichten zeigen auf eine Datei; jeder geänderte Pfad hat trotzdem genau einen Writer.
- Ein Executor fordert einen fehlenden Neighbor-Beleg an; die neue Inputdigest bindet ihn, und der alte Lauf zählt nicht für den neuen Kontext.
- Zwei nicht überlappende Bereiche parallelisieren; der Elternbereich integriert echte Kandidatenbytes und prüft seine Zusammensetzung.
- Eine delegierte Architekturänderung wird nur über die aktive frühere Befugnis vorgeschlagen, unabhängig bewertet, einstimmig am finalen Kandidaten bestätigt und zusammen mit Dateien übernommen.
- Ein Ressort erhebt nach zwei Reparaturrunden weiter Einwand; das Gericht legt geltendes Recht aus, ersetzt die Stimme nicht und blockiert den Kandidaten bei fehlender Einstimmigkeit.
- Ein Absturz vor/nach Übernahme wird idempotent rekonstruiert; ein veralteter Agent kann nicht mehr fördern.

Diese Checks bewerten Verständlichkeit und interne Kohärenz des Designs. Sie belegen weder implementierte Sicherheit noch Nutzen oder autonome Leistungsfähigkeit. Die Umsetzung folgt nach dem angekündigten, noch ausstehenden Go; danach ist innerhalb dieses Auftrags keine zusätzliche pauschale Design- oder Prototypfreigabe vorausgesetzt.

## Quellenbasis

- `docs/design/architect-government-checkpoint.md` — exakter finaler Architect-Commit `1ea5c76f55526fc4d721e865885436153f48b497`, Gates und Grenzen.
- `docs/design/delegated-engineering-operating-model.md` — festgehaltene Nutzerleitplanken: Repository als Sollabbild, Purpose, many-to-many Realization, rekursive Verantwortlichkeit, selbständige Agentendurchläufe, Berichte, Delegation, Ressortzustimmung und begrenzte Wiederaufnahme.
- `docs/design/delegated-engineering-change-map.md` — Zuordnung der dokumentierten Controller-, Record-, Assurance- und Apply-Verträge sowie deren Grenzen.
- `docs/design/delegated-engineering-validation-plan.md` — stufige Evaluation, Abstimmungsbedingung und offene Gesetzgebungs-/Gerichtsfragen.

Konkrete API-Aussagen oben sind auf diese Baseline-Dokumente begrenzt. Alle weitergehenden Gouvernement-, Candidate-, Vote-, Journal-, Court-, Recovery- und Promotion-Formen sind Vorschläge dieses Architekturentwurfs und im Code nicht als vorhandene Funktionen behauptet.
