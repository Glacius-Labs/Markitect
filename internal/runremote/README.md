# The Run remote and its hooks

Every Run of [concept 2.0](../../docs/concepts/v2-concept.md) has its own Git remote: one Linux container with a bare repository, the server-side hooks and the required checks (§6.3, §9). The Managers' containers clone from it and push to it over the Run's container network. The hooks hold the guarantees, so no agent can skip them. This page is the canonical description of that remote, including the verdict trailer format, which §14 left to the build. It is step T3 of the [delivery plan](../../docs/implementation-plan.md#concept-20-build).

| Package | Role |
| --- | --- |
| `internal/runremote` | The plan, the trailer format and `Decide`, the hook's decision over repository facts. It runs no Git and no process. |
| `internal/adapters/remoteserver` | The program inside the container: Git smart HTTP with one credential per agent, the pre-receive hook, the required checks, seeding and export. |
| `internal/adapters/remotecontainer` | The host side: starts and removes the container with the Docker CLI only, natively on Windows, Linux and macOS. |
| `cmd/markitect-remote` | The container's entrypoint (`seed`, `serve`, `hook`, `ready`, `export`, `log`). |

## The plan

The host writes one plan per Run before it starts ([`plan.go`](plan.go)); it never changes while the Run runs, because the model, its registry and `execution.yaml` cannot change in a Run and so the base states them for the whole Run.

- `base`: the commit the Run starts from.
- `managers`: every Package, working or not, with its parent, its path patterns at the base, its registry page and, for a working Manager, its checklist (from impact, step T1).
- `agents`: one maker and one checker per working Manager (checkers only in `verify`), each with a user name and the SHA-256 of its token. The remote never holds a token.
- `protected`: path patterns no push may change: `.markitect/**` in practice, which covers the model, the manifest, the lock, `execution.yaml`, Libraries and Results.
- `checks`: the required checks of `execution.yaml`.

`Validate` refuses an inconsistent plan; a Run never starts on one.

## Branches

The remote holds `base` (read-only, the starting commit), `root` for Root and `managers/<path>` for every other working Manager, with each `/` of the Package path written as `.` (`members/cards` → `managers/members.cards`), because Git cannot hold both `a` and `a/b` as branches. A plan whose Packages map to one branch, or to an invalid ref name, is refused. Every branch starts at the base. Its maker and its checker push to it; everybody reads every branch.

## What the hooks refuse

The pre-receive hook decides every push, one push at a time, and prints the reason and a next step for the pusher. The decision log (`markitect-remote log`) keeps every push with its outcome and checks.

1. **Another ref.** A push may update exactly one ref, the agent's own branch. Other branches, `base`, new branches, tags and notes are refused.
2. **Rewritten history.** Deleting the branch, or a push the old tip is not an ancestor of (force push, rebase, amend of pushed commits), is refused.
3. **Files outside the area.** The new tip must keep the Manager's area:
   - a file the Manager owns is free;
   - a file owned by a Package below it must be exactly as in the newest approved commit of the child toward that Package that the tip contains, or as at the base when it contains none; so a parent can neither alter a child's approved work nor slip edits into a child's area, and unapproved child work cannot be merged;
   - a file outside its subtree must be exactly as in the newest state of its parent's branch that the tip contains, or as at the base; so siblings meet only through their parent, and a child cannot undo a parent's change that would reach the parent at its merge;
   - a registry page belongs to its own Package, whoever owns the paths around it;
   - a protected file stays as at the base, for every Manager, Root included.

   "Owns" uses the base's registry: the most specific pattern wins, Root owns what no pattern claims (concept §4). The rule compares content, not commit identity, so a content-equal copy of approved work is accepted. The parent's states and the children's approvals come from the remote's reflog, which Git writes in the same transaction as every update and which no agent can reach.
4. **Trailers in the wrong hands.** Review lines from anyone but the branch's checker, verdict lines from a checker in `apply`, and trailers that name another Manager are refused in every new commit.
5. **An incomplete submission.** A maker's tip that is a submission must give every checklist item exactly one verdict and nothing else.
6. **A failing submission.** The required checks then run on the tip's full repository state, in order, until one fails; a failure, a timeout or checks that cannot run refuse the push. Work commits without verdicts run no check. A parent's submission is its merged state, so the checks run on that as well.
7. **A review that is no review.** A checker pushes exactly one commit, whose only parent is the current tip, which must be a submission of this Run; the commit changes no file. An approval confirms every submitted verdict and disputes none; a rejection disputes at least one. A checker never edits.
8. **Verify that changes something.** In `verify` a checker pushes one commit on its tip with the base's tree and a complete set of verdicts; no checks run, since nothing changes.

The bare repository's own settings hold as a second line: no deletes, no non-fast-forward updates, `fsck` on received objects, no push options, no automatic garbage collection.

## Verdict trailers

Verdicts and reviews are [Git trailers](https://git-scm.com/docs/git-interpret-trailers) in the last paragraph of a commit message. Other trailers such as `Co-Authored-By` may stand beside them; a `Markitect-` line anywhere else is refused. A line that starts with white space continues the one before.

| Key | Value | In |
| --- | --- | --- |
| `Markitect-Manager` | the Package ID of the Manager (`.` for Root) | every submission and review |
| `Markitect-Verdict` | `ITEM \| conforms\|changed\|model-problem \| EVIDENCE` | a submission |
| `Markitect-Review` | `approved` or `rejected` | a review |
| `Markitect-Check` | `ITEM \| confirmed\|disputed \| EVIDENCE` | a review |

`ITEM` is a checklist item as the plan names it (a Definition ID, or a Finding's item); it holds no `|`. `EVIDENCE` is required and may hold anything, `|` included. A maker's submission:

```text
Realize the loan limit

Markitect-Manager: loans
Markitect-Verdict: loans/loan | changed | limit 3 in src/loans/limit.go; loan_test.go covers 3 and 4
Markitect-Verdict: rules/limits | conforms | no other limit in src/loans
```

Its checker's answer, as `git commit --allow-empty` on the submission:

```text
Review loans

Markitect-Manager: loans
Markitect-Review: rejected
Markitect-Check: loans/loan | disputed | loan_test.go never tests a member at the limit
Markitect-Check: rules/limits | confirmed | read src/loans
```

In `verify` the checker writes the `Markitect-Verdict` lines itself, without a maker. [`trailers.go`](trailers.go) parses and writes the format.

## Required checks

The hook writes the tip's tree into a fresh directory and makes it a Git repository whose one commit has exactly that tree; it verifies the tree ID, so a check sees the branch's full state as a clean checkout (submodules are not supported). Each check runs with `/bin/sh -c COMMAND` in its directory, with its timeout, and with only `PATH`, a fresh `HOME` and `TMPDIR`, `LANG=C.UTF-8` and the check's own `environment`; nothing else of the image's environment reaches it, so `execution.yaml` states what a check needs (for example `GOTOOLCHAIN` or `GOFLAGS`). The Run's network is internal, so checks run offline.

Checks run code the agents wrote, so they never run as the remote. Each check run takes a fresh unprivileged user (uid 61000 and up) that can neither read the plan and the decision log nor write the repository, the hook or the program; afterwards every process of that user is killed, also those that left their process group, and the copy is removed. Checks are the first victims of the container's memory limit. Pushes are taken one at a time, so checks never run side by side.

## The container

`remotecontainer.Start` creates the Run's internal network `markitect-run-<run>` and the container `markitect-run-<run>-remote` with the alias `remote`, so the repository is `http://remote:8080/run.git` on that network. The container runs the remote as root with only the capabilities it needs (`CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `KILL`, `SETUID`, `SETGID`), `no-new-privileges`, an init process and the given CPU, memory and process limits. The host copies the program and the plan in as a tar stream, starts the container, streams the start bundle into `seed` and waits until it serves; `StartBundle` makes that bundle from the user's repository without changing it. `Export` streams a bundle of every branch out, `Log` the decision log. `Close` removes the container and the network; `Sweep` removes everything that carries the Run's label, also after a host that stopped without cleaning up. A failed start removes what it created. No folder is mounted.

The image must hold Git, a POSIX shell, coreutils and what the required checks need; it is the project's check image. The program is `markitect-remote` built for Linux on the engine's architecture (`Engine.Arch`); how the host obtains it is the native host's matter (step T5).

## Open interfaces

- **Step T1.** The plan's ownership, checklists, registry page paths, Package IDs and checks come from T1's registry, impact and `execution.yaml`. Until T1 lands, `PatternOwnership` stands in for its pattern semantics, which T1 owns; the hook then uses T1's.
- **Step T4.** The engine gives each agent its credentials and puts Manager containers on the Run's network beside the network they reach their provider through. Landing must also check that Root's approved commit contains every working Manager's latest approval and holds no account credential. Messages, the Result page and landing are T4's.

## From the G0 prototype

T3 starts from the [G0 prototype](https://github.com/Glacius-Labs/Markitect/tree/dev/g0-prototype/prototypes/g0) without its referee: the guarantees sit in the hook, and handoffs, reviews and merges are commits. Of its [pre-run review](https://github.com/Glacius-Labs/Markitect/blob/dev/g0-prototype/prototypes/g0/review-20261028.md), protected inputs (4) are rule 3; orphaned Runs (8) are `Sweep` and the cleanup of a failed start; the resource limits of (9) are `Limits`; checks no longer run beside the credentials, and the remote is exported as a bundle. Check placement (1) is decided otherwise by the concept: every submission passes the required checks on its branch's full state (§6.3), so a child keeps the repository green and takes its parent's shared changes when it needs them. The referee's acceptance bookkeeping, deadlock watch, nudges and token counting (2, 3, 5, 6, 10) do not exist here; their successors belong to T4. The secret scan (7) belongs to landing (T4).

## Tests

`internal/runremote` has unit tests for the trailers, the plan, branch names and ownership. `internal/adapters/remoteserver` has a test for every refusal with scripted Managers that push over real Git smart HTTP to a real hook. `internal/adapters/remotecontainer` starts real Run containers when `MARKITECT_TEST_ENGINE` (absolute path of the Docker CLI) and `MARKITECT_TEST_REMOTE_IMAGE` (an image with Git, a shell and coreutils) are set: scripted Managers in their own containers try to get around the hooks (other branches, force push, deletion, paths, anonymous and wrong credentials), a check tries to read the plan, write the hook, the program and the log, kill the remote and outlive itself, the result is exported, and nothing of the Run remains.
