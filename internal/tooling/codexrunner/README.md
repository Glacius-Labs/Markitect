# Codex execution adapter

This standalone standard-library Python adapter accepts the closed agentexec.Invocation JSON value on stdin and emits exactly one response JSON object on stdout. It rejects duplicate or unknown request fields, verifies explicit artifact byte digests, and uses separate Executor, Verifier, and infer-only instructions. It does not embed Markitect ontology or write candidate files.

The caller supplies --model, --codex-executable, --codex-script when the executable is a runtime such as Node, and --codex-version. Executable and script paths must be absolute. It invokes codex exec with --ignore-user-config, --sandbox read-only, --ephemeral, --json, --skip-git-repo-check, --disable plugins, an output schema, an output-last-message file, the empty temporary --cd directory, and stdin -. modelOptions arrive from the Host as explicit JSON and become literal --config key=value argv pairs; no shell parses them.

The Host supplies MARKITECT_AGENT_CONFIG_JSON and MARKITECT_AGENT_PRIVATE_LOG. Codex JSONL stdout events and stderr are retained in that private log, bounded to 16 MiB; the adapter emits only the final protocol response on stdout and a generic error on stderr. It parses tool-call events and provider-reported token usage. Missing usage stays missing.

Install Python 3.10 or later as a caller-managed dependency. Run the boundary tests from this directory with python -m unittest. The wrapper itself can be inspected without invoking Codex; the real CLI is an external dependency and is exercised separately by a controlled integration experiment.
