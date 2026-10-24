# Owner instructions for the Agent runtime

Date: 17 October 2026, morning. Verbatim owner messages from a Claude Code session, in their original order.

Recorder's note: The session first analyzed the overnight rebuild work. Most time went to coordination and waiting. The largest open risk for the coming session work was the agent runtime: the Codex Host tools depended on an experimental App Server field, a Windows desktop binary was pinned, ambient user configuration was not controllable, and the earlier native smoke had needed 28 attempts. The assistant recommended a small real handshake before the session design hardened.

## Run the handshake and design a better runtime

> Ja, leg los mit dem Handshake-Test. Ansonsten denke dir gerne ein besseres Konzept aus wenn du hier Probleme erwartest. Wir können auch über die Claude CLI gehen, wollen aber beides unterstützen. Eventuell wären auch Container eine Idee. Wir brauchen eine solide Runtime und wollen hier nicht mit irgendwelchen Windows- oder Pfadproblemen oder Berechtigungen kämpfen.

Recorder's note: The [handshake record](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/runtime-handshake-20261017.md) holds the results. Both CLIs worked with a Host tool, on the host and in a Linux container. On the host, both inherited the user's configuration; in the container they ran clean. The assistant proposed one Linux container per Agent session from a pinned image, with authority given by mounts, workspaces on a Linux volume, Host tools over MCP for both providers through a session helper on one stdio connection, a Codex and a Claude adapter, dedicated runtime credentials, and checks in the same image. Docker or a compatible engine becomes required for real Apply runs.

## Adopt the runtime and implement it

> Ja, schreib das ins Konzept und bring es auf main. Wenn du der Meinung bist dass der Codex Coordinator es nicht anfässt, kannst du es auch gerne umsetzen. Oder du setzt es dann um und ich gebe dem Codex Coordinator eine Nachricht von dir damit er bescheid weiß

Recorder's note: Recorded as [DEC-037](../register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers) with the open question [OQ-010](../register.md#open-questions) on subscription logins.
