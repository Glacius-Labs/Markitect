# Markitect Desktop: the first real chat — 2 November 2026

Until now the [chat](../../internal/adapters/desktop/README.md#the-chat) ran only against scripted agents, since Docker stayed free for the evaluations. The owner freed it on 2 November 2026 ("Docker is now free"). This run puts the chat against a real container and a real provider, through the app's own page. It is a source check of the chat, not a Run: no Manager starts and nothing lands.

## Questions

- **Q1, a turn:** does a message reach a real agent in its container, with what it does showing while it works, and its answer in the panel?
- **Q2, the proposal:** does a change the agent makes reach the views as a proposal, and does an accept write only that into the working tree, with no commit?
- **Q3, the same conversation:** does a second message continue the provider's conversation, remembering the first?
- **Q4, a reject:** does a reject drop the proposal and leave the working tree alone?
- **Q5, outside the model:** is a change outside `.markitect/` kept out of the proposal and named to the owner?
- **Q6, Stop:** how long does a real agent take to stop, is its container gone afterwards, and does the next message start a fresh agent that knows the conversation?
- **Q7, quitting:** does quitting the app remove the agent, and does the conversation continue after a restart?

## Dispatch

| Item | Value |
| --- | --- |
| Source | `main` at `3c707992`. `markitect.exe` and the Manager image `markitect-manager:desktop-3c707992` (`sha256:f30220f7…`) are built from it. A fix a run shows gets a rebuilt `markitect.exe`, and a rebuilt image if it reaches the image, and is named under Results |
| Project | A scratch repository outside this one, `%LOCALAPPDATA%\markitect-desktop-chat\library`: the Library M1 model from `tools/evaluation-tasks/library/m1`, one commit, no remote |
| Command | `markitect desktop --listen 127.0.0.1:7350 --no-open`, with the page in the session's browser pane, so nothing opens on the owner's screen |
| Login | Claude `claude-sonnet-5-5`, effort `high`, the account login only, never an API key. Every `ANTHROPIC_*`, `CLAUDE*`, `OPENAI_*` and `CODEX_*` variable is removed from the app's environment; `APPDATA` points at a scratch folder, so the owner's project list stays untouched |
| Limits | The host's for an agent: 1 CPU, 2 GiB, 256 processes, 30 minutes per turn. At most 12 turns in all and 2 hours of wall time |
| Engine | Docker Desktop's regular engine. Before the first start, `docker ps` lists no container |
| Lock | `NATIVE-RUN.lock` in Build's notes folder, created exclusively before the image build and removed after the post-checks |
| Abort | The owner's or Build's word, a failed login, a quota stop, or a credential found in the evidence |
| Cleanup | The app removes each agent's container and volumes. Afterwards nothing labelled `markitect.chat` may remain. The `markitect-provider` network and the image stay, like after a Run |
| Evidence | Outside the repository, `%LOCALAPPDATA%\markitect-desktop-chat\evidence`: the app's log, its saved conversations, the project's final state and a scan of every file for the login's tokens |

## Pass criteria

Each question passes when the behavior it names is seen on the real agent. A failed attempt stays recorded below, with its cause and fix.

## Results

Run on 2 November 2026 from 15:06 to 15:26 UTC, 11 provider turns and one send that never reached the provider. Before the first start the engine ran no container. Both fixes below reach only the host, so the image stayed the same. All seven questions passed, after two defects that no scripted agent could show.

**Attempts:**

| Build | What it ran | Result |
| --- | --- | --- |
| `3c707992` | The first message | **Failed at start:** `control message "initialize": exactly its own complete body is required`. `harness.Chat` rendered no `Context`, and agentd requires both of Claude Code's prompt files. Scripted sessions accepted it, so no Claude chat could ever have started. The failed start left no container and no volume |
| Fix 1: the chat's rules in `Base`, its skill in `Context` | Six turns: Q1 to Q6, and a clean quit for Q7 | The questions below passed, except that **a running agent did not know about a reject**: asked whether the rejected sentence was in the model, it said yes, from memory. It was not |
| Fix 2: a running agent hears what the app said since its last answer | Four turns: a restart, the reject again, a hard kill, a working tree with CRLF line ends | Passed |
| Fix 2, native app under the [window guard](#the-native-window) | One turn | Passed |

**Questions:**

| Question | Verdict |
| --- | --- |
| Q1, a turn | **Passed.** "Raise the loan limit from 5 to 7" took 14 s. The panel showed "Starting the chat's agent", the searches, "Editing .markitect/model/lending/loan-policy.md", "Running markitect check; markitect lint; markitect impact", "Reading what changed" and the answer. The container ran on `markitect-provider` with the two labelled volumes, at 141 MiB |
| Q2, the proposal | **Passed.** The views showed `7` with the working tree still at `5`, and the hub said that a proposal waits. The accept wrote that one line and made no commit, and `markitect check` passed |
| Q3, the same conversation | **Passed.** Asked which value it should keep, the agent named the loan period. This is the first native use of `--resume` |
| Q4, a reject | **Passed** with fix 2. The working tree stayed as it was and the copy went back, its outside file gone too. The agent then answered: "No, you rejected that change", after searching the model |
| Q5, outside the model | **Passed.** Told to write `NOTES.md` at the project root, the agent did. The proposal held only the model file, and the panel said "Also changed outside the model, and not proposed: NOTES.md" |
| Q6, Stop | **Passed.** The turn ended about one second after the click, not the ten the code allows. Its container and volumes were gone, its edits were not in the working tree, and the panel said so. The next message started a fresh agent, which told the accepted change from the rejected one |
| Q7, quitting | **Passed.** A clean quit took 0.5 s with an agent running and left nothing. After the restart the conversation went on with a fresh agent. A hard kill during a turn removed the container and left the two volumes, without the login in the home volume; the conversation's next start swept them |

**Line ends:** with the working tree in CRLF, as Git checks it out on this Windows machine, an accepted change showed as one line in `git diff`, and the file kept CRLF throughout.

**Afterwards:** nothing labelled `markitect.chat` remained, no other container ran, and none of the 28 files of the evidence, the app's folder and the project held either of the login's tokens.

### The native window

One turn ran in `markitect-desktop`, started without a console like a double click, with the debugging port added through a build overlay. The guard saw its control window and then counted **0 new console windows** across the agent's start, its turn, the reject and the conversation's delete, which also removed the agent. The page reported no errors.

### Seen beside the criteria

- **A hard kill's volumes** stay until the same conversation starts again. A conversation that never does keeps them, even when it is deleted. They hold no login.
- **A killed turn** leaves the owner's message without an answer or a note in the conversation.
- **A turn that changes nothing** still numbers a new proposal with the same content. The panel shows only the open one, so the owner does not see this.

### Not covered

Codex in a chat, a chat with the Extract skill, Linux and macOS, a turn reaching its 30-minute limit, a login expiring during a conversation, and a large project. A chat keeps no transcript, so the turns' tokens were not counted.
