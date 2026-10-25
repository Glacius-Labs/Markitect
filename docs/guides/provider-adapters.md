# Provider adapters

The CLI exposes structural `check`, `context` and guarded `docs`, with explicitly requested static Runtime metadata, and `apply`, which runs complete fresh Assessment, Implement and Verify Runs and publishes the stored Verify approval. The [Run](../../internal/run/README.md) owns the complete Manager hierarchy and its Session, Git, check and record ports; it has no publication path. Provider qualification and owner acceptance are separate claims.

<a id="role-profiles"></a>

## Explicit profiles and static metadata

Build `cmd/markitect` and use its CLI from the coding environment. `check` and `context` are structural by default. Request static metadata with `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH`. Runtime profile selection is explicit; the product does not discover accounts or infer provider configuration.

The closed `markitect.runtime/v1` document contains native Agent and Check profiles. Outer composition selects native profiles. Project's `markitect.execution/v2` declarations contain Package instructions and scoped checks, without provider roles, teams, strategies or invocation budgets. Invocation limits and normal account-login references belong to the explicit outer run configuration.

Codex profile version 2 supports metadata only; version 3 adds the closed native session settings. The [Codex adapter](../../internal/adapters/codexappserver/README.md) documents exact fields and preparation evidence. Codex is the only Agent adapter in V1 ([D18](../concepts/v1-concept.md#23-provider-scope-d18)); the Runtime profile refuses any other Agent section. [Check profiles](../../internal/adapters/checks/README.md) bind pinned images, literal commands, environment policy and finite limits.

Codex metadata preparation may inspect a selected image and copy bounded whole `/opt/markitect/runtime.env` bytes from a created, never started container. It does not launch an Agent Session or provider. Check metadata preparation is process-free. Neither establishes effective native instructions, login, runtime support or semantic completion. Structural output never reports metadata as implicitly passed.

Fixed previews refuse unbound present-day external profile or dependency inputs, even for HEAD. Working previews retain observed bytes. Neither preview supplies an artifact publication guard.

<a id="container-runtime"></a>

## Native Sessions and Git workspaces

The concrete Codex adapter implements `run.Sessions`: one fresh private home and Manager container per Manager per Run, with an ordinary persistent native conversation until closure. It consumes the actual Linux volume/subpath supplied by [LinuxGit](../../internal/adapters/gitrepo/WORKSPACES.md), without a writable Windows bind or a host credential home. One held Git service and private volume serve each Run; each Manager has its own repository.

The pure [harness](../../internal/harness/README.md) supplies checked Project context, complete hierarchy, exact Manager, mandate and Findings. The native adapter supplies its actual rendering and loader controls. Nine Run tools (`delegate`, `await`, `review`, `ask`, `reply`, `judge`, `check`, `read_notes`, `append_note`) have one implementation each in Run; the adapter only maps them. `judge` records the decision separately from final prose. The [session helper](../../internal/agentd/) bridges native provider tools; the [container runtime](../../internal/adapters/container/README.md) owns concrete resource lifetime.

[Container checks](../../internal/adapters/checks/README.md) use separate disposable input/home copies with verified bytes, modes and required directory inventories. Run retains independently reopenable candidates and terminal records after resource closure. [Publication](../../internal/adapters/gitrepo/PUBLISHER.md) reads them in a separate host repository after every Run has closed.

<a id="bring-your-own-executor"></a>

## Evidence and alternate runtimes

Controlled Sessions and real local Git establish bounded source behavior. The [V1 qualification record](../validation/v1-qualification-20261024.md#current-qualification-evidence) owns the native evidence: what the Codex account-login path with the recorded runtime image has shown, on which source, and which proofs are still missing. Source tests and metadata do not establish account refresh or expiry, other providers, hosted CI, release or owner acceptance.

An alternate runtime needs a real consumer adapter and bounded capability evidence. No provider or container run follows from reading this guide.
