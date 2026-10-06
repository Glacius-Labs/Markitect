# C6 Codex proposal scope packet — corrected, authoritative inventory

This is the authoritative corrected scope inventory for any human review of a future Codex-backed C6 proposal. The earlier `blocked-codex-propose-scope.md` packet is **INVALID for approval scope**: it named obsolete fixture paths for the Foundation 1.2.0 and Codex projection modules. Preserve that earlier file only as the contemporaneous record of the automatic-review rejection; do not use its source inventory to approve a payload. The automatic-review decision and quoted reason in that historical record remain unchanged.

## Why the earlier inventory was wrong

The earlier note mixed in paths from an older fixture layout (`modules/foundation/.../schemas/foundation.yaml` and `modules/agent-rules-codex/...`). Those paths are not selected by the exact `canonical-codex-only.yaml` at `f256ab776987c808fb8d537758c2dfd2b1831352`, and must not be treated as payload inputs. The immutable config actually selects `modules/foundation-v1.2.0/{module.yaml,schema.yaml}`, `modules/workflow/{module.yaml,schema.yaml}`, and `modules/agentrules-codex-v1/module.yaml`. The corrected inventory below is derived from `git show` of the exact config and `git ls-tree` metadata at the frozen commits, not from the old packet.

## Frozen commits and selection

- Implementation source/build input commit: `17ad4205f15efbf7ae6ccd04b2d1085201602cd3`.
- Proof fixture/config commit: `f256ab776987c808fb8d537758c2dfd2b1831352`.
- Disposable external clone control revision: `a997ec8791c3237cf5e37c6f2e24e074add53926`; read-only inspection confirms its `canonical-codex-only.yaml` blob equals the f256 config and its HEAD is a997.
- Selected Source config: `examples/canonical-workflow/canonical-codex-only.yaml`.
- Config selects these nine Definition paths: four Codex ProjectionPolicies, one Projection, and four canonical workflow Definitions, all enumerated in the table below. No other Definition paths are selected by this config.
- Exact identities selected: Rule `markitect.workflow/v1 example retain-source-authority`; Process `markitect.workflow/v1 example review-workflow-change`; Responsibility `markitect.workflow/v1 example workflow-owner`; Gate `markitect.workflow/v1 example source-review`; and Projection `markitect.foundation/v1 example workflow-codex`. The four Policies target the Codex projection for those four source Kinds.
- Module pins declared by this exact config: `markitect-foundation@1.2.0` digest `sha256:6e42fe4d786693af2824a4fe7bbaed3cce9b2e372055f40938215cef6cf95fb4`; `canonical-workflow-example@1.0.0` digest `sha256:b8cef0bd63f766413de019f7b9de1db3727a67ad028aab2cc7fd124e62ca6995`; `markitect-agent-rules-codex@1.0.0` digest `sha256:9607dbdec580016b6b5a250a0b75112d5f58bebfbb8fb15a60c1cefd5deca966`.
- Projection binding in config: exact Projection key `["markitect.foundation/v1","Projection","example","workflow-codex"]` to module `markitect-agent-rules-codex`.
- Read-only model request digest previously returned for this selection: `sha256:04b545214125293a2b1859c23cbeeb5047bc89845966b15d7b65925c9d5d07d6`.
- Fixed check: `canonical-workflow-check`, argv `go run ./examples/canonical-workflow/check/main.go`; the runtime explicitly names the check program and four canonical Definition inputs below.
- Expected selected output: `agent/AGENTS.md`, mode `100644`, under target prefix `agent/`. Runtime explicitly excludes `agent/owner-note.md` with a reason. `agent/unowned.txt` is the unowned sibling control.

## Exact source/config/check/module inventory

Each path below is taken from the config's `definitions`, `modules[].manifest`, `modules[].files`, or check/runtime inputs, plus the check program that the check argv runs. Git mode, size, Git blob ID and SHA-256 are read from the exact f256 tree/blob bytes. The disposable clone's source entries have the same blob IDs at a997.

