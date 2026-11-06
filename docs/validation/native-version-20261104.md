# The native version: real sessions while building it, 4 November 2026

The owner chose to build [draft concept 4.0](../concepts/v4-concept.md) on a branch and test it by using it. This page records the real sessions that build needs: the mechanisms the concept only reads from the documentation, tried before and while building. It is bounded evidence on small changes, not acceptance, and says nothing about how often a run fails.

**Since this record:** later the same day the owner removed required checks ([draft concept 4.0](../concepts/v4-concept.md#3-the-tools)). The sessions below ran before that, and no real session has run on the build without them.

## Questions

- **Q1:** does a background session start from inside another session, in its own worktree based on the current commit?
- **Q2:** do the settings given at launch hold in that session: permission mode, allow and deny rules, hooks, the model, and no personal rules or memory?
- **Q3:** does the session start a reviewer agent and wait for it, with the reviewer's tools narrowed, and does the agent's type reach a hook?
- **Q4:** does a Stop hook send the session back, and does a provider habit of pushing stay off?
- **Q5:** does a whole change land through the tool, with nobody watching?

## Dispatch

| Item | Value |
|---|---|
| Source | Branch `dev/native-apply` at the commit each attempt names; throwaway scripts frozen into each attempt's folder |
| Provider | Claude Code 2.1.318 on the owner's stored login, unmodified. Haiku for mechanism sessions, `claude-sonnet-5-5` at effort `high` for a whole change |
| Target | Throwaway repositories under `C:\Users\Consiliari\workdir\markitect-native`, each with a local bare `origin` and no real remote. No project of the owner's |
| What a session may do | Edit files and run any shell command as the user, on the owner's machine, with a deny rule on `git push` and `gh`. The target holds nothing of value, and no session reads outside input |
| Limits | At most 12 mechanism sessions, each under 10 minutes, and 3 whole changes, each under 45 minutes. At most 5 points on the weekly reading, which stood at 15 before the first session |
| Abort | A session acting outside its target, any push reaching a real remote, a login failure, or a limit reached. The session is stopped by its own ID with `claude stop`; no process is killed by name |
| Cleanup | `claude stop` and `claude rm` for each session this dispatch started, by ID, and `git worktree remove` for its worktrees. The target folder stays as evidence until the branch is judged. Other sessions in `claude agents` are left alone |
| Evidence | Outside the repository, in `C:\Users\Consiliari\workdir\markitect-native`: hook logs, session IDs, `claude agents --json` snapshots and the target's refs |

## Results

All on Windows, on the owner's stored login, in a throwaway repository with a local bare `origin`. The weekly reading went from 15 to 16 through the mechanism session and the first three changes. Nothing reached a remote in any attempt.

| Attempt | Started by | Change | Time | State | Repair rounds |
|---|---|---|---|---|---|
| Mechanism session | `claude --bg` from inside a session | None; Haiku | under a minute | done | - |
| Library 1 | `markitect apply` | M1 from nothing | under a minute | incomplete | 0 |
| Library 2 | `markitect apply` | M1 from nothing | 7 minutes | landed | not counted, a bug since fixed |
| Library 3 | `/markitect-apply` in a Manual-mode session | M2, loan limit 5 to 3 | 3 minutes | landed | 1 |
| Verify | `markitect verify --full` | None | a few minutes | verified, 19 items | - |
| Library 4 | `markitect apply` from a linked worktree | M3, short-loan copies | 4 minutes | landed on the owner's branch | 0 |
| Library 5 | `markitect apply` from a linked worktree, after the review's fixes | M4, one renewal per loan | 3 minutes | landed on the owner's branch | 0 |

**Library 1 failed on a bug in the tool:** a required check's folder that does not exist yet, as in a new project, counted as a broken environment. It stopped within seconds and landed nothing, as it should have.

**Q1: yes.** A background session starts from inside another session, also through the skill's injected command in Manual mode and from a linked worktree, in its own worktree on the commit just made. `worktree.baseRef: head` is needed for that.

**Q2: yes, with two findings.** `dontAsk`, the allow and deny rules, the hooks, the model and the forced subagent model all held, and every call in the transcripts ran on Sonnet. `--setting-sources project` did not keep the user's own `CLAUDE.md` out of a background session. `claudeMdExcludes` does, and the session's own transcript then holds none of it. The account's email still reaches the session's context. Writes to protected folders such as `.husky` and `.vscode` are refused in `dontAsk`, also through the shell, so a spec that asks for such a file fails with no defect in the spec.

**Q3: yes.** The session started the reviewer, which had only its narrowed tools and whose `agent_type` reached the hook. It runs in the background, and the session picks up its report when it ends. A hook refused `markitect answer` from the implementer's type.

**Q4: yes.** The Stop hook sent the session back. The session did not push, and the deny rule refused `git push` when it was told to try.

**Q5: yes, on these small changes.** Each change landed with nobody watching. Reviewers found a real fault twice: a lone surrogate escape turned silently into a replacement character in Library 2, and a README still saying a member may hold 5 loans after the limit became 3 in Library 3, which is the kind of drift the method is for.

**Trust.** `claude --bg` refuses a folder whose own trust question was never answered, even when a normal `claude` opens there without asking because a parent folder is trusted. Only a real terminal shows the question. This needed the owner once, and the error now says what to do. A linked worktree of a trusted folder needed nothing.

**One oracle run, a mistake of scope.** The Library's independent oracle passed on Library 3 (12 of 12 cases, 2,838 steps, the two limits told apart). The owner did not want hidden cases for this version, and said so: their own use is the test. No further oracle, suite or change-series run belongs to this dispatch.

**Review.** An independent read of the code confirmed that nothing destroys pushed commits or uncommitted edits, and found twelve defects, all fixed with tests or listed in the [draft](../concepts/v4-concept.md#14-what-is-not-yet-shown), among them that a stop did not stop a running `submit`, that merging a stopped run by hand skipped its review for good, and that a later landing deleted a verify's Findings.

**Not shown.** Any rate: this is a handful of small changes on one small project. A real project's mess, a repair limit reached for real, a usage limit, a reboot and a run on Linux or macOS have not been met. The resume path and the deadline hook are tested only without a provider.
