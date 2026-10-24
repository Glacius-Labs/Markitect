# Markitect V1 concept

**Model-based realization through a derived Manager organization**

| | |
|---|---|
| **Version** | 1.0 – recorded concept |
| **Date** | 24 October 2026 |
| **Scope** | Narrow V1: Model → Artifacts |
| **Basis** | The owner's concept [v0.18](sources/v1-concept-v018-20261024.md) and decisions D10–D18 accepted on 24 October ([decision record](sources/v1-concept-decisions-20261024.md), [DEC-068](register.md#dec-068-v1-concept-10)); English text per [DEC-069](register.md#dec-069-documentation-is-written-in-english) |

> **The canonical model describes the intended project world. The Manager organization is derived from it. The Managers realize, review and integrate that world; the host provides a few clear technical guarantees for it.**

This version condenses v0.18 and closes its open items with decisions. The decision record governs D10–D18; the owner's v0.18 governs all other content. Any other divergence in this version is an editing error and is reported to the coordinator. The document describes the target. The [delivery plan](../implementation-plan.md) tracks implementation status and known deviations of the code. It grants no implementation, release, deployment or acceptance approval.

## Reading rules

| Category | Meaning |
|---|---|
| **Decided** | Accepted product semantics or a responsibility boundary that follows directly from it. This is the normal case in this document. |
| **Technical design** | The goal is fixed; API, storage form, provider mapping or package cut need a reasoned implementation. Collected in section 28. |
| **Later** | Deliberately not part of V1. Must not shape the core through premature compatibility obligations. |

"Must" denotes a domain or technical invariant, not a prescribed file. "For example" names a suitable form, not a mandatory technology. Owner decisions determine product semantics; an architecture proposal, an audit or a green test does not change them silently. Current code does not become the target merely because it is implemented or passes its tests, and explicitly rejected earlier requirements do not return.

---

## 1. Product idea

### 1.1 The problem is not a lack of text production

AI agents produce code, tests, documentation and working instructions quickly. The hard part is keeping their meaning consistent over time. An architecture decision often affects code, interfaces, tests, configuration, documentation and agent rules at once; a change that succeeds locally can break obligations elsewhere.

The starting point was growing, increasingly contradictory collections of rules, skills and workflows. Maintaining text alone and repeating complete AI reviews were not a sufficient basis for consistency.

### 1.2 One canonical model instead of competing truths

Markitect represents the intended project world in a canonical model: domain requirements, architecture, relationships, rules, responsibilities and quality obligations. Code, tests, documentation and configuration are realizations of that intent.

```text
Intent and architecture in the canonical model
    → deterministically checked structure
    → Manager organization derived from it
    → realization and integration
    → independent verification
    → artifacts published under control
```

The canonical language uses typed YAML Definitions with meaningful Markdown prose. The existing language reference governs its syntax.

### 1.3 Compiler and agents complement each other

The compiler checks expressible structural properties. The Managers judge semantic questions and realize the model. A language model should not guess trivial reference errors, and the host should not force domain work into a deterministic process. The analogy to typed languages extends to early detection of expressible structural errors; it does not prove that natural-language requirements are realized correctly.

### 1.4 Product thesis and limits of evidence

An authoritative model, bounded responsibility, hierarchical reviews, independent verification Runs and controlled integration should make consistent changes more reliable than a single agent that must keep every obligation in view alone. This is a thesis to be tested. Reliability, less human review effort and advantages over simpler agent approaches must be shown through comparisons and real projects.

## 2. V1 scope

### 2.1 Deliverable

V1 delivers **Model → Artifacts**: deterministic preparation, Assessment, Implement, Verify, fresh correction Runs, durable results and controlled publication. V1 is operated through a narrow Apply entry point and the model tools Check, Context and Docs.

It includes registry and ownership maintenance, a harness that works in practice, normal CLI account authentication, Run-local containers and Git working copies, persistent Shared Notes and the necessary technical and domain evidence.

### 2.2 Not commissioned

Extract and other Artifacts→Model operations, the Cockpit or any graphical UI, interactive model authoring and the migration of these old consumers are not part of V1. Their earlier interfaces impose no requirements on the execution core.

Setup is not part of V1. Tested explicit configuration examples make the Apply call possible; no separate onboarding product is built. Ordinary reading of existing artifacts remains necessary. What is excluded is turning them into a new canonical truth during Apply.

### 2.3 Provider scope (D18)

V1 is accepted with **Codex** and normal account login. Claude is the next provider mapping. It is built on the stabilized session and tool contract (sections 13 and 14) and qualified on its own. Until then the product carries no non-functional Claude path.

The approach stays provider-neutral. Every further provider needs its own adapter and its own evidence; a successful Codex run proves nothing about another provider.

### 2.4 Later

Further commands and user-defined workflows should be able to combine Runs in the future. This requires the independent Run boundary, but no workflow DSL in V1. Longer-term Manager memory, semantic search, automatic knowledge condensation, a graphical workflow designer, dynamic organization, long-lived agent containers and a custom code-mode runtime are not V1 requirements.

## 3. Quality principles

### 3.1 Architectural quality before reuse

The target architecture determines the implementation. Old types, package boundaries and runtime paths are not requirements; historical work supplies insights and regression expectations. A rewrite is not an end in itself either: coherent foundations are kept, and a newly written, over-abstracted system would miss the goal just as much.

### 3.2 Pragmatic determinism

Facts that are technically unambiguous and relevant are established deterministically: identities, references, fixed Git states, file accounting, actual check results and resource closure. This creates no duty to classify, approve or accompany every work step with evidence protocols. The benefit of a control must justify its complexity.

### 3.3 Manager autonomy with clear responsibility

Managers work independently in their area, plan locally, coordinate their children and assess their results. The host supports this organization. It makes no domain architecture decisions and does not replace the parent with a repair or approval machine.

### 3.4 Simplicity and cohesion

Few clear responsibilities, fitting dependency directions and readable code are acceptance criteria. More packages, interfaces or status values do not mean more quality. A single large orchestrator is not a simplification either.

### 3.5 Focus and preservation

An assignment covers the valid model, actual Findings and necessary consequences. Conforming artifacts are preserved. Optional improvements are noted, not implemented unasked. An unchanged, conforming candidate is a valid result.

### 3.6 Honest results

Domain approval, technical success, stored results, working provider integration and owner acceptance are different statements. Uncertain or incomplete states are not reinterpreted as success. Later successes do not change earlier failures.

