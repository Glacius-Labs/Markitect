# Roombook installation and schema handoff

Source `2b7b22bafb72de1d1ca3bf54e5870c281a171956` remains clean and pushed on `codex/model-first-operations`. Exactly one Windows build was made. Its binary SHA-256 is `abf877849379900cf2f1cf6c19f3d5d1e67a134592ac810cedc437191e9a5f29`. This is an unreleased source candidate.

The installation uses only the authorized public initial model. All ten original hashes and original bytes are retained. Two model files change:

- `booking-contract.yaml`: normalize the unsupported category `behavior` to the existing `rule` category. The behavioral text, identity and references are unchanged.
- `project.yaml`: declare `workflowMode: guided` and `acceptancePolicy: committed-model`, the current initialization defaults required by the intended guided acceptance workflow.

No business rules, paths, checks, required flags or station scope were changed. No application, test placeholder or later-station implementation was written. The real model contains one Manager, four Statements, three Artifacts and one Check.

Normal guarded setup/onboarding and documentation generation created the runtime, both native entrypoints/Skills, shared workflow and readable view. The installed adapter is a source-identical copy of the existing Codex adapter. Manager and local reviewer use `gpt-6-luna` with effort `high`; applicable integration and full audit use the Manager binding. Actual inner transport is the native Codex CLI through Python, with read-only/tools-disabled requests, a 300-second role timeout, 24 starts and 1200 seconds per run, depth two and parallelism one. Study-wide shared accounting and absolute phase deadlines remain unbound; these per-run settings do not establish the shared 72-start envelope.

The installed repository is clean at `d7a004172f98ebb6bc59bf7e93f9c709fb2ebc86`, tree `01b64bd79d5554ffde6f8df3333746b190630abf`. All 18 tracked files match their raw Git blobs; native onboarding repeats as unchanged. The schema hash is `30c8fe1855afe37bd424e61c969f7cbab104848f2c0d95c01d8a6bcb4795cd62`, runtime hash `a99b619fda2226ccd820458b4bc501459b6ad05ba417338942074fead7a0ef98`.

`project check` is **incomplete**, with exactly the required-but-absent `app.py` and `tests/` findings. Coverage is accounted and nonconforming. This establishes schema compatibility and installation facts after the declared migration; it does not establish readiness or a successful autonomous Run/Review/Verify/Apply.

The sealed owner-preparation interval is conservatively measured at 1385.854584 seconds within the separate 1800-second task. This cost is retained; no claim is made that it fits the future 900-second study setup window. No actual study setup/cell ran. One build, zero new model/agent/product/trial/transport-probe starts. Normal setup performed executable-version metadata discovery; authentication, model serving and costs remain unverified. All known own mechanical jobs terminated.

At seal, source CI `37908655579` passed Linux and Windows was running fixed dogfood verification. No full platform PASS or fresh independent review is claimed. The three previous product proof jobs remain exhausted and failed. The sixth study cell stays unallocated/disabled, and the 10800-second post-handoff suite has not started.

Machine-readable handoff: `handoff.json`, SHA-256 `0cf49b54ae019ab2ab2cb6dd3e863c995823a49137a375a9e5e4547ee8aa4cb2`. Archive: `roombook-model-installation-2b7b22ba.zip`, SHA-256 `1f29af68c6ec2b295c6930c76512deac93a3c565dd6f1fbef1539db5a78918b0`. It contains the original model, declared delta, installed model/views/Skills/runtime, adapter, schema, binary and inventories. Absolute runtime pins bind this installation on this Windows host; relocating the package requires normal setup/rebinding before any separately authorized execution.

Root/Scientist still own common case inputs, S1/state, the fresh study repository and empty ledger, shared role/deadline binding, accepted readiness and any separately allocated execution grant. No private Scientist source or evaluator content was read.
