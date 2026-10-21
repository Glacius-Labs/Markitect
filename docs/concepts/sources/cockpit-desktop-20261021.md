# Cockpit as desktop app, 21 October 2026

Source: the owner's direct messages to Claude Code in the cockpit session (Claude Code session `8dba53fb`), 21 October 2026, after the hub-and-areas cockpit (#556) and the Markitect mark (#558) were merged. The quotations are verbatim. Recorder context and interpretation are labelled separately.

## Question

> Was wäre wenn das Cockpit sowas wäre wie Claude Desktop für die claude CLI ist?

Recorder context: Claude answered that much of it already exists — the cockpit is its own surface over the same program as the CLI — and that four things make a desktop app:

1. starting without flags: open the app, choose a project from a list, with guided setup of run configuration, Runtime profile and credentials instead of `--run-config`/`--runtime-profile`;
2. starting and stopping runs, which [DEC-065](../register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration) had deliberately not adopted because a start spends provider budget; Claude recommended a start only with an explicit confirmation showing the budget, and stopping always allowed;
3. background work and notifications when a run asks a question;
4. a native window with installer and automatic updates.

Claude proposed the order 1, then the decision on 2, the native window last, and noted that a desktop app makes Markitect more accessible but not more reliable while Apply is still being stabilized. It named the real decision: not the technology, but whether the app may start runs itself.

## Decision

> Die eigentliche Entscheidung ist dabei aber nicht die Technik, sondern ob die App selbst Läufe starten darf => Ja klar, darf sie.
> Halte das ganze so fest und setze es dann mithilfe von Sonnet Subagents um

Interpretation: the owner adopts all four parts and lets the cockpit start and stop Apply, recorded as [DEC-066](../register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply). This amends DEC-065's exclusion of starting and stopping Apply from the cockpit. Implementation is assigned to Claude with Sonnet subagents. Claude's item 3 named notifications for a run's question; the owner's later closed-Run decision ([#560](https://github.com/Glacius-Labs/Markitect/pull/560)) removes questions from a running Apply to the owner, so the notifications cover a run's end and decisions that wait for the owner instead. The decision grants no provider run by itself: a run starts only when the owner starts it, under the configured credentials and budget. Installer and automatic updates are scheduled with the pinned pilot build ([#503](https://github.com/Glacius-Labs/Markitect/issues/503)), so that one installable Markitect carries the cockpit, and starting and stopping comes before the project list because the rest depends on it; this sequencing is the recorder's design choice within the decision, not an owner statement.
