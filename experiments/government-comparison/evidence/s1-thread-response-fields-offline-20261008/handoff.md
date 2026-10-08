# Offline-Übergabe: vier Antwortvergleiche

Schema und Gate widersprechen sich nicht: Das Schema erlaubt mehr Werte, als
die konkrete Anfrage akzeptiert. Das Gate fordert anschließend unverändert
exakte Gleichheit für `cwd`, `model`, `modelProvider` und `approvalPolicy`.
Die Pfadbeschreibung im Schema ist keine Prüfung der Dateisystemidentität.
Welche Felder im abgeschlossenen R2-Lauf abwichen, bleibt unbekannt.

Die additive Offlineableitung erfasst nach erfolgreicher Schemaprüfung alle
vier Vergleiche mit Feldname, erwartetem/tatsächlichem JSON-Typ und Gleichheit.
Hinzu kommt ausschließlich `cwdLexicallyEquivalent`: ein Vergleich der
Windows-Pfadschreibweisen mit Separatorersetzung und Unicode-Casefold, ohne
Auflösung, Dateizugriff oder Änderung der Annahmeregel. Der Ein-Antwort-Consumer
bleibt auch bei einem Fehler terminal und hält diese Metadaten abrufbar.
Tatsächliche Werte, Pfade, Raw-RPC und Inhaltsfingerprints werden nicht ausgegeben.

Ein späterer separater Grant könnte genau einen `thread/start`-Versuch mit den
bisherigen fünf Methoden, denselben Eingaben und unveränderten Stopregeln
erlauben. Dafür wäre ausschließlich dieses begrenzte Vergleichsobjekt in den
terminalen Metadaten zu ergänzen. Ein Mismatch bliebe terminal vor jedem Turn;
es bräuchte keinen privaten Vergleichsauszug und keine automatische Anerkennung
lexikalisch ähnlicher Pfade. Dieses Paket enthält keine Laufgenehmigung und
reserviert keinen Slot.

Jetzt erfolgen ausschließlich Quell-/Schemaanalyse und synthetische Prüfungen.
Die tatsächliche Historie bleibt bei neun nativen Bäumen, sechs Actorreservationen
mit fünf historischen Modellversuchen und acht CLI-Metadatenaufrufen. Die 53331
bekannten Tokens, unbekannte Gesamtnutzung und native Werte 15/16/13/2250 bleiben
unverändert. Alte Grants, Artefakte und Studienpins werden nicht verändert.
