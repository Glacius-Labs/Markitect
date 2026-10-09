# Normaler App-Einstieg für Produkt-Praxisnachweise

Fallunabhängiger Koordinationsvertrag vom 9. Oktober 2026. Er enthält keine Studienaufgaben, Falllösungen, Bewertungsdaten oder zusätzliche Ausführungsquote. Der jeweilige konkrete bestehende Grant und seine echten Receipts bestimmen die Grenzen.

## Einstieg und Rollen

Der Operator benutzt die vorhandenen nativen `collaboration`-Tools unmittelbar. Ein frischer gemessener Actor wird mit `spawn_agent`, `fork_turns="none"`, `model="gpt-6-luna"` und `reasoning_effort="high"` gestartet. Unterstützte Felder sind `task_name`, `message`, `fork_turns`, `model`, `reasoning_effort`. Es gibt kein `cwd`-Argument und keine hier behauptete Python-/CLI-App-API. Der Auftrag benennt das eigene absolute Projektrepository und seine normalen installierten Anweisungen. Datei-/Shellaufrufe verwenden dieses Repository ausdrücklich. Das ändert keinen geerbten Kontext und stellt keine OS-Isolation her.

Der Actor erhält den normalen Arbeitsauftrag und seine echten Produktanweisungen. Markitects Modellpflege, Bereitschaft, Run, Review, Apply und Verify müssen durch seine tatsächlich vorhandenen unterstützten Mechanismen erfolgen. Direkte Implementierung als Ersatz für verweigerte notwendige Produktaktionen ist kein erfolgreicher Produktnachweis. Entwürfe und Exploration erhalten; den normalen Client-Freigabeweg nutzen oder den konkreten Blocker dokumentieren. Keine künstlichen Commithelfer, vorgegebenen Antworten, erfundenen Rollenentscheidungen oder Produktnachimplementierung durch den Operator.

Eine Fortsetzung adressiert ausschließlich den vom eigenen Toolaufruf zurückgegebenen Handle mit `followup_task`. `send_message` dient Ressourcen-, Umfangs- und Stopkoordination, nicht fachlicher Evaluatorhilfe. Eine weitere Aktivierung eines ruhenden Actors zählt als weiterer Start. Hilfs- und Produktrollen sowie Freigabe-/Reviewrollen erhalten dieselbe tatsächliche Luna-High-Bindung und zählen in den bestehenden gemeinsamen Rahmen. Ein benötigter interner regulärer Transport bleibt sichtbar; ein App-Primäractor beweist nicht automatisch, dass alle inneren Rollen ebenfalls App-Actors sind.

## Erfassung und Grenzen

Vor jedem Start/erneuter Aktivierung wird im eigenen bestehenden Ledger reserviert. Danach werden tatsächlicher Toolrequest/-result, eigener Handle, beobachteter Start/Ende und Fehler gebunden. Reservation, Taskerstellung, Modellturn, Toolausführung und Providerrequest sind verschiedene Ereignisse. Fehlgeschlagene Requests bleiben gezählt; eine Reservation wird nicht als belegte Modellrechnung ausgegeben. Keine zusätzliche Accounting-Plattform ist Voraussetzung. Vorhandene Erfassung erweitern, wenn eine tatsächlich genutzte unterstützte Rolle noch fehlt; unbekannte Starts ausdrücklich benennen statt Null oder Vollständigkeit zu behaupten.

Der aktuelle Design-Produktgrant bleibt maximal drei Praxisjobs, 7200 Sekunden aktive native Laufzeit und 128 Modell-/Rollenstarts einschließlich Helfern, Fehlern und Reviews. Zwei Jobs sind geschlossen: insgesamt zwei äußere Starts, null beobachtete innere Starts und 495.4756093 Sekunden äußere Laufzeit. Genau ein ursprünglicher Job bleibt. Keine neuen Quoten, keine Übernahme der anderen Grenzen eines Studienledgers. Vor diesem letzten Job eigenen Source-/Binary-/Konfigurationspin, Entry/Args, Rollenplan, Ledger, harte äußere/innere Fristen und Cleanupgrenze an Root melden. Aktive äußere/innere Zeiten und Überlappung getrennt berichten; unbekannte Sampling-/Token-/Serving-/Kostenwerte nicht aus Startzahlen ableiten.

Jeder echte Lauf hat vor Beginn eine absolute harte Frist und ausschließlich eigene bekannte Handles/PIDs/Prozesshandles. Warteaufrufe bleiben höchstens 60 Sekunden; vor und nach ihnen Zeit prüfen. An der Frist nur eigene Actors unterbrechen und bekannte eigene Shell-/Produktjobs beenden bzw. ihren terminalen Stand belegen. Turnunterbrechung beweist keine Prozess-/Providerbeendigung. Ungeklärtes Cleanup verhindert den nächsten Lauf. Keine globale Prozess-/Agenteninventur zur Eigentumssuche, keine fremden Kills, keine persistent abgekoppelten Jobs.

## Kontext und Ergebnis

Normale System-/Developer-/Tool-/Workspaceanweisungen, allgemeine App-Setup-/Memoryhinweise ohne fremde Falllösungen oder private Bewertung und legitime eigene Fortsetzung sind erlaubt. Actors lesen nicht gezielt andere Fälle, Chats oder Memoryquellen. Konkrete fremde Lösung bzw. private Bewertung ist terminal; ein bloßes History-/Memorylabel beweist diese Exposition nicht. Herkunft/Umfang/Unsicherheit erfassen, ohne private Inhalte in öffentliche Logs zu kopieren. Keine Sanitization oder harte Isolation behaupten.

Normale Schutzregeln bleiben aktiv. Kein Policybypass, keine globalen Freigabeänderungen, Transport-/Capabilityproben oder automatische Wiederholung aus diesem Dokument. Ein konkreter Ausführungsblocker bleibt ein Ausführungsbefund; fehlendes Apply bleibt fehlendes Apply. Eigenen Produktproof, deterministische Regressionen, unabhängigen Quellenreview, CI und gematchte Studienqualität getrennt ausweisen.

Root koordiniert schwere/native eigene Hostarbeit mit tatsächlichen Messzellen. Leichte Code-/Dokumentationsarbeit und gehostete CI können weitergehen. Die Produktreadiness ist keine Wartebedingung für einen unabhängigen Conventional-Start. Dieses Dokument erteilt keine Startfreigabe und ersetzt keinen Kandidatenpin.
