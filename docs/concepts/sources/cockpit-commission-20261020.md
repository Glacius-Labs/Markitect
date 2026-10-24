# Cockpit commission of 20 October 2026

Source: direct owner messages in the Codex chat "Apply Run visualisieren", thread `01a1597c-31cf-7750-8170-25d2b450983d`, 20 October 2026. The quotations below are verbatim. Recorder context and interpretations are labelled separately.

## Visualization requests

> Welche Möglichkeiten haben wir, um zu visualisieren wie ein Apply Run läuft und was gerade passiert und wer was macht?

> Zeig mir mal einen interaktiven Entwurf der lokalen Webansicht mit Managerbaum, Zeitverlauf und Schrittdetails anhand eines beispielhaften Apply Runs. Noch keine Implementierung im Repository.

> Kannst du den Beispielrun automatisch abspielen lassen, sodass man sieht, wer gerade arbeitet und worauf die anderen warten? Weiterhin nur als Entwurf, keine Implementierung im Repository.

> Kannst du im Entwurf auch die Nachrichten und Übergaben zwischen den Managern sichtbar machen, sodass man erkennt, warum jemand wartet? Weiterhin keine Implementierung im Repository.

> Kannst du die Nachrichten während der Wiedergabe auch als animierte Pfeile zwischen den Managern darstellen? Weiterhin nur im Entwurf, ohne Änderungen im Repository.

Recorder context: an interactive HTML example was created outside the repository. Its Library Run, times, participants, messages, candidate changes and finding F17 were synthetic. Browser checks covered interactions and responsive presentation; they were not real Apply evidence or acceptance of its older process vocabulary.

## Cockpit direction and Claude assignment

> Stelle hierfür ein Konzept zusammen. Claude soll sich dann um die Umsetzung kümmern und die Darstellung designen und alles. Am Ende des Tages wollen wir hier eine Art Cockpit haben.

Recorder context: the assistant proposed a project entry point connected to Run inspection, organization, timeline, causal messages, owner questions and candidate-bound evidence. It proposed recorded inspection first, live observation next and existing execution actions through their controller afterward. It explicitly left architecture, technology and visual design to Claude and distinguished later personal AI/model-editing extensions.

## Documentation delivery

> Kannst du es auf main bringen und mir den Prompt für Claude geben?

Interpretation: this authorizes integrating the [cockpit concept](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/cockpit.md) and [Claude handoff](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/development/claude-cockpit-handoff-20261020.md) as documentation. It does not ask this chat to implement the product UI, start Claude, resume an old session or run providers. The requested Claude prompt supplies the later explicit assignment. The scoped cockpit direction refines the former broad deferral; it does not dispatch the other parked Government/personal-AI ideas or replace Apply's current behavioral owner.

Source reconciliation for the documentation: the originating checkout was on an older design branch. Fresh Main at `8eedb5385e03683d454a705ad7cdd9a3c589f299` contains the DEC-064 Pass implementation and current file-based runtime evidence. The concept therefore follows Plan/Prepare/Implement and independent Verify, Orchestrator/Runtime vocabulary and the actual read boundary, rather than turning the synthetic example's Explore/local-review phases or historical SQLite proposal into a new product contract.

## Working through the cockpit

Source: the owner's direct message and option choices to Claude Code in the cockpit implementation session (Claude Code session `40a900e8`), 20 October 2026, after the run inspector, live observation and cockpit entry were merged (Main `7cf8647e`). The quotation is verbatim.

> Idealerweise würden wir das Model über das Cockpit anpassen können durch Prompts, die dann direkt an die claude oder Codex cli gehen. Ausserdem sollten dann auch die Agenten Teams konfiguriert etc werden können. So könnten wir über das Cockpit arbeiten.

Recorder context: Claude answered that this is feasible, that the concept so far kept conversational model editing as a later extension and the delivery plan prioritizes Apply stabilization, and proposed configuration through forms first and agent-proposed model changes second, with the owner accepting every change. It named two guardrails: Agents run in Linux containers per DEC-037 rather than through the CLI on the host, and every session costs provider budget and needs the owner's approval with a limit („Freigabe samt Limit“). Claude asked two questions with fixed options. The owner chose:

- When to start: „Jetzt beginnen (Empfehlung)“, described as recording the decision in the concept and delivering team and execution configuration without an Agent first, then model proposals by an Agent, while the Apply hardening stream continues independently.
- Where the Agents run: „Container (Empfehlung)“, described as through the existing Claude and Codex adapters in Linux containers ([DEC-037](../register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)) with Runtime profile and budget.

Interpretation: this adopts the [cockpit concept's](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/cockpit.md#working-through-the-cockpit) working-surface slices recorded as [DEC-065](../register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration). It does not adopt Government, delegated model maintenance, a personal AI beyond these sessions, an automatic Apply after a change, or a provider turn without the owner's prompt and the configured budget limits.
