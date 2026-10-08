# Government executor response contract assessment

Assessment date: 2026-10-08  
Purpose: offline attribution of the retained R3 Government executor failure.  
Method: read-only inspection of pinned source and saved run artifacts. No product, controller, wrapper, delegate, model, provider, or test process was started for this assessment.

## Pinned source and contract

The product source under review is Government commit `04e225d5caee78c2a198607143863fca1e829750` (the R3 request binds this source). Source paths below are relative to that commit; Git blob IDs pin the inspected files:

| Source | Blob ID | Relevant lines |
| --- | --- | --- |
| `internal/host/agentexec/types.go` | `664a36e5984ae2a3360b733f50b40b3c18f41cbb` | 57-90 |
| `internal/host/agentexec/runner.go` | `84893e927a96edd787152a769d5ae55b042e506a` | 225-238, 369-493 |
| `internal/host/agentexec/json.go` | `7b33564ce88a06e615e994187b7179f885cd2b46` | 26-48, 51-69, 199-220 |
| `internal/host/government/execution/run.go` | `f8ca2ca796a9b4f81990dd10fcae2f88d042f790` | 40-72, 347-364, 385-430 |

`Response` declares the identity and outcome strings, `candidateFiles`, `evidenceRefs`, `verifierObservations`, `uncertainty`, optional `candidateJson`, and optional `usage` (`types.go:77-90`). Strict decoding rejects duplicate keys, case-folded protocol-key aliases, unknown fields, invalid UTF-8, and trailing JSON (`json.go:26-48,51-69`). Response validation then checks exact invocation identity and requires all four arrays to be non-null, bounded arrays (`runner.go:369-380`). An empty array is valid; omission or JSON `null` decodes to nil and is rejected.

The role-specific contract is:

| Role | Allowed outcomes | Required/allowed response content | Forbidden content |
| --- | --- | --- | --- |
| Executor | `proposed`, `failed`, `incomplete`, `escalated` | All four required arrays remain present. `proposed` requires at least one `candidateFiles` entry. Evidence references, uncertainty, and candidate files otherwise follow their field validators. | `verifierObservations` must be empty; `candidateJson` must be omitted. |
| Verifier | `passed`, `failed`, `incomplete`, `escalated` | All four arrays remain present. `passed` and `failed` require at least one explicit `verifierObservations` entry. | `candidateFiles` must be empty; `candidateJson` must be omitted. |
| Infer | `proposed`, `failed`, `incomplete`, `escalated` | All four arrays remain present. `proposed` requires `candidateJson`, which must be a JSON object. | `candidateFiles` and `verifierObservations` must be empty. |

The exact guards and protocol error strings are in `runner.go:382-416`. For every role, evidence references must be unique and drawn from request artifacts, scope IDs, or policy IDs (`:443-465`); observations require nonempty subject/detail and a valid outcome (`:466-476`); uncertainty entries must be nonempty (`:478-482`). Candidate-file paths must be canonical portable paths, unique, use an allowed mode, contain valid UTF-8, and stay within the aggregate byte bound (`:418-442`).

`candidateJson` is `json.RawMessage` with `omitempty`, so its absence has zero length while literal JSON `null` remains four bytes. Executor and Verifier both reject any nonempty `candidateJson`, including `null` (`runner.go:389-396`). Infer proposals require it, and any supplied value must canonicalize to a JSON object; `null`, scalar, or array values do not satisfy that rule (`:401-416`; `json.go:199-220`). Optional `usage` may be omitted or `null`; when supplied, it must declare `source: "provider-reported"`, contain only nonnegative counters, and provide at least one counter (`runner.go:483-490`). Null identity/outcome strings decode as zero values and fail the invocation/outcome checks. These null distinctions are source-contract findings, not inferred product behavior.

No inference fixture expansion is part of this assessment: the `Infer` row documents the pinned host contract only. The retained R3 case exercises Executor only.

## Retained R3 evidence and causal attribution

The raw role stdout is `government/role-evidence/579807ab283d8dfd5484b28076523cabd6e666050cdec8ec91cf58732e0dc9f7/stdout.log`, SHA-256 `7d434a9c103ced86de2886a74bdc7ce589c7956b0f4f5bb356fc896f6b9c6092`. It contains a correctly bound `executor` response with `outcome: "proposed"`, two candidate files, supplied evidence references, and empty `verifierObservations` and `uncertainty` arrays. It also contains the field `"candidateJson": null`. That single field violates the Executor contract and matches the exact native error `executor response contains verifier or inference output` (`runner.go:389-390`). The other retained response content is not needed to trigger this rejection.

The role process record is `.../role-evidence/579807ab283d8dfd5484b28076523cabd6e666050cdec8ec91cf58732e0dc9f7/process.json`, SHA-256 `87ceea22df9d4e1b09a565dbf6de33110c54067f69fe36ea739b21ec7a86e7ea`; its stdout digest matches the raw bytes above. The controller's saved process record is `government/controller-evidence/process/process.json`, SHA-256 `e3b8536a1c3cf4f1f6c76f98d1f1c64ab028f40e5a2994a9f843a82a8c50e55a`; its saved stdout is SHA-256 `0f58f63b4445361a80d0f0dad3ed9ad9e7af3c457fea99a3f1e741dee1e2b525`. That stdout records an OS process return code of zero but a Government queue status of incomplete, with one job in incomplete state and the same executor error. The process exit code therefore does not indicate successful job completion.

At `runner.go:225-237`, a decoded response is validated before it is assigned to `RunResult.Response`. On validation failure, the runner returns the receipt and error without assigning the response. That control flow explains why the retained actor record has a zero-value `Response`, while its `Receipt` retains the raw stdout digest and incomplete outcome.

The retained native report is `government/results/run-state/government-run--3ff444b05f0b7d0b5293051ba3dbf93c/report.json`, SHA-256 `b01dfcca88a170f34932c708061a8b761b2eb3242c97c6014e99a5ba6ce47e98`. It records `status: "incomplete"`, `stage: "execute"`, and the protocol error; it contains no candidate, checks, votes, or evidence identity. The actor record at the corresponding `actor-0001.json` has SHA-256 `a994a9def09e62d1f3ff143d8ea6d657965e4c12ce359852a3a5e8f90eab3f82` and preserves that error and the receipt.

The absence of report evidence identity is downstream of the protocol rejection, not a separate native evidence-generation defect. In the non-recursive execution path, `run.go:347-352` immediately returns when the executor invocation errors, before proposal application (`:357-364`). Evidence is assigned only after candidate creation, fresh technical checks, and independent review (`:385-427`). `Report.Evidence` is optional with `omitempty` (`:40-65`), so the earlier incomplete report correctly has no evidence identity: execution never reached the evidence-producing stage. Any adapter note about missing evidence identity describes that incomplete report but is secondary to the saved executor protocol error. The source and retained artifacts support this causal ordering; they do not establish semantic quality of the proposed files or any later review result.
