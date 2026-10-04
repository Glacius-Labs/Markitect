# Azure DevOps captured-metadata Module

This independent source-only experimental consumer compares Core semantic IR with explicitly mapped Azure DevOps Git repository metadata. Module-owned configuration records organization/project/repository identity and exact capture paths. `Request` carries decoded config, supplied bytes/errors and typed prior observation/plan; `CapturePaths` declares paths for Host acquisition.

The Module performs no filesystem/network IO. Host owns strict outer protocol parsing and bounded regular-file/path acquisition. Observe/Plan/Verify emit deterministic findings bound to model/mapping/target/captured evidence. Supplied identity/branch records do not authenticate an owner or prove live provider state, permissions, runtime behavior or complete governance. Apply is unsupported.

Provider DTOs/configuration stay in this subtree. Unit fixtures test malformed/mismatched captures and narrow verification; Host tests invoke the real thin executable with parsed fixed models and refusal cases. See [shared contracts](../../../docs/development/shared-contracts.md), [first-wave evidence](../../../docs/validation/parallel-development-wave-1.md) and [Module laws](../../../docs/development/modules.md).
