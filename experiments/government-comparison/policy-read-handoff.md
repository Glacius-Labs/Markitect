# Scientist: one authorized config-read session, stopped at handshake

Base `f3eeb57e54408f3ba127c22681f39c2c31491ca9`; separate authorization from
Overseer chat `01a11367-a781-7683-a20f-46e12614dcb4`. The previous six metadata
calls remain consumed. This package records **one additional local app-server
process tree**, with no repeat, Actor dispatch, rejected-read retry, permission
change, installation, full suite or Q1 expansion.

## Result

The native app-server acknowledged `initialize`. The client sent `initialized`,
then encountered an unexpected server message carrying a `method` field and
stopped **before sending either `config/read` or `configRequirements/read`**.
The guard discarded the entire unexpected frame without answering it. No
configuration or managed-requirement values were obtained.

| Observation | Bound value |
|---|---|
| Native binary | Codex CLI 0.160.1, SHA-256 `3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916` |
| Native exit / client exit | `0` / `1` |
| Native session / outer controlled tree wall time | `0.1427724999 s` / `0.5027772000 s` |
| Allowed messages sent | `initialize` once; `initialized` once |
| Config / requirements reads sent | `0` / `0` |
| Stop reason | `unexpected-server-method; contents discarded` |
| Raw protocol bytes consumed then discarded / stderr bytes | `420` / `0` |
| Retry / extra app-server trees | `0` / `0` |

The unexpected method's identifier was not retained. It therefore cannot be
classified retrospectively as Actor activity, authentication, or a benign
initialization notification. The safe stop is evidenced; the identity/cause is
not. Native exit success does not mean the requested read task succeeded. This
does not identify the earlier exec rejection or establish any policy option that
requires a user choice. S1 remains open.

## Exact request, differences and redaction

[Request](evidence/policy-read/run-1/request.json) SHA-256
`8bae84ef28ef37904711d9725756c82d95d48e5d61a23a622de0c6440578f0a1`
contains full argv and the four planned messages. The source-bound
[one-shot client](evidence/policy-read/run-1/read-once.py) SHA-256
`998480229ae2573264904b00c8077ba2a6bed11349b04749cefdc6cd79724390`
adapts the existing stdio transport; the original client was unsuitable because
it forwards raw replies and automatically sends account/model queries. Its source
and the existing process-tree controller are hash-bound in the request.

The new argv was the unchanged binary plus `app-server --stdio` and exactly the
original exec invocation's fifteen `--config` pairs, in their original order.
The client rechecked the frozen process receipt/hash, argv, cwd and its own source
hash before starting. No guessed key or permission profile was added.

The original Actor cwd was both the app-server process cwd and the planned
`config/read` cwd. The exec-only ignore-user-config/ignore-rules, sandbox/model,
ephemeral/JSON/Git-check flags and Actor prompt were not transferred; app-server
help does not document those invocation controls. No equivalent precedence or
policy was claimed. In particular, ordinary user/project/managed layers could
participate in this metadata route even though the failed exec requested that
user config/rules be ignored. The existing inline `approval_policy="never"`
was unchanged, but its runtime effect was not observed. No additional rights
were selected or any global setting changed.

The [official app-server documentation](https://learn.chatgpt.com/docs/app-server)
describes the initialization handshake and the two read methods. The previously
exported schema from this exact binary also declares them. Its
`configRequirements/read` request has no required params object, so this request
omitted params rather than assuming the current web example's shape. No account,
model, auth/login, thread/turn, command, filesystem, sandbox setup or write RPC
was sent.

Raw responses and native stderr were held only in bounded transient memory and
discarded before persistence/display. The allowlist permitted only relevant
approval/sandbox/permissions/Windows values, selected feature flags, policy layer
origins and managed restrictions. File/path and network-target identifiers were
retained only if needed as policy/provenance data; provider config, headers,
credentials, arbitrary strings, unknown fields, initialization user-agent data
and error payloads were excluded. No raw configuration digest was saved. The
32-frame queue and 4,000,000-byte ingestion threshold bound input, with transient
line-buffer overhead; this is not an exact peak-RAM guarantee. The outer existing
Windows Job controller imposed a 60-second tree limit; inner deadline was 50
seconds. No direct credential file was inspected; internal native service/file
activity beyond the observed protocol was not traced.

## Blocker and smallest next reviewable step

The authorized session ended at an **unclassified protocol message**, before any
policy query. The current evidence cannot determine the exec policy, inherited
read/network rights, selected Windows sandbox, applicable managed constraints,
or external-file access. No supported configuration correction follows.

The smallest next change is a code-only refinement of this one-shot stop record:
retain only a recognized protocol method identifier/class from the frozen
notification/request schema, never its payload, and still stop without answering
or allowing it. An unknown identifier can remain redacted. This would make a
future safe stop diagnosable; it cannot recover the discarded frame from this
session. No such change or rerun is applied here. A fresh session or a change to
the allowed inbound message policy needs a separate explicit coordinator scope
decision; no permissions choice or new search/diagnostic loop is implied.

[Independent review](public/policy-read-review.md) and
[validation](evidence/policy-read/run-1/validation.json) bind the client/request,
sanitized result, process receipts, single-use marker and unchanged predecessor
evidence/ledgers. Q1 and its snapshots remain unchanged. The five historical Actor
starts, known token subtotal 53,331 and unknown complete historical total remain.
Scientist and one independent reviewer supplied shared preparation; their active
time/token/cost totals are unavailable, not zero. Hold for Overseer.
