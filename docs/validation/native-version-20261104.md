# The native version: real sessions while building it, 4 November 2026

The owner chose to build [draft concept 4.0](../concepts/v4-concept.md) on a branch and test it by using it. This page records the real sessions that build needs: the mechanisms the concept only reads from the documentation, tried before and while building. It is bounded evidence on small changes, not acceptance, and says nothing about how often a run fails.

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

Recorded as they happen, with failed attempts.
