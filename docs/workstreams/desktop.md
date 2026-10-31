# Desktop

The Desktop session designs and builds Markitect Desktop, the app over the CLI that users know as Markitect ([DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli), [design](../design/markitect-desktop.md)). The owner started it on 31 October 2026 so that work goes on in parallel while the evaluations run. It builds only the app's own scope: `desktop/`, `internal/adapters/desktop` and `cmd/markitect-desktop`. Changes to the Run engine, such as the event port for the live view, go through the [Build and evaluation session](build.md).

## Now

- **Decided:** the app, its name and its technology (DEC-089).
- **The old cockpit is gone:** its code left Main with [PR #637](https://github.com/Glacius-Labs/Markitect/pull/637) on 23 October. The postponed model explorer was folded into the design, and the leftover acceptance cockpit folder in WSL was removed on 31 October.
- **Design:** the [design page](../design/markitect-desktop.md) is a proposal; nothing is built.

## Next

1. D1, projects and hub, once the questions below are answered.
2. D2, the native shell, with a probe of Wails v3 and Tauri 2 on Windows, Linux and macOS.

## Questions for the owner

1. **The design system's license.** Markitect is public under Apache 2.0. The design system is a private package with no license. The app ships the design system's CSS, fonts and icons inside Markitect's repository and binary, which publishes them. Recommendation: allow it, with a license for the built package that permits redistribution with Markitect. The fonts, Inter and Roboto Mono, already have open licenses.
2. **Where the chat works.** Claude Code Desktop edits the user's folder directly. Markitect's agents work only in containers and never touch the working tree ([concept 2.0 §6.5, §9](../concepts/v2-concept.md#65-landing)). Recommendation: the chat follows Markitect's rule. Its changes arrive as a proposal to accept, and the user pulls them like any landed change.
