# C6 Codex proposal blocked before invocation

Status: blocked by automatic review before process start. No approval to resume provider-backed work is recorded. This packet fixes the exact proposed scope so a human can make an informed decision.

## Frozen source, proof fixture, and selection

- Implementation source and binary build input: `17ad4205f15efbf7ae6ccd04b2d1085201602cd3` (clean source commit used for build).
- Frozen proof input: `f256ab776987c808fb8d537758c2dfd2b1831352`, additive fixture/config commit. The external Codex clone added only the two controls below and was at control revision `a997ec8791c3237cf5e37c6f2e24e074add53926` for the proposed call.
- Frozen executable: `C:\Users\Consiliari\AppData\Local\Temp\markitect-live-proof-1hs4ntmh\markitect-workflow-modes.exe`, SHA-256 `8d179620a679fdfa04c92888ce2639755aa2bd91d15b7531ed709e17cb64bf76`.
- Build receipt: `C:\Users\Consiliari\AppData\Local\Temp\markitect-live-proof-1hs4ntmh\build-receipt-workflow-modes.json`, SHA-256 `4a7ca77cfbfb83c1e1b5e3f98b755f1de128d072372499438cd9e2d2ab0f3375`.
- Target config: `examples/canonical-workflow/canonical-codex-only.yaml` at proof input commit; one Projection `workflow-codex`, module `markitect-agent-rules-codex@1.0.0`, target prefix `agent/`, expected artifact `agent/AGENTS.md` mode `100644`.
- Exact selected source identities (and no other Definition identities):
  - `markitect.workflow/v1 Rule example retain-source-authority`
  - `markitect.workflow/v1 Process example review-workflow-change`
  - `markitect.workflow/v1 Responsibility example workflow-owner`
  - `markitect.workflow/v1 Gate example source-review`
- Exact selected Projection identity: `markitect.foundation/v1 Projection example workflow-codex` (serialized key `["markitect.foundation/v1","Projection","example","workflow-codex"]`). Four selected ProjectionPolicies target those four selected source Kinds and the `codex` target technology.
- Exact model request digest from the frozen binary's read-only request action: `sha256:04b545214125293a2b1859c23cbeeb5047bc89845966b15d7b65925c9d5d07d6`.
- Fixed check: `canonical-workflow-check`, argv `go run ./examples/canonical-workflow/check/main.go`; explicit check inputs are listed below.
- This configuration is an explicit one-Projection overlay because the controller has no Projection selector; assurance roots do not narrow proposals.

## Source file inventory (proof-input commit)

The following are the exact selected Source/config/check inputs and supporting package files, with Git mode, byte count, and SHA-256:

| Path | Mode | Bytes | SHA-256 |
| --- | ---: | ---: | --- |
| `examples/canonical-workflow/canonical-codex-only.yaml` | `100644` | 1897 | `0005e7ecce4ec35cc35b51b7f97d1f0553afdaa5eec8651c44e669d2960fcbda` |
| `examples/canonical-workflow/definitions/workflow.rule.yaml` | `100644` | 370 | `dbd7787f42a780d8ee50133591fdc1b493f67633872034c7ed18019c7d211645` |
| `examples/canonical-workflow/definitions/workflow.process.yaml` | `100644` | 816 | `15266c867668569fac2d3088f8d88a9351d85abfe5232d12b7dfdeaf9e45bec5` |
| `examples/canonical-workflow/definitions/workflow.responsibility.yaml` | `100644` | 237 | `ec843e6bc49ea692eb5b4bc963e8ed03f9dc420c930aeea1bb370c8791e79a23` |
| `examples/canonical-workflow/definitions/workflow.gate.yaml` | `100644` | 282 | `5647326d9410885a01e6403fdd6338f096352f444d49d3c145fddac3e1be3efe` |
| `examples/canonical-workflow/definitions/projection.workflow-codex.yaml` | `100644` | 1315 | `a512ba1e5098c629507827117f28561f94220d66845a5e41b6e8866c3bef9089` |
| `examples/canonical-workflow/definitions/policy.rule-to-codex.yaml` | `100644` | 672 | `a79042922abe2f946ceca1883c365b4591856831485da098cba8b38170b3167f` |
| `examples/canonical-workflow/definitions/policy.process-to-codex.yaml` | `100644` | 681 | `337cf33abd4bc81b5a836cc21f2129d6bb88066b93adc565f7574ef43fbb531d` |
| `examples/canonical-workflow/definitions/policy.responsibility-to-codex.yaml` | `100644` | 702 | `a8089434c16e0c1f88b01a750c5ef76c95c1fd2162e604120b612e94d287c220` |
| `examples/canonical-workflow/definitions/policy.gate-to-codex.yaml` | `100644` | 672 | `74bd492201a8db706d359c8598ee246f7185676fbfe6531c42ecaa24dc16a760` |
| `examples/canonical-workflow/modules/foundation/module.yaml` | `100644` | 302 | `b3d40a2ba115204d936727faeb4e0ccb75b4841c2c7bdbfb1cfa3fc3d9b65514` |
| `examples/canonical-workflow/modules/foundation/schemas/foundation.yaml` | `100644` | 2907 | `38bed4d42fb207669dc996abca3bc067e629315c74d59c4d54e049dd7aa2397e` |
| `examples/canonical-workflow/modules/workflow/module.yaml` | `100644` | 296 | `a0dd2666faa904a6052ef6cee71a73c8d67558811bbf5f25b1d693bde4e174c3` |
| `examples/canonical-workflow/modules/workflow/schemas/workflow.yaml` | `100644` | 1799 | `4d4d8546cf01c84e38747c66a5a50d676183593645200dedbe7f1ea95057bb47` |
| `examples/canonical-workflow/modules/agent-rules-codex/module.yaml` | `100644` | 803 | `a0c544ed715669acc2698b1dff546b8290ce09af1f4ac611f42a5ef6d1561cf6` |
| `examples/canonical-workflow/check/main.go` | `100644` | 1383 | `b0d39b7809efdd30c7a67bc52f45006b55b12263a65ccf992613f3e8def2fefd` |

