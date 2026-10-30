# The Run remote and its hooks

Every Run of [concept 2.0](../../docs/concepts/v2-concept.md) has its own Git remote: one Linux container with a bare repository, the server-side hooks and the required checks (§6.3, §9). The Managers' containers clone from it and push to it over the Run's container network. The hooks hold the guarantees, so no agent can skip them. This page is the canonical description of that remote, including the verdict trailer format, which §14 left to the build. It is step T3 of the [delivery plan](../../docs/implementation-plan.md#concept-20-build).

| Package | Role |
| --- | --- |
| `internal/runremote` | The plan, the trailer format and `Decide`, the hook's decision over repository facts. It runs no Git and no process. |
| `internal/adapters/remoteserver` | The program inside the container: Git smart HTTP with one credential per agent, the pre-receive hook, the required checks, seeding and export. |
| `internal/adapters/remotecontainer` | The host side: starts and removes the container with the Docker CLI only, natively on Windows, Linux and macOS. |
| `cmd/markitect-remote` | The container's entrypoint (`seed`, `serve`, `hook`, `ready`, `check`, `start`, `export`, `log`). |

## The plan

The host writes one plan per Run before it starts ([`plan.go`](plan.go)); it never changes while the Run runs, because the model, its registry and `execution.yaml` cannot change in a Run, so the base states them for the whole Run. `PlanFor` builds it from the project at the base and the change's impact, both from the [model language](../../docs/guides/model-language.md#impact-and-checklists):

- `base`: the commit the Run starts from.
- `managers`: every Package (Root's ID is `""`) with its path patterns at the base; a working Manager, one impact names, also with its checklist.
- `pages`: the registry pages a Run may change, each with the Package whose Manager changes it: a live Package's page, whether it exists yet or not, and the page of a Package that no longer exists, which Root cleans up.
- `agents`: added by the host; one maker and one checker per working Manager (checkers only in `verify`), each with a user name and the SHA-256 of its token. The remote never holds a token.
- `checks`: the required checks of `execution.yaml`.

`Validate` refuses an inconsistent plan; a Run never starts on one.

**Checklist items** are named as verdict trailers name them (`Items`): a Definition by its ID, own or inherited; the n-th Finding of the Manager as `finding n on ID`; stale realization by its code, subject and place, such as `stale.file src/old.go at .markitect/registry/members.md:12`. A parent that only integrates has no items.

## Branches

The remote holds `base` (read-only, the starting commit), `root` for Root and `managers/<path>` for every other working Manager, with each `/` of the Package path written as `.` (`members/cards` → `managers/members.cards`), because Git cannot hold both `a` and `a/b` as branches; Package names hold no `.`, so no two Packages share a branch. Every branch starts at the base. Before a Manager below Root starts, the host may move its branch, still at the base, to a state its parent's branch held, its parent's approval (`StartFrom`, `markitect-remote start`; [parent first](../run/README.md#schedule)); the decision log records it as user `host`, kind `start`. No agent can do that. Its maker and its checker push to it; everybody reads every branch.

## What the hooks refuse

The pre-receive hook decides every push and prints the reason and a next step for the pusher. The decision log (`markitect-remote log`) keeps every push with its outcome and checks. Pushes are taken one at a time, so each is decided against a repository no other push changes; a push waits while another's checks run.

1. **Another ref.** A push may update exactly one ref, the agent's own branch. Other branches, `base`, new branches, tags and notes are refused.
2. **Rewritten history.** Deleting the branch, or a push the old tip is not an ancestor of (force push, rebase, amend of pushed commits), is refused. The tip must be a commit.
3. **Files outside the area.** The new tip must keep the Manager's area:
   - a file the Manager owns is free;
   - a file owned by a Package below it must be exactly as in the newest approved commit of the child toward that Package that the tip contains, or as at the base when it contains none; so a parent can neither alter a child's approved work nor slip edits into a child's area, and a child's later unapproved work cannot reach the parent's tip;
   - a file outside its subtree must be exactly as in the newest state of its parent's branch that the tip contains, or as at the base; so siblings meet only through their parent, and a child cannot undo a parent's change that would reach the parent at its merge;
   - a registry page belongs to the Package `pages` names;
   - everything else below `.markitect/` stays as at the base, for every Manager, Root included: the model, the manifest, the lock, `execution.yaml`, the ignore page, Libraries and Results;
   - a file that two Packages claim with equally specific patterns, which the compiler refuses, is refused;
   - a changed path that differs from another path of the tip only in case is refused, because a checkout on Windows or macOS holds both as one file. Unicode normalization is not compared.

   "Owns" uses the base's registry with the compiler's own pattern rules (`realization.Pattern`): the most specific pattern wins, Root owns what no pattern claims. The rule compares content with the newest approval the tip contains, not commit identities. The parent's states and the children's approvals come from the remote's reflog, which Git writes in the same transaction as every update and which no agent can reach; the repository uses the files ref format, and a missing reflog fails the push.
4. **Trailers in the wrong hands.** Review lines from anyone but the branch's checker, verdict lines from a checker in `apply`, and trailers that name another Manager are refused in every new commit.
5. **An incomplete submission.** A maker's tip that is a submission must give every checklist item exactly one verdict and nothing else.
6. **A submission that does not compile.** The hook compiles the tip's full repository state as `markitect check` does and refuses it with the compile errors, so a file that enters unregistered, even in the Manager's own area, never reaches a submission.
7. **A failing merged state.** The required checks then run on the tip's full repository state, in order, until one fails. They refuse only a merged state ([DEC-087](../../docs/concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)): a submission that contains an approval of one of the Manager's children, as rule 3 finds them. A failure or a timeout there refuses the push. Any other submission, a child's or a parent's first one, is a partial tree that a check of the whole repository may fail when the build needs a child's files: it is accepted, the pusher sees the failure as a note, and the decision log keeps the results, which the engine gives its checker as evidence ([Run engine](../run/README.md#the-result)). Checks that cannot run refuse every submission. Work commits without trailers are neither compiled nor checked. The state that lands is checked by the host, not the hook (`Check` below).
8. **A review that is no review.** A checker pushes exactly one commit, whose only parent is the current tip, which must be a submission of this Run; the commit changes no file. An approval confirms every submitted verdict and disputes none; a rejection disputes at least one, unless the submission has no items. A checker never edits.
9. **Verify that changes something.** In `verify` a checker pushes one commit on its tip with the base's tree and a complete set of verdicts. Since every such commit holds the same tree, the required checks run once, on Root's verdicts, as in every Run, and the decision log keeps their results; a failing check does not refuse the push, since nothing changes and the checker cannot fix it, but checks that cannot run do. It is not compiled again: its tree is the base, which the host compiled before the Run started.

The bare repository's own settings hold as a second line: no deletes, no non-fast-forward updates, `fsck` on received objects, no push options, no automatic garbage collection.

## Verdict trailers

Verdicts and reviews are [Git trailers](https://git-scm.com/docs/git-interpret-trailers) in the last paragraph of a commit message. Other trailers such as `Co-Authored-By` may stand beside them; a `Markitect-` line anywhere else, or a `Markitect-` key in another case, is refused. A line that starts with white space continues the one before.

| Key | Value | In |
| --- | --- | --- |
| `Markitect-Manager` | the Manager's Package ID, `.` for Root | every submission and review |
| `Markitect-Verdict` | `ITEM \| conforms\|changed\|model-problem\|deviates \| EVIDENCE` | a submission |
| `Markitect-Review` | `approved` or `rejected` | a review |
| `Markitect-Check` | `ITEM \| confirmed\|disputed \| EVIDENCE` | a review |

`ITEM` is a checklist item as above; it holds no `|`. `EVIDENCE` is required and may hold anything, `|` included. For `conforms` and `changed` it names how each example in the item's Definition is met ([DEC-088](../../docs/concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)); the hook cannot see examples, so the checker holds the maker to it. A submission is a commit with a `Markitect-Manager` line and its verdicts; a Manager without items submits with the `Markitect-Manager` line alone. A maker's submission:

```text
Realize the loan limit

Markitect-Manager: members/loans
Markitect-Verdict: members/loans/loan | changed | limit 3 in src/loans/limit.go; loan_test.go covers 3 and 4
Markitect-Verdict: loan-limit | conforms | no other limit in src/loans
```

Its checker's answer, as `git commit --allow-empty` on the submission:

```text
Review members/loans

Markitect-Manager: members/loans
Markitect-Review: rejected
Markitect-Check: members/loans/loan | disputed | loan_test.go never tests a member at the limit
Markitect-Check: loan-limit | confirmed | read src/loans
```

In `verify` the checker writes the `Markitect-Verdict` lines itself, without a maker, and `deviates` says that the realization does not conform to the item, with the defect as evidence; the engine turns it into a Finding. In `apply` a maker never writes `deviates`: it makes the realization conform, or reports `model-problem`. [`trailers.go`](trailers.go) parses and writes the format.

## Required checks

A check's `environment` is the image it runs in ([build decisions](../../docs/guides/model-language.md#build-decisions)), and the remote runs every check of a Run inside its own container, so all checks of a project name one image, which is the remote's image; a plan with checks in two images is refused.

The hook writes the tip's tree into a fresh directory and makes it a Git repository whose one commit has exactly that tree; it verifies the tree ID, so a check sees the branch's full state as a clean checkout (submodules are not supported: the checks then cannot run). Each check runs with `/bin/sh -c COMMAND` in its directory, with its timeout, in the image's environment as the remote recorded it at seeding, with a fresh `HOME` and `TMPDIR`; every other variable that pointed into the image's home, `PATH` excepted, points into the fresh one, so caches there are not shared between check runs. The Run's network is internal, so checks run offline.

Checks run code the agents wrote, so they never run as the remote. Each check run takes the next of 2000 unprivileged users (uid 61000 and up, reused in turn), which can neither read the plan and the decision log nor write the repository, the hook or the program. At seeding the remote takes write access for others from every file of the image and records the folders every user may write. After a check run it kills every process of that user, also those that left their process group, removes everything the user left in those folders and its System V IPC objects, and removes the copy; if anything survives, the push fails. The hook and its checks are the first victims of the container's memory limit, and a check cannot lower that; the serving remote is the last.

## The container

`remotecontainer.Start` creates the Run's internal network `markitect-run-<run>`, in isolated gateway mode so that the bridge has no address on the engine's host and no container of the Run reaches the host (this needs Docker 28 or newer; an older engine is refused before anything is created), and the container `markitect-run-<run>-remote` with the alias `remote`, so the repository is `http://remote:8080/run.git` on that network. The image is the checks' image, or `Spec.Image` when the project has none; it holds Git, a POSIX shell, coreutils and `ipcrm` besides what the checks need. The container runs the remote as root with only the capabilities it needs (`CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `KILL`, `SETUID`, `SETGID`, `SYS_RESOURCE`), `no-new-privileges`, an init process and the given CPU, memory and process limits (for `markitect apply` and `verify`, the [host's](../agents/README.md#composition)). The host copies the program and the plan in as a tar stream, starts the container, streams the start bundle into `seed` and waits until it serves; `StartBundle` makes that bundle from the user's repository without changing it. `Export` streams a bundle of every branch out, `Log` the decision log. `Check` is the host's landing check (concept §6.5): it streams in a bundle holding a commit only the host has, such as the merge of a moved target branch, and runs the required checks on that commit's full state as the hook runs them, taking turns with the hook's checks; it changes no branch of the Run, and the decision log records it as user `host`, kind `landing-check`. A failing check is a result; an error means the checks could not run. `Close` removes the container and the network, as far as they exist; `Sweep` removes everything that carries the Run's label, also after a host that stopped without cleaning up. A failed start removes what it created. No folder is mounted.

The program is `markitect-remote` built for Linux on the engine's architecture (`Engine.Arch`); how the host obtains it is the native host's matter (step T5).

## Open interfaces for step T4

- The engine gives each agent its user and token and puts the Manager containers on the Run's network beside the network they reach their provider through. Tokens travel as HTTP Basic credentials on that network, so a Manager container must not be able to spoof the remote: it runs without `NET_RAW` and `NET_ADMIN` (best with `--cap-drop ALL`).
- Landing must check that Root's approved commit contains every working Manager's latest approval and holds no account credential; the hooks do not.
- Messages, the Result page and landing are T4's.

## From the G0 prototype

T3 starts from the [G0 prototype](https://github.com/Glacius-Labs/Markitect/tree/dev/g0-prototype/prototypes/g0) without its referee: the guarantees sit in the hook, and handoffs, reviews and merges are commits. Of its [pre-run review](https://github.com/Glacius-Labs/Markitect/blob/dev/g0-prototype/prototypes/g0/review-20261028.md), protected inputs (4) are rule 3; orphaned Runs (8) are `Sweep` and the cleanup of a failed start; the resource limits of (9) are `Limits`; checks no longer run beside the credentials, and the remote is exported as a bundle. Check placement (1) is decided by the concept: the required checks run on every submission's full state, but refuse only merged states (§6.3, rule 7), since a parent's first submission is a partial tree. The referee's acceptance bookkeeping, deadlock watch, nudges and token counting (2, 3, 5, 6, 10) do not exist here; their successors belong to T4. The secret scan (7) belongs to landing (T4).

## Tests

`internal/runremote` has unit tests for the trailers, the plan and `PlanFor` over a project and its impact, checklist items, branch names, ownership, and the pushes no Git client sends. `internal/adapters/remoteserver` has a test for each refusal above with scripted Managers that push over real Git smart HTTP to a real hook, on a fixture project that compiles, and for the acceptances that matter most: a child's and a parent's first submission that fail a check, accepted with the result in the log, a parent integrating two approved children, approved work climbing two levels, a child taking its parent's state, a parent without items, and verdicts in `verify` whose check fails. `internal/adapters/remotecontainer` starts real Run containers when `MARKITECT_TEST_ENGINE` (absolute path of the Docker CLI) and `MARKITECT_TEST_REMOTE_IMAGE` (an image with Git, a shell, coreutils and `ipcrm`) are set: scripted Managers in their own containers try to get around the hooks (other branches, force push, deletion, paths, anonymous and wrong credentials); a check tries to read the plan, write the hook, the program and the log, kill the remote, leave its memory-limit rank, outlive itself and leave files and shared memory for the next check, which then finds none of it; the result is exported, and nothing of the Run remains.