### 3.7 Tasks, not process (D13)

Managers receive their role, organization rules, context, task and answer format – not the host's process control. The host manages admission, wake-ups, drain, completion and similar states itself. When it refuses an action, the tool result names the concrete reason and the possible next steps.

A recurring failure in the interaction between agent and host is fixed in host state or in the tool contract, not with additional prompt text.

## 4. Terms

| Term | Meaning |
|---|---|
| **Canonical model** | Authoritative description of the intended project world. |
| **Package / Definition** | Organizational units and typed statements of the canonical language. |
| **Artifact** | Concrete realization, for example code, test, documentation, build or configuration file. |
| **Manager** | Agent with a responsibility derived from the model. |
| **Assignment** | Domain work of one Manager within a Run. Not the same as a message. |
| **Run** | Independently executable mandate through the complete Manager organization. |
| **Mandate** | Task and expected result of a Run; in V1 Assessment, Implement or Verify. |
| **Workflow** | Composition of Runs, their inputs, results and transitions. |
| **Command** | Named application operation including preconditions, overall budget and, where needed, a workflow. |
| **Harness** | Prepared working environment specific to Manager, mandate and provider. |
| **Workspace** | Mutable, private working copy of a Manager during a Run. |
| **Handoff** | Fixed work or review state of a Manager with the context its review needs. |
| **Candidate** | Fixed, durably referenceable Git realization state. |
| **Judgment** | A Manager's domain decision about its own assignment: approval, rejection or model blocker; in Implement, the submission of a candidate. Implement never approves itself. |
| **Review** | A parent's decision about the fixed handoff of a direct child. |
| **Finding** | A statement or correction required for the current mandate. |
| **Shared Note** | Persistent, non-canonical observation without automatic effect on assignments. |
| **Publication** | Controlled transfer of an accepted candidate into the user's repository. No automatic commit or push. |

**A Run is not a workflow. A command can contain a workflow. A technical message is neither a Run nor automatically a new assignment.**

## 5. Canonical model and deterministic preparation

### 5.1 Meaning of the model

The model can describe domain concepts, use cases, architecture boundaries, rules, relationships and development obligations. Identities, kinds, fields, imports and references follow the existing language; free prose complements the typed structure where meaning cannot sensibly be reduced to decidable fields.

This concept defines no additional model grammar and no new mandatory primitives. A project convention such as "one use case, exactly one handler" can be modeled without being built into the core.

### 5.2 Model authority

An Apply works against a fixed, successfully checked model basis and its controlled inputs. Realization agents do not silently change requirements, ownership rules or ignore exceptions to remove a conflict. A necessary material model decision becomes visible as a blocker, is settled outside the Apply and is realized again only on a newly checked model basis.

### 5.3 Initial gate

Before the agentic workflow, the model basis and relevant project structures are checked deterministically. With an invalid or incompletely captured starting state no Manager starts; the actual diagnostics are returned. Project preparation connects the existing compiler with registry, accounting, organization derivation and context selection. No second compiler arises inside the agent runtime.

### 5.4 Limits of the check

Structural validity is not semantic completeness. A correctly registered file is not automatically right in domain terms; a valid Definition does not prove that its behavior is implemented. Artifacts not yet produced are no model error in a fresh realization – Assessment distinguishes missing realization from a missing model decision. Existing unexplained files remain an accounting problem of the starting state.

### 5.5 Protected foundation and self-model

The semantics of language, compiler, snapshot values, organization derivation and realization accounting stay protected; structural improvements must not change these guarantees in passing. Markitect's own canonical self-model describes the architecture actually chosen. Maintaining it is not a change of the model language.

## 6. Registry, ownership and complete file accounting

### 6.1 Local shards and global view

Each responsible Manager maintains the registry shard of its Package. It maps its artifacts to the relevant Definitions and gives a reason for each mapping. An artifact can realize several Definitions but has exactly one responsibility.

The global registry is derived deterministically from the shards, not maintained as a second truth. When a child's result is integrated, its registration stays in the child's shard.

### 6.2 Complete accounting

For every relevant observed file it is unambiguous which category it belongs to.

| Category | Treatment |
|---|---|
| Realization artifact | Valid registration and responsibility required. |
| Canonical model input | Authoritative project intent, protected during Apply. |
| Markitect control input | Controlled input, for example execution or ignore rules. |
| Operational state | Technically separated from the product candidate, not artificially registered. |
| Explicitly ignored file | Explicit exception, not a hidden way around a model obligation. |

`.markitect/ignore.yaml` governs; `.gitignore` does not replace this accounting. Canonical model and registry files do not disappear from validation through a blanket directory exception.

### 6.3 Registry maintenance is realization, project intent is not

Managers register new artifacts, update mappings when artifacts move and clean them up on permitted deletions. That is part of realization. Requirements, ownership policy and ignore rules, by contrast, must not be changed to get around a check. "Model unchanged" does not mean "registry frozen".

### 6.4 Accounting is not authorship

Accounting says to whom a file is assigned and which Definitions it addresses – not who changed it. Handoffs therefore contain the actual task base, the fixed result state and the complete diff.

Review and Verify take accepted child integrations into account; these are not unauthorized work by the parent. Child files are not removed wholesale from the parent's diff, though, or later edits of the parent's own would become invisible. Git bases and diffs are enough for this. A per-write ACL system or a global provenance ledger is not intended.

## 7. Model-derived Manager organization

### 7.1 The model is the architecture

The organization is derived from the valid model. There is no second, manually maintained Manager hierarchy. It comprises Root and every local Package Manager, including empty and intermediate Packages. The existing rules for external model dependencies are not extended in passing into an additional executable organization.

### 7.2 Complete participation in every mandate

Assessment, Implement and Verify use the same complete organization; the mandate changes the task, not the tree of responsibility. Root-only is correct only when the model contains only Root. Participation means actually handling the responsibility, not just a started container.

### 7.3 Responsibility of Root and local Managers

A local Manager realizes or reviews its own obligations. A parent is additionally responsible for delegating to its direct children, aligning their interfaces, reviewing them and integrating upward.

Root forms the integrated overall result. It does not routinely take over its children's domain work, does not skip their responsibilities and does not replace required child decisions with its own approval.

### 7.4 No second delegation organization

Native provider sub-agents must not bypass the modeled organization. In the V1 product path, delegation runs through Markitect's Manager structure. The chats that develop Markitect may use sub-agents within their assignments; development organization and product runtime are separate levels.

