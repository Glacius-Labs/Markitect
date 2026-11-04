# Workstreams

Since 28 October 2026, Markitect work runs in a few sessions. The owner replaced the eleven lanes of 27 October with one session for the build and one for the product topics. On 31 October 2026 the owner started a third session, Desktop, to build Markitect Desktop ([DEC-089](concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli)). On 4 November 2026 the evaluation suite and the variants were removed ([DEC-094](concepts/register.md#dec-094-a-fresh-start-for-realization)).

This page lists the sessions and where each keeps its state. It holds no status itself.

## Sessions

| Session | Does | Builds | State page |
|---|---|---|---|
| Build | Builds the product's code and integrates its PRs | yes, everything except Desktop's scope | none; the build page left with the evaluation suite |
| Topics | Discusses the product topics with the owner (model language and skills, compiler, commands, continuous change, platform) and records the decisions | no; it writes docs only | [topics.md](workstreams/topics.md) |
| Desktop | Designs and builds Markitect Desktop with the owner | yes, `desktop/`, `internal/adapters/desktop` and `cmd/markitect-desktop`; shared files one writer at a time with Build | [desktop.md](workstreams/desktop.md) |

The sessions sit in the sidebar folder "Markitect". The topic lane pages were folded into the topics page on 28 October 2026; the [commands page](workstreams/commands.md) stays as the canonical page for the commands frame.

## Rules

- **One state page per session that holds state.** It keeps three short parts, updated whenever a step ends: **Now**, **Next** and **Questions for the owner**, each question with a recommendation. Chats, messages and dashboards are not state.
- **Few sessions at once.** At most five sessions work at the same time (owner, 28 October 2026; earlier about three). Build starts short sub-sessions for concrete jobs, one job and one PR each, and ends each one when its PR is merged.
- **Topics hands work to Build.** When an owner decision needs code, Topics sends Build one clear task.
- **The usual rules still apply.** [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the [integration policy](implementation-plan.md#integration) govern every session. Each real container or provider run needs a bounded dispatch the owner approves, one at a time on the shared engine.
