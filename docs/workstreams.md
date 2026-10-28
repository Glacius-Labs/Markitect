# Workstreams

Since 28 October 2026, Markitect work runs in two sessions. The owner replaced the eleven lanes of 27 October: "instead of all those parallel session we need one session which is responsible to coordinate and discuss the evaluation suite and the three variants. We need one session for discussion of the various topics."

This page lists the sessions, the variants and where each keeps its state. It holds no status itself.

## Sessions

| Session | Does | Builds | State page |
|---|---|---|---|
| Build and evaluation | Coordinates the evaluation suite and the variants, runs their dispatches and integrates their PRs | yes, the only session that builds | [build.md](workstreams/build.md) |
| Topics | Discusses the product topics with the owner (model language and skills, compiler, commands, continuous change, platform, Cockpit) and records the decisions | no; it writes docs only | `workstreams/topics.md` |
| Lane 1: host variant | Finishes its current task, then hands arm M over to Build and ends | until the handover | [host-variant.md](workstreams/host-variant.md) |

Both sessions sit in the sidebar folder "Markitect". The lane sessions 2 to 11 end once the owner agrees. The evaluation, plugin and Git variant pages fold into the build page as their PRs land; the topic lane pages stay as they are until the owner opens their discussion.

## Variants

The evaluation suite treats every variant as a black box with the same task in and the same judging out.

| Arm | Variant | Branch |
|---|---|---|
| M | Host variant: today's V1, the host coordinates the Manager hierarchy | `main` |
| P | Plugin variant: Markitect's Apply method as a Claude Code plugin | `variant/plugin` |
| G | Git variant: Managers collaborate through a Run-local Git remote ([DEC-078](concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)); prototype on `dev/g0-prototype` | `variant/git` |
| K | Baseline: classic agentic coding, one Claude Code session with the task | part of the suite ([PR #793](https://github.com/Glacius-Labs/Markitect/pull/793), not yet on `main`) |

## Priority

The owner, 27 October 2026: "The working streams and evaluator are currently our primary goal i think. It should achieve that each of them gets a stable version and can be evaluated and then we can decide which one of them performs well in which parameters of a project."

Three steps:

1. **One yardstick.** The evaluation suite's variant command is the one contract for every variant: the same task folder in, a result bundle out, the same judging.
2. **A stable version per variant.** One Library task runs end to end through the suite three times in a row without manual help. Order: K and M first, then P, then G.
3. **Comparison.** A small task set along project traits (size, kind of change, cost) runs on every variant with the same model. The result is one table per area: which variant does well where. There is no overall score.

## Rules

- **One state page per session.** It keeps three short parts, updated whenever a step ends: **Now**, **Next** and **Questions for the owner**, each question with a recommendation. Chats, messages and dashboards are not state.
- **Few sessions at once.** At most about three sessions work at the same time. Build starts short sub-sessions for concrete jobs, one job and one PR each, and ends each one when its PR is merged.
- **Topics hands work to Build.** When an owner decision needs code, Topics sends Build one clear task.
- **Shared code lives on `main`.** The compiler, the model, the Libraries and the evaluation suite are shared. A variant branch holds only that variant's code and merges `main` regularly.
- **The usual rules still apply.** [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the [integration policy](implementation-plan.md#integration) govern every session. Each real container, provider or Apply run needs a bounded dispatch the owner approves, one at a time on the shared engine.