## 8. Collaboration within a Run

### 8.1 Local planning and shared prerequisites

Each Manager plans within its assignment. Parents settle shared interfaces and prerequisites before dependent child assignments start. There is no global planning Run and no host that plans the domain work.

Independent children work in parallel. If a child needs a sibling's integrated result, the common parent coordinates the order. A private copy that has already started does not receive later changes of its parent or siblings live, only at clear handoff boundaries, for example with a correction delegation.

### 8.2 Communication along the hierarchy (D10)

Delegation, questions, answers, alignment and escalation follow the tree. A Manager asks only its direct parent or its direct children; answers go back to the asker. Siblings align through their common parent, who asks on or decides; alignment across several levels runs step by step through the Managers in between. Root asks no one outside the Run.

The host derives the permitted relationships from the topology and refuses other addressees with a reason. There is no separate ACL or routing language.

Answers can arrive asynchronously. A waiting Manager handles incoming questions instead of blocking its partner. The host transports and correlates messages; it does not answer domain questions.

### 8.3 Handoff, review and integration

A child delivers a fixed state and the information needed to assess it. The parent reviews the actual handoff – the diff, not a success story – and accepts it, requests a correction or escalates a model blocker.

Accepted results are integrated upward as soon as they are available, without waiting for all siblings. Dependencies or conflicts can justify a different order; the responsible Manager coordinates it, not a global host planning framework.

### 8.4 Faulty handoffs are the parent's responsibility (D11)

In Implement, every submitted child candidate is fixed and handed to the parent together with the result of the deterministic check – also when the check fails. The host does not silently send an invalid submission back to the child.

From the diagnostic the parent sees whether a local correction, an interface clarification or a different integration order is needed. It rejects the submission with Findings, settles the interface or escalates. It can neither accept nor integrate an invalid candidate.

Local self-correction remains possible: a child can run the same deterministic check on its current state through a tool before submitting (section 14). This is an offer, not a mandatory step. Root has no parent; the host returns Root's diagnostic directly to Root (the checkpoint before Root completion).

In Assessment and Verify a child handoff is a judgment about the fixed input candidate. Helper files in the private copy are not adopted. A check result about the input candidate informs the judgment and is no integration gate; a reasoned rejection of an invalid candidate is a regular result.

### 8.5 Correction within a Run and between Runs

A return by the parent during Implement stays within the same Run. The child keeps its conversation and working state and receives the Findings and the parent's current base.

A rejection by the independent Verify, by contrast, leads to a new Implement Run; earlier Implement conversations are not continued. The two kinds of correction are kept apart technically and in reports.

## 9. Run

### 9.1 Independent call boundary

A Run executes exactly one mandate through the Manager organization. It can be executed and tested independently of Apply; Assessment or Verify can be examined in isolation without separate orchestration paths.

Its inputs are a checked model and project binding, the derived organization, a fixed starting candidate, the mandate, optional mandatory Findings, execution and harness configuration and finite resources.

### 9.2 Responsibility of the Run

The Run is responsible for building its environment, session and workspace lifetime, hierarchical collaboration, handoffs, integration, the checkpoints within the Run, fixing the result, securing the candidate, cleanup and storing its final result.

It does not choose the next Run and publishes nothing into the user's repository. Its successful return does not depend on a caller finishing closing work, also in standalone use.

### 9.3 Lifecycle

```text
Valid bound Run inputs
    → prepare the Run environment and Manager harnesses
    → start the Run-local Git Workspace Service and the Managers
    → work the assignment through the complete organization
    → fix the actual handoffs and the overall result
    → secure the candidate durably and make it independently reopenable
    → close all Run resources under control
    → store the technical and domain outcome and read it back
    → return the closed result to the caller
```

This is a host responsibility, not an evidence protocol for the agent to fill in. One executor with clear helper responsibilities is enough; a cascade of lifecycle services is not intended.

### 9.4 Reachability within the Run

The end of a provider response does not end the Manager session. A Manager that has submitted stays reachable for questions and corrections within the same Run; its fixed handoff and later communication remain distinguishable. Only the end of the Run closes the environment. New Runs get new conversations and working environments.

## 10. Command, workflow and Apply

### 10.1 Command

The command provides the named product operation. It binds inputs, triggers the initial check, owns the overall budget, calls the workflow and is responsible for publication and the overall result. The CLI and later interfaces use the same application operation; the CLI parser contains no domain logic. The command implements neither Manager communication nor Git merges.

### 10.2 Workflow

A workflow describes the order, inputs and result handovers of Runs, branches, repetitions, success and abort. It stays understandable and composable later. It knows no containers, provider processes or working copies and works only with closed Run results.

### 10.3 Validity of closed results (D15)

Whether a Run result is complete, bound and cleanly closed – binding, technical completion, closure of all Run resources, a decision valid for the mandate and mandatory Findings – is checked by exactly one validation. It belongs to the Run contract. Workflow and publication use it without knowing the Run's internal resource names.

### 10.4 Apply flow

```text
Initial model and structure check
    ├─ invalid / incomplete → end without starting agents
    └─ valid
         → Assessment
             ├─ Reject / model blocker → end without publication
             └─ Approve
                  → Implement
                      ├─ not completed / model blocker → end
                      └─ valid finished candidate
                           → Verify
                               ├─ realization Reject
                               │    → new Implement with Findings
                               │    → new Verify
                               ├─ model blocker / technical abort → end
                               └─ Approve
                                    → final checks and guards
                                    → controlled publication
```

All iterations share the original command budget. A new Run ID resets neither time nor message resources. The loop runs until approval or an honest abort within finite limits.

### 10.5 Extensibility (D16)

V1 knows exactly three mandates. The Run call boundary can be reused by further commands. New mandates and their result or capability requirements arise only from an actual later command. V1 needs neither a DSL nor a graphical workflow system.

## 11. Mandates

### 11.1 Assessment

Assessment examines the semantic readiness of the structurally valid model. Each Manager looks at its area and relevant interfaces. Material contradictions, necessary open decisions and ambiguities that cannot be resolved are returned with Findings.

Investigations, commands and, for example, a reasoned build attempt are allowed. Assessment implements no realization and does not change the project intent; helper files in private copies are not adopted. Missing or outdated artifacts are implementation needs, not a reason for rejection. Approve means "a sufficient model basis for realization", not "the product is correct", and does not skip Implement.

### 11.2 Implement

