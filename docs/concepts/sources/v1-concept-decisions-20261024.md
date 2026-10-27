# Owner decisions: V1 concept 1.0

Date: 24 October 2026. Source: the owner's direct messages in a Claude Code session. The owner statements below are verbatim. The assistant proposals are quoted verbatim from the assistant's reply and labeled as such. Recorder's notes are marked.

## Owner request

The owner supplied the consolidated concept v0.18 ([verbatim](v1-concept-v018-20261024.md), SHA-256 `7a1dee4fce1c5d0edeb24d0c5c45b496e0671cbdf4264be0636405536ce82ba0`) and wrote:

> Prüfe den aktuellen Stand von Markitect gegen das Konzept hier. Unser Ziel ist es dieses Konzept noch einmal final zu reviewen und zu verfeinern und anschließend festzuhalten. Dann wollen wir die v1 stabil und sauber implementieren. Die höchste Priorität liegt auf Qualität, Sauberkeit und exzellenter Architektur. Wir präferieren generell Simplicity. Bevor du anfängst etwas zu implementieren gib mir dein Assessment und deinen Plan wie wir weiter vorgehen sollten

English translation: "Check the current state of Markitect against the concept here. Our goal is to review and refine this concept one final time and then record it. Then we want to implement V1 stably and cleanly. The highest priority is quality, cleanliness and excellent architecture. We generally prefer simplicity. Before you start implementing anything, give me your assessment and your plan for how we should proceed."

Recorder's note: the assistant compared v0.18 with Main `334e7c3133ff294c0b827e623d80185e3d3ac9af` and the supplied [static audit](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/concept-conformance-audit-20261024.md). It confirmed the audit's A1, A2, A3, A5 and R2 observations in source and found R1 already fixed by PR #672.

## Assistant proposals (quoted)

> **Was ich für v1.0 am Konzept ändern würde:**
> - **Kapitel 28 schließen** statt offen lassen. Meine Vorschläge:
>   - A1: Fragen nur an Vater und Kinder.
>   - A2: Eine ungültige Abgabe geht mit Diagnose an den Vater; Approve und Integration bleiben gesperrt. Optional bekommt das Kind ein `check`-Tool zur Selbstprüfung.
>   - A3: Ein Urteil gilt für Auftrag plus Kandidat. Eine bloße Antwort braucht kein neues.
>   - R3: erledigt sich mit A2/A3 über einen klaren Manager-Zustand.
>   - A4: Native Base-/Developer-Instruktionen sind die V1-Form des Harness.
>   - A5: Die Ergebnisprüfung wandert in den Run-Vertrag.
>   - A6: feste Mandatsmenge in V1.
>   - R2: Notes seitenweise und stabil sortiert.
> - **Neu aufnehmen:**
>   - den kleinen Werkzeugsatz der Manager als Vertrag; dass er fehlt, hat A3 erst möglich gemacht,
>   - die Regel „Agenten bekommen Aufgaben, nicht Prozess“,
>   - den ausdrücklich benannten Providerumfang.
> - **Kürzen:** viele Verneinungen und Historie raus; die Historie gehört ins Register.
>
> **Plan:**
> 1. Konzept v1.0 im Repo festhalten: Quelle plus Register-Eintrag. Dazu AGENTS.md und den Plan auf die geltenden Regeln kürzen, das Evidenz-Log nach `docs/validation` verschieben und das Selbstmodell anpassen.
> 2. A1, A2, A3 und R3 als ein zusammenhängender Schnitt mit einem Writer, getestet an den Gegenbeispielen aus §14.3.
> 3. Parallel in anderen Dateien: A5, R2 und die Bereinigung beim Provider.
> 4. Native Neuqualifikation mit Codex. Die alten Nachweise gelten für den geänderten Vertrag nicht mehr. Dabei wird auch der noch offene Zweig nachgewiesen: Verify-Reject → frische Korrektur → Publication über den öffentlichen Apply.
> 5. Gates, Doku und eine Provider-Matrix mit den bekannten Grenzen, danach deine Abnahme.
>
> **Eine Produktentscheidung brauche ich für v1.0:** Welche Provider gehören in V1? Ich empfehle V1 nur mit Codex. Claude käme als nächster Schritt auf dem neuen Session-Vertrag, und den heutigen Claude-Code würde ich bis dahin entfernen. Oder soll Claude Teil der V1-Abnahme sein?

