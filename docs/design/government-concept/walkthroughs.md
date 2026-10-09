# Durchgängige Modellfälle

**Entwurfsbeispiele, keine ausgeführten Produktfälle.** Namen wie `M0`, `P1`, `D1` und `A1` sind anschauliche Platzhalter für die vollständigen Identitäten aus [Verträge](contracts.md), keine tatsächlichen Commits oder Belege. Die Beispiele verwenden genau deren CasePhase, ControlStatus und EvidenceOutcomes. Ihre Projektregeln dienen dem Verständnis; sie werden keinem adopting project automatisch vorgeschrieben.

## Ausgangsordnung des Beispielprojekts

Ein Shop besitzt Orders, Inventory und Payment als fachliche Managerbereiche; Commerce ist ihr gemeinsamer Vorfahr. Engineering besitzt gemeinsame technische Standards. Der Präsident hat das Ausgangsmodell `M0` angenommen. Die projektinterne Grundordnung schützt nachvollziehbare Geldbewegung, unveränderte Ausführungsbefugnis und begrenzte Ausgaben. Commerce darf eine ausdrücklich benannte Klasse neuer Bestell- und Stornierungsabläufe annehmen, solange diese Schutzregeln, externen Zusagen und reservierten Ziele unverändert bleiben. Jeder materielle Vorschlag braucht einen unabhängigen Modellreview und die durch Impact erforderliche Anhörung.

Das ist eine **echte** Delegation zur Solländerung. Die zulässige Klasse umfasst neue fachliche Regeln, nicht bloß Tippfehler. Der Manager kann sie vorschlagen; die Annahmefunktion wird durch ein vorab legitimiertes Mandat ausgeübt. Der Host entscheidet deterministisch über erklärte Referenzen, Scope, Digests und Befugnisgrenzen. Fachliche Zwecktreue und unbeabsichtigte Folgen werden zusätzlich durch unabhängige Prüfung beurteilt.

## Fall 1: Eine gewöhnliche neue Stornierungsfunktion

Der Auftrag lautet: „Kunden sollen noch während packing stornieren können.“ Bisher erlaubt das Modell Stornierung nur vor diesem Zustand.

| Schritt | Ergebnis und verantwortliche Funktion |
|---|---|
| Aufnahme | Die Geschäftsstelle friert den Originalauftrag ein und ordnet ihn als Solländerung ein. Ein bisher verbotener Übergang ist kein Implementierungsbug. |
| Zuständigkeit | Orders besitzt den Bestellablauf, Inventory die Reservierungsfreigabe, Payment die finanziellen Folgen. Commerce koordiniert; kein Integrationsministerium übernimmt seine Verantwortung. |
| Entwurf | Die Owners erstellen ein gemeinsames begrenztes Delta: Stornierung bis zum belegten Versandpunkt, einmalige Freigabe der Reservierung und nachvollziehbarer Rückerstattungsstatus. Offene externe Zahlungsannahmen bleiben benannt. |
| Feste Prüfung | Ein isolierter ProposalCommit `P1` erhält Struktur-/Impactprüfung und unabhängigen Review gegen Originalanliegen, `M0`, beide Modelldigests und die alte Befugnis. Er ist kein angenommenes Gesetz. |
| Anhörung | Betroffene Manager werden gehört. Sicherheits- oder Nutzerressorts wirken mit, wenn deren geltendes Prüfmandat diese Änderung erfasst. Ein zusätzlicher Fachreview muss eine konkrete Folge oder begründete Nichtbetroffenheit liefern. |
| Beschluss | Die delegierte Gesetzgebungsinstanz prüft die geschützten Zahlungsregeln und nimmt genau `P1` innerhalb der alten Klasse an. Ein beratender UX-Einwand bleibt mit der Begründung dokumentiert. |
| Verkündung | Der Host reserviert den Vorgang, prüft Frische, schreibt das beschlossene Delta und erzeugt den tatsächlichen Modellcommit `M1`. Nachfolgender Receipt `A1` und Cursorfortschritt bestätigen die Annahme. |
| Vollzug | Die Manager arbeiten auf dem angenommenen Modell: Plan, Kandidaten, eigene Elternintegration, unabhängige Realisierungsprüfung, Verify und guarded Apply. |
| Rechenschaft | Das Präsidentenbriefing nennt Zweck, Modellrevision, fortbestehenden Einwand und tatsächliche technische Ergebnisse. Menschliche Abnahme wird nur ausgewiesen, wenn sie wirklich stattgefunden hat. |

