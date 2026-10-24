# Provider adapters

The CLI exposes structural `check`, `context` and guarded `docs`, with explicitly requested static Runtime metadata. S3 source composes `apply` over complete fresh Runs and stored-candidate publication; full native Apply qualification and owner acceptance remain separate. The standalone [Run](../internal/run/README.md) owns the complete Manager hierarchy and direct Session, Git, check and record ports. It provides no product publication path.

<a id="role-profiles"></a>

## Explicit profiles and static metadata

Build `cmd/markitect` and use its CLI from the coding environment. `check` and `context` are structural by default. Request static metadata with `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH`. Runtime profile selection is explicit; the product does not discover accounts or infer provider configuration.

The closed `markitect.runtime/v1` document contains native Agent and Check profiles. Outer composition selects native profiles. Project's `markitect.execution/v2` declarations contain Package instructions and scoped checks, without provider roles, teams, strategies or invocation budgets. Invocation limits and normal account-login references belong to the explicit outer run configuration.

Codex profile version 2 supports metadata only; version 3 adds the closed native session settings. The [Codex adapter](../internal/adapters/codexappserver/README.md) documents exact fields and preparation evidence. Codex is the only Agent adapter in V1 ([D18](concepts/v1-concept.md#23-providerumfang-d18)); the Runtime profile refuses any other Agent section. [Check profiles](../internal/adapters/checks/README.md) bind pinned images, literal commands, environment policy and finite limits.

Codex metadata preparation may inspect a selected image and copy bounded whole `/opt/markitect/runtime.env` bytes from a created, never started container. It does not launch an Agent Session or provider. Check metadata preparation is process-free. Neither establishes effective native instructions, login, runtime support or semantic completion. Structural output never reports metadata as implicitly passed.

Fixed previews refuse unbound present-day external profile or dependency inputs, even for HEAD. Working previews retain observed bytes. Neither preview supplies an artifact publication guard.

<a id="container-runtime"></a>

## Native Sessions and Git workspaces

The concrete Codex adapter implements `run.Sessions`: one fresh private home and Manager container per Manager per Run, with an ordinary persistent native conversation until closure. It consumes the actual Linux volume/subpath supplied by [LinuxGit](../internal/adapters/gitrepo/WORKSPACES.md), without a writable Windows bind or a host credential home. One held Git service and private volume serve each Run; each Manager has its own repository. There is no legacy Pass/Turn execution fallback.

The pure [harness](../internal/harness/README.md) supplies checked Project context, complete hierarchy, exact Manager, mandate and Findings. The native adapter supplies its actual rendering and loader controls. Seven direct Run tools provide delegation, awaiting input, exact child review, questions/replies and durable Notes. A provider-local judgment tool records the decision separately from final prose. The [session helper](../internal/agentd/) bridges native provider tools; the [container runtime](../internal/adapters/container/README.md) owns concrete resource lifetime.

[Container checks](../internal/adapters/checks/README.md) use separate disposable input/home copies with verified bytes, modes and required directory inventories. Standalone Run retains independently reopenable candidates and terminal records after resource closure. [Publication](../internal/adapters/gitrepo/PUBLISHER.md) remains the S3 boundary.

<a id="bring-your-own-executor"></a>

## Evidence and alternate runtimes

Controlled Sessions and real local Git establish bounded source behavior. The [V1 qualification record](validation/v1-qualification-20261024.md#early-native-evidence) records native Codex harness loading, normal-login fan-out, crossed questions, child repair and tracked-writer drain on its specific early fixture. Subsequent native Library M1/M2 Apply on `1ef8b910` completed the full six-Manager hierarchy in Assessment, Implement and Verify, retained closed results/candidates and published from stored acceptance. Independent outside Oracles passed for both actual candidates. Later bounded proofs established strict no-change and a genuine native Verify rejection followed by fresh correction and approval; the latter has an independently diagnosed outer-controller classification failure, preserved separately from its passing component result. The current source examples also completed a native public Apply with normal login and strict no-change on `6a46bd8a`, using declared placeholders and one explicit project-specific Check-profile name mapping. The [qualification record](validation/v1-qualification-20261024.md#current-qualification-evidence) binds sources, image, attempts, limitations and outstanding proofs.

This evidence qualifies the observed Codex 0.162.0 account-login path with `gpt-6-sol`/high and the exact recorded runtime image. It does not establish account refresh/expiry, universal instruction reliability, detached-writer containment, any other provider or final owner acceptance. Earlier [handshake](validation/runtime-handshake-20261017.md) and [capability probes](validation/runtime-capabilities-20261017.md) remain historical observations with their original scopes.

An alternate runtime needs a real consumer adapter and bounded capability evidence. No provider or container run follows from reading this guide. Source tests and metadata do not establish measured provider cost, performance, product Apply, hosted CI, release or owner acceptance.