Recorder's note: the quoted proposals are in German. In English they propose: questions only to parent and children (A1); an invalid submission goes to the parent with its diagnostic, approval and integration stay blocked, and the child optionally gets a `check` tool for self-checking (A2); a judgment applies to assignment plus candidate and a mere answer needs no new one (A3); R3 is resolved together with A2/A3 through a clear Manager state; native base/developer instructions are the V1 harness form (A4); result validation moves into the Run contract (A5); a fixed set of mandates in V1 (A6); Notes page by page and stably sorted (R2); add the small Manager tool set as a contract (its absence made A3 possible), the rule "agents get tasks, not process" and an explicitly named provider scope; remove many negations and the history, which belongs in the register. The plan has the five steps of the [delivery plan](../../implementation-plan.md#steps). The provider question recommends a Codex-only V1 with Claude as the next step on the new session contract, removing today's Claude code until then, and asks whether Claude should instead be part of V1 acceptance.

## Owner decision (verbatim)

> Ja, starte mit Schritt 1: Konzept v1.0 mit deinen Entscheidungsvorschlägen schreiben und im Repo festhalten.

English translation: "Yes, start with step 1: write concept v1.0 with your proposed decisions and record it in the repository."

## Recorded meaning

Recorder's interpretation: the owner accepts the proposals above as V1 product decisions, recorded as D10–D18 in [concept 1.0](../v1-concept.md#27-decisions):

| ID | Decision |
|---|---|
| D10 | Questions go only to the direct parent or direct children; other coordination goes through the Managers in between. |
| D11 | Every Implement child candidate reaches the parent with its deterministic check result; an invalid candidate cannot be approved or integrated; a child can use a check tool for optional self-correction. |
| D12 | A judgment belongs to the assignment and the assessed state, not to a message; the host keeps an explicit assignment state per Manager. Closes A3 and R3. |
| D13 | Agents get tasks, not process; a fixed small Manager tool set with one tool per purpose. |
| D14 | The V1 harness uses native base/developer instructions. Recorder's note: "skills or loadable context only on demonstrated need" restates v0.18 section 15.4; the generic AGENTS.md representation planned in v0.18 section 15.5 stays planned. |
| D15 | Validation of closed Run results belongs to the Run contract. |
| D16 | V1 has exactly three mandates. |
| D17 | Notes are read in bounded, stably ordered pages within the transport limit, without silent truncation. The concrete interface follows a controlled reproduction of the size conflict, as v0.18 R2 asks. |
| D18 | V1 acceptance covers Codex; Claude follows on the stabilized contract; no non-functional Claude path in the product until then. |

The owner also accepts the five-step plan and starts step 1. The concept text may be condensed as proposed.

Recorder's note on D18: the assistant phrased the provider scope as a question with a recommendation. The owner's reply accepts "deinen Entscheidungsvorschlägen" without answering the question separately. It is recorded as accepted with the recommendation, and the assistant's report states this so the owner can revise it.

## Clarifications after the Astra review

The owner forwarded an independent review of concept 1.0 at Main `c9f7cb45`, written by Astra, with the words:

> Hier ist das Urteil von Astra:

English translation: "Here is Astra's verdict:"

Recorder's note: only this summary of the review is kept; the review text itself is not stored in the repository. The review recommends concept 1.0 as the working basis and asks for three precisions before implementation. They were incorporated as clarifications of the accepted decisions, without new product semantics:

1. D12: a message that only handles questions or answers needs no judgment, also before the first judgment of the assignment. The end of a provider response is not a completion attempt; completion validity is checked when the Manager judges and before the host relies on an earlier judgment, for example at Root completion. This restates v0.18 §14.2, which forbids blanket forcing of an identical new decision at every message end.
2. D11: invalidity of a child candidate refers to the checks that apply to that fixed state; integration still outstanding elsewhere is not invalidity, and checks that did not run are not reported as passed. This follows v0.18 §13.1 and §13.3.
3. Plan: step 2 includes one bounded native contract proof before the changed session contract is fixed, as v0.18 §24.2 and the architecture's early native gate require; the full requalification stays in step 4.

The review also noted that an explicit assignment state is a means, not proof of quality, and a stale reference to two provider adapters in the architecture; both were corrected.

This record grants no new execution authority. The [standing evidence authorization](v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization) is unchanged. Release and deployment remain excluded.