Implement reconciles the existing realization with the model, required checks and actual Findings. Managers change the necessary artifacts within their responsibility, maintain their registry, delegate and integrate reviewed child results.

Necessary follow-up changes are part of it. Extra test ideas, cosmetic changes or refactorings unrelated to applicable obligations are not made just to produce a diff. Implement delivers a fixed integrated candidate and does not approve its own publication. If no changes are needed, submitting the unchanged candidate is valid after complete delegation, checking and review.

### 11.3 Verify

Verify is a fresh Run over exactly the candidate under review. The complete organization independently assesses model conformance, domain behavior, interfaces, integration, relevant tests and realization accounting.

Verify does not repair the candidate; its own investigations and check files stay local. The result is Approve, Reject with the necessary realization Findings, or a model blocker. A Reject is a regular verification result, not a technical failure.

### 11.4 Corrections after Verify

A realization Reject hands the model binding, the last candidate and the actual Findings to a new Implement Run, followed by a new Verify. The old Implement organization is not continued, and the new verifier does not receive its self-assessment as authority. A material model contradiction discovered late ends the command without publication; the model question is settled outside.

## 12. Checking strategy and four checkpoints

### 12.1 One shared checking mechanism

The same compiler, registry and accounting mechanisms are applied at different points to suitable fixed states; there are no four diverging validators. Cheap structural checks run often. Not every isolated child copy must run the complete project test suite: the origin of a checking obligation and the sensible place to run a test are different properties.

### 12.2 The four checkpoints

| Checkpoint | Check and consequence |
|---|---|
| **Before the workflow starts** | Valid model basis and applicable project structure. On failure, abort before the agentic workflow and before Run resources. |
| **At child handoffs** | Fixed handoff state and accounting. In Implement the diagnostic always travels with the handoff to the parent, and an invalid candidate cannot be integrated (D11). |
| **Before successful Root completion** | Integrated overall state, relevant checks and bindings. No successful Implement and no Approve with unresolved technical invalidity. |
| **Before publication** | Exactly the accepted candidate, required checks and actual source and target guards. No impermissible transfer. |

### 12.3 Intermediate states and checking mandates

Intermediate states can be incomplete while other children are still working; diagnostics make that context visible. The host does not turn temporary incompleteness into a command abort. Verify may reject a faulty candidate as a regular result and need not change it to make its own completion check green. The validity conditions stay binding for an approval.

### 12.4 No check rituals

The host establishes technical results and makes them available to the Managers. Managers do not retype known identities and produce no check receipts, disputes or evidence matrices in order to be allowed to continue ordinary work.

## 13. Assignment, message and judgment

### 13.1 The basic separation

A Manager has an assignment. Messages, questions and answers serve to fulfill it; they are not new assignments. The end of a provider response is not the domain completion of the Manager or the Run. Free closing prose is no technical proof of success. The Manager records the domain result through a small explicit decision, the judgment.

### 13.2 The judgment belongs to the assignment (D12)

A judgment applies to the current state of the assignment – task and mandatory Findings – and to the assessed state: in Implement the Manager's own working state at the time of the judgment, in Assessment and Verify the fixed input candidate. It stays valid until one of these changes. It becomes invalid through:

- a rejection or correction delegation by the parent with Findings,
- new mandatory Findings that affect the assignment,
- a change of the assessed state after the judgment,
- a late mandatory Finding, Reject or model blocker from a child about an already accepted handoff.

A message that only handles questions or answers requires no new judgment; a valid judgment stays in force. If a provider response ends while an active assignment has no valid judgment, the host makes the concrete reason visible, for example "your working state changed after your judgment" or "the review of child X is missing". This is a diagnostic by the host, not a general prompt rule; its form is a technical design matter.

For this, the host keeps a small, explicit assignment state per Manager instead of scattered individual flags. Every invalidation follows from a nameable event.

### 13.3 Decisive counterexamples

| Situation | Decided consequence |
|---|---|
| A question is answered | No new assignment and no new judgment. |
| An answer the Manager is waiting for is still pending | Incoming questions remain answerable; no deadlock. |
| New mandatory Findings arrive | The previous judgment becomes invalid; a new one is needed. |
| The assessed state changes | Approval of the old state does not apply to the new one. |
| A fixed handoff has been made | Later communication does not change the reviewed state. |
| A late finding refutes an assumption | A late Reject or model blocker with Findings stays in force. The parent's acceptance of that handoff and every judgment above it that rests on it become invalid and must be made again before Root can complete successfully. A model blocker prevents successful completion. |

## 14. Manager tools (D13)

### 14.1 A fixed small tool set

| Tool | Purpose |
|---|---|
| `delegate` | A parent gives a direct child a task with Findings; after a review also a correction. |
| `await` | Receive the next event: assignment, question, answer or fixed child handoff with its check result. |
| `review` | A parent decides on a direct child's fixed handoff: accept, reject or model blocker. |
| `ask` / `reply` | Questions and answers along the hierarchy, asynchronously. |
| `judge` | Record the Manager's own judgment about its assignment, with Findings. |
| `check` | Run the deterministic check of the Manager's own current state, for an optional self-check before submitting. |
| `read_notes` / `append_note` | Read Shared Notes page by page and append observations. |

Each domain purpose has exactly one tool.

### 14.2 Principles

- The host does not ask for what it knows: the actual Manager, Run, mandate and the Manager's own commits. Findings need no sender field. An unambiguous reference to the reviewed state stays allowed in reviews, so that no stale approval arises.
- Tools do not drive the host through receipts. A refusal names the concrete reason and what is possible (section 3.7).
- The semantics of each tool has exactly one implementation in Run. Provider adapters only map it. MCP, CLI or typed APIs are transports, not an architectural principle.

## 15. Harness

### 15.1 Purpose

The harness prepares a Manager for its responsibility like an employee: role, organization chart, responsibility, model context, working rules and tools are reliably available. The assignment message stays short and focused on the current work. A harness is not just a text with a hash: the provider actually started must load it and use it in the container.

### 15.2 Five controlled sources

| Source | Content and responsibility |
|---|---|
| **Markitect core** | Versioned general rules for delegation, collaboration, ownership, registry, reviews, Findings and Notes. Not to be redefined by every project. |
| **Project** | Canonical domain, architecture, policies, quality obligations and explicitly bound project conventions. |
| **Manager** | Its own role, parent, direct children, relevant colleagues, responsibilities, registry shard, interfaces and project navigation. |
| **Run** | Mandate, fixed inputs, relevant Findings, permitted effects and expected result. |
| **Provider** | Implementation through actually supported native instructions, tools and configuration. |