Der Präsident braucht für diesen Routinefall keinen Einzelentscheid. Seine bisherige Delegation ist die Legitimation. Entsteht während der Umsetzung ein Race bei der Reservierungsfreigabe, repariert Inventory gegen dasselbe Soll. Soll wegen technischer Schwierigkeit künftig doch mehrfach freigegeben werden dürfen, ist das ein neuer normativer Fall; der Executor ändert den Maßstab nicht selbst.

## Fall 2: Ministerien streiten über einen Datenexport

Ein Work Item verlangt „alle Kundendaten für Support exportieren“. Das Nutzerressort erwartet schnellere Bearbeitung. Das Sicherheitsressort erhebt einen bindenden Einwand und verweist auf eine bestehende Projektpflicht: Zugriff nur für bezeichneten Zweck und Empfängerkreis.

Zunächst fehlen Datenklassen, Empfänger, Berechtigungen und der tatsächliche Supportzweck. `ControlStatus: awaiting-evidence` verhindert eine Entscheidung aus bloßen Meinungen. Eine begrenzte Untersuchung klärt diese Fakten. Die Instanz benutzt die Projektpflicht; das Beispiel entscheidet keine allgemeine rechtliche Zulässigkeit eines Datenexports.

Danach bleiben zwei Varianten: ein eng begrenzter Datensatz für eine befugte Supportrolle oder ein unbeschränkter Export an einen externen Empfänger. Die erste Variante könnte die geltende Pflicht erfüllen. Der behauptete Einwand gegen **jede** Exportmöglichkeit ist nun eine Auslegungsfrage. Das Gericht prüft Scope, Norm und belegte Fakten, weist die pauschale Behauptung gegebenenfalls zurück und bindet sein Urteil an genau diesen Vorschlag. Es hat kein neues Datenschutzgesetz beschlossen.

Die Gesetzgebungsinstanz trifft anschließend einen eigenen frischen Annahmeentscheid für die zulässige Variante. Möchte der Auftraggeber dennoch den unbeschränkten Export, braucht er einen anderen Normvorschlag beim Standardowner und die ausdrücklich zuständige Änderungskompetenz. Ist diese Grundsatzfrage reserviert, erhält der Präsident eine Vorlage mit Konsequenzen, Optionen und fehlenden Belegen. Weder eine Ministeriumsmehrheit noch das Gericht kann diese Kompetenz erzeugen.

## Fall 3: Höhere Instanz und neue Priorität

Architektur bevorzugt einen gemeinsamen Dienst, Wartbarkeit eine lokale Lösung. Beide erfüllen die harten Regeln. Die geltende Projektpräferenz priorisiert Verständlichkeit vor maximaler Skalierung; die lokalen Ziele und belegten Lastannahmen werden dagegen geprüft.

Ein befugter Entscheider kann die lokale Lösung innerhalb dieses Spielraums wählen und begründen. Ein Appeal behauptet, eine erhebliche gemeinsame Lastannahme sei übersehen worden. Die nächste unabhängige Instanz prüft den konkret bezeichneten Evidenzfehler. Sie bestätigt, ändert die zulässige Wahl oder verweist zur neuen Untersuchung zurück. Der interne Instanzenzug ist endlich; eine stärkere Modellinvocation allein ist keine höhere Gerichtskompetenz.

Sollen skalierende öffentliche Dienste nun generell Vorrang vor Einfachheit haben, entsteht eine **neue Zielpriorität**. Das ist Gesetzgebung und möglicherweise ein vorbehaltener Präsidentenentscheid. Ein alter Präzedenzfall für interne Prototypen rechtfertigt diesen Wechsel nicht automatisch. Seine Gründe bleiben auffindbar, ihre Anwendungsbedingungen unterscheiden sich jedoch.

## Fall 4: Widerruf nach Commit und vor Annahme

