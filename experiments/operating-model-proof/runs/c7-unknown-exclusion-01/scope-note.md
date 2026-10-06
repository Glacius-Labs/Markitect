# C7 unknown and exact exclusion preparation

This packet prepares a fresh external clone at the public aligned C4 evidence commit, with two deterministic additions under the configured `src/Orders/` target prefix. `UnknownBehavior.cs` is unowned and is expected to remain unknown. `legacy-note.md` is the one exact target exclusion, with a technical-test-only reason. Both files are recorded by path, Git mode, object ID, size, and SHA-256; their bytes are not included in this packet.

The runtime is a structured copy of the verified C5-03 input. It keeps `auditAll=false`, uses the exact `src/Orders/legacy-note.md` exclusion, and points to a new isolated record store and empty private-log directory. The seven prior event files and `store.json` were copied byte-for-byte. Executor and verifier `args[0]` each equal their first `runtimeFiles` path; the actual wrapper bytes match the declared SHA-256 and mode.

No Propose command or provider has been invoked. The direct command template is frozen for review, and the C7 unknown/exclusion result remains unobserved. This packet makes no global coverage, semantic verification, or human-acceptance claim. The fixture and runtime stage are external disposable inputs; the original public C4/C5 source fixture and earlier trials were not changed.
