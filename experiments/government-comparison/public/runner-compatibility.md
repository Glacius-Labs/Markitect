# Runner compatibility, 2026-10-07

Preferred next candidate: the already installed desktop binary **codex-cli
0.160.1**, keeping **gpt-6.1-sol/high**. No model alias, installation or global
configuration change is needed to select it. This is a supported local candidate,
not proof of account entitlement or successful inference. S1 remains open; the
two authorized Actor starts remain consumed and none was added here.

| Question | Source-bound finding |
| --- | --- |
| Desktop name versus CLI ID | Official OpenAI documentation maps GPT-6.1 Sol to `gpt-6.1-sol`, including `codex exec`. The current binary's bundled record uses that exact slug and display name `GPT-6.1-Sol`; `high` is explicitly listed. No different model name is substituted. [Official models](https://learn.chatgpt.com/docs/models). |
| Old runner/protocol | The pinned npm 0.130.0 bundled catalog has six legacy entries and no target. Its saved remote refresh contains only `gpt-5.5` and `codex-auto-review`; `max` in the latter causes the demonstrated enum decoding failure. This establishes protocol incompatibility with that returned catalog; it does not establish a target alias or the cause of model rejection. |
| Current local compatibility | 0.160.1's bundled catalog contains 11 entries, including the exact target with `low/medium/high/xhigh/max/ultra`. Non-Agent `features list` accepts the corrected target/high configuration and separately accepts `model_reasoning_effort=max`. This demonstrates local option parsing, without proving successful decoding of a fresh authenticated catalog or inference protocol. |
| Account entitlement | The old invocation received HTTP 400; no target entry or successful reply was received. A bundled catalog is client metadata, not an account grant. Current-client entitlement remains unknown; official availability depends on client, rollout and workspace/account settings. [Official models](https://learn.chatgpt.com/docs/models). |

The preferred executable is:

```text
C:\Users\Consiliari\AppData\Local\OpenAI\Codex\bin\5ea220ae823df3d7\codex.exe
version: codex-cli 0.160.1
SHA-256: 3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916
```

Native `exec --help` confirms `--model`, `--config`, `--ignore-user-config`,
`--ignore-rules`, `--sandbox read-only`, `--ephemeral`, `--json`,
`--skip-git-repo-check`, `--cd` and stdin `-`. The config check accepted explicit
`model_provider="openai"`, `model_reasoning_effort="high"` and the corrected
settings from attempt 2, without reserved `model_providers.openai` definitions.
`--ignore-user-config` still leaves auth at CODEX_HOME, according to native help.

The minimal proposed correction is to select this absolute executable and pin
its version/hash in a future reviewed runner candidate, retaining the exact model,
reasoning and authenticated built-in provider. A plain `codex` currently resolves
to the older npm shim first. No runtime pin or dispatcher was changed by this
read-only assignment, and the historical 0.130.0 evidence stays intact.

There is one additional compatibility gap: the new runner's `features list`
reports `unified_exec=true` and `unified_exec_tty=true` despite the passed
`features.unified_exec=false`; it reports `shell_tool=false`. Therefore the old
configuration cannot be called an equivalent effective tool boundary solely
because it parses. The documented flags are separate controls; their effective
composition here remains unresolved. [Configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference).

The next decision is whether to adopt this pinned candidate and authorize a
separate, bounded follow-up that first settles effective tool settings and current
account metadata. The existing two-start grant provides no further inference or
access probe. `model/list` is a documented discovery interface, but it was not
invoked here and its examples are not account evidence. [App-server model discovery](https://learn.chatgpt.com/docs/app-server#list-models-modellist).

All six new native commands were version/help/bundled-catalog/config-feature
metadata operations with an empty credential-free CODEX_HOME and no model refresh.
Credentials were neither read nor copied; no install, purchase or global setting
changed. Catalog instruction text was treated solely as untrusted diagnostic data.
Raw receipts and extracted metadata are in
`evidence/runner-compatibility-r1/`; the manifest binds the report and receipts.
Independent inspection confirmed the old catalog findings and the new feature
flag discrepancy. No full suite, live trial or additional Actor ran.