Markitect defines how its organization works; the project model defines which world it realizes. Provider adaptations do not change this meaning. The host's process control belongs in none of these sources (section 3.7).

### 15.3 Reproducibility and identity

The same bound inputs – rule version, model state, Manager, mandate, provider profile, templates and tool definitions – produce the same effective harness configuration. A traceable harness identity shows which instructions and capabilities were provided. Fresh operational names, container identifiers or secrets are not part of it. Deterministic generation does not mean deterministic model behavior.

### 15.4 Context

The Manager receives a compact orientation and the context its responsibility needs. Binding information is neither silently truncated nor hidden. Historical notes and process descriptions are not copied into every start instruction.

### 15.5 V1 form of the harness (D14)

In V1 the harness is provided through the provider's native base and developer instructions. They contain the Markitect rules and the complete statically selected mandatory context. Automatically loaded repository instructions are handled deliberately, for example switched off; project-bound instructions reach the environment only through the controlled harness. This way neither repository files nor provider profiles override the canonical intent or the Markitect rules uncontrolled. Semantically unclear conflicts between instructions stay visible instead of being hidden by a policy framework. If the mandatory context exceeds a limit, preparation fails visibly; nothing is truncated.

Modular files, skills or loadable context are added as soon as a concrete size or usability case shows their benefit. For unknown compatible providers a generic AGENTS.md harness is planned as a best-effort representation of the instructions. It replaces no executable adapter, no Manager communication and no necessary technical capabilities; an unknown agent is not supported merely because this file exists. Whether it belongs to the V1 acceptance scope is recorded with the provider matrix (section 28.2).

### 15.6 Technical boundaries stay outside the instructions

Important invariants are checked independently of the agent's obedience: model binding, registry, fixed Git states, actual checks, resource closure and publication preconditions. This justifies no per-write file approvals and no evidence protocols.

## 16. Providers and authentication

### 16.1 Provider boundary

The provider adapter starts and ends the native runtime, transfers messages, mediates tools and observes actual response and process states. It decides neither the next workflow stage nor model conformance. Native protocol terms such as `turn/start` exist only in the adapter; the core derives no domain Turns, empty-Turn classifiers or Pass machine from them.

### 16.2 Account login and protection of secrets

The V1 product path uses normal CLI account authentication. An API-key product variant is not carried along. Credentials do not belong in candidates, harness texts, Shared Notes or unprotected logs; they are prepared and removed under control within the Manager environment. A fleet of short-lived auth containers is not a standard solution.

Login, expiry and refresh properties need their own evidence per provider. Missing capabilities are recognizable before support is claimed.

## 17. Containers, Git working copies and durable candidates

### 17.1 Lifetime per Run

Each Manager has its own container, conversation, harness and Git working copy per Run. Within the Run they persist across delegation, questions and corrections; they are not reused between Runs. The Git Workspace Service also lives exactly one Run. Later Runs need no volumes or working copies reserved in advance.

### 17.2 Git Workspace Service

The service prepares working copies, provides starting commits, fixes handoffs, supports parent-child integration and the update of child copies and reports technical Git conflicts. It is not an agent, not a general helper and not a place for domain decisions; provider login, project checks and workflow control are other responsibilities.

### 17.3 Isolation and collaboration

Manager copies are independent. Parents integrate results into their own copy after review; changes are not propagated through a shared writable directory. Isolation enables the agreed collaboration; it does not replace the ownership review.

### 17.4 Workspace and candidate are different states

A workspace is mutable and bound to the running execution. A candidate is a fixed Git state with durably available objects. Model identity, commit identity and tree content are different quantities and stay bound appropriately. Persistence does not depend on running containers, temporary volumes or live objects.

### 17.5 Storage form

Self-contained Git bundles with bound metadata are the intended V1 approach; an equivalent simpler form would be justifiable. The guarantee is independent reopening, not the format: a persistence proof removes the temporary working copies and opens exactly the candidate in a new environment from the stored data alone. Verify and publication need no live reference to the producing Run. A Git server platform is not intended.

### 17.6 Checks and remaining resources

Separate short-lived check executions are allowed where they provide necessary isolation; final checks before publication have their own clearly owned resources and do not continue Manager containers. Uncertainty about leftover resources is reported; an unknown leftover container is not deleted wholesale. A general recovery system after a hard host abort is not part of V1.

## 18. Binding results and handovers between Runs

### 18.1 Two separate kinds of result

Binding Run results determine further workflow execution; Shared Notes preserve optional information. Both are persistent but have different authority.

A Run result contains only the necessary bound facts: Run identity and mandate, model and project binding, input candidate, technical completion, domain judgment, result candidate, mandatory Findings and necessary technical observations including actual cleanup problems.

### 18.2 No live state in the result

Sessions, active Manager trees and temporary workspace handles are no precondition for reading or publishing a completed result. A new Run builds its environment from the durable inputs itself. A successful return requires storage and confirmed readability. Failed or incomplete Runs may keep diagnosable data without pretending a successful candidate.

### 18.3 Findings as an actionable handover

A Finding shows which obligation is affected, what was observed, what is expected and which place enables the correction. A small understandable structure is enough. The host determines the origin from the actual Manager or checking mechanism. The next organization receives the necessary Findings explicitly, not hidden in an earlier chat history.

### 18.4 Consistency across Run boundaries

A new Run works on exactly the referenced candidate; approval and candidate match. A new model basis does not make earlier bindings valid. Continuity comes from model, candidates, results and Notes, not from replaying old conversations.

## 19. Shared Notes and independent verification

### 19.1 Shared Notes are part of V1

Managers durably record side findings, improvement suggestions, test gaps and helpful insights. Notes survive Runs and commands within a project. Every Note has understandable content and traceable origin: Run, Manager, affected area and suitable references. Competing writers do not overwrite each other.

### 19.2 No automatic authority

A Note is not a delegation, not a model obligation and not a ticket to be executed automatically. It bypasses neither hierarchy nor review. Mandatory problems must not be moved into Notes to make an approval possible; what counts is their meaning for the mandate, not where the agent filed them.

### 19.3 Bounded retrieval (D17)

Notes are not copied wholesale into every start prompt, but stay readable when needed, also as their number grows over many commands. `read_notes` returns bounded pages in a stable order; every page fits the transport limit. No silent truncation, no data loss, no deletion of old Notes, no vector database or semantic search. Sort key, page size, filter and the handling of individual very large Notes are chosen after a controlled reproduction of the size conflict (section 28.1).