Check input paths supplied to the configured check:

- `examples/canonical-workflow/check/main.go`
- `examples/canonical-workflow/definitions/workflow.rule.yaml`
- `examples/canonical-workflow/definitions/workflow.process.yaml`
- `examples/canonical-workflow/definitions/workflow.responsibility.yaml`
- `examples/canonical-workflow/definitions/workflow.gate.yaml`

External control files in the disposable Codex clone at the control revision were `agent/owner-note.md` (60 bytes, SHA-256 `aa692977a8fe49d499719d2bdafa6a5f4a77a7b38a316571b89e087fb7ec0385`) and `agent/unowned.txt` (27 bytes, SHA-256 `8361436a4445ba4a05d70df293a20ff51b3beb6428355f7c3205d15f7b2b105`). The former was explicitly excluded with a reason in the runtime; the latter was the unowned sibling control. Both were intended as ordinary `100644` files.

## Runtime, provider, and external destination

- Runtime configuration: `experiments/operating-model-proof/runs/c6-preparation/runtimes/runtime-codex.json`, SHA-256 `da3d36f2709e7b71c4b9fa03193d4349b3bf1bf07004ec52495842f14e88bc81`.
- Runtime template (configuration only): `C:\Users\Consiliari\AppData\Local\Temp\markitect-operating-model-20261006-c4-aligned-v2\runtime.json`, SHA-256 `8385a59915dec560d65ee3a1e954b1c9c07caa37365c51b3d055daeb0f8147fa`. The C6 runtime has distinct, fresh C6 private-log and record-store paths; it does not reuse the template's C4 state paths.
- Destination/provider: Codex CLI `0.130.0`, model `gpt-5.5`, reasoning effort `high`. Executor and Verifier are configured to use `C:/Python313/python.exe` plus the runner wrapper below. A configured external provider invocation could transmit the selected repository payload to that destination.
- Runtime wrapper: `C:\Users\Consiliari\.codex\worktrees\c4-aligned-proof\Markitect\internal\tooling\codexrunner\runner.py`, SHA-256 `7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`.
- Native Codex executable: `C:\Users\Consiliari\AppData\Roaming\npm\node_modules\@openai\codex\node_modules\@openai\codex-win32-x64\vendor\x86_64-pc-windows-msvc\codex\codex.exe`, SHA-256 `280cb1c4e3375d94dbdcba1a191f4f6adbf73c293be1e4f16c74b006662b9c54`.
- Disposable target clone: `C:\Users\Consiliari\AppData\Local\Temp\markitect-c6-proof-17ad-20261006\codex\repo`.
- Fresh record-store leaf: `C:\Users\Consiliari\AppData\Local\Temp\markitect-c6-proof-17ad-20261006\codex\state\record-store` (confirmed absent after rejection).
- Private log directory: `C:\Users\Consiliari\AppData\Local\Temp\markitect-c6-proof-17ad-20261006\codex\private-logs` (empty after rejection).

## Exact proposed command

This was a read-only `controller-propose` command (no `--write`):

```text
C:/Users/Consiliari/AppData/Local/Temp/markitect-live-proof-1hs4ntmh/markitect-workflow-modes.exe canonical --repo C:/Users/Consiliari/AppData/Local/Temp/markitect-c6-proof-17ad-20261006/codex/repo --config examples/canonical-workflow/canonical-codex-only.yaml --runtime C:/Users/Consiliari/.codex/worktrees/c6-proof-preparation/Markitect/experiments/operating-model-proof/runs/c6-preparation/runtimes/runtime-codex.json --action controller-propose --base f256ab776987c808fb8d537758c2dfd2b1831352 --revision a997ec8791c3237cf5e37c6f2e24e074add53926
```

## Automatic review decision and observed boundary

Automatic review rejected this exact command before the executable started. Stated reason: “The controller-propose action uses a runtime configured with a Codex executor and may send repository contents to an external provider; the transcript does not establish authorization for that payload and destination, so sensitive egress must be denied. Do not bypass this rejection through a workaround or indirect execution. Continue with a safer alternative, or carry out checks to prove that the action is authorized or low risk before trying again. Complete unaffected work without asking for confirmation. Report anything that remains blocked, clarify why it was blocked by auto-review, inform the user of the risk and ask for approval.”

Observed result: zero executable starts, zero provider requests, zero proposal output, zero ledger creation, zero target materialization, and zero repository writes from this attempt. The record-store leaf remains absent and the private log directory remains empty. No alternate execution path was used. Any subsequent provider-backed action requires a direct human response to the exact payload/destination scope described here.
