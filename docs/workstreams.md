# Workstreams

Since 28 October 2026, Markitect work runs in a few sessions. The owner replaced the eleven lanes of 27 October: "instead of all those parallel session we need one session which is responsible to coordinate and discuss the evaluation suite and the three variants. We need one session for discussion of the various topics." On 31 October 2026 the owner started a third session, Desktop, to build Markitect Desktop while the evaluations run ([DEC-089](concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli)).

This page lists the sessions, the arms and where each keeps its state. It holds no status itself.

## Sessions

| Session | Does | Builds | State page |
|---|---|---|---|
| Build and evaluation | Builds Markitect, runs its dispatches and integrates its PRs; until 6 November 2026 that was concept 2.0 and the evaluation suite that measured it | yes, everything | [build.md](workstreams/build.md) |
| Topics | Discusses the product topics with the owner (model language and skills, compiler, commands, continuous change, platform) and records the decisions | no; it writes docs only | [topics.md](workstreams/topics.md) |

The sessions sit in the sidebar folder "Markitect". Lane 1 handed the host variant over to Build on 28 October 2026, and its page folded into the build page. The other lane sessions end once the owner agrees. The suite's contract was the [evaluation suite guide](https://github.com/Glacius-Labs/Markitect/blob/2fd2d33747de2e69e5f4968432792d543003041a/docs/guides/evaluation-suite.md), until its runner was removed on 7 November 2026 ([archive](archive.md#removed-with-the-evaluation-runner)). The topic lane pages were folded into the topics page on 28 October 2026; the [commands page](workstreams/commands.md) stays as the canonical page for the commands frame.

## Arms

Since 7 November 2026 the target is [concept 4.0](concepts/v4-concept.md) ([DEC-098](concepts/register.md#dec-098-concept-40-is-the-target)), whose aim is one version that works reliably on Sonnet, with no arms to compare ([§1](concepts/v4-concept.md#1-the-aim)). The rest of this section is a record. From 4 November the target was [concept 3.0](concepts/v3-concept.md) ([build steps](implementation-plan.md#concept-30-build)). From 28 October it was [concept 2.0](concepts/v2-concept.md). The owner chose one implementation of it, compared with the baseline, the same day; the plugin variant and V1's open steps stopped, and the Git variant's prototype became the start of the 2.0 Run remote ([build steps](implementation-plan.md#concept-20-build)). The evaluation suite treated each arm as a black box with the same task in and the same judging out.

| Arm | What | Where it was |
|---|---|---|
| 2.0 | Markitect as concept 2.0 describes it, built in steps T1 to T9 | `main` until 6 November 2026, now the branch `archive/main-concept-3-20261106` |
| K | Baseline: classic agentic coding, one Claude Code session with the task | the suite, whose runner was removed on 7 November 2026 ([archive](archive.md#removed-with-the-evaluation-runner)) |

The concept 1.0 arms stopped where they were, for reference only: M (V1 on `main` until concept 2.0 replaced it), P (`variant/plugin`) and G (`dev/g0-prototype`).

## Priority

Concept 4.0's aim is the yardstick: the spec as it stands is applied reliably on Sonnet, with no babysitting ([§1](concepts/v4-concept.md#1-the-aim)). What "reliable" means is still open ([§16, question 10](concepts/v4-concept.md#16-questions-for-the-owner)). From 28 October to 4 November concept 2.0's acceptance (§15) was the yardstick.

## Rules

- **One state page per session.** It keeps three short parts, updated whenever a step ends: **Now**, **Next** and **Questions for the owner**, each question with a recommendation. Chats, messages and dashboards are not state.
- **Few sessions at once.** At most five sessions work at the same time (owner, 28 October 2026; earlier about three). Build starts short sub-sessions for concrete jobs, one job and one PR each, and ends each one when its PR is merged.
- **Topics hands work to Build.** When an owner decision needs code, Topics sends Build one clear task.
- **The usual rules still apply.** [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the [integration policy](implementation-plan.md#integration) govern every session. Each real container, provider or Apply run needs a bounded dispatch the owner approves, one at a time on the shared engine.
