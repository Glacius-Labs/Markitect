> English translation of the German original at commit a1a308fc287b687f9f73219bcf70dc051ee33cc8.

# Markitect V1 – Concept conformance and dominant problems

**Audit date:** 24 October 2026  
**Fixed source state:** `6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998`  
**Comparison basis:** Shared concept decisions D1–D9, consolidated plan v0.17 and the narrow S0–S5 scoping that was explicitly accepted afterwards.  
**Type:** Independent static code and contract review. No implementation approval and no change to the repository.

## 1. Result

The load-bearing architecture is actually implemented: preserved model foundation, independent project preparation, shared hierarchical Run executor, direct Apply composition, fresh sessions and working environments, durable candidates and results, Shared Notes and Publication after the end of the Run.

This is **not the same as complete implementation of all agreed ways of working**. Concrete differences exist in the communication paths, in the handling of invalid child handoffs and in the binding of domain completions to individual technical messages. The Harness fulfills reproducibility and native use of instructions, but not completely the originally intended modular, file/skill-based working environment from which material can be loaded in a targeted way.

In addition, the confirmed loss of Context in the Publication guard and a statically traceable size conflict between Notes storage and tool transport remain. These findings are to be distinguished from missing native qualification and from subjective refactoring wishes.

**Recommendation:** No renewed complete rebuild. Preserve the existing core, decide explicitly on or correct concrete deviations from the concept, and secure the affected transitions with targeted negative tests. A green Library run does not replace a review of the organization and responsibility rules.

## 2. Sources and limits

The original plan `markitect-v1-plan-v0.17.md` was read again. Particularly relevant are §4 responsibilities, §5.2 collaboration, §6.2 control points, §8 Harness/provider and §9 Notes/Verify. Earlier proposals on preserving Extract/UI have been superseded by later owner decisions and are not counted here as unfulfilled requirements.

The current Main was resolved to the same SHA at the beginning and at the end of the review. Central production files were read directly via GitHub, in particular Run, Workflow, Apply, Harness, Codex/Claude adapters, project validation, Runstore, Publication and Git guards. In addition, Git subtrees of the preserved foundations were compared and the documented qualification limits were read.

In this review, **no Go, container or provider tests were run**. The local Go version is older than the repository requirement; an independent checkout was not available. Documented native successes are reproduced as repository evidence, not as attempts repeated by the reviewer. The complete local transcripts and evidence packages on the Windows host were not inspected. This is not a complete race, security or line-by-line review of the entire repository.

The categories mean:

- **Implemented:** Directly recognizable in the reviewed product path.
- **Deviation:** Concrete difference from an agreed way of working.
- **Partial/different:** A partial goal is achieved; the originally intended form or extensibility is not completely achieved.
- **Risk:** Statically substantiated failure condition; no claim of a newly executed runtime failure.
- **Evidence limit:** Implementation or component proof exists; broader product evidence has not yet been provided.

## 3. Requirements matrix

| Concept requirement | Finding in the fixed code | Classification |
|---|---|---|
| Preserve model, compiler, snapshot, organization and Realization | Four Git subtrees are identical to `447ac3b1` | Implemented |
| Model→Artifacts; no model change as an escape route for repair | `Prepared.ValidateCandidate` binds model/control sources; registry may change | Implemented |
| Deterministic check before agent start | A valid `Prepared` is a prerequisite for Command and Executor | Implemented |
| One Executor for Assessment, Implement, Verify | All three use `run.Executor.Execute` | Implemented |
| Entire modeled hierarchy in every mandate | Environments and sessions are created for all topology Managers; child work/review is expected | Implemented |
| Run independent of Apply | Own Execute contract, persistence and closure in the Run | Implemented |
| Workflow coordinates only Runs | Sequence is small; `ValidateClosed` additionally knows concrete resource names | Largely; limited responsibility leak |
| Fresh conversations and workspaces per Run | New Open calls, native threads and private homes per Execute | Implemented |
| One Git Workspace Service per Run | Run opens and closes its WorkspaceService instance | Implemented |
| Parent delegates/reviews only direct children | Corresponding parent checks in `Delegate` and `Review` | Implemented |
| Questions and coordination along the hierarchy | `Ask` permits every other Manager of the same Run | Deviation A1 |
| Child check as diagnosis to the parent | With an invalid Submitted state, the host keeps correcting in the child before its handoff is delivered | Deviation A2 |
| Successful Root requires a valid candidate | Implement check, final aggregation, fixed candidate binding | Implemented |
| New Implement/Verify Runs after Reject | Workflow creates new Run IDs and passes on actual findings/candidates | Implemented; public native branch proof open |
| Technical failure, Reject and model blocker separate | Technical status and semantic decision are separate values | Implemented |
| Short assignment plus prepared Harness | Task/feedback compact; background knowledge is injected as extensive Base/Developer context | Partial/different A4 |
| Provider-native files/skills and generic fallback | Codex uses native instruction parameters and MCP; automatic project files are disabled; no productive generic fallback demonstrated | Partial/different A4 |
| Autonomy without a domain turn process machine | Old turn engine removed; active messages nevertheless each require a new `judge` | Tension A3 |
| Ownership and registry | Static assignment and protection of controlled inputs; actual compliance with responsibilities remains parent/Verify review | Corresponds to the explicitly permitted boundary without per-write ACL |
| Durable candidates/results after resource end | Retain before cleanup, terminal save/read afterwards; no live session in the result | Implemented |
| Shared Notes across Runs/Commands | Independent persistent records, append and read | Implemented; retrieval limit R2 |
| Normal CLI authentication without API-key path | Codex accepts account login and provisions via the Manager container | Implemented |
| Claude and further providers | Claude.Open always returns unavailable; a generic instruction does not replace a runtime adapter | Explicit capability limit, not merely a missing test |
| Publication from stored approval | Stored Verify result is compared; candidate reopened and checked | Implemented |
| Safe handling of user changes | Drift is rejected; partial transfer is reported; no automatic semantic merge | Accepted V1 simplification |
| Finite deadlines until the end of Publication | Context is lost in older guard/Git helpers | Confirmed defect R1 |
| Small, coherent components | Good package boundaries; central Run file bundles many lifecycle/communication states | Quality risk, no evidence of a general architectural failure |

