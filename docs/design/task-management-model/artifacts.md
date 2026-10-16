# Planned Artifacts

All entries describe the future task application's repository, not files already implemented in Markitect. Paths are relative to that future repository root. These logical scopes assign one owner without forcing one file per record. Exact files and language-specific manifests must be declared during realization before claiming full file accounting.

<a id="art-cli"></a>
## ART-CLI — Command-line implementation

Type: Artifact. Planned path scope: `src/cli/`.

Source for CLI parsing, selected-file resolution, formatting and outcome translation. Owned by COMP-CLI. Include executable startup wiring without putting storage or task-action policy into the parser.

<a id="art-management"></a>
## ART-MANAGEMENT — Task implementation

Type: Artifact. Planned path scope: `src/tasks/`.

Source for domain meaning and the offered task actions. Owned by COMP-MANAGEMENT. Its responsibilities are stable even if the implementation chooses functions rather than classes.

<a id="art-storage"></a>
## ART-STORAGE — File storage implementation

Type: Artifact. Planned path scope: `src/storage/`.

Source for file decoding, encoding and safe persistence. Owned by COMP-STORAGE. Realize the selected representation without moving task-action policy into file operations.

<a id="art-tests"></a>
## ART-TESTS — Verification implementation

Type: Artifact. Planned path scope: `tests/`.

Automated test implementations for the machine-checkable Check records. Owned by COMP-APP for integrated coverage; a test may examine a child component. Keep independent expected results and isolated temporary task files. Human review Check procedures need no fabricated automated test.

<a id="art-user-guide"></a>
## ART-USER-GUIDE — User guide

Type: Artifact. Planned path: `README.md`.

Explain the audience, scope, supported runtime/tool prerequisites selected during realization, setup/build/start commands, all offered commands with examples, task-file location and override, exit outcomes and recovery steps for storage errors. Describe version 1's concurrency/crash limits. Link the canonical specification for detailed contracts rather than copying it wholesale. Owned by COMP-CLI.

<a id="art-agent-guide"></a>
## ART-AGENT-GUIDE — Agent entrypoint

Type: Artifact. Planned path: `AGENTS.md`.

Route a development agent to the canonical specification, responsibility boundaries, developer guide and test entrypoint. Do not independently restate domain behavior or shared development policy. Owned by COMP-APP.

<a id="art-developer-guide"></a>
## ART-DEVELOPER-GUIDE — Developer guide

Type: Artifact. Planned path: `CONTRIBUTING.md`.

Describe how to propose an intent change, realize a repair, locate affected contracts and outputs, run checks, review a candidate and record evidence. Give actual build/run/test commands for the chosen implementation, instructions for storage-failure tests and the human architecture, documentation and user-observation Check procedures. Route to REQ-CHANGE and REQ-DOCS as authoritative rules. Owned by COMP-APP.

<a id="art-tooling"></a>
## ART-TOOLING — Build and test configuration

Type: Artifact. Planned path scope: `scripts/`; additional toolchain-native manifest paths are selected with the implementation language.

Provide reproducible entrypoints or tool-native commands for build, running the CLI and the automated Check suite. Record versions and prerequisites in the appropriate guide. The concrete configuration and output paths must be identified once the implementation is selected. Owned by COMP-APP. No hosted CI service or package registry is mandated by this example.

<a id="art-ignore"></a>
## ART-IGNORE — Runtime/output exclusions

Type: Artifact. Planned path: `.gitignore`.

Exclude the default `tasks.data`, selected implementation build outputs and test temporary data from source control. A user-selected file elsewhere remains user-controlled; do not claim this ignore file prevents all possible accidental commits. Owned by COMP-APP.

## Canonical specification placement

For this design exercise, these Markdown documents own intent. If the example becomes a separate application repository, move the canonical package there or establish an explicit pinned source route before implementation; do not create a second independently maintained specification. That placement action does not imply an implemented Markitect import mechanism.

The task data file is runtime state, not a source Artifact or a second source of application intent.
