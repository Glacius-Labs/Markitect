# Start-stand drift is repaired, not refused — owner, 19 October 2026

Recorder's note: owner answer in the Claude session that tests and stabilizes Apply. A drift scenario on the library (RA-R1, Main `cc11ff5a`) started from a published A1 with an unregistered file `realization/members/notes.txt`. `markitect apply` stopped in preparation before any Agent ran, with diagnostics that named the registry shards (`registry.source_uncaptured`) instead of the file. The concept then said: "Preflight blocks on missing/duplicate registrations or ambiguous ownership." [DEC-062](../register.md#dec-062-start-stand-drift-is-the-owners-first-work) records the decision.

## The question

> Wie soll Apply mit einer nicht registrierten Datei im Ausgangsstand umgehen (z. B. jemand legt realization/members/notes.txt von Hand an)? Heute bricht Apply schon in der Vorbereitung ab, bevor ein Agent läuft, mit einer irreführenden Meldung. Das Konzept sagt bisher: „Preflight blocks on missing registrations“.

Options offered:
- **Als Drift reparieren (Recommended):** Apply startet. Unter dem Layout hat jede Datei einen Owner; der bekommt die Abweichung als erste Aufgabe (registrieren oder entfernen), Verify prüft wie üblich. Passt zu DEC-058 („Abweichungen im eigenen Bereich korrigieren“). Nur echte Kontrollfehler (ungültiges Layout, geänderte Modell-Dateien) brechen weiter ab.
- **Weiter abbrechen:** Wie heute: Der Mensch bringt die Registry vorher in Ordnung. Ich verbessere nur die Fehlermeldung, damit sie die Datei und den Grund nennt.

## The answer

> Als Drift reparieren (Recommended)

Recorder's note: the implementation keeps a file that no Manager owns under the layout (`layout.no_owner`) as a refusal. Only the owner can change the persistent layout, and removing such a file is no repair.