### 19.4 Independence of Verify

Verify receives the valid model, the exact candidate and necessary technical information. It takes over neither implementation conversations nor their self-assessment. Notes provide hints with their origin, but no predetermined truth about the candidate. Hidden expectations of independent external oracles do not reach implementers. Agent-written tests do not by themselves replace the independent domain quality proof.

## 20. Errors, budgets and controlled termination

### 20.1 Domain and technical outcomes

| Outcome | Meaning and consequence |
|---|---|
| **Domain-accepted** | The mandate was judged positively. Usable only together with a valid technical completion. |
| **Realization Reject** | The realization violates requirements; Findings enable a correction. |
| **Model blocker** | The bound intent is materially insufficient or contradictory; no endless repair attempt in the code. |
| **Failed** | An actual error prevents execution. Not a domain Reject. |
| **Incomplete** | A reliable completion cannot be established, for example on timeout, abort or uncertain cleanup. |

The names of status values are an implementation detail; their meanings are not mixed. A positive agent claim does not heal a transport or storage error.

### 20.2 Shared finite resources

The command sets a finite overall budget; new Runs and corrections create no new budget. Each Run consumes from this shared budget without knowing the overarching composition. Per-message limits bound technical calls but do not replace the domain completion contract. Budgets are not raised wholesale to hide a deadlock or protocol error that is not understood.

### 20.3 Deadlines reach the blocking operation

Cancellation and deadlines reach the controlled subprocesses, in particular in Git, guard and publication paths. An outer deadline check is not enough when an inner process keeps running without bound. Cleanup and final persistence get a small separate grace bound to secure diagnosable results. The grace period allows no new domain work and no hidden extension of the assignment's resources.

### 20.4 Honest cleanup

Partially started resources are closed as far as possible. An unclear completion is stored as unclear; a later empty container list does not retroactively turn an earlier failed closure proof into success. Ordinary error paths clean up reliably. A guarantee for arbitrary kernel, file system or host failures is not claimed.

## 21. Controlled publication

### 21.1 Approval of the exact candidate

Publication consumes a durably stored candidate and its stored, bound Verify approval. Implement does not approve itself; an approval of candidate A does not apply to candidate B. Before transfer, bindings and required checks are checked again. All Run resources are closed by then; the publisher starts no Manager coordination.

### 21.2 User changes and conflicts

The target state must match the bound starting point. User changes made in the meantime are not overwritten; with a relevant divergence V1 ends with a clear conflict instead of an invented semantic merge. If a later function produces a merge that differs from the reviewed candidate, it needs its own new review. The last deterministic check does not replace independent domain approval of new content.

### 21.3 Partial transfer and readback

A multi-file transfer is not automatically atomic. On error or abort it stays visible which changes are complete; pending ones do not appear as transferred. The final readback confirms the published content. An error at this boundary is reported neither as success nor as an unchanged state. Blind rollback under possible user drift is not a safe substitute.

### 21.4 No automatic user commit

Publication creates no commit, push, release or deployment in the user's project. Private Git commits that fix candidates and development PRs while building Markitect are separate. A no-change result also goes through the binding and target checks and then publishes no changes.

## 22. Logical target architecture

### 22.1 Responsibilities

| Responsibility | Task | Not part of it |
|---|---|---|
| **Model / project state** | Model compilation, registry, accounting, snapshots, organization and context. | Provider start, workflow or publication. |
| **Project preparation** | Combine explicit inputs into a checked, fixed execution basis. | A second compiler or the old runtime. |
| **Command application** | Initial check, budget, workflow and overall result. | Manager communication or Git merge logic. |
| **Workflow** | Coordinate Runs and their domain transitions. | Containers, provider protocols or resource management. |
| **Run execution** | One mandate through the organization, with lifecycle, result and the validation of closed results. | Choosing the next Run or publishing. |
| **Harness** | Compose rules, context and working tools. | Git integration or workflow decisions. |
| **Workspace / candidate storage** | Private copies and independent durable candidates. | Review decisions or a general helper. |
| **Run information** | Store and provide binding results and separate Notes. | Creating tasks from Notes. |
| **Publication** | Transfer exactly the accepted state under control. | Domain repair or starting agents. |
| **Infrastructure adapters** | Providers, containers, Git, checks, file system. | Project domain. |

A logical responsibility is not automatically a Go package, interface, service or container. The cut follows cohesion, actual consumers and reasoned infrastructure boundaries.

### 22.2 Dependency direction

```text
CLI / other entry point
    → command application
         → project preparation and deterministic foundation
         → workflow
              → Run contract and Run execution
                   → model organization and harness
                   → workspace, session, check and storage contracts
         → publication

Concrete infrastructure adapters implement the required boundaries.
Composition happens at the outer application edge.
```

The compiler knows neither Docker nor Codex nor Claude. The Run does not know the Apply correction loop. The workflow implements no Manager coordination. Provider details stay out of the organization.

### 22.3 Domain, application and infrastructure

Domain, application and infrastructure are a logical orientation, not a duty to split every feature into three file groups. Narrow interfaces arise where a dependency actually has to be separated. No universal repository, event, policy or workflow framework just for the sake of a pattern; old package names are neither preserved automatically nor replaced cosmetically.

## 23. Handling existing code

Model language, compiler semantics and relevant deterministic guarantees are the foundation worth keeping. The historical agentic execution architecture has no protection as existing code. For affected components the choice is Keep, Refactor, Replace or Remove, decided by responsibility, coupling, understandability, testability and required behavior, not by time already invested.

Adapters only for old internal APIs, permanent compatibility layers, competing runtime paths and special cases for excluded consumers are no acceptable default strategy. Guiding question: **Would we design this building block this way if the earlier runtime did not exist?**

A rewrite is not a reflex. The need for correction is measured by demonstrated contract and reliability problems. A large file is a hint of overloaded responsibility, not an automatic defect; spreading misunderstood state complexity over more files improves nothing. First the semantics become clear.

Self-model, registry shards, generated documentation and actual responsibilities are kept consistent together. Active entry documents contain current rules, not a sequence of historical work orders. Historical insights and failures stay archived but do not become binding again by being close to the current plan.

## 24. Development organization

### 24.1 Coordinator and implementation chats