### Independent evidence of the preserved foundation

The following Git tree identities match between the starting state `447ac3b152de79b52de80513e14dc94349a060bd` and the reviewed Main. This evidences the same respective file tree, not merely an asserted semantic equivalence.

| Directory | Identical Git tree |
|---|---|
| `internal/model` | `ded9b19b4e7572b2fe04adebf4e9ed0094943fa8` |
| `internal/snapshot` | `c2d6a7249fc4c6244dc105f1a04a7232ba593e9a` |
| `internal/organization` | `06731ff5babb4dd33620780392919e82305665f0` |
| `internal/realization` | `6f65fddf872ec3c1d35fb910ba782990c18d9204` |

This does not assert an unchanged runtime or an unchanged canonical self-model. Their adaptation to the new architecture was explicitly intended. Sources: [S01], [S02].

## 4. Concrete deviations

### A1 – Siblings and distant Managers can talk to each other directly

**Target:** The original decisions require communication along the hierarchy. The v0.17 plan names sibling coordination via shared parents and also keeps required communication hierarchical in the Notes section.

**Actual:** `run/inbox.go: ask` checks the existence of the recipient, a sender different from the recipient, and non-empty text. A parent/child relationship is not checked. `codexappserver/tools.go` correspondingly describes the tool as a question to another Run Manager. By contrast, delegation and review actually enforce the direct parent-child relationship. [S03], [S04], [S05]

**Consequence:** The tree is binding for tasks and integration, but for questions only a selection of possible addresses. A sibling can ask the other directly; a grandchild can skip levels. This is not external communication with the owner, but it is a different internal organization rule.

**Why:** The implementation permits this explicitly. A documented owner approval for the relaxation was not found in the reviewed specifications. The asynchronous Ask/Await/Reply protocol is justified by the demonstrated provider limits. It does not follow from this, however, that all Managers must be directly reachable: hierarchical neighbors, too, can talk to each other asynchronously.

**Recommendation:** Derive the permitted neighbor relationships from the existing topology and test them, or have the owner explicitly accept the relaxed rule. No new general ACL/routing platform. The counter-check example must cover two siblings as well as Root/grandchild, not only Root and direct child.

### A2 – Invalid child states are held back before the parent handoff

**Target:** §6.2 of the v0.17 plan requires, for every child handoff, the diagnosis to the parent, with correction and coordination within the Run. The final Root state remains a hard gate.

**Actual:** After `submitted`, `execution.work` creates a snapshot and runs `x.check`. On `ErrCandidateInvalid`, the host sends findings directly in a new message of the same child and repeats the loop. A normal handoff reaches the parent only after a successful check. The handoff structure contains no separate report of this earlier failed check. [S05], [S06]

**Consequence:** Local self-correction is possible, but the intended automatic flow of information to the parent is missing. The child can, it is true, itself ask questions or deliver a Reject; it is not cut off from every kind of communication. The host nevertheless decides that an invalid Submitted result initially stays with the child. Cross-area causes can thereby reach the responsible party later.

