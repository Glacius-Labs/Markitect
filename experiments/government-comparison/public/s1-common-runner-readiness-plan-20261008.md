# Common real runner: one bounded next-operation proposal

Date: 2026-10-08. Assignment `s1-common-runner-readiness-plan-20261008`.
**Documentation only; not a run grant.** Base delivery
`91bb8cf7b3ae242adc02e7f466bf17ca9120a1c1`. No profile is applied, checkout
provisioned, case prepared, admission/freeze performed, or experimental process
started. The [current S1 matrix](s1-adapter-mechanics.md) separates implemented
mechanics from real tools, provider observations and semantic acceptance.

## Recommendation and new information

Recommend **one separately granted, prospective app-server policy metadata
read**, using a strictly parsed built-in read-only permission default on the
new common route and a reviewed presence-aware sanitized result. Do not start a sixth real Actor yet. The new question
is: does pinned 0.160.1 accept the explicitly requested `:read-only` permission
default under `--strict-config`, what default and constraints does it report at
the designated public cwd, and is a legacy sandbox setting or managed requirement
reported that prevents treating that default as a prospective selection?

The old client already had allowlist branches for permission defaults/profiles
and these typed requirements. Their absence in its sanitized result cannot
distinguish missing, null or shape-filtered fields; it is not evidence that the
client never requested them. Raw responses were discarded. The new input delta
is strict parsing plus an explicit read-only default at a new public cwd, with
presence/null/invalid recorded separately. Repeating the old argv/filter, account/
model listing, help, feature listing, or either denied file command would add no
relevant evidence. This is not a retry of either denied command. It will **not** identify the earlier exec's effective configuration,
explain its policy denials, establish OS access, or clear S1.

The proposed common future transport is the same pinned app-server / built-in
OpenAI / existing ChatGPT-auth route for Conventional, Classic and Government,
with fresh contexts and the same ordinary tools, permissions and observation
policy. It is a new prospective route, not a claim that it reproduces the earlier
exec with ignore-user-config/ignore-rules. No arm gets an App-only tool surface,
extra persistent chat history, hidden instructions, or an easier permission path.
The actual role bridge is still absent and is not implemented here.

## Historical runner barrier

The [fifth Actor observation](../evidence/context-tools-live/run-1/observation.json)
and [tool-attempt record](../evidence/context-tools-live/run-1/tool-attempts.json)
show a completed answer and process/dispatcher exit 0, yet both sentinels null.
Stderr records exactly two `exec_command` requests targeting the released context
file and Actor-owned file; both were rejected at CreateProcess as blocked by
policy. No shell execution or successful read was observed. The stdout-only
counter was zero; it does not erase stderr tool attempts. No denied command is
replayed, no shell is switched, and no protection is removed to work around this.
The precise policy layer or platform constraint responsible remains unknown.

Historical real Actor starts: **5**; known input-plus-output subtotal **53,331**;
all-history tokens, provider requests and internal retries **unknown**. The fifth
reported 22,478 input + 343 output = 22,821 tokens, separately retaining cached
input and reasoning counters without adding them again. Six CLI metadata calls
and three app-server policy-read trees are consumed. Of those policy-read trees,
the first two stopped before usable config; the third received the two reads.
No quota is refilled and no unknown total becomes zero.

The [third policy-read result](../evidence/policy-read-disabled-status/run-1/sanitized-result.json)
reports approval `never`, null legacy sandbox mode, Windows sandbox `elevated`,
ordinary tool flags true, and selected other features false; its user layer
supplied the Windows setting. The sanitized requirements retained only `network=null`; the client already
supported the other policy fields but did not preserve their absence/null/invalid
classification.
Null is not proof of unrestricted access or absent policy. This invocation loaded
user/system layers and omitted exec-only ignore, sandbox and ephemeral controls.
It does not establish the historical exec's effective settings. The
[dated post-review](policy-disabled-status-postrun.md) remains unchanged.

## Exact pinned interfaces and precedence evidence

