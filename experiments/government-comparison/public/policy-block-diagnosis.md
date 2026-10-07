# Context smoke policy-block diagnosis

## Finding

The fifth context run did not prove file access. It completed one model turn and
the outer CLI process exited `0`, but the answer returned both sentinels as
`null`. The JSONL stdout contained zero structured tool events. Raw stderr
records two attempted `exec_command` calls for the exact allowed files; both
were rejected at `CreateProcess` with `blocked by policy`. The PowerShell
commands therefore did not run and no successful file read is evidenced.

The run has a concrete policy configuration to investigate, but the
available evidence does not establish a pinned-version cause. The frozen argv
enabled `features.shell_tool=true` and `features.unified_exec=true`, selected
`--sandbox read-only`, and set `approval_policy="never"`. Current OpenAI Codex
guidance describes the legacy read-only sandbox as permitting inspection while
requiring approval to run shell commands, and `never` as not pausing for
approval. That is a plausible explanation for why the requested PowerShell
commands were rejected. A separate permissions-profile guide describes the
built-in `:read-only` profile as permitting read-only local command execution.
These descriptions concern different configuration systems and do not prove
which interpretation Codex CLI 0.160.1 applied to this run. The second file is
outside the Actor workspace, which is another boundary to resolve.

The captured error has no Windows error code, sandbox log, or OS denial detail.
It does not identify whether the rejection came from the CLI's read-only
approval path, filesystem scope, Windows sandbox, or an additional host policy.
The unsupported PowerShell shell snapshot warning is not evidence of the
cause. Process exit success is not task success, and stderr attempts are not
structured tool events.

## Smallest documented correction to review

Current OpenAI documentation describes a named permissions-profile mechanism
for read-only local commands and explicit filesystem grants. It also says
permission profiles do not compose with the older `sandbox_mode` selector.
This is a possible design direction, not an applied correction. Extending
`:read-only` and adding two path entries does not establish exclusive access
to only those files: inherited read permissions may be broader. No proposed
profile is an exact effective boundary until its resolved rules are inspected.

The captured `codex exec --help` belongs to the pinned 0.160.1 binary and lists
`--sandbox` and `--config`, but not `--permission-profile`. That help output is
not enough to rule out a global option, or the `default_permissions` config
path, because the option could be documented by global help and configuration
overrides are supported. Compatibility with this binary is therefore unknown,
not disproven.

The minimal next decision for the Overseer is whether to authorize a
no-inference compatibility and effective-policy inspection of the pinned
0.160.1 setup. It should establish how the exact CLI resolves read-only shell
execution under `approval_policy="never"`, whether a named profile can be
selected without legacy sandbox settings, and the complete inherited and
explicit filesystem/network rules. The candidate must preserve read-only
access and the unchanged approval policy, and explicitly account for the
external context file. If that inspection cannot show an acceptable bounded
policy, report the blocker and request a scope decision; do not infer support
from current documentation or weaken permissions.

The documented Windows `/sandbox-add-read-dir` command is another mechanism
for granting sandbox read access to a directory. The existing record does not
show whether it is usable by this noninteractive pinned invocation, how that
grant interacts with its read-only/approval settings, or what other reads it
would expose. It remains an unverified alternative, not the proposed fix.

## Configuration checked

The frozen invocation explicitly passed `--ignore-user-config` and
`--ignore-rules`, followed by per-run inline configuration. A selective local
read found the user's `config.toml` contains `windows.sandbox = "elevated"`;
that setting is not evidence of the Actor's effective sandbox because the
invocation asked the CLI to ignore user configuration. A user rules file is
present, but was not opened because the invocation also asked the CLI to ignore
rules. The Actor workspace had no `.codex/config.toml`, `.codex/requirements.toml`,
`.codex/rules/default.rules`, or `AGENTS.md` at the checked paths. No user
`requirements.toml` was present. No source here establishes whether a managed
or system-level policy was present or affected execution. No credentials or
authentication files were read.

## Bound evidence

The local evidence is pinned by delivery source `098a7931f9f350b007046e858b6b348bee08feec`,
execution source `56d2e578ce105a77e4f8be4e3364925efc9d813d`, and the unchanged
Codex CLI binary SHA-256 `3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916`
(version 0.160.1). The raw process receipt reports return code 0 and 26.037335
seconds; the Actor result reports both sentinels null and usage 22,478 input +
343 output tokens. The request SHA-256 is
`39a62a4da246ab3e48ff4ee1bb21f0265753c0ecbaa98f3077567d862d484970`; the
stderr SHA-256 is
`1b17bf8f6d36c77fe081014ff363b45ea5ed3c88a4d84f6e47c83502b464934c`; stdout
SHA-256 is
`386a00e490fdc8d5f38fb0420cd1c25e000d867148ef035501bdc8611052d9f1`.

The two rejected targets were `actor-workspace/actor-own.txt` and
`released/context.txt`, whose expected content hashes remain bound in the
frozen Request and evidence manifest. No retry or further run is part of this
diagnosis.

## Official references

Retrieved 2026-10-07 for this diagnosis. These are current official pages; they
may describe behavior newer than the pinned 0.160.1 CLI.

- [Codex sandbox modes and approval policies](https://learn.chatgpt.com/docs/sandboxing), sections “Configure defaults” and “What the sandbox does”.
- [Codex permission profiles](https://learn.chatgpt.com/docs/permissions), sections “Define and select a profile”, “Extend a profile”, and “Configuration spec”.
- [Windows sandbox](https://learn.chatgpt.com/docs/windows/windows-sandbox), sections “Grant sandbox read access” and “Troubleshooting and FAQ”.
- [Codex CLI commands and flags](https://learn.chatgpt.com/docs/developer-commands), `codex exec` flags.
