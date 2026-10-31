# Desktop

The Desktop session designs and builds Markitect Desktop, the app over the CLI that users know as Markitect ([DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli), [design](../design/markitect-desktop.md)). The owner started it on 31 October 2026 so that work goes on in parallel while the evaluations run. Its own scope is `desktop/`, `internal/adapters/desktop` and `cmd/markitect-desktop`. Shared files, such as `internal/bootstrap`, the CLI's `desktop` verb, the architecture rule table with the code map, the source packager and CI, change one writer at a time with Build. Changes to the Run engine go through the [Build and evaluation session](build.md).

## Now

- **Decided:** the app, its name and its technology (DEC-089), with its two clarifications: Markitect ships the design system under its own notice, and the chat works in a container on a copy of the project, writing its proposal into the working tree when the owner accepts. On 31 October the owner went through every area one question at a time; the [design page](../design/markitect-desktop.md) holds the result.
- **Build order:** the model first, then surroundings when blocked, and the Runs only once the evaluations settle how Runs execute ([delivery](../design/markitect-desktop.md#delivery)).
- **The old cockpit is gone:** its code left Main with [PR #637](https://github.com/Glacius-Labs/Markitect/pull/637) on 23 October. The postponed model explorer was folded into the design, and the leftover acceptance cockpit folder in WSL was removed on 31 October.
- **Docker stays free:** while the evaluations run, this session starts no container, builds no image and runs no test that reaches the container engine (owner, 31 October 2026). The chat's container work in D4 is built against a fake and runs for real only once Docker is free.
- **Built:** D1, projects and hub ([Markitect Desktop](../../internal/adapters/desktop/README.md)): `markitect desktop` opens the app in a browser window; the project list; opening, creating and cloning projects; `init`; and the hub with the model check. Tried on Windows in the browser pane at 1440 and 390 pixels wide. The areas are shown but not built.

## Next

1. D2, the native shell, with a probe of Wails v3 and Tauri 2 on Windows, Linux and macOS.
2. D3, the four Model views and hand edits.
3. With D4, the Extract skill gains its stop at the Package cut. The skill is a shared file, changed one writer at a time with Build.

## Questions for the owner

None open now. For the Runs phase: how a project created in the app gets the remote that landing needs, since a Run without one keeps its work on its Run branch. Recommendation, to confirm then: creating a project also creates its remote, a bare repository in the app's data folder until GitHub repositories can be created, and the project pulls from it like from any remote.