| Item | Binding / evidentiary limit |
|---|---|
| Native executable | `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe`; `codex-cli 0.160.1`; SHA-256 `3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916`. Hash locally before any later permitted launch; never resolve plain `codex` to the older shim. |
| Desired model / reasoning | `gpt-6.1-sol` / `high`, built-in `openai`, existing ChatGPT auth. Prior authenticated model-list advertisement is retained, may be cached, and is not entitlement or serving-model proof for a new invocation. No account/model refresh is proposed. |
| Pinned API schema | [Archived schema ZIP](../evidence/policy-compatibility/run-1/protocol-schema.zip), SHA-256 `cd24042eec4696f6b73368cfc6ce930726f3e10c28bae63cdbd1b2f502ff1d97`. Members `v2/ConfigReadParams.json`, `v2/ConfigReadResponse.json`, `v2/ConfigRequirementsReadResponse.json`, `v2/ThreadStartParams.json`, `v2/ThreadStartResponse.json`; member hashes in [source evidence](../evidence/s1-common-runner-readiness-plan-20261008/source-evidence.json). |
| Metadata request capability | `config/read` types optional `cwd` and `includeLayers`; response has config/origins and optional layers. `configRequirements/read` types nullable `defaultPermissions`, `allowedPermissionProfiles`, `allowedSandboxModes`, `allowedApprovalPolicies`, `allowedApprovalsReviewers`, `allowedWindowsSandboxImplementations`, and feature requirements. Presence in schema proves the interface, not that a particular response will populate it. |
| Permission-profile capability | Generic Config allows additional properties but does not type `default_permissions` or `permissions`. Do not turn that permissiveness into a proven config contract. Thread-start schema has mutually exclusive named `permissions` and legacy `sandbox`; its response can report `activePermissionProfile`. These future thread fields are **not invoked** by the proposed read. Reading requirements cannot establish an active per-thread profile. |
| Pinned local precedence observation | Third read's origins identify the session flags for the selected feature/approval values and the user layer for Windows sandbox. Layer-source schema names packaged defaults, system, managed, user/profile, project and session flags; variants alone do not prove their complete precedence. No runtime/client source commit for this Codex binary is established by the study packet. |