**Why:** This procedure protects the property “a normally submitted Implement handoff is technically valid”. That is a plausible technical motivation, but not the same distribution of responsibility as “handoff plus diagnosis to the parent”. The documented failure in which a complete project test ran in an incomplete child copy shows that this boundary is practically relevant. [S07]

**Recommendation:** Decide explicitly before a blanket concept acceptance. An alignment need not permit invalid merging: the parent can see the diagnosis and the state while successful integration and Root completion remain blocked. Do not introduce a new multi-stage approval protocol.

### A3 – Domain completion still depends heavily on the end of technical messages

**Target:** Runs and Manager assignments, no domain turn machine and no report-to-continue. A small explicit semantic decision is permitted; it should not become a ritual for ordinary communication.

**Actual:** Every active `Exchange` continuation requires a fresh decision. In the Codex adapter the previous `judgment` is reset per message. If a semantic result is missing after the message, `work` ends the assignment with an error. The general Harness rule explicitly requires the new decision even when an active continuation only handles outstanding questions/answers. Only messages after an already fixed handoff receive the special treatment of answer without new judgment. [S05], [S08], [S09]

The short description of the `judge` tool at the same time says generally that ordinary question/answer messages need no new judgment. The difference between “active continuation” and “late message after a fixed handoff” is not made there as precisely as in the general rules. This is at least an avoidable ambiguity of the delivered protocol. [S04]

**Consequence:** The agent must correctly distinguish the technical phases of the host. It can answer correctly in domain terms and still count as technically failed. Several historical failed runs concern exactly old tasks, unexpected waiting, review decisions that were already registered, and missing repeated judgments. [S07]

**Why:** The rules are meant to prevent stale approvals and to synchronize provider messages with fixed Git handoffs. The background is legitimate. The stabilization so far has, however, built up additional instructions and state distinctions. This is not a return of the old productive turn engine, but it is a conceptual tension with the desired simple collaboration.

**Recommendation:** Assess assignment, message and validity of the last domain completion separately. A task that has actually changed, a changed candidate or new mandatory findings must not inherit an old approval. A purely communicative answer, on the other hand, should not have to formulate the same approval again unnecessarily. First examine counterexamples and current state transitions, then simplify in a targeted way; do not remove `judge` blindly, and do not use natural prose as proof of success.

A small additional note on avoidable protocol work: native findings must contain a `manager` field, although in the normal work/review path the host afterwards enters the actual Manager itself. This is not a main blocker, but a good probe for the principle “do not ask the model again for what is technically known”. [S04], [S05]

### A4 – Harness as native instruction injection instead of a fully modular working environment

**Target:** Deterministic five-part composition, short task prompts, controlled native use, compact map and targeted context access. Originally AGENTS.md/CLAUDE.md, skills, rules, scripts and a generic fallback were discussed concretely. v0.17 does not require every native function, but an actually usable prepared environment.

**Actual:** `harness.Document` contains two strings, `Base` and `Developer`. The builder puts together the complete topology, ownership, Manager, mandate, findings, all selected definition bodies/fields/provenance and selected instruction files. The Codex adapter passes these at thread start via native Base/Developer instructions. The actual work message, by contrast, correctly consists only of task and feedback. MCP tools and private environment are really integrated. [S08], [S09], [S10]

Automatic loading of project instructions is explicitly disabled (`project_doc_max_bytes=0`, empty fallback file names). The private Codex home contains the explicitly provisioned login data; in the reviewed execution path no corresponding generated AGENTS/skill structure arises as the primary source of knowledge. An executable generic provider fallback does not exist. [S11]

**Assessment:** This is not a “missing Harness” and not a mere dummy. Native instruction parameters are real provider mechanisms, and the central rules remain separate from task messages. Disabling uncontrolled repository instructions is understandable as a measure for priority/determinism.

Nevertheless, the original idea “map and retrievable working materials instead of a large permanent context guide” is only partially implemented. The necessary statically selected context is injected completely at the beginning; for model context there is no reloadable read operation in the Run toolset. With a larger model this is a question of scaling and clarity, not an already measured general loss of quality.

**Recommendation:** Document and accept the actual replacement explicitly. Add modularity and targeted reloading only where concrete scale/usability evidence shows their benefit. No obligation to build in every provider feature; but also no claim that the file/skill/fallback variant discussed originally has already been completely implemented.

### A5 – Workflow checks internal resource names of the Run

The actual Apply sequence is small and correct in domain terms. `workflow.ValidateClosed` additionally checks `session:<package>`, `copy:<package>`, `manager-workers` and `git-service`. The Workflow thereby has to know technical closure names, although the Executor already owns its closed return contract. Apply in turn uses this function to check stored approval. [S12], [S13]