A coordinator is responsible for architecture, dependencies, assignments, integration order and quality control; it does not write competingly in other chats' files. Implementation chats work in their own worktrees and branches with an exact starting commit, unambiguous write responsibility and verifiable acceptance criteria. Shared contracts, registry shards and generated files have exactly one writer at any time; sub-agents do not bypass this boundary.

### 24.2 Parallelism by actual decoupling

At most two writing workstreams at a time. Read-only analysis and independent reviews can run in parallel. Dependent implementation starts only once the shared contracts are clear; no interface hierarchy without consumers. Early vertical proofs test central assumptions before much is built on them.

### 24.3 Review and integration

Substantial changes need an independent complete-diff review of the commit that will actually be integrated. Changes after the review need a new review. After larger integration steps the overall architecture is reviewed. Others' work is neither silently discarded nor overwritten by a reset.

### 24.4 Authority

Routine decisions within approved limits can be delegated. Changes to product semantics, significant architecture conflicts and owner acceptance are handled explicitly. A standing approval exists for useful real proofs; every proof names source, limits, abort, cleanup and actual result. Release and deployment remain separate decisions.

### 24.5 Change of coordinator

Long chats can be archived after a controlled handoff. A handoff names Main, open work, confirmed evidence, limitations and next tasks. A new coordinator takes over the repository and evidence state, not the old conversation history.

## 25. Acceptance

### 25.1 Kinds of evidence

| Kind of evidence | Shows | Does not show |
|---|---|---|
| Static code and architecture review | Responsibilities and recognizable contract violations. | Runtime behavior, freedom from races, provider obedience. |
| Deterministic tests | Behavior of the tested cases on fixed code. | Native integration or freedom from errors. |
| Controlled sessions with real Git | Interplay of executor, candidates, reviews and error paths. | That a real provider serves the same flow reliably. |
| Native component test | One concrete provider capability or Run chain. | Public workflow selection and publication. |
| Public native Apply | The product path on a named model, candidate and runtime profile. | Universal reliability, other providers, arbitrary project sizes. |
| Independent oracle | Behavior of the produced candidate against independent expectations. | Untested properties or migrations. |
| Owner acceptance | Deliberate acceptance of a clearly bounded product scope. | Release or deployment approval. |

### 25.2 Acceptance criteria

| ID | Required evidence |
|---|---|
| **V01** | An invalid model or initial structure prevents the agentic workflow. |
| **V02** | The complete hierarchy works in Assessment, Implement and Verify, including grandchild, intermediate and empty Package. |
| **V03** | Parallel child work; questions only along permitted paths, sibling and level-skipping questions are refused with a reason; crossed questions and abort work. |
| **V04** | Review and integration concern exactly the actual task base and fixed handoff. |
| **V05** | An invalid Implement child candidate reaches the parent with its diagnostic and cannot be integrated; Root cannot complete unresolved technical invalidity as success. |
| **V06** | A Verify Reject produces new Implement and Verify Runs without old conversations and without a reset overall budget. |
| **V07** | Model blockers recognized late end without publication and without endless code repair. |
| **V08** | Candidates can be reopened exactly after the temporary environments are removed. |
| **V09** | Standalone Runs deliver persisted actual final results; storage and cleanup errors stay visible. |
| **V10** | A reproducible harness is used natively: short tasks, mandatory context, normal account authentication, no API-key variant. |
| **V11** | Notes survive Runs and commands, lose no entries under parallel access, do not extend the assignment and are completely readable page by page at the transport limit. |
| **V12** | Fresh Library, M1→M2 with limit 5→3, preservation, strict no-change and independent oracle. |
| **V13** | A seeded defect is detected and independently reviewed again after correction; an ordinary Implement fix and a genuine Verify-Reject branch stay distinguishable. |
| **V14** | Own changes, other areas' responsibility and accepted child integration are reviewed appropriately. |
| **V15** | Publication binds stored approval and candidate, handles drift and reports partial transfer and readback errors correctly. |
| **V16** | Timeout, provider error, budget exhaustion, start failure and uncertain cleanup produce no false successes. |
| **V17** | The counterexamples of section 13.3 are covered as controlled tests; harness and host messages do not explain internal host states (section 3.7). |

### 25.3 Final state and repeatability

Complete local tests, architecture and dependency gates, Linux and Windows compatibility, race checks and approved hosted CI are shown on the cleaned final state. Native evidence names provider, version, runtime, candidate and limits. The number of independent repetitions is set with a reason; repeating until the first success replaces no reliability assessment. Earlier successes do not qualify changed contracts without checking the impact.

### 25.4 Quality is more than a green example

A Library demonstration tests the idea concretely but does not replace the product definition. Acceptance additionally assesses understandability, cohesion, state complexity, extensibility within the target picture and the honest support boundary. Capabilities that are not qualified – further providers, login refresh, hard host aborts, live data migration – are not implicitly promised by the name V1.

## 26. Reference scenarios

### 26.1 Fresh realization

The valid Library model contains obligations, responsibilities and checks but no realization yet. The initial check establishes structural validity, Assessment the feasibility. Implement produces and registers the artifacts, Verify reviews independently, then publication follows. The external oracle assesses the candidate outside the Manager context.

### 26.2 Change from five to three loans

The starting point is a valid realization A1. In model M2 only the loan limit changes from five to three. Markitect produces an adjusted realization A2 and preserves the other obligations. The number of changed files is not fixed in advance; every change is justified by the model, checks or necessary consequences. A migration of existing runtime data would be a separate requirement.

### 26.3 No-change

An unchanged model meets a conforming realization. All mandates and reviews run completely, optional ideas go into Notes, and the result contains no unjustified artifact change.

### 26.4 Faulty child handoff

A child delivers an artifact without a valid registry entry or with an interface problem. The handoff reaches the parent with the actual diagnostic. The parent sees whether local registry maintenance, a shared decision or a different integration order is needed and rejects with Findings or settles the interface. The child repairs locally. Nothing is published because an agent writes "done", and no global repair workflow starts.

### 26.5 Genuine Verify correction loop

Verify finds a realization defect in the Implement candidate and returns Reject with Findings. The public workflow starts a new Implement Run with fresh conversations and then a new Verify; only its approval carries the publication. A repair already made in the first Implement is a different scenario, and a manually chained component test does not prove that the public workflow chooses the right branch.

### 26.6 User change during Apply

The user changes the target repository while the Managers work in isolation. The guards detect the divergence; V1 ends with a recognizable conflict instead of overwriting blindly.

### 26.7 Abort during publication