Current [official configuration documentation](https://learn.chatgpt.com/docs/config-file/config-basic)
describes CLI overrides above trusted project layers, selected profile, user,
managed defaults, system and built-ins, subject to enforced requirements. The
[official permissions documentation](https://learn.chatgpt.com/docs/permissions)
describes named permissions and legacy-sandbox interaction, including managed
profile allowlists. These are current documentation read on 2026-10-08; they are
not a frozen 0.160.1 implementation audit or proof of historical exec behavior.
Managed/system requirements remain constraints; a requested value is not an
ability to override them. No global file or requirement is edited.

## Proposed operation, not an executable authorization

Use one new external, public, arm-independent location, proposed exact cwd:
`C:/Users/Consiliari/Documents/Scientist-Probes/s1-common-runner-prospective-20261008/actor`.
It does not exist as a prepared case in this package. A later metadata grant must
bind its actual existence, clean public contents and cwd; missing prerequisites
stop rather than fall back to a project or historical Actor workspace. Only
metadata is requested, with no task prompt, private rubric, study data or earlier
Actor history. Process cwd and `config/read.cwd` must match exactly. Existing
CODEX_HOME/auth is not copied, cleared or redirected; no credentials are read or
persisted by the study client.

The exact proposed native argv is the pinned executable followed by:

```json
[
  "app-server", "--stdio", "--strict-config",
  "--config", "default_permissions=\":read-only\"",
  "--config", "approval_policy=\"never\"",
  "--config", "apps._default.enabled=false",
  "--config", "features.apps=false",
  "--config", "features.goals=false",
  "--config", "features.hooks=false",
  "--config", "features.memories=false",
  "--config", "features.multi_agent=false",
  "--config", "features.plugins=false",
  "--config", "features.shell_tool=true",
  "--config", "features.unified_exec=true",
  "--config", "forced_login_method=\"chatgpt\"",
  "--config", "model=\"gpt-6.1-sol\"",
  "--config", "model_provider=\"openai\"",
  "--config", "model_reasoning_effort=\"high\"",
  "--config", "project_doc_max_bytes=0",
  "--config", "web_search=\"disabled\""
]
```

This retains the prior fifteen inline settings, adds the desired model, and
requests `default_permissions=":read-only"` only in this proposed future process.
Pinned app-server help documents `--strict-config` rejection of unrecognized
config fields. Current official documentation names that built-in, while the
pinned API proves named-profile interfaces but not this config key's implementation.
Its strict acceptance and reported value are therefore the new observations to
seek, not assumed facts. No user/system file, legacy sandbox key, bypass switch,
endpoint or provider changes. Unknown access remains unknown. If managed requirements conflict with
these requests, report that conflict and stop; do not relax requirements.

Send exactly these messages once, in order, awaiting each response where present:

```json
{"id":0,"method":"initialize","params":{"clientInfo":{"name":"scientist_common_runner_policy_read","version":"1.0"}}}
{"method":"initialized","params":{}}
{"id":1,"method":"config/read","params":{"includeLayers":true,"cwd":"C:/Users/Consiliari/Documents/Scientist-Probes/s1-common-runner-prospective-20261008/actor"}}
{"id":2,"method":"configRequirements/read"}
```

No thread/turn, command/file, account/model, authentication or config-write RPC.
A permission default is requested only under the later grant; no thread-level
profile or tool command is applied by these RPCs. Nothing is applied by this
planning package. The proposed allocation is **one app-server process tree,
zero Actors/turns/model calls, parallelism one, zero retries**; 33 seconds active
RPC work, a 38-second owned-process execution deadline, at most 50 seconds
through native/controller completion including the existing up-to-ten-second
cleanup allowance, and 60 seconds outer wall through sanitization and terminal
receipt readback. Any missing/late terminal receipt is nonpositive; the execution
deadline alone is not claimed as the cleanup completion time. Retain the existing Windows Job
kill-on-close and bounded ingestion: queue 32 frames, 4,000,000 raw bytes, at most
16 notifications. Native internal authentication/network behavior is not proven
absent by the RPC list; transport metadata is not provider-inference evidence.

Reuse the existing classified client's exact disabled-status notification rule:
only a schema-valid `remoteControl/status/changed=disabled` can be discarded
without response. Connecting/connected/errored/malformed or unknown notifications,
server requests, auth activity, RPC errors, warnings/provisional config, resource
limits, missing or invalid responses and deadlines terminate this operation.
No reply to a server request, retry, alternate shell/runner or follow-on session.
The client's broader legacy warning handling must not be silently mistaken for
this stricter proposed stop policy. A later narrow client change and independent
review need separate implementation/preparation authority; none occurs here.

## Sanitized result contract and exit decision

Persist a bounded allowlist, with **missing / null / present kept distinct**:

- Executable/request/client/schema hashes, exact argv/cwd reference, start/end,
  wall time, return/stop status, sent method IDs, warning/provisional markers and
  discarded raw-byte counts. No raw frames or stderr/config text is persisted.
- Effective config: typed model/provider/reasoning, approval/reviewer, legacy
  sandbox, Windows implementation and selected feature booleans. For untyped
  permission keys, record missing/null/invalid/present separately and retain only
  the known built-in default `:read-only` when exactly reported; profile map
  contents stay summarized/aliased. Do not claim resolved active permissions or
  store arbitrary values.
- Config layers/origins: source-type enum, version digest, selected-profile
  presence and sanitized known fields, without raw config, instructions, source
  paths, account identifiers or profile names. Preserve reported order as an
  observation; do not invent a missing priority.
- Typed requirements: defaultPermissions and allowedPermissionProfiles rendered
  as built-in IDs or stable opaque aliases for custom names, with alias equality
  preserved between default and allowlist; allowed sandbox/approval/reviewer/
  Windows implementation enums, selected featureRequirements, and network
  presence/enabled only. Unknown/custom text, paths, domains, provider definitions,
  developer instructions and credentials are discarded. Exact custom IDs cannot
  be selected from this sanitized public record alone; the policy owner supplies
  any later concrete profile binding through a new reviewed grant.
- `activePermissionProfile`, effective Actor tool inventory, read/write/build/test
  access, serving model, provider requests/retries and inference usage stay
  `not-observed` / null. Zero requested turns is a client behavior boundary, not a
  claim that native internals made no network requests.

The new metadata can establish strict acceptance/rejection of the requested
read-only config key/default, and report applicable requirements/legacy settings
with presence distinctions the old result lacks. A non-null incompatible legacy
sandbox or managed constraint makes this proposed selection insufficient; no
protective setting is removed. Config acceptance is not active-thread permission
or OS enforcement. If it cannot, its finite result is
**insufficient**, with no automatic diagnosis continuation. The missing fact is
the active permission/tool boundary of a new 0.160.1 Actor in the designated cwd;
its owners are the Codex client/Windows enforcement and managed-policy boundary.
Scientist cannot infer or grant it. A typed restriction is for its policy owner
to interpret, not for the harness to bypass. Overseer decides any next finite
allocation after this one result; this plan requests no blanket user permission.

Even a clean metadata result permits no real Actor by itself. A future separately
granted fresh session would have to observe its actual thread permission/model/
reasoning fields and ordinary tools, then demonstrate the specifically allowed
capabilities on public, case-independent inputs. Successful reading would not
clear writing, builds or tests. Serving-model and provider counters absent from
the protocol remain unknown; no self-report or impossible receipt requirement
is introduced. Common real delegate wiring and the six-cell study need their own
subsequent review and authority. There is no automatic second operation here.

## Accounting and completion of this planning package

This package adds zero CLI/app-server/metadata/Actor/provider/native/controller/
wrapper/delegate/study starts, case preparations, admission actions, freezes or
ledger writes. Native fixture history remains 15 starts / 16 wrapper attempts /
13 deterministic delegates / 2,250 reserved seconds. The five historical real
Actor starts, 53,331 known tokens and unknown total, six CLI metadata calls and
three policy-read trees remain consumed. R4/R6 incomplete and R5 NOT ADMITTED
remain unchanged. S1 remains open; all six cells NOT RUN; no Conventional reference
or method comparison, semantic quality or human acceptance is claimed.

The package retains source and schema hashes, historical document bytes, an
independent bounded readiness/methods review, and local documentation-only
verification. It performs no product test and publishes nothing externally.