The independent check of persisted results is sensible and must remain. Its knowledge of the technical resources should, however, sit in a jointly used Run result validation instead of coupling the Workflow layer to these names. This is a limited architectural leak, not a second orchestration and no reason for a new framework.

### A6 – Universal for today's mandates, not already for arbitrary new assignments

`Request` and `harness.Build` accept only Assessment, Implement and Verify. The Root assignment is fixed in the Executor. A further Workflow can combine these Runs; a new mandate type requires changes to Core/Harness checks. [S06], [S08]

For the narrow V1 this is defensible: a general Workflow DSL was explicitly deferred. It is, however, narrower than the original long-term picture, in which mandates are largely prepared assignments for the same mechanism. Do not hastily replace the existing direct composition with a universal engine. Merely name the extension point honestly.

## 5. Confirmed defects and concrete reliability risks

### R1 – Publication guards lose the call deadline

The Publisher has a Context. `CaptureV1Files` and `ApplyOrderedChecked`, however, do not accept it. Their Git guards use, among others, `IdentifyGit`, `BranchName` and `GitOutput`; the shared Git start uses `exec.Command` instead of an execution bound to the call. Checks in the outer callbacks cannot abort an inner Git process that is already blocked. [S14], [S15], [S16]

This confirms the new Coordinator's P2. The correction must cover the entire guard path that is actually called, not just rename one function. Partial-write report, branch/identity/file guards and readback are preserved. An isolated abort test must actually reach the blocking Git query; the earlier collective-test timeout does not prove its concrete cause.

No hang of our own was provoked in this audit. A guaranteed immediate termination in case of an arbitrary kernel/filesystem stall would be a different property, not asserted here.

### R2 – The complete Notes retrieval can exceed the tool transport

The store permits records up to 16 MiB and reads project-wide up to 10,000 entries or 16 MiB of text/context in total. `read_notes` has no filter and no pagination and serializes all results into a single tool reply. The agentd tool channel transports one ToolReply per frame; `MaxPayload` is 1 MiB. [S17], [S04], [S18]

Validly persisted Notes can therefore no longer be delivered as a whole well below the storage limit. JSON/MCP embedding reduces the usable space further. This is a statically traceable size incompatibility; a corresponding new native failure was not executed here.

Similar whole-package limits exist for extensive definition contexts and text diffs. The binary patch error that has already been fixed shows the practical relevance of the transport limit, but does not qualify arbitrarily large text/Notes replies. [S19]

**Recommendation:** Write a small controlled boundary test; bind the retrievable amount of data to the actual transport limit. For example bounded pages with a context filter or explicit result references. No silent truncation, no deletion of historical Notes, no new vector database. The optionality of the content must not mean that arbitrarily growing optional history regularly blocks later work.

### R3 – Process complexity sits in a few heavily loaded places

`run/execute.go` contains environment setup, resource end, tool callbacks, delegation states, review/integration, correction loop and check collection. `inbox.go` adds wakeups, question correlation, trailing judgments and abort. Several Boolean states must stay coherent together with mutexes, WaitGroups and channels. [S05], [S06], [S03]

This synchronization is not automatically superfluous. The host has to handle genuinely parallel providers and write copies. A blanket race/deadlock accusation would be impermissible without reproduction. The dominant maintainability risk is nevertheless the combination of domain assignment states with technical reply/drain states.

The first quality goal should be clearer semantics of these states, not a cosmetic distribution of the same complexity over more files.

## 6. Deliberate limitations and missing evidence

### Accepted or disclosed V1 limits

Extract, Cockpit/UI, authoring and Setup were removed from the active scope. That was explicitly approved later and is not an unnoticed loss of function in this audit. Likewise, conflict abort instead of semantic merge for user changes, persistent bundles instead of long-lived services, and MCP instead of a new Code Mode runtime are compatible with the narrow concept.

The absence of per-write protection is explicitly intended. From correct registry accounting it nevertheless does not follow that every acting Manager changed only its own files. The complete diff, task basis and accepted child integration give the domain review the corresponding context. A missing deterministic single-write ACL is not to be counted here as a defect.

### Provider

Claude is not merely untested: `claudecode.Agents.Open` always returns `ErrUnavailable` in the current product code. Codex intentionally does not copy any refresh authority into the Manager environment. Codex-native successes therefore mean neither Claude functionality nor qualified login renewal. [S20], [S21]

A generic AGENTS.md fallback would in any case be only an instruction representation, not an executable unknown provider. That must be kept separate from an asserted provider-independent product usability.

