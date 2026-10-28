# Provider adapters

The [project workflow](project-workflow.md) describes the commands and where they run. This guide covers the provider side of them: explicit profiles and static metadata, native Sessions, the account login and the evidence. The [Run](../../internal/run/README.md) owns the complete Manager hierarchy and its Session, Git, check and record ports; it has no publication path. Provider qualification and owner acceptance are separate claims.

<a id="role-profiles"></a>

## Explicit profiles and static metadata

Build `cmd/markitect` and use its CLI from the coding environment. `check` and `context` are structural by default. Request static metadata with `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH`. Runtime profile selection is explicit; the product does not discover accounts or infer provider configuration.

The closed `markitect.runtime/v1` document contains native Agent and Check profiles. Outer composition selects native profiles. Project's `markitect.execution/v2` declarations contain Package instructions and scoped checks, without provider roles, teams, strategies or invocation budgets. Invocation limits and normal account-login references belong to the explicit outer run configuration.

Codex profile version 2 supports metadata only; version 3 adds the closed native session settings. The [Codex adapter](../../internal/adapters/codexappserver/README.md) documents exact fields and preparation evidence. Codex is V1's Agent adapter ([D18](../concepts/v1-concept.md#23-provider-scope-d18)). An Agent profile selects exactly one section, `codexAppServer` or `claudeCode`; the [Claude section](#claude-code) below describes the addition. [Check profiles](../../internal/adapters/checks/README.md) bind pinned images, literal commands, environment policy and finite limits.

Agent metadata preparation may inspect a selected image and copy bounded whole `/opt/markitect/runtime.env` bytes from a created, never started container. It does not launch an Agent Session or provider. Check metadata preparation is process-free. Neither establishes effective native instructions, login, runtime support or semantic completion. Structural output never reports metadata as implicitly passed.

Fixed previews refuse unbound present-day external profile or dependency inputs, even for HEAD. Working previews retain observed bytes. Neither preview supplies an artifact publication guard.

<a id="container-runtime"></a>

## Native Sessions and Git workspaces

The Apply host that starts the Sessions runs on Linux, natively or in the [Linux runner](project-workflow.md#apply-in-the-linux-runner). The concrete Codex adapter implements `run.Sessions`: one fresh private home and Manager container per Manager per Run, with an ordinary persistent native conversation until closure. The Claude adapter implements the same port with one container and conversation per Manager. Both consume the actual Linux volume/subpath supplied by [LinuxGit](../../internal/adapters/gitrepo/README.md#run-git-workspace-service), without a host bind or a host credential home. One held Git service and private volume serve each Run; each Manager has its own repository.

The pure [harness](../../internal/harness/README.md) supplies checked Project context, complete hierarchy, exact Manager, mandate and Findings. The native adapter supplies its actual rendering and loader controls. Nine Run tools (`delegate`, `await`, `review`, `ask`, `reply`, `judge`, `check`, `read_notes`, `append_note`) have one implementation each in Run; the adapter only maps them. `judge` records the decision separately from final prose. The [session helper](../../internal/agentd/) bridges native provider tools; the [container runtime](../../internal/adapters/container/README.md) owns concrete resource lifetime.

[Container checks](../../internal/adapters/checks/README.md) use separate disposable input/home copies with verified bytes, modes and required directory inventories. Run retains independently reopenable candidates and terminal records after resource closure. [Publication](../../internal/adapters/gitrepo/README.md#publication-boundary) reads them in a separate host repository after every Run has closed.

## Account login

The provider runs with its CLI's normal account login only; there is no API-key route. The run configuration names a dedicated copy of that login as a credential reference outside the repository ([credential references](project-workflow.md#dedicated-credential-references)): `dedicated-runtime-login` for Codex, `subscription-credentials` for Claude Code. The Codex adapter admits only a login with `auth_mode: chatgpt`, account ID, ID token and access token, and refuses mixed API-key authority, desktop mode and malformed logins ([login limits](../../internal/adapters/codexappserver/README.md#login-and-evidence-limits)). The Claude adapter admits only a Claude Code `.credentials.json` with a `claudeAiOauth` account login ([login](../../internal/adapters/claudecode/README.md#login)).

Natively, the host's resolver reads the selected login file. In the [Linux runner](project-workflow.md#apply-in-the-linux-runner), the launcher checks each login the same way and mounts each file on its own, read-only, never its directory; the login stays outside the runner's clone, so it cannot enter a candidate, and the runner never prints it. In both forms a Manager receives access-only material through one bounded private setup message before its provider starts, without refresh authority, and the source login is never rewritten. Login expiry, refresh and fresh-login behavior need their own evidence.

## Claude Code

The [Claude adapter](../../internal/adapters/claudecode/README.md) is an addition beyond V1's Codex scope: D18 keeps V1 acceptance on Codex, and the owner scoped Claude as its own slice K1 ([owner decisions](../concepts/sources/owner-night-decisions-20261027.md)). Increment 1 implements it in source with fixture tests. The [K1b record](../validation/v1-k1b-claude-qualification-20261028.md) holds its first native evidence: one Library Apply inside the [Linux runner](../design/linux-runner.md) passed with the launch-control fixes of [PR #807](https://github.com/Glacius-Labs/Markitect/pull/807), and failed at the first Manager without them.

Select it with a `claudeCode` section in place of `codexAppServer`:

```yaml
agentProfiles:
  claude-agent:
    claudeCode:
      profileVersion: 1
      engineExecutable: /usr/bin/docker
      imageID: sha256:...
      agentdVersion: ...
      instructions: []
      credentialRef: claude-account
      maxWireBytes: 4194304
      session: {version: 1, model: sonnet, effort: high, baseInstructions: '', developerInstructions: '', store: markitect-claude, cpus: 1, memoryBytes: 2147483648, pids: 192, tmpBytes: 268435456, network: ''}
```

The run configuration maps `claude-account` to provider `claude-code` with mode `subscription-credentials`: a Claude Code `.credentials.json` account login, mounted read-only in the runner. It is the only Claude mode; a long-lived token mode needs owner acceptance first. Apply refuses an environment that sets `ANTHROPIC_API_KEY` or another API-key, provider or endpoint variable. The [base image](../../packaging/runtime/README.md) pins Claude Code `2.1.308` next to Codex.

Each Manager gets one Claude Code conversation in its own container: every response is a fresh `claude --print` process with one stream-json message, the first creating the conversation and later ones resuming it. Before each start, agentd checks that the Manager has not changed its private instruction files or added user-scope Claude Code settings, and each launch must report the expected login, version, permission mode and tools. The adapter's README lists the launch controls, the login and refresh limits, the transient-error rules it shares with the Codex adapter and the CLI assumptions, which the K1b record answers.

<a id="bring-your-own-executor"></a>

## Evidence and alternate runtimes

Controlled Sessions and real local Git establish bounded source behavior. The [V1 qualification record](../validation/v1-qualification-20261024.md#current-qualification-evidence) owns the native evidence: what the Codex account-login path with the recorded runtime image has shown, on which source, and which proofs are still missing. The [runner qualification](../validation/v1-l1-runner-qualification-20261027.md) adds one Library Apply from Windows through Docker Desktop with the login mounted read-only and unchanged afterwards. Source tests and metadata do not establish account refresh or expiry, other providers, hosted CI, release or owner acceptance.

An alternate runtime needs a real consumer adapter and bounded capability evidence. No provider or container run follows from reading this guide.