| Config role | Exact path | Mode | Bytes | Git blob | SHA-256 |
| --- | --- | ---: | ---: | --- | --- |
| Source config | `examples/canonical-workflow/canonical-codex-only.yaml` | `100644` | 1897 | `117385ed0ac05881a0a55a3876f00f4254f68881` | `0005e7ecce4ec35cc35b51b7f97d1f0553afdaa5eec8651c44e669d2960fcbda` |
| Definition: Policy | `examples/canonical-workflow/definitions/policy.rule-to-codex.yaml` | `100644` | 672 | `b0de2f6020e27228e28eec3fc76a39ca2721177f` | `a79042922abe2f946ceca1883c365b4591856831485da098cba8b38170b3167f` |
| Definition: Policy | `examples/canonical-workflow/definitions/policy.process-to-codex.yaml` | `100644` | 681 | `10dc73927b39c47217b2c7114ff2c9470a9272ce` | `337cf33abd4bc81b5a836cc21f2129d6bb88066b93adc565f7574ef43fbb531d` |
| Definition: Policy | `examples/canonical-workflow/definitions/policy.responsibility-to-codex.yaml` | `100644` | 702 | `da3f3289dda7e5e4fe36fd8d8850aeae0187a06d` | `a8089434c16e0c1f88b01a750c5ef76c95c1fd2162e604120b612e94d287c220` |
| Definition: Policy | `examples/canonical-workflow/definitions/policy.gate-to-codex.yaml` | `100644` | 672 | `caae5aeed49fc211a79e09d1f85c9cd309a8afc6` | `74bd492201a8db706d359c8598ee246f7185676fbfe6531c42ecaa24dc16a760` |
| Definition: Projection | `examples/canonical-workflow/definitions/projection.workflow-codex.yaml` | `100644` | 1315 | `c93ba15ca81ba78693c2e89fc7b95bacde2975a6` | `a512ba1e5098c629507827117f28561f94220d66845a5e41b6e8866c3bef9089` |
| Definition: Rule | `examples/canonical-workflow/definitions/workflow.rule.yaml` | `100644` | 370 | `fbf3abfe9f4e654c8739fc48b0315b243efa9559` | `dbd7787f42a780d8ee50133591fdc1b493f67633872034c7ed18019c7d211645` |
| Definition: Process | `examples/canonical-workflow/definitions/workflow.process.yaml` | `100644` | 816 | `65b5d1f96f63da31fe53f6f1be5ddd221f585506` | `15266c867668569fac2d3088f8d88a9351d85abfe5232d12b7dfdeaf9e45bec5` |
| Definition: Responsibility | `examples/canonical-workflow/definitions/workflow.responsibility.yaml` | `100644` | 237 | `e91930912b7f519259a45af07baae16ba1efdee7` | `ec843e6bc49ea692eb5b4bc963e8ed03f9dc420c930aeea1bb370c8791e79a23` |
| Definition: Gate | `examples/canonical-workflow/definitions/workflow.gate.yaml` | `100644` | 282 | `f9c995ad8679e7d99bb7fdaa40cfaf6cd52c4c0c` | `5647326d9410885a01e6403fdd6338f096352f444d49d3c145fddac3e1be3efe` |
| Foundation 1.2.0 manifest | `examples/canonical-workflow/modules/foundation-v1.2.0/module.yaml` | `100644` | 302 | `d3413bbb14e63967a95fca31b6ee9b771488145a` | `b3d40a2ba115204d936727faeb4e0ccb75b4841c2c7bdbfb1cfa3fc3d9b65514` |
| Foundation 1.2.0 schema file | `examples/canonical-workflow/modules/foundation-v1.2.0/schema.yaml` | `100644` | 2907 | `49c754aad8c493b8b08826289d0b0e355f8bf300` | `38bed4d42fb207669dc996abca3bc067e629315c74d59c4d54e049dd7aa2397e` |
| Workflow manifest | `examples/canonical-workflow/modules/workflow/module.yaml` | `100644` | 296 | `707d6f8fca7faee105b15abf72700ad65c4dbf79` | `a0dd2666faa904a6052ef6cee71a73c8d67558811bbf5f25b1d693bde4e174c3` |
| Workflow schema file | `examples/canonical-workflow/modules/workflow/schema.yaml` | `100644` | 1799 | `c25961d36b4dd64d27e971ca0c5c96af53fe0eae` | `4d4d8546cf01c84e38747c66a5a50d676183593645200dedbe7f1ea95057bb47` |
| Codex projection manifest | `examples/canonical-workflow/modules/agentrules-codex-v1/module.yaml` | `100644` | 803 | `37367328aa9ddc4577a4f47360e4ca11585e900b` | `a0c544ed715669acc2698b1dff546b8290ce09af1f4ac611f42a5ef6d1561cf6` |
| Check program | `examples/canonical-workflow/check/main.go` | `100644` | 1383 | `b3a7f1ed8c8719e7833d5402de8c1836f301291d` | `b0d39b7809efdd30c7a67bc52f45006b55b12263a65ccf992613f3e8def2fefd` |

Runtime-declared check input paths are exactly:

- `examples/canonical-workflow/check/main.go`
- `examples/canonical-workflow/definitions/workflow.rule.yaml`
- `examples/canonical-workflow/definitions/workflow.process.yaml`
- `examples/canonical-workflow/definitions/workflow.responsibility.yaml`
- `examples/canonical-workflow/definitions/workflow.gate.yaml`

