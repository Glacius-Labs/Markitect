# .NET evidence Module

`Run(Request)` consumes Core semantic IR, decoded project-file mappings, exact captured bytes/errors, and this Module's typed prior observation/plan. `CapturePaths(Config)` returns only configured project-file paths for Host acquisition. The Module performs no filesystem access or process execution.

Host owns strict adapter-request decoding, exact-path regular-file/size controls and stdout/exit behavior; the thin command delegates to that runtime. Module-owned configuration and typed evidence stay here, outside Core. Observe/Plan/Verify report mapped MSBuild ProjectReference declarations and bounded literal XML checks. Apply is unsupported.

This proves correspondence to captured declarations, not runtime dependency architecture, business semantics or absence of alternative dependency mechanisms. Unmapped files are not implicitly scanned. Unit fixtures live in `module_test.go`; Host runtime and executable examples exercise the public command protocol.

See [canonical engineering](../../../examples/canonical-engineering/README.md), [architecture](../../../docs/architecture.md) and [Module laws](../../../docs/development/modules.md). Public protocol/Domain meaning does not change with the internal package move.