### Qualification

The repository records contain actual native M1 and M2 realization, independent oracles, preservation, Notes use, repair of a deliberately inserted defect and strict no-change. These are substantial successes. The component sequence Verify-Reject → fresh Implement → fresh Verify-Approve was likewise observed natively; the outer controller report remains failed because of a separately diagnosed date comparison error. [S22]

The native proof that the **public Apply Workflow itself** selects this Reject branch and then publishes the candidate repaired from it is separate from this and still open. A normal repair in the initial Implement is no proof of this branch. Likewise, broad repeatability and statements for arbitrary project sizes are missing. This is an evidence limit, not a functional defect proven from it.

## 7. Which problems dominate?

### 7.1 Observed: communication between agent and host

The documented failed runs cluster around completion, questions, tasks that have already been consumed, handoff/review states and the placement of checks. This is a qualitative classification of the reports at hand, not a statistical failure rate. The connection of A2 and A3 is more important than additional product features: a simple domain goal is in part overlaid by several technical communication rules. [S07]

### 7.2 Observed: Git, Windows, container and transport boundaries

Long paths, Git configuration effects, file modes, large binary patches and cleanup under abort have triggered real corrections. These are predominantly infrastructure/lifecycle problems, no evidence that the model-based organization is unsuitable in domain terms. The current R1 and the scaling R2 show that composite technical boundaries continue to matter. [S19], [S14], [S15], [S16], [S17], [S18]

### 7.3 Conceptual: Documented target rule and implemented contract drift apart

Code can match the updated repository architecture text even though the latter has meanwhile expressed an originally stricter owner resolution less clearly. “Communication within the Run” is not the same as “communication along the hierarchy”. “Error repairable in the child” is not the same as “diagnosis to the parent”. A comparison against the original decisions is therefore necessary, not only against current package tables.

### 7.4 Long term: scope and usability

Large up-front contexts, complete Notes retrievals and extensive procedural notes do not automatically scale as well as the cleanly separated Go packages. Current Library successes do not answer this question. This is a justified next measurement point, not a call for a large optimization wave.

## 8. Prioritized next steps

1. **Close R1 completely:** Context through to all Publication-guard Git calls actually used; observed abort regression; no lost partial paths.
2. **Cross-check A1/A2 against the owner's intent:** Demonstrate the communication graph and the information flow for an invalid child submission concretely. Then align, or record an explicit deliberate deviation. Do not interpret current docs alone as approval.
3. **Simplify A3 with a few counterexamples:** Answer without new domain work, active state with a new mandatory finding, already fixed handoff and late question. No reuse of stale approval; no unnecessary new decision for mere communication.
4. **Reproduce and limit R2 in a controlled way:** Stored Notes below the storage limit but above the tool frame; a suitable bounded read interface instead of larger blanket limits.
5. **Decide A4/A5/A6 as scope and quality topics:** Name native instruction injection clearly, keep Workflow result validation closer to the Run contract, modularize only demonstrated context needs. No general Workflow/skill platform in advance.
6. **Prove the public native correction branch in a targeted way:** On a fixed corrected state, without reinterpreting successful old evidence. Afterwards a bounded Codex acceptance with a clear provider and runtime matrix.

A new complete rebuild is not justified by this review. Equally, the statement “only the Git timeout remains, otherwise everything as designed” would be too sweeping. The right next work is a few verifiable contract slices.

## 9. Sources – bound to the reviewed commit

The following links contain the fixed commit, not a moving Main reference. Function names in the text are the specifically reviewed places.

[S01]: https://api.github.com/repos/Glacius-Labs/Markitect/git/trees/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998:internal
[S02]: https://api.github.com/repos/Glacius-Labs/Markitect/git/trees/447ac3b152de79b52de80513e14dc94349a060bd:internal
[S03]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/run/inbox.go
[S04]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/tools.go
[S05]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/run/execute.go
[S06]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/run/contracts.go
[S07]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/docs/implementation-plan.md#v1-implementation-organization
[S08]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/harness/harness.go
[S09]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/agents.go
[S10]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/project/prepared.go
[S11]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/profile.go
[S12]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/workflow/apply.go
[S13]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/apply/apply.go
[S14]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/gitrepo/publication.go
[S15]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/tooling/guardedwrite/guardedwrite.go
[S16]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/tooling/source/git_snapshot.go
[S17]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/runstore/store.go
[S18]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/agentd/frame/frame.go
[S19]: https://github.com/Glacius-Labs/Markitect/pull/667
[S20]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/claudecode/agents.go
[S21]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/agents_credential.go
[S22]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/docs/implementation-plan.md#current-qualification-evidence
