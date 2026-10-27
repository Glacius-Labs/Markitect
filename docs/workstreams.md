# Workstreams

Since 27 October 2026, Markitect work runs in independent lanes, so that one chat never mixes several topics. The owner asked for it: "Id like to partition those so that we have independant workstreams on all of that and dont mix everything in one chat. Also each variant or prototype of Markitect should get an own coordinator or responsible session, which can create subsessions for parallel work. We dont want to loose track here".

This page lists the lanes and where each one keeps its state. It holds no status itself: each lane owns its page under `docs/workstreams/` and updates only that page.

## Lanes

| # | Lane | Kind | Responsible session | Branch | State page |
|---|---|---|---|---|---|
| 1 | Host variant: today's V1, the host coordinates the Manager hierarchy | variant | Lane 1: Host variant lead (the Coordinator) | `main` | `workstreams/host-variant.md` |
| 2 | Plugin variant: Markitect as a Claude Code plugin (arm P) | variant | Lane 2: Plugin variant lead | `variant/plugin` | `workstreams/plugin-variant.md` |
| 3 | Git variant: Managers collaborate through a Git remote; the host only starts and stops ([DEC-078](concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)) | variant | Lane 3: Git variant lead | `variant/git` | `workstreams/git-variant.md` |
| 4 | Evaluation suite: the same job and the same judging for every variant; classic agentic coding is the baseline arm | shared | Lane 4: Evaluation suite | `main` | `workstreams/evaluation.md` |
| 5 | Windows and Linux execution | topic | Lane 5: Windows / Linux execution | `main` | `workstreams/platform.md` |
| 6 | Canonical model language: Kinds, Libraries, quality | topic | Lane 6: Canonical model language | `main` | `workstreams/model-language.md` |
| 7 | Modeling skills: how people and agents design and change a model | topic | Lane 7: Modeling skills and model design | `main` | `workstreams/modeling-skills.md` |
| 8 | Compiler: what `markitect check` checks and reports | topic | Lane 8: Compiler capabilities | `main` | `workstreams/compiler.md` |
| 9 | Agentic commands run through the Manager hierarchy | topic | Lane 9: Agentic commands | `main` | `workstreams/commands.md` |
| 10 | Continuous change: outside triggers become tasks, model changes and Applies | topic | Lane 10: Continuous change from triggers | `main` | `workstreams/continuous-change.md` |
| 11 | Cockpit or IDE for seeing and understanding the model and its runs | topic | Lane 11: Cockpit / IDE | `main` | `workstreams/cockpit.md` |

The state page paths are plain text, so a lane can add its page without touching this one.

## Priority

The owner, 27 October 2026: "The working streams and evaluator are currently our primary goal i think. It should achieve that each of them gets a stable version and can be evaluated and then we can decide which one of them performs well in which parameters of a project."

Lanes 1 to 4 build, in three steps:

1. **One yardstick.** Lane 4's arm interface becomes the one contract for every variant: the same task file in, the same `run.json` and bundle out, the same judging.
2. **A stable version per variant.** One Library task runs end to end through that interface three times in a row without manual help. Order: classic baseline and host first, then plugin, then Git.
3. **Comparison.** A small task set along project traits (size, kind of change, cost) runs on every variant with the same model inside a suite. The result is one table: which variant does well on which trait.

Topic lanes 5 to 11 pause once their state page is up, until the owner opens a discussion.

## State pages

A lane's state page is the only place for its state; chats and messages are not. The page keeps three short parts, updated whenever a step ends:

- **Now:** what the lane is doing.
- **Next:** its next step.
- **Questions for the owner:** each decision it needs, with its recommendation.

## Rules

- **Shared code lives on `main`.** The compiler, the model, the Libraries and the evaluation suite are shared. A variant branch holds only that variant's code and merges `main` regularly, so the variants never fork the shared parts.
- **One lane, one scope.** A lane changes only its own files. When it needs another lane's files, it asks that lane's session.
- **Leads may delegate.** A variant lead may start its own sub-sessions for parallel pieces with disjoint files.
- **Topic lanes discuss first.** They take stock, write their state page and discuss with the owner before they build.
- **The usual rules still apply.** [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the [integration policy](implementation-plan.md#integration) govern every lane. Real container, provider or Apply runs need a bounded dispatch.
