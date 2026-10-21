# Markitect cockpit

**Status: owner-commissioned concept and Claude implementation handoff, 20 October 2026.** The owner requested an Apply visualization, refined an interactive example, then asked for a cockpit concept, Claude-led implementation/design and documentation on Main. The [source record](../concepts/sources/cockpit-commission-20261020.md) preserves the commission. This delivery documents the concept; it implements no product UI or runtime behavior. [VIS-001](../concepts/register.md#vis-001-cockpit-personal-ai-and-autonomous-project-organization) retains the broader vision.

**Implementation, 20 October 2026:** recorded Run inspection with replay, live observation and the cockpit entry are implemented as `markitect cockpit` in the [cockpit adapter](../../internal/adapters/cockpit/README.md). Their validation records ([inspection](../validation/cockpit-run-inspector-20261020.md), [live](../validation/cockpit-live-observation-20261020.md), [entry](../validation/cockpit-entry-20261020.md)) separate checks, real history, re-fed demonstrations and open points; no active Apply has been observed yet. Starting and stopping Apply were first left out, because a start spends provider budget and needed an owner decision; the owner adopted both on 21 October 2026 together with the cockpit as desktop app ([DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply), [below](#the-cockpit-as-desktop-app)). Starting and stopping the one run of a configured cockpit are implemented ([adapter](../../internal/adapters/cockpit/README.md#starting-and-stopping-a-run), [validation](../validation/cockpit-run-control-20261021.md)); the app window with desktop notifications ([adapter](../../internal/adapters/cockpit/README.md#app-window-notifications-and-background), #567) and the project list with guided setup ([adapter](../../internal/adapters/cockpit/README.md#projects-and-guided-setup), [validation](../validation/cockpit-projects-20261021.md)) are implemented too. A first bounded acceptance on a real provider and engine set a project up, started, followed and stopped Apply from the cockpit ([record](../validation/cockpit-desktop-acceptance-20261021.md)). Installer and automatic updates are not implemented; they come with the pinned pilot build.

Claude owns the visual design, frontend/backend choices and implementation within this scope. The [handoff](../development/claude-cockpit-handoff-20261020.md) supplies delivery steps and a startup prompt. The [delivery plan](../implementation-plan.md) remains the owner of shared-file reservations and the Apply stabilization checkpoint; this document starts no session or provider run.

## Purpose and learning

The cockpit is the intended central working surface for Markitect: understand the project and its organization, follow work, resolve the decisions actually addressed to the owner and inspect the evidence behind an outcome. The first complete surface is an Apply Run inspector in a local web application.

**Observation:** the owner repeatedly asked who works, what happens next, why others wait and which messages or deliveries explain that waiting. The interactive example made those questions concrete, including a documentation finding and a correction/review cycle. It did not observe a real Apply or measure usability.

**Challenged assumption:** a phase label, activity animation or collection of green checks is sufficient to orient the owner. Those views alone do not explain a dependency or establish that an outcome was independently verified and published.

**Design response:** connect organization, time, communication, decisions and candidate-bound evidence through the same recorded identities. Make a wait navigable to its cause. Keep detailed records available without requiring routine raw-log reading.

**Observable improvement:** using an actual recorded Run, the owner can identify the active actor, a blocked responsibility, its concrete dependency, the next required actor and the evidence for the final result without assistance from the implementation chat. Replay and live observation must give the same account at the same recorded boundary. This remains an acceptance goal, not a measured benefit.

## Scope selected in the conversation

The owner requested a local web view with a Manager tree, timeline and step details; automatic playback with active work and wait reasons; messages and handoffs; and animated directional arrows between participants. The cockpit expands that Run inspector into a project entry point. The example's particular layout, colors and implementation are not a required design system.

The following organization and delivery sequence are implementation recommendations. Claude may improve them while retaining the requested capabilities, existing authority boundaries and inspectable evidence. A new product authority or a change to Apply semantics is a separate decision.

## Current Apply semantics govern the view

Use [DEC-064](../concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams), the [accepted workflow](canonical-model/manager-apply-workflow.md) and the [implemented runtime contract](canonical-model/manager-apply-runtime.md). Do not implement the example's older Explore/work/local-review task machine or revive its historical SQLite-journal design.

The view must follow Model check and File check, the strategy's work Passes (Implement in `direct`; Plan, Prepare and Implement in `staged`), quiescence and immutable capture, mandatory closing checks, independent Verify, correction rounds where needed, and guarded Publish with read-back. A Pass, Manager result, Turn ending and Run outcome are distinct. Render actual recorded participation, state and acceptance; do not invent progress from elapsed time.

Use Orchestrator for the deterministic controller and Runtime for execution infrastructure. One logical Manager owns each required local Package; Root is an ordinary Manager. Workers, reviewers and specialists belong to the responsible Manager's team and add no independent ownership. Show parallel branches when actually observed, rather than imposing the example's serial execution.

Questions and answers travel between direct parents and children, including ancestor-routed cross-branch exchanges. No question or answer crosses Root to the user or the invoking AI during a Run ([closed Runs](canonical-model/manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs)); the cockpit offers no answer route. An ordinary clarification, an unanswered or lapsed question and an actual model defect have different consequences. The UI grants no authority to override the model.

## Connected cockpit views

### Project overview

Show the project, bound model/source identities, active Run, recent completed/interrupted Runs, open decisions and concrete blockers. Prefer a short factual account such as "Implement is correcting F17; Root still needs the child's accepted result" over generic activity scores. Time, usage and cost are useful secondary observations; unknown values stay unknown.

Do not turn this into a second development coordination backlog. Product Run evidence and the repository's implementation plan serve different purposes.

### Organization and current work

The Manager tree follows the required Package hierarchy and states what each responsibility owns, who is active, and what releases a wait. Distinguish Manager, team member, fresh reviewer, provider session/Turn, Orchestrator and Runtime. A Manager waiting for Verify must not appear to be implementing merely because its reviewer is active.

A spatial organization view may complement the tree. Highlight the actual sender, receiver and direction of recent messages; keep reviewer identities attached to their historical messages when another reviewer is now active. Show concurrent work without implying that all branches can advance independently of their prerequisites.

### Messages and causal waits

Each visible exchange identifies its sender, recipient, purpose, observed time/order, correlation and relationship to a question, report, proposed basis, accepted result, candidate or finding where applicable. Keep Agent-authored communication separate from Orchestrator-delivered control/acceptance events.

Sent, confirmed delivery, response, parent acceptance and publication are separate facts. Display only states the records establish; otherwise say unobserved or unknown. Do not infer that a participant read a message or acted on it from delivery alone.

A typical navigation chain is:

> Root waits for the child's result → that result was returned or remains unaccepted → F17 names a concrete model violation → its owner submits a correction → fresh independent Verify still has to close F17 on the new candidate.

The cause must link back to the actual question, dependency or evidence. An arbitrary age threshold does not prove a deadlock. When several conditions block progress, show them separately; receiving one reply does not imply the whole responsibility is complete.

### Timeline and detail inspection

Use one time axis for actual work, checks, independent review, orchestration and waits. Event sequence and causal references govern order; timestamps alone do not flatten parallel branches into a false serial history.

Selecting a Manager, Turn, event or message opens its relevant context, supplied inputs, actual report, candidate difference, findings, checks and acceptance. Retain rejected, reopened and interrupted attempts. Evidence links identify the exact source/model/candidate and scope; a verdict for an older stand cannot become green for changed bytes by presentation alone.

### Owner decisions and results

A Run asks the owner nothing, so the cockpit has no owner-question view and no answer action. A question to the user that an earlier Run recorded is shown read only, as recorded. A model defect ends the Run Failed; the owner amends the model outside the Run and starts a new one. What waits for the owner here are proposals for the model and the execution settings.

The outcome view distinguishes what was requested, changed, independently assessed, accepted, published and read back, with remaining findings or unknown effects. Partial publication or an uncertain read-back cannot become success. A successful source check, provider result or prototype does not establish release or owner acceptance.

## Live observation, replay and animation

Live and replay use the same recorded facts. Pin a historical event boundary when inspecting the past; later questions, answers and verdicts must not leak into that slice. Preserve the user's selection as new live events arrive, indicate the newer data and offer an explicit return to current observation.

Replay provides Play/Pause, speed, event jumps and optional automatic following of the active participant. A manual selection pauses or pins presentation. **Presentation pause never stops the real Run.** Run stop/start actions, when implemented, have separate labels and go through existing Orchestrator commands and acknowledgements.

Animated arrows identify the exchange and its direction, pause with replay, and open the corresponding message. Arrow movement illustrates a send event, not measured transport duration or proof of receipt. During quiet work, do not invent communication to keep the graph moving. Avoid endless loops; preserve keyboard access and a useful static representation with reduced motion.

Claude chooses the layout, visual hierarchy, typography, responsive behavior and interaction details. A large hierarchy must remain understandable through navigation and selection rather than rendering every label and transcript at once.

## Observation boundary and implementation choices

Start from the current runtime's evidence boundary: `<state>/runs/<runID>/events.jsonl`, `transcripts/<turn>.jsonl`, the terminal `report.json`, and current owner-question records. See [Evidence and state](canonical-model/manager-apply-runtime.md#evidence-and-state). The final report's absence during execution is not a failed Run.

Inspect the actual producer and readers before choosing an adapter. Prefer a small local read interface and derived projections over adding a second authoritative store. Any optional UI cache must be disposable and bound to its source identity. A replay, browser reconnect or read refresh must cause no Agent dispatch, question answer, state repair or publication.

The consumer needs Run/Pass/Manager/team/Turn identities, ordered observations and causal links, message/question correlations, basis/result acceptance, candidate/finding/check bindings, and publication/read-back observations where supplied. This is an information requirement, not a new fixed event schema. Audit missing fields against the real producer. Expose gaps instead of fabricating identifiers, correlations, receipt states or semantic explanations.

Read bounded consistent prefixes. Retain unknown tails, partial records, unavailable/redacted payloads, changed source identity and connection loss as visible limitations. Live reconnect must neither duplicate events nor silently skip an observed boundary. Existing recorded state and current connection health are separate.

Use local delivery by default. Access to transcripts must respect existing redaction and visibility limits. Raw prompts need not dominate the normal view; the available recorded inputs remain reachable for investigation.

## Delivery sequence and boundaries

1. **Recorded Run inspection.** Implement a complete historical Run through the real read boundary, with organization, causal waits, messages, candidate/evidence details and replay. Keep synthetic examples explicitly labelled. Demonstrate a correction and independent re-verification, not just a happy-path animation.
2. **Live observation.** Reuse the same projections for an actual active Run. Demonstrate updates, selection pinning, reconnection, incomplete evidence and interruption without adding execution side effects.
3. **Cockpit entry point and existing actions.** Add the project overview, outcome history and the view of proposals that wait for the owner. Wire available start/stop operations only through their actual Orchestrator interfaces; show unsupported capabilities honestly and define any missing seam through its owner. Do not change Apply to satisfy a UI shortcut.

The owner assigned design and implementation to Claude. The [handoff](../development/claude-cockpit-handoff-20261020.md) is intended for a new explicit bounded assignment, not automatic resumption of an ended Claude session. Read and reconcile the current stabilization plan and shared-file reservations before writing. New provider series, unbounded spending, releases and deployments are not included.

Personal AI, Government, approval reuse and gamification remain later extensions. Model editing through the cockpit was a later extension too until the owner adopted it in bounded form ([DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration)); the next section owns it. The [working method](../development/working-method.md) still requires a complete usable workflow before unrelated expansion.

## Working through the cockpit

**Owner decision, 20 October 2026:** the owner wants to work on the project through the cockpit: adjust the model by prompts that go to Claude or Codex, and configure the Agent teams ([source](../concepts/sources/cockpit-commission-20261020.md#working-through-the-cockpit), [DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration)).

**Observation and design response:** the cockpit already shows how Apply realized the model; changing the model and the team setup still meant leaving it for an editor and a terminal. Both changes become proposals with one acceptance path, so the owner reviews the actual difference and its checks before anything is written.

**One proposal path.** A proposal is a changed stand of the project's `.markitect` inputs against the captured committed source; a working tree with uncommitted changes refuses a new proposal. The cockpit runs the Model check and the File check on it, shows the changed files and the changed Definitions, and offers acceptance only when both pass. Acceptance writes the exact change through Apply's guarded publication: the captured source must be unchanged, every written path is read back, nothing is committed, and no Apply may hold an observed state directory meanwhile. Only an applied publication accepts the proposal. A stale source refuses it before any write; a partial or unknown outcome is shown with every path's read-back state, the proposal is not accepted, and the owner inspects the working tree. Discarding writes nothing. Accepting starts no Apply; the owner starts Apply separately.

**Configuration.** The execution settings' budget, role profiles, per-Package profile overrides and team (helper roles with their profiles, write rights and instruction references, and the helper limit) are edited in a form. The form produces the changed `.markitect/execution.yaml`; product decoding and the structural check judge it as they judge a hand-edited file. Ownership, the top-level and per-Package instructions, scope and checks are not edited in the form and stay as written; they change by hand.

**Model sessions.** A prompt starts or continues an Agent session on a workspace materialized from the captured committed source. The session runs through the existing Runtime adapters in a Linux container with the Execute profile and its provider, model and limits ([DEC-037](../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)); its write plan permits only `.markitect/model/` and `.markitect/registry/`. Follow-up prompts continue the same provider conversation. The Agent receives the language contract's essentials, the current model and the request; it can run the checks on its live workspace and must submit its proposal with a summary. Each prompt is one provider turn whose usage and transcript are retained as session evidence under the state directory and shown. A turn is bounded by the execution budget's `turnTimeout`, and a session ends at the budget's `agentTurns` limit; its last proposal stays reviewable. The Agent never writes the owner's working tree; a model it judges contradictory is reported in its summary, not repaired silently.

**Boundaries:** no Government or delegated maintenance, no automatic Apply, no change to Apply semantics, and no change to Runtime profiles, credentials or the run configuration other than through the function behind `markitect setup` ([desktop app](#the-cockpit-as-desktop-app)). The owner starts and stops Apply as the desktop-app section describes. Every session needs a configured Runtime profile; a provider turn happens only when the owner sends a prompt.

**Delivery:** the fourth slice delivers configuration proposals through the form and the shared acceptance path, the fifth Agent model sessions in containers. Each slice records source checks, UI behavior and real or demonstrated use separately.

## The cockpit as desktop app

**Owner decision, 21 October 2026:** the cockpit becomes to `markitect` what a desktop app is to a command-line tool, and it may start runs itself ([source](../concepts/sources/cockpit-desktop-20261021.md), [DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply)).

**Observation and design response:** the cockpit shows a project, its runs and its decisions, but using it still needed a terminal: `markitect setup` with absolute paths, `markitect cockpit` with `--state`, `--run-config` and `--runtime-profile`, and `markitect apply` to start the run the cockpit then watches. The earlier assumption was that starting provider work belongs outside the cockpit. The owner judged that the start is the owner's own action and can happen where the owner already works, with the budget in view. What stays: Apply itself, its inputs, checks, budget, evidence and publication; the proposal path; and that only the owner causes provider spending.

**Projects without flags.** A per-user project list in the operating system's user configuration directory (`markitect/cockpit/projects.json`) names each project's repository and its runtime directory, the directory `markitect setup --runtime` fills with `runtime.yaml`, `run.json` and `state/`. Opened without flags, the cockpit shows the list; the existing flags still open one project directly. Switching the project happens only while the current one runs nothing.

**Guided setup.** A form asks for the inputs of `markitect setup` — runtime directory, workspace image, credential file, Agent model and effort, check profiles — and calls the same function. It writes exactly what setup writes and refuses where setup refuses, for example while an Apply holds the state directory. The formats of the run configuration and the Runtime profile stay owned by setup.

**Starting and stopping.** The Runs area offers *Start a run* for the chosen project. The confirmation names the execution budget of the committed `.markitect/execution.yaml` (run time, turn timeout, Agent turns, Passes), the run configuration's deadline and the Agent profile with its model. The cockpit then runs the same execution function as `markitect apply`, in its own process, with a context only it can cancel. *Stop* cancels that context: the run ends incomplete with its report and evidence, exactly like an interrupted `markitect apply`, and nothing is restored. A start is refused with uncommitted changes, while another run or a proposal publication holds the state directory, or without a configured project. Runs started elsewhere are observed as before and stopped where they were started. A run started in the cockpit lives in the cockpit's process: closing the window keeps it running, ending the cockpit with an interrupt stops it the same way, and the cockpit warns before that. A killed cockpit process, like a killed `markitect apply`, leaves the run without its report.

**Window and notifications.** The cockpit opens its local address in an app window of an installed Chromium browser (Edge or Chrome) and falls back to the default browser; a flag suppresses opening. Nothing extra is bundled. With the owner's permission, the page notifies when a run ends, when a proposal waits for the owner's decision and when a model session answers, and the window title counts what waits. Installer and automatic updates come with the pinned pilot build ([#503](https://github.com/Glacius-Labs/Markitect/issues/503)).

**Observable result:** the owner opens the cockpit without arguments, sets up or chooses a project, starts a run with its budget in view, follows it, stops it with its evidence kept, and is notified when it ends or a decision waits — with no terminal command beyond opening the cockpit. Source checks, a scripted run and a real run with the owner's credentials are recorded separately.

## Acceptance

Use this bounded set through the actual observation adapter: accepted and returned results; correlated question/answer including lapse; simultaneous independent branches; a no-change result; a correction and fresh Verify on a new candidate; interruption or unknown effects; and successful Publish with exact read-back. Synthetic coverage is legitimate UI evidence, but must not be labelled real runtime proof. Reuse existing real evidence where appropriate; a new provider Run needs its own existing applicable grant.

The owner must be able to explain current work, each demonstrated wait, the next required actor and the final evidence without reading raw logs. Verify that history does not reveal later events, pause/reconnect causes no execution, animation agrees with message identity/direction, keyboard/reduced-motion operation remains useful, and final success is bound to the published/read-back candidate.

For the desktop app, show: a run starts only after the owner's confirmation and names its budget; a stop ends the run incomplete with its report and evidence; a start is refused with uncommitted changes and while the state directory is held; setup through the cockpit writes the same files as `markitect setup`; and opening without flags reaches the project list.

For working through the cockpit, also show: a proposal that fails a check cannot be accepted; a discarded proposal writes nothing; acceptance is refused while an Apply holds an observed state directory and when the committed source changed since the proposal; an accepted proposal reads back exactly, is not committed and starts no Apply; a partial or unknown publication, synthetic if necessary, is shown with each path's read-back state and stays unaccepted; a dirty working tree refuses a new proposal; and a model session spends only on the owner's prompts, stays within its limits and changes only the model and registry.

Record source checks, UI behavior, real history inspection, live observation and owner assessment separately. Keep unresolved data or runtime gaps explicit rather than broadening the work until an attractive demonstration appears complete.
