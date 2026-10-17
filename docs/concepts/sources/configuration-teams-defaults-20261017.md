# Project teams and product defaults — 17 October 2026

Recorder's note: direct owner discussion in the Codex chat **Config**, thread `01a14ac0-a822-72c1-aa65-3ebd0b1cc613`. The quoted owner messages below retain their original wording. The assistant developed the examples and numeric proposal in response. Acceptance concerns the design; no source implementation or runtime result was established. The final instruction authorizes publishing the concept on Main, where other contributors retrieve it; it does not require messaging other chats.

## Owner requirements

> Ich möchte pro Manager Modell und Effort, eigene Anweisungen und die Reviewer konfigurieren können. Wie würden wir das mit sinnvollen Projektdefaults möglichst einfach aufbauen?

> Das gefällt mir. Ich möchte aber auch mehrere Reviewer pro Manager konfigurieren können, jeweils mit eigenem Modell, Effort und Anweisungen. Wie können wir das genauso einfach mit Projektdefaults lösen?

> Wer übernimmt diese unabhängige Klärung konkret? Brauchen wir dafür einen zusätzlichen Reviewer als Schiedsrichter, den ich ebenfalls pro Projekt und Manager konfigurieren kann?

> Das klingt gut. Zeig mir bitte eine vollständige Beispielkonfiguration für die Bibliothek mit Projektdefaults, mehreren Reviewern und dem Schiedsrichter sowie gezielten Overrides für Lending.

> Die Teamkonfiguration gefällt mir. Müssen die ganzen Limits und Budgets in jedem Projekt stehen, oder kann Markitect dafür sinnvolle Defaults liefern, sodass ich nur Abweichungen konfigurieren muss?

## Accepted proposal and publication instruction

Recorder's note: the assistant proposed `defaults.manager`, named `defaults.reviewers` and `defaults.reviewArbiter`, with exact Package overrides. Model/effort replace individual fields; instructions append. At least one independent reviewer remains active. The arbiter assesses an explicitly disputed finding independently, cannot change intent or implementation, and does not form a recursive appeal hierarchy. One submitted candidate is rejected at most once per review purpose regardless of reviewer count.

The simplified Library file omitted all budget blocks. Proposed assignment defaults were 3 runs, 80 starts, 160 steps and 8h active time; ceilings were 5, 160, 320 and 16h; the included completion reserve was 24 starts, 40 steps and 2h. The step default/ceiling were 10m/20m, control timeout 2m, rejection default/ceiling 3/3, context 400000 bytes, conversation limit 40 steps and proven-non-dispatch retries 2. Creation roots were `[realization]`, instruction lists and selected owner approvals empty, and batched no-change reviews false. These were expressly starting values, not measured sufficiency or monetary ceilings. The current [configuration owner](../../design/canonical-model/manager-apply-configuration.md) defines the accepted effects and avoids copying a second specification here.

The owner then said:

> Das passt. Gib mir einen Prompt für den Architect, der unsere Teamkonfiguration und Produktdefaults zusammenfasst und die nötigen Änderungen am bestehenden Config-Vertrag klar abgrenzt.

After receiving that handoff, the owner instructed:

> Arbeite es einfach ins Konzept in main ein. Jeder soll sich dort das Ergebnis abholen

Recorder's interpretation: this is explicit authority to integrate this bounded concept change despite the earlier concept freeze and Architect reservation. It is not authority to expand unrelated process design or run Docker, providers, credentials, Apply, comparisons or studies. Record required implementation follow-ups without claiming they are delivered.
