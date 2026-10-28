# Workstreams

Since 28 October 2026, Markitect work runs in two sessions. The owner replaced the eleven lanes of 27 October: "instead of all those parallel session we need one session which is responsible to coordinate and discuss the evaluation suite and the three variants. We need one session for discussion of the various topics."

This page lists the sessions, the arms and where each keeps its state. It holds no status itself.

## Sessions

| Session | Does | Builds | State page |
|---|---|---|---|
| Build and evaluation | Builds concept 2.0 and the evaluation suite that measures it, runs their dispatches and integrates their PRs | yes, the only session that builds | [build.md](workstreams/build.md) |
| Topics | Discusses the product topics with the owner (model language and skills, compiler, commands, continuous change, platform, Cockpit) and records the decisions | no; it writes docs only | [topics.md](workstreams/topics.md) |

Both sessions sit in the sidebar folder "Markitect". Lane 1 handed the host variant over to Build on 28 October 2026, and its page folded into the build page. The other lane sessions end once the owner agrees. the suite's contract is the [evaluation suite guide](guides/evaluation-suite.md). The topic lane pages were folded into the topics page on 28 October 2026; the [commands page](workstreams/commands.md) stays as the canonical page for the commands frame.

## Arms

Since 28 October 2026 the target is [concept 2.0](concepts/v2-concept.md). The owner chose one implementation of it, compared with the baseline, the same day; the plugin variant and V1's open steps stopped, and the Git variant's prototype became the start of the 2.0 Run remote ([build steps](implementation-plan.md#concept-20-build)). The evaluation suite treats each arm as a black box with the same task in and the same judging out.

| Arm | What | Where |
|---|---|---|
| 2.0 | Markitect as concept 2.0 describes it, built in steps T1 to T5 | `main`, as it lands |
| K | Baseline: classic agentic coding, one Claude Code session with the task | the suite on `main` |

The concept 1.0 arms stay where they stopped, for reference only: M (V1 on `main` until 2.0 replaces it), P (`variant/plugin`) and G (`dev/g0-prototype`).

## Priority

Concept 2.0's acceptance (§15) is the yardstick: correct against hidden tests, no drift after a change series, nothing skipped, stops right on model defects, and at least as trustworthy as one strong agent. Consistency with the model and quality come first; time and token cost second.

## Rules

- **One state page per session.** It keeps three short parts, updated whenever a step ends: **Now**, **Next** and **Questions for the owner**, each question with a recommendation. Chats, messages and dashboards are not state.
- **Few sessions at once.** At most five sessions work at the same time (owner, 28 October 2026; earlier about three). Build starts short sub-sessions for concrete jobs, one job and one PR each, and ends each one when its PR is merged.
- **Topics hands work to Build.** When an owner decision needs code, Topics sends Build one clear task.
- **The usual rules still apply.** [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the [integration policy](implementation-plan.md#integration) govern every session. Each real container, provider or Apply run needs a bounded dispatch the owner approves, one at a time on the shared engine.
