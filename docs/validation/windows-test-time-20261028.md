# Why the Go tests are slow on Windows — 28 October 2026

The owner asked on 27 October 2026: "Yes, find out why the tests are slow on Windows and what would fix it. Measure and report only, no code changes yet." The platform lane measured it on the shared Windows development machine on head `7db54909`. This record keeps the numbers; the fix decision is tracked on the [topics page](../workstreams/topics.md).

## Setup

- **Command:** `go test ./... -count=1` on Windows 11, compared with Linux CI on the same source.
- **Second run:** every Git start was counted through a logging wrapper on `PATH` (scratch tooling, not in the repository).

## Result

| | Windows | Linux CI |
|---|---|---|
| Full suite | about 50 min | about 3 min |
| `internal/adapters/gitrepo` | 2,985 s | about 119 s |
| One Git process start | about 23 ms idle, 35–55 ms under test load | about 1 ms |

- **Cause: Git process starts.** The suite started Git about 90,000 times, 77,000 of them from `gitrepo`'s tests. That accounts for nearly all of `gitrepo`'s Windows time.
- **Where the `gitrepo` starts come from:** `rev-parse` 22,000, `mktree` 18,000, `ls-tree` 17,000, `ls-files` 7,000; everything else is under 2,000 each. `mktree` and `ls-tree` come from rebuilding directory trees one level at a time after every snapshot (`withDirectoriesUsing` in `internal/adapters/gitrepo/local_run.go`). `rev-parse` resolves revisions one process at a time.
- **Two Windows extras:** `git` on `PATH` is Git for Windows' launcher, which starts the real `git.exe` as a second process (about 23 ms instead of 10 to 12 ms per start). The 103 `gitrepo` tests run one after another; parallel starts give about 3.5 to 4 times the throughput.
- `codexappserver`, `claudecode` and `bootstrap` also took 3 to 5 minutes each on Windows, mostly through the same Git starts.

## Fix options (not built)

1. Build the directory trees in one step or in Go (about 46 % of `gitrepo` starts).
2. Resolve revisions in one long-lived Git process, as blob reads already do with `cat-file --batch` (about 29 %).
3. Call the real `git.exe` instead of the launcher.
4. Run independent `gitrepo` tests in parallel.

Options 1 and 2 alone cover over 80 % of the starts. A further question is whether empty folders need to exist in Git at all; a placeholder file would remove the directory rebuild.

## Limits

One machine, one measurement per configuration, under the load of other sessions. The per-start costs are averages from the counting run, not a controlled benchmark.