A Git call blocks while the deadline expires. The deadline reaches the subprocess, no further domain work starts, completed paths are reported correctly and incomplete readbacks do not count as success. The test actually reaches the blocking call.

## 27. Decisions

### 27.1 Basic decisions D1–D9

| ID | Decision | Meaning |
|---|---|---|
| **D1** | Universal Run executor | One independently usable mechanism for the complete organization and all V1 mandates. |
| **D2** | Workflow as Run composition | Lean coordination of Runs; custom commands possible later, no DSL obligation. |
| **D3** | Containers and Workspace Service per Run | Fresh Manager environments and a Git service with the same Run lifetime. |
| **D4** | Four deterministic checkpoints | Initial check, child handoff, Root completion and publication. |
| **D5** | Deterministic harness from five sources | Product, project, Manager, Run and provider stay separate sources of a usable environment. |
| **D6** | Binding results and Shared Notes | Explicit handovers between Runs and durable non-canonical observations in V1. |
| **D7** | Architectural quality before reuse | Domain and technical quality count for acceptance, not only functional tests. |
| **D8** | Selective clean-slate rebuild | Keep the model foundation, replace unsuitable runtime; no repository reset and no continuing rewrite compulsion. |
| **D9** | Coordinated multi-chat development | Coordinator, separate worktrees, clear writers, independent reviews and controlled integration. |

### 27.2 Decisions of 24 October 2026 (D10–D18)

They close the items A1–A6 and R1–R3 left open in v0.18. R1 (deadline in publication) was already implemented.

| ID | Origin | Decision | Section |
|---|---|---|---|
| **D10** | A1 | Questions only to the direct parent and direct children; siblings align through their common parent, several levels step by step through the Managers in between. | 8.2 |
| **D11** | A2 | Every Implement child candidate goes to the parent with its check result; invalid candidates cannot be integrated; optional self-check through a tool. | 8.4 |
| **D12** | A3, R3 | A judgment belongs to the assignment and the assessed state, not to a message; explicit assignment state in the host. | 13.2 |
| **D13** | New | Tasks, not process; a fixed small tool set with exactly one tool per purpose. | 3.7, 14 |
| **D14** | A4 | V1 harness through native base/developer instructions; skills and loadable context only on demonstrated need. | 15.5 |
| **D15** | A5 | The validation of closed Run results belongs to the Run contract. | 10.3 |
| **D16** | A6 | Exactly three mandates in V1; new mandates only with an actual command. | 10.5 |
| **D17** | R2 | Notes page by page and stably ordered, every page within the transport limit, without silent truncation. | 19.3 |
| **D18** | New | V1 acceptance with Codex; Claude as the next provider mapping on the stabilized contract; no non-functional Claude path until then. | 2.3 |

### 27.3 Clarifications compared with early versions

| Discussed earlier | Governing |
|---|---|
| A single agent for Assessment or Verify | The complete model hierarchy in every mandate. |
| Containers per Apply or long-running | Manager containers and Git Workspace Service per Run. |
| A general helper for Git, auth and checks | Separate clear responsibilities; no tiny containers per operation. |
| Preserving Extract/UI as a migration duty | Excluded; no influence on core contracts. |
| Reusing the assignment runtime | Suitability decides; no protection as existing code. |
| A fixed number of new packages | No artificial upper or lower limit. |
| Ignoring `.markitect` wholesale | Canonical and controlled inputs stay visible; operational data is handled separately. |
| Harness necessarily as specific files | Native instruction parameters are the V1 form (D14). |
| Questions to any Manager in the Run | Only along the hierarchy (D10). |
| A new judgment after every active message | A judgment applies until the assignment or the assessed state changes (D12). |
| A new complete rebuild for every audit finding | Improve the existing core in a targeted way. |

## 28. Technical design and limits of evidence

### 28.1 Technical design

| Topic | Fixed goal | Open for implementation |
|---|---|---|
| Assignment state (D12) | Every invalidation follows from a nameable event; no scattered flags. | Concrete states and transitions, form of the host diagnostic for a missing judgment, passing late findings to the parents. |
| `check` tool (D11) | Optional self-check with the same deterministic mechanism as the handoff check. | Scope (structural, project checks), place of execution and cost. |
| Notes retrieval (D17) | Completely readable, bounded, stably ordered. | First reproduce the size conflict under control, then choose the smallest read interface: sort key, page size, cursor form, context filter, handling of very large individual Notes and of storage errors. |
| Result validation (D15) | One validation in the Run contract. | Package cut and call sites. |
| Generic AGENTS.md harness | Best-effort representation of instructions for unknown providers, not an adapter. | Timing and membership in V1 acceptance, recorded with the provider matrix. |
| Self-model | Describes the chosen architecture. | Package cut for project preparation, workflow and harness. |

### 28.2 Limits of evidence and open acceptance decisions

Before owner acceptance, provider scope, runtime limits, repetition evidence and known limitations are recorded explicitly in a provider matrix. Evidence on older contracts does not qualify the contracts changed by D10–D17. The public native Verify-Reject correction branch with subsequent publication must be shown separately. A clean final state can be accepted with limits without promising universal reliability or unimplemented capabilities.

## 29. Sources and versioning

| Source | Role |
|---|---|
| [Owner concept v0.18](sources/v1-concept-v018-20261024.md), 24 October 2026 | Verbatim text of the consolidated owner version with D1–D9 and its sources Q1–Q4, in German. |
| [Decision record](sources/v1-concept-decisions-20261024.md), 24 October 2026 | Owner acceptance of the proposed decisions D10–D18. |
| [Conformance audit](../development/concept-conformance-audit-20261024.md), 24 October 2026 | Static review at commit `6c7d758c`; origin of A1–A6 and R1–R3. Not a replacement concept. |
| [V1 scope clarification](sources/v1-scope-clarification-20261023.md), [source implementation approval](sources/v1-source-implementation-approval-20261023.md) | Earlier owner foundations, consolidated by v0.18 and this version. |

Changes to this document show whether they change the target, choose a technical variant or only improve the text. A change of the target needs an owner decision with a register entry. Implementation status and evidence belong in the delivery plan and in validation reports, not here. Like all Markitect documentation, this document is written in English ([DEC-069](register.md#dec-069-documentation-is-written-in-english)).

---

**Guiding idea:** The model determines the organization. The Managers carry the domain responsibility. The host keeps the technical guarantees. Quality comes from this clear separation – not from as many agents, rules, interfaces or approval steps as possible.
