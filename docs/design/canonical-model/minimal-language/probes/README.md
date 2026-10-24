# Archived design probes

These sources were temporary check programs during the design conversations. They are archived here byte for byte so that another session can trace the reports. They are not Markitect product components and not a complete implementation of the current [language contract](../language-contract.md).

The Python prototypes have the same content digests as the sources bound in bootstrap/probe.json, bootstrap/compact-probe.json and identities/probe.json. The two Go programs and inputs.json archive the older vertical slice against the existing Core after adaptation from definition to purpose. The archives are not adapted after the fact to newer design decisions.

From the repository root with Python and PyYAML:

```text
python -B docs/design/canonical-model/minimal-language/probes/replay.py
```

With the Go version that matches CONTRIBUTING, additionally:

```text
python -B docs/design/canonical-model/minimal-language/probes/replay.py --with-go
```

The replay checks input/prototype digests and runs the probes in a new temporary working directory. Newly generated Python reports are compared with the historical reports except for sourceRevision. For the older Go probes, their generated report part is compared with the stored report; its later added provenance fields are preserved. The originals are not changed. Unchanged results at the current head are a renewed bounded check, not a retroactive change of the historical evidence.

The Python archives expect their bootstrap import of that time under TEMP. The replay provides exactly this environment only for its child processes. A Go overlay creates the virtual entry point under src/cmd/markitect so that the existing internal imports stay valid; no source file is created in the repository in the process. The Go sources are therefore stored as .go.txt and do not accidentally become a new Go package.

The compiler prototype understands no complete body link grammar and no semantic index. The M3/M4 reviews in the language contract are still assumed results of a design case. No Connector, no broker and no agent provider is started.