## External target clone controls at a997

The clone is `C:\Users\Consiliari\AppData\Local\Temp\markitect-c6-proof-17ad-20261006\codex\repo`; target-control revision `a997ec8791c3237cf5e37c6f2e24e074add53926` is its HEAD. Read-only Git metadata reports these exact controls:

| Role | Path | Mode | Bytes | Git blob | SHA-256 |
| --- | --- | ---: | ---: | --- | --- |
| Explicitly excluded manual sibling | `agent/owner-note.md` | `100644` | 60 | `f3e422f8e4634d76b7d0c0ddbf3fb8537b10610a` | `aa692977a8fe49d499719d2bdafa6a5f4a77a7b38a316571b89e087fb7ec0385` |
| Unowned sibling control | `agent/unowned.txt` | `100644` | 27 | `66299840db18f74a51bcffef31816407a00c5d04` | `8361436a4445ba4a05d70df293a20ff51b3beb6428355f7c3205d15f7b2b105` |

## Frozen binary and runtime destination

- Executable: `C:\Users\Consiliari\AppData\Local\Temp\markitect-live-proof-1hs4ntmh\markitect-workflow-modes.exe`, SHA-256 `8d179620a679fdfa04c92888ce2639755aa2bd91d15b7531ed709e17cb64bf76`.
- Build receipt: `C:\Users\Consiliari\AppData\Local\Temp\markitect-live-proof-1hs4ntmh\build-receipt-workflow-modes.json`, SHA-256 `4a7ca77cfbfb83c1e1b5e3f98b755f1de128d072372499438cd9e2d2ab0f3375`.
- C6 runtime config: `experiments/operating-model-proof/runs/c6-preparation/runtimes/runtime-codex.json`, SHA-256 `da3d36f2709e7b71c4b9fa03193d4349b3bf1bf07004ec52495842f14e88bc81`.
- Template runtime config: `C:\Users\Consiliari\AppData\Local\Temp\markitect-operating-model-20261006-c4-aligned-v2\runtime.json`, SHA-256 `8385a59915dec560d65ee3a1e954b1c9c07caa37365c51b3d055daeb0f8147fa`; it is configuration only. C6 uses new, per-clone state/log paths, not the template's C4 paths.
- Destination configured by runtime: Codex CLI `0.130.0`, model `gpt-5.5`, reasoning effort `high`, through `C:/Python313/python.exe` and the runner wrapper below.
- Runner wrapper `C:\Users\Consiliari\.codex\worktrees\c4-aligned-proof\Markitect\internal\tooling\codexrunner\runner.py`, SHA-256 `7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`.
- Native Codex executable `C:\Users\Consiliari\AppData\Roaming\npm\node_modules\@openai\codex\node_modules\@openai\codex-win32-x64\vendor\x86_64-pc-windows-msvc\codex\codex.exe`, SHA-256 `280cb1c4e3375d94dbdcba1a191f4f6adbf73c293be1e4f16c74b006662b9c54`.
- Per-target record-store leaf: `C:/Users/Consiliari/AppData/Local/Temp/markitect-c6-proof-17ad-20261006/codex/state/record-store`; it was absent after automatic review.
- Per-target private logs: `C:/Users/Consiliari/AppData/Local/Temp/markitect-c6-proof-17ad-20261006/codex/private-logs`; it was empty after automatic review.

## Exact proposed command and review outcome

The exact command proposed was a read-only `controller-propose` (no `--write`):

```text
C:/Users/Consiliari/AppData/Local/Temp/markitect-live-proof-1hs4ntmh/markitect-workflow-modes.exe canonical --repo C:/Users/Consiliari/AppData/Local/Temp/markitect-c6-proof-17ad-20261006/codex/repo --config examples/canonical-workflow/canonical-codex-only.yaml --runtime C:/Users/Consiliari/.codex/worktrees/c6-proof-preparation/Markitect/experiments/operating-model-proof/runs/c6-preparation/runtimes/runtime-codex.json --action controller-propose --base f256ab776987c808fb8d537758c2dfd2b1831352 --revision a997ec8791c3237cf5e37c6f2e24e074add53926
```

Automatic review rejected that command **before process start** because the configured Codex executor might send repository contents to an external provider and authorization for that payload and destination was not established. No executable started, no provider request occurred, no proposal output or ledger was created, no target materialization occurred, and no repository files were written by the attempt. No retry or alternate execution path was used. Any provider-backed action still requires direct human response to this corrected payload/destination scope.
