# Desktop

The Desktop session designs and builds Markitect Desktop, the app over the CLI that users know as Markitect ([DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli), [design](../design/markitect-desktop.md)). The owner started it on 31 October 2026 so that work goes on in parallel while the evaluations run. Its own scope is `desktop/`, `internal/adapters/desktop` and `cmd/markitect-desktop`. Shared files, such as `internal/bootstrap`, the CLI's `desktop` verb, the architecture rule table with the code map, the source packager and CI, change one writer at a time with Build. Changes to the Run engine, such as the event port for the live view, go through the [Build and evaluation session](build.md).

## Now

- **Decided:** the app, its name and its technology (DEC-089).
- **The old cockpit is gone:** its code left Main with [PR #637](https://github.com/Glacius-Labs/Markitect/pull/637) on 23 October. The postponed model explorer was folded into the design, and the leftover acceptance cockpit folder in WSL was removed on 31 October.
- **Design:** the [design page](../design/markitect-desktop.md) is a proposal; nothing is built.

## Next

1. D1, projects and hub, once question 1 below is answered.
2. D2, the native shell, with a probe of Wails v3 and Tauri 2 on Windows, Linux and macOS.

## Questions for the owner

1. **The design system's license.** The design system is proprietary: it may be used only in Glacius Labs projects, which Markitect is. But Markitect is public under Apache 2.0, and the app ships the design system's CSS inside Markitect's repository and binary, which publishes it. Its fonts (Inter and Roboto Mono, Open Font License) and icons (Lucide, ISC) are already open. Recommendation: allow Markitect to redistribute the built package under its own proprietary notice, kept apart from Apache 2.0.
2. **Where the chat works.** Claude Code Desktop edits the user's folder directly. Markitect's agents work only in containers and never touch the working tree ([concept 2.0 §6.5, §9](../concepts/v2-concept.md#65-landing)). Recommendation: the chat follows Markitect's rule. Its changes arrive as a proposal to accept, and the user pulls them like any landed change.
