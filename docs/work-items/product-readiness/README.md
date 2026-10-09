# Product readiness and Main integration



Direct user mandate, 9 October 2026. This backlog replaces the earlier Design-only configuration exception and stop on Main integration. Build a stable, correct, usable Markitect, finish the native agent experience, add MCP and Codex App Server adapters, clean the repository and integrate the validated product into Main. Case Studies and retired Classic/Government work remain stopped. No additional release publication is implied.



This directory is the durable queue. `backlog.yaml` owns statuses, dependencies, assignments and evidence pointers; the numbered documents own scope and acceptance. Product Integration (chat `01a121a0-0417-71c1-9e4b-be742f8c1146`, branch `codex/product-integration-20261009`) owns P06–P10, integration and this queue. Designer completed P01/P02 at the shared dispatch base `1495e1be7b0046711531fa42c8407fba67b8e814` and no longer owns new implementation. Root created the initial handoff and coordinates material completion/blockers; chat history is not the task database. On context reduction or restart, read this directory, the current source/status and the last evidence entry, then continue the first eligible item. Do not revert to an old waiting prompt or require repeated human prompting.



## Execution



Finish the current compatibility/native checkpoint as P01 before starting overlapping changes. Then P02 establishes only the concrete shared contracts needed by P03/P04/P05. Workspace, App Server and MCP work can proceed in parallel with disjoint ownership and focused independent reviews. Product Integration owns shared DTOs, Host orchestration, dependencies, canonical rules, wiring and integration. Subagents own named adapter/workspace files and tests; use isolated feature worktrees when needed. Do not let separate agents concurrently edit shared contracts. Integrate and test each workstream before marking it done. Publish coherent checkpoints normally; no Force push or branch-protection bypass.



Current path references describe the pre-cleanup source. P08 updates the queue and all build/import/package/schema/agent references when moving production source into `src/`. Keep one current maintained path and workflow. Historical releases/Git pins remain evidence; compatibility-only runtime paths are removed. Protect unrelated adopting-project files and existing WIP, not obsolete product APIs.



Use statuses `ready`, `blocked`, `in_progress`, `review`, `done`. Record actual commit, targeted checks, independent review and limitations for each completion. Dependencies are completion conditions, not permission questions. Already authorized normal model edits, checks, repairs, candidate reviews and Apply should continue autonomously within the assigned Work Item. A typed diagnostic should enable repair and recheck; it must not automatically terminate ordinary work. Ask the human only for a material unresolved intent, new purchase/paid commitment, or a concrete action that cannot be completed under the mandate.



## Required user experience



After setup, a normal prompt such as "Implement these Work Items and merge the result to main" is sufficient. Markitect installation supplies current provider-native instructions and operation skills. The outer agent develops the canonical YAML within its task authority and uses actual Markitect operations. Every Manager is a real fresh independent agent with the project context and tools needed for its task; reviewers are independent agents. Workspaces support Git/history, shell, tests, additions, modifications, deletions and renames. Native helpers are real tracked agents rather than disabled placeholders. Required context is not limited to an artificial few-file view; acceptance of changes still respects ownership and the chosen base.



CLI and MCP call the same product services. Codex App Server runs Managers/reviewers/helpers with actual independent contexts. Skills describe how to work; adapters enforce concrete tool/protocol contracts; neither replaces the canonical model or fabricates votes, results or reviews. A helper must not duplicate a Host-scheduled Manager. Recovery preserves real completed work and reports uncertain execution without blind replay.



## Product validation authority



Focused source/protocol/negative/regression tests are authorized as needed for these items. Required final platform gates run on the integrated final candidate; avoid repeated unchanged full suites. Existing failed native proof and closed 600-second lease remain preserved and consumed.



This new mandate supplies a separate finite product acceptance allocation, not a Case Study: at most three actual native acceptance jobs, each at most 1,800 seconds wall time, total at most 5,400 seconds, and at most 64 Manager/helper/reviewer start requests across all jobs. Count failed requests and all nested starts. Use the existing authenticated Codex installation and Luna High for measured product roles. One job validates App Server operation, one validates real parallel Managers/helpers/independent review, and one validates the ordinary end-to-end delivery/repair/recovery workflow. Product Integration records an exact source/config, finite reservations, start/deadline, process/thread handles, results and remaining quota before each actual job. No automatic quota refill or reuse of old allowances. Protocol stubs and mock tests are not native acceptance. Missing cost receipts remain unknown; they do not prevent functional validation. No purchases, new provider API/key integration, installation without need, or weakening ordinary user rules/security.



Developer implementation subagents and read-only reviews are authorized separately from those measured product-role starts; they are not study observations or product proofs. Use them only for bounded useful workstreams, account for their effort, and keep the existing user-selected model/profile for actual product execution. A reproducible defect receives a source fix and focused regression before another permitted product job; no blind retry loop.



## Main and coordination



P10 may update PR89, push the feature branch and merge normally after all readiness criteria, required hosted gates and independent review are positive. This is directly authorized by the user; do not ask for a second blanket Main approval. No release/tag/package publication follows automatically. Root verifies the material handoff and actual remote Main SHA/checks. Do not declare the queue done from a push, green mocks, or a merge alone.



Report material completed items, concrete blockers or needed decisions exactly once to Overseer `01a11367-a781-7683-a20f-46e12614dcb4` on host `local` using `send_message_to_thread`. The user's explicit coordination request authorizes these callbacks. Include item IDs, exact source/remote SHA, checks/review, real proof versus mocks, and next eligible work. No ACK loops or routine progress polling; continue eligible authorized items after reporting. Root can follow up on the same explicit mandate. Case Studies resume only on a later user instruction after product readiness.


Current integration checkpoint and next dependencies are recorded in [integration progress](integration-progress-20261009.md). This is source/developer evidence; actual acceptance remains in the separate ledger.
