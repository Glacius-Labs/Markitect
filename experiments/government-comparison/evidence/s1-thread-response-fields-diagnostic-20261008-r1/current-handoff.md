# Terminaler Befund: exakter cwd-Vergleich

Der einmalige prospektive Versuch stoppte vor jedem Turn mit
`thread-response-identity-or-policy-mismatch`. Nur `cwd` war exakt ungleich;
`model`, `modelProvider` und `approvalPolicy` waren exakt gleich. Alle vier
Felder hatten erwarteten und tatsächlichen JSON-Typ `string`.
`cwdLexicallyEquivalent` war `true`: Separatorersetzung und Unicode-Casefold
lieferten im begrenzten Windows-Stringvergleich Gleichheit. Tatsächliche
Antwortwerte wurden nicht ausgegeben oder archiviert. Daraus folgt weder
Dateisystemidentität noch eine zulässige Lockerung des Equality-Gates.

Die Metadatengates waren positiv, alle fünf erlaubten RPCs vollständig
geschrieben. Das ursprüngliche Boundary-Receipt blieb erhalten und meldet
`responseValidated: false`. Nachgelagerte Sandbox-/Parent-/Threadprüfungen
wurden wegen des Equality-Stops nicht erreicht. Eine vollständig validierte
Thread-Antwort liegt nicht vor. Der Befund erklärt den alten R2-Lauf nicht.

Sechs neue synthetische Integrationstests bestanden in einem Aufruf. Zwei
Vorbereitungsberichtigungen und die reine EOF-Formatkorrektur sind dokumentiert;
Produktionslogik wurde dabei nicht repariert. Unabhängige Quell-, Bindungs-
und Nachreviews fanden keinen materiellen Blocker. S ist
`d2d3b2b43412b9043e628f8e2d253788f2bc5098`, F ist
`a3168b367c2e2345d9a185359768cb4ef3a21fe7`; 45 Quellen, 56 Freezeinputs und
acht öffentliche Receipt-Paare sind unverändert verifiziert.

Ein neuer nativer Baum und eine Diagnostikreservation bringen die Baumhistorie
von neun auf zehn. Keine neuen Actorreservationen, Turns, Tools, CLI-Metadaten,
Studienzellen oder Retries. Historisch bleiben sechs Actorreservationen mit
fünf Modellversuchen, acht CLI-Metadatenaufrufe, 53331 bekannte Tokens und native
15/16/13/2250; Gesamt-/aktuelle Nutzung und Billing bleiben unbekannt.
Der eigene private Auszug wurde ohne Inhaltsread, JSON-Parsing oder Hashing
gezielt entfernt. Grant geschlossen, Rest null, Slot ausdrücklich freigegeben.
Keine Folgeausführung; S1/Tools/ServingModel/OS-/Billing-/Methodenqualität offen.
