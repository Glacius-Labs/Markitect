# Provider adapters

The CLI exposes structural `check`, `context` and guarded `docs`, with explicitly requested static Runtime metadata. Public `apply` reports unavailable until S3. The standalone [Run](../internal/run/README.md) owns the complete Manager hierarchy and direct Session, Git, check and record ports. It provides no product publication path.

<a id="role-profiles"></a>

## Explicit profiles and static metadata

Build `cmd/markitect` and use its CLI from the coding environment. `check` and `context` are structural by default. Request static metadata with `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH`. Runtime profile selection is explicit; the product does not discover accounts or infer provider configuration.

The closed `markitect.runtime/v1` document contains native Agent and Check profiles. Outer composition selects native profiles. Project's `markitect.execution/v2` declarations contain Package instructions and scoped checks, without provider roles, teams, strategies or invocation budgets. Invocation limits and normal account-login references belong to the explicit outer run configuration.

Codex profile version 2 supports metadata only; version 3 adds the closed native session settings. The [Codex adapter](../internal/adapters/codexappserver/README.md) documents exact fields and preparation evidence. Claude retains static profile preparation and protocol codecs, but its current `run.Sessions.Open` returns unavailable before credentials or containers are acquired. Static metadata does not qualify Claude execution. [Check profiles](../internal/adapters/checks/README.md) bind pinned images, literal commands, environment policy and finite limits.

Codex metadata preparation may inspect a selected image and copy bounded whole `/opt/markitect/runtime.env` bytes from a created, never started container. It does not launch an Agent Session or provider. Check metadata preparation is process-free. Neither establishes effective native instructions, login, runtime support or semantic completion. Structural output never reports metadata as implicitly passed.

Fixed previews refuse unbound present-day external profile or dependency inputs, even for HEAD. Working previews retain observed bytes. Neither preview supplies an artifact publication guard.

<a id="container-runtime"></a>

## Native Sessions and Git workspaces

The concrete Codex adapter implements `run.Sessions`: one fresh private home and Manager container per Manager per Run, with an ordinary persistent native conversation until closure. It consumes the actual Linux volume/subpath supplied by [LinuxGit](../internal/adapters/gitrepo/WORKSPACES.md), without a writable Windows bind or a host credential home. One held Git service and private volume serve each Run; each Manager has its own repository. There is no legacy Pass/Turn execution fallback.

The pure [harness](../internal/harness/README.md) supplies checked Project context, complete hierarchy, exact Manager, mandate and Findings. The native adapter supplies its actual rendering and loader controls. Seven direct Run tools provide delegation, awaiting input, exact child review, questions/replies and durable Notes. A provider-local judgment tool records the decision separately from final prose. The [session helper](../internal/agentd/) bridges native provider tools; the [container runtime](../internal/adapters/container/README.md) owns concrete resource lifetime.

[Container checks](../internal/adapters/checks/README.md) use separate disposable input/home copies with verified bytes, modes and required directory inventories. Standalone Run retains independently reopenable candidates and terminal records after resource closure. [Publication](../internal/adapters/gitrepo/PUBLISHER.md) remains the S3 boundary.

<a id="bring-your-own-executor"></a>

## Evidence and alternate runtimes

Controlled Sessions and real local Git establish bounded source behavior. The [accepted architecture](design/v1-execution-architecture.md) records the narrow Linux copy boundary proof that passed on frozen `623eb190`. It does not qualify later Git conflict corrections or the current composed Run. Effective harness loading, normal-login fan-out, native full-hierarchy completion and background-writer drain still require their own source-bound proof. Earlier [handshake](validation/runtime-handshake-20261017.md) and [capability probes](validation/runtime-capabilities-20261017.md) remain historical observations; they do not qualify this Run or a full Apply.

An alternate runtime needs a real consumer adapter and bounded capability evidence. No provider or container run follows from reading this guide. Source tests and metadata do not establish measured provider cost, performance, product Apply, hosted CI, release or owner acceptance.
