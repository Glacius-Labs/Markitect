# Markitect Desktop: the app builds its chats' image — 2 November 2026

The [first real chat](desktop-first-real-chat-20261102.md) showed that a chat starts only when `MARKITECT_MANAGER_IMAGE` is set. The owner decided on 2 November 2026 that the app builds the image itself ("App builds it itself"). This run tries that in the native window, on the real engine. It is a source check of the chat, not a Run: no Manager starts and nothing lands.

## Questions

- **Q1, the first chat:** with no image set, does the first message build the image from the source the app carries, show that while it runs, and then get a real agent's answer?
- **Q2, a later chat:** does the next start use the image without building?
- **Q3, Stop during the build:** does Stop end the build, leave no tag and no build process behind, and does the next message build again?
- **Q4, a new source:** does an app built from changed source build a new image and remove the old one?
- **Q5, no windows:** does the whole build open no console window on Windows?
- **Q6, how long:** how long does a build take here with no cached step?

## Dispatch

| Item | Value |
| --- | --- |
| Source | Branch `dev/desktop-chat-image` from `main` at `1d263834`. `markitect-desktop` is built from it with a debugging port added through a build overlay |
| Project | The scratch Library from the [first real chat](desktop-first-real-chat-20261102.md), `%LOCALAPPDATA%\markitect-desktop-chat\library` |
| Command | `markitect-desktop` started like a double click, driven through WebView2's debugging port, under the window guard, which ends the app at the first new console window |
| Login | Claude `claude-sonnet-5-5`, effort `high`, the account login only. `MARKITECT_MANAGER_IMAGE` and every `ANTHROPIC_*`, `CLAUDE*`, `OPENAI_*` and `CODEX_*` variable are removed; `APPDATA` points at a scratch folder |
| Limits | The host's for an agent: 1 CPU, 2 GiB, 256 processes, 30 minutes per turn; one hour per build. At most 6 turns and 1 hour of wall time |
| Engine | Docker Desktop's regular engine, freed by the owner the same day. Base images and the engine's build cache stay, since other sessions use them, so Q6 uses `--no-cache` and cannot measure the first download |
| Lock | `NATIVE-RUN.lock` in Build's notes folder, created exclusively before the first build and removed after the post-checks |
| Abort | The owner's or Build's word, a failed login, a quota stop, or a new console window |
| Cleanup | The app removes each agent. The image the app built stays, as it would for a user, and images of Q4's earlier source go with the later build. Q6's image is removed. Nothing labelled `markitect.chat` may remain |
| Evidence | Outside the repository, `%LOCALAPPDATA%\markitect-desktop-chat\evidence` |

## Results

Not run yet.