Ein Beschluss `D4` ist unter PolicyEpoch `p7` und AuthorityEpoch `a12` gültig. Der Host hat die freigegebenen Bytes geschrieben; der tatsächliche Commit `M4` existiert. Receipt und Cursorfortschritt sind noch offen.

Der Präsident suspendiert die delegierte Annahmebefugnis über den bereits autorisierten Owner-Control-Kanal. Das dauerhaft bestätigte Control-Ereignis erhöht die AuthorityEpoch auf `a13`. Normtexte und PolicyEpoch `p7` bleiben unverändert. Beim nächsten Annahmeschritt passt `D4` nicht mehr zur wirksamen Kontrollgeneration. `M4` wird **nicht** angenommen, der alte Cursor bleibt auf `M3`, und abhängige Delivery wartet. Der Host wiederholt den Commit nicht.

Der Owner kann den geschriebenen Zustand zurücknehmen lassen oder eine neue frische Entscheidung veranlassen. Wiederfreigabe allein macht den alten Beschluss nicht automatisch aktuell. Erfolgt der Widerruf erst nach bestätigter Annahme, bleibt diese historisch gültig; neue Ausführungsschritte unter widerrufener Befugnis stoppen trotzdem. Liegt eine bereits angewendete externe Wirkung vor, ist eine neue Kompensation nötig. Diese Reihenfolge verhindert sowohl rückwirkende Erfolgsbehauptung als auch eine versteckte Wiederholung.

## Fall 5: Konkurrierende Entwürfe und verlorene Antwort

Zwei Fälle `C5a` und `C5b` beginnen bei `M5`. Ihre Dateien unterscheiden sich, doch beide beziehen sich auf dieselbe gemeinsame Rückerstattungsnorm. Vorbereitung und Review laufen parallel. `C5a` wird zuerst angenommen und bewegt den Cursor nach `M6`.

Der erwartete Anker von `C5b` stimmt nicht mehr. Der Host setzt `stale` und verlangt eine begründete Impact-/Frischeprüfung gegen `M6`. Ein blindes Rebase oder Dateidisjunktheit genügt nicht. Unbetroffene Stellungnahmen können mit expliziter Bindung weiterverwendet werden; semantisch betroffene Aussagen benötigen neue Prüfung.

Geht später eine Applyantwort verloren, führt die Akte `effect-unknown` mit dem ursprünglichen Deliveryvorgang. Recovery liest dessen Workspace, Turn, Journal und beobachtete Zielbytes. Ein fehlendes Resultat wird weder als Fehler noch als Erfolg gewertet. Solange nicht feststeht, ob die Wirkung bereits eintrat, startet kein zweiter Apply unter neuer CaseID. Das gemeinsame Budgetkonto bewahrt auch diese Versuche.

## Fall 6: Das Modell wächst und soll vereinfacht werden

Der Rechnungshof findet mehrere befristete Ausnahmen, ähnliche Qualitätsregeln und hohe Pflegekosten. Er meldet die Belege und schlägt eine Modellpflegeprüfung vor. Die Owners untersuchen ursprüngliche Schutzwirkung, betroffene Fälle und fortgeltende Ziele. Ein Vorschlag fasst zwei äquivalente Normen zusammen und entfernt eine überholte Ausnahme; andere Regeln bleiben wegen verschiedener Schutzbereiche bestehen.

Der Vereinfachungsentwurf durchläuft denselben unabhängigen Gesetzgebungs- und Annahmepfad wie eine neue Funktion. Gegenfälle prüfen insbesondere, ob das vermeintlich überflüssige Detail bisher einen seltenen Fehler verhindert hat. Das Audit setzt keine Regeln außer Kraft. Weniger Text, weniger Agentenfragen oder eine kleinere Regelzahl sind Anzeichen, aber keine ausreichenden Beweise für Modellqualität.

Dieser Fall schließt den Pflegekreislauf: Government darf aufräumen und korrigieren, ohne sein eigenes Regelwachstum oder seine Erfolgszahlen zum neuen Projektzweck zu machen. Der spätere Nachweis braucht Folgeänderungen und unabhängige fachliche Kriterien aus dem [Umsetzungsplan](implementation-plan.md).
