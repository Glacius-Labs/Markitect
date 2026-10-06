# .NET projector

`Evaluate(Input)` preserves the existing candidate shape check against exact paths and project-owned guidance. Accepted bytes remain `candidate-unverified`; this result does not establish semantic correctness.

`Propose(Input)` consumes only the selected Core Schema/Definitions, matching .NET `ProjectionPolicy` guidance, registered roots, and Host-translated prior and observed state. It returns `work`, `no-op`, or `escalate`; work contains a bounded `ExecutorTask` with ontology purposes/contracts, policies, target prefix, registered roots, supported extensions, and observed currently-owned artifacts. The Executor chooses filenames from that task. The module does not hard-code Kind mappings, generate candidate files, or access the filesystem.

A no-op requires a complete inspected inventory and byte/mode agreement with the active record. A current supplied verification binding clears `EvidenceRefreshRequired`; an older materialization request digest alone does not keep unchanged artifacts stale. unknown, retired, incomplete, or ambiguous ownership escalates. The Host runs declared checks and records their evidence. The module never deletes artifacts or claims verification.

When the Host supplies validated `RepairEvidence` for a current completed semantic failure, unchanged complete owned artifacts may produce a new bounded Executor task with the exact finding subjects/details and opaque record/result IDs. The module checks only the evidence shape and size; the Host owns freshness, provenance, and failure classification. Missing or malformed repair evidence never authorizes semantic repair, and the module does not retry, apply, or verify the resulting candidate.
