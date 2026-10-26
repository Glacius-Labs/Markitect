# Owner approval: product fixes before the next study

Date: 26 October 2026. Source: the owner's direct message in the coordinator's Claude Code session, and two earlier owner statements that the Architect session relayed to the coordinator the same day. The direct statement confirms the relayed ones. The owner statements are verbatim; translations and the recorder's notes are labeled.

## Owner statement in the coordinator session (verbatim)

> Ja, ich bestätige den Fix-Plan v2 (WP1–WP4) samt der Konzeptklarstellung WP4b/c. Starte S1–S3 jetzt; bis Phase C fertig ist nur CI, keine schweren lokalen Testläufe.

English translation: "Yes, I confirm fix plan v2 (WP1–WP4) including the concept clarification WP4b/c. Start S1–S3 now; until Phase C is finished, only CI, no heavy local test runs."

## Owner statements relayed by the Architect session (verbatim as relayed)

> Ja, die bekannten technischen Defekte von M (Provider-Retry, Apply-weites Nachrichtenbudget bei Frage-Relay, Uhr pro Nachricht bei Root, Änderungsminimalität bei unverändertem Modell) sollen vor der Studie behoben werden; danach F einfrieren.

English translation: "Yes, M's known technical defects (provider retry, Apply-wide message budget with question relay, per-message clock at Root, change minimality on an unchanged model) are to be fixed before the study; then freeze F."

> Ja: WP4b/c als kleine Konzeptklarstellung freigegeben. Übergib den Fix-Plan v2 an den Coordinator, damit er ihn in den Delivery-Plan aufnimmt und die Sessions S1–S3 startet.

English translation: "Yes: WP4b/c approved as a small concept clarification. Hand fix plan v2 to the coordinator so that the coordinator adds it to the delivery plan and starts sessions S1–S3."

## Recorded meaning

Recorder's interpretation:

- The four defects were found in the comparison study's Phase C on F_final: a transient provider overload ended the whole Apply (WP1); the per-message clock cut off an actively coordinating Root (WP2); question relay and a correction pair without a budget check exhausted the message allowance (WP3a, WP3c); and Implement changed an already verified, unchanged state (WP4). The Architect's fix plan v2 designs the fixes; WP3b (`delegate.after`) is deferred and changes nothing now.
- WP4b and WP4c are the approved concept clarifications, recorded as [DEC-073](../register.md#dec-073-change-minimality-on-an-unchanged-model) and written into concept §11.3, §18.3 and §19.4.
- Phase C stays untouched on F_final. Native controls of the fixes run after Phase C, outside any study. After the fixes merge, F is frozen (commit and image) before any material of the next study is shared. The [delivery plan](../../implementation-plan.md#product-fixes-before-the-next-study-owner-approval-26-october) tracks sessions S1–S3.
