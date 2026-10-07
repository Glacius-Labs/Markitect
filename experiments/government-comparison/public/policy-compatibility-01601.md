# Pinned CLI 0.160.1: bounded local compatibility passage

Status: **six metadata calls completed; no effective-policy or file-read success**.
This follows the [earlier diagnosis](policy-block-diagnosis.md), which remains a
dated hypothesis report. No Actor, model, provider probe, rejected read-command
retry, alternate shell/path, permission change, installation, or global-config
change was performed. `approval_policy="never"` was not weakened.

## Exact local observations

All six calls used the unchanged binary
`C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe`,
SHA-256 `3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916`,
previously pinned as `codex-cli 0.160.1`. Arguments were explicit lists reviewed
before dispatch, stdin was closed, and each process had a 30-second timeout.
No call was an empty invocation, `exec`, a TUI, daemon or server start.

| Call | Arguments after binary | Native exit | Purpose/result |
|---|---|---|---|
| 1 | `--help` | 0 | Global CLI help; no `--permission-profile` option listed. `--profile` layers a named config file; it is not documented here as a permission profile. |
| 2 | `debug --help` | 0 | Lists models, app-server and prompt-input helpers; no isolated effective-policy dump advertised. Those helpers were not executed. |
| 3 | `app-server --help` | 0 | Advertises static protocol schema export separately from server/daemon operations. |
| 4 | `doctor --help` | 0 | Doctor covers config, auth and runtime; no config-only/no-auth switch listed. Doctor inspection was not executed. |
| 5 | `app-server generate-json-schema --help` | 0 | Confirms local schema export with required `--out` and optional experimental fields. |
| 6 | `app-server generate-json-schema --experimental --out <bound study evidence directory>` | 0 | Static export: 440 JSON files, archived byte-for-byte with per-file hashes. No app-server RPC was sent. |

Raw call records, exact argv, timings, archive and its manifest are under
[`evidence/policy-compatibility/run-1`](../evidence/policy-compatibility/run-1/01-global-help.json).
The first collector saved its complete native receipt before terminal printing
failed on a Unicode character; native exit was 0. The saved receipt was read
without repeating the call. A later archive-inspection script requested an
absent optional schema filename; this was a local file lookup, not a CLI call.
Neither postprocessing error caused a retry or inference.

Existing `exec --help` and the fifth-run argv were read first. They establish
explicit `--ignore-user-config`, `--ignore-rules`, legacy `--sandbox read-only`,
ordinary shell/unified tools enabled and `approval_policy="never"`. The earlier
selective config inventory found a user `windows.sandbox="elevated"` setting,
but ignored user configuration does not establish the Actor's effective mode.
No additional configuration, rules or credentials were opened in this passage.

## What the exact binary does expose

The exported schema is version-bound evidence of an interface, beyond current
web documentation. [Schema observations](../evidence/policy-compatibility/run-1/schema-observations.json)
bind the original schema members by SHA-256:

- `v2/ThreadStartParams.json` has a nullable string `permissions`: a named profile
  id, explicitly incompatible with `sandbox`. `approvalPolicy` is a separate field.
- `v2/CommandExecParams.json` has nullable string `permissionProfile`, explicitly
  incompatible with `sandboxPolicy`; omission defaults to configured permissions.
- `v2/ConfigRequirementsReadResponse.json` exposes `allowedPermissionProfiles`
  and `defaultPermissions`, plus sandbox/approval/Windows implementation constraints.
- `v2/ConfigReadResponse.json` has `config`, `layers` and `origins`; its `Config`
  explicitly lists legacy sandbox/approval fields and permits additional properties.
  Absence of a declared `default_permissions` field therefore does not disprove
  support. `ConfigReadParams` describes resolving project layers for a given cwd.
- Legacy `SandboxPolicy` includes a `readOnly` variant with `networkAccess=false`
  by default. That shape contains no explicit two-file read allowlist. Its shape
  alone establishes neither all inherited reads nor OS enforcement.

Thus **named-profile concepts are present in this binary's app-server protocol**.
No exported schema resolves actual configuration, proves a CLI selector or TOML
key is applied, or identifies the effective Windows/managed policy in the failed
`exec` run. No RPC, thread, command-execution request, or configuration parse was
executed. App-server profile fields are not evidence that the frozen `exec` route
uses them. In particular, adding a config value to a help/export invocation would
not establish its runtime effect, so no such pseudo-validation was performed.

## Concrete blocker and required decision

The exact policy rejection remains known; the exact rejecting rule/layer is not.
There is no verified configuration correction from this passage. The missing
evidence is the **resolved policy of this exact exec route**: recognized profile
selection and precedence, read-only shell behavior with unchanged `never`,
complete inherited file/network rights, the two allowed sentinel paths including
the external released file, selected Windows implementation and applicable
managed/system constraints. Both rejected files must be explained; external scope
alone cannot account for the Actor-owned file's denial.

The bounded metadata allowance is exhausted (6/6). Do not start another discovery
loop or Actor. Overseer must obtain an authoritative, sanitized effective-policy
and rejection-layer explanation from the environment/runner owner for this exact
binary and frozen argv, or explicitly decide on a separately bounded owner-led
diagnosis. Such work must avoid authentication/provider access and preserve the
read-only/`never` boundary; no permission or global change is preauthorized here.
If the owner cannot establish support and acceptable complete rights, the context
smoke remains blocked on environment/authority rather than being repaired by an
unverified TOML profile or broader sandbox permission. A later supported policy
would still need a separately authorized access proof before S1 can pass.

The five historical Actor starts, known token subtotal 53,331 and unknown complete
historical total remain unchanged. No sixth Actor or study cell is authorized.
