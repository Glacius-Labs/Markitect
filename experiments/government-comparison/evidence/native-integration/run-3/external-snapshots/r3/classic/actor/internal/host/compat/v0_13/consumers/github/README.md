# GitHub captured-metadata Module

This independent source-only experimental consumer compares Core semantic IR with explicitly mapped, supplied repository-metadata captures. `Request` owns decoded configuration, captured bytes/errors and typed prior observation/plan; `CapturePaths` declares exact paths for Host to acquire. The Module performs no filesystem/network IO and imports no sibling capability.

Host owns strict outer adapter-protocol decoding and bounded exact-file acquisition. Observe/Plan/Verify retain source/model/mapping/capture identity and deterministic findings. The captured API/version, repository identity and default-branch evidence describe supplied records; they do not establish live GitHub state, authentication, branch protection completeness or semantic correctness. Apply is unsupported.

Configuration and provider DTOs remain module-owned rather than extending Core or sharing mutable provider state. Unit fixtures live here; Host tests build/invoke the thin command against parsed fixed-model evidence. See [shared contracts](../../../docs/development/shared-contracts.md), [first-wave evidence](../../../docs/validation/parallel-development-wave-1.md) and [Module laws](../../../docs/development/modules.md).
