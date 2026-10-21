# Cockpit app window and notifications — 21 October 2026

**Scope:** the window and notification part of the [desktop-app Rule](../design/cockpit.md#the-cockpit-as-desktop-app) ([DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply)), implemented in the [cockpit adapter](../../internal/adapters/cockpit/README.md#app-window-notifications-and-background). Starting and stopping runs, projects and guided setup are other slices. No provider run was involved.

## What was built

- `markitect cockpit` opens the address it prints in an app window of an installed Chromium browser unless `--no-open` is given. `WindowLaunches` chooses the command from the platform, the environment and the installed programs; `OpenWindow` starts the first that works, detached. A failure is reported on stderr and never stops the cockpit.
- The page notifies, with the owner's permission, when a run ended, when the model session answered or halted and when a proposal waits for a decision. Each part of the observation takes its baseline when it first arrives; nothing is raised while the window is visible and focused. A hidden window keeps polling once notifications are allowed and still notifies, delayed by the browser's throttling; a closed window cannot notify. The bell in the top bar is a status (turn on, on, blocked) and asks only on a click.
- The window title carries the count of proposals that wait for a decision.
- Closing the window stops nothing: no page script or server route ties the server to the window, and the README says so.

## Checks

Run on Windows from the worktree of branch `dev/cockpit-window-notify`:

- `go test` of `internal/adapters/cli`, `internal/adapters/cockpit` and `internal/bootstrap`, with `go vet`. New tests: the opening commands for Windows (Edge, Chrome, path, none), macOS, Linux, WSL (the Windows browsers probed through `/mnt/c` and run directly, then the default browser) and the default-browser fallback; the CLI opens by default, not with `--no-open`, and reports a failed opening on stderr with exit code 0 while the address is still printed.
- The notification comparison (`snapshot`, `changes`) was exercised in Node on a constructed before and after observation: one notification each for the ended run (with its outcome), the answered or halted session and the new proposal, none for an unchanged observation, and none for a part that arrived late (it only sets its baseline).
- `gofmt`, the architecture gate, `markitect check`, `node --check` of the changed modules, `check-docs.py` and `git diff --check`: see the commit report.

## Not established

- No real browser was started by the tests: the choice is verified, the process start is a thin wrapper. Opening on macOS and under WSL was not tried on those systems.
- Notification permission, the bell states and the click targets were not exercised in a real browser window, and no real run, session or proposal triggered a notification.
- Whether the owner finds the notifications timely or noisy is open; the four-second poll bounds their delay.
