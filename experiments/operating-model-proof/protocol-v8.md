# Real-agent operating-model proof protocol v8

Protocol ID: `operating-model-proof/v8`. This source-only amendment precedes new calls. v1-v7 records, binaries, runner copies, failures and blocked attempts retain their exact meaning and bytes.

The runner is pinned to sha256:b632fd3b7b5766183fcdb8bfccb3de9ad81053fda5f6acf35bac288b0b21a594. When a controller supplies `context.requiredObservationSubjects`, the prompt uses those exact typed scope, policy, artifact and fixed-check subjects. A passing response must account for every subject with a concrete passing observation. Negative or incomplete responses may preserve partial observations. Observation identities and evidence references remain distinct: evidenceRefs still use the complete scope-ID, policy-ID and artifact-path union, never typed check tokens.

The adapter validates the bounded subject list and constructs the full prompt before provider process creation or workspace writes. Legacy requests without the list retain the earlier scope/policy prompt contract. This fallback does not allow a new Host controller to reuse old evidence: the Host receipt-check version and full verifier-input digest bind the changed contract.

No observation count establishes semantic truth, human acceptance, independent reasoning, benefit or privacy beyond the tested acquisition boundary. No provider call is authorized by this tooling amendment. Build-receipt validation, fresh-process roles, literal runtime bindings, complete prompt-submission witness, guarded Apply and all existing bounds remain unchanged.
