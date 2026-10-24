# Procedure proposals for context, impact and semantic index

**Not chosen and not normative.** This derivation from the design conversation is not part of the [language specification v1](language-contract.md). Context building, change assessment, semantic index, updating, realization mapping, verification planning and Apply are procedure proposals; their records and flows are neither mandatory language constructs nor additional Core Kinds.

The original section numbers 11.1–11.14 and historical notations are kept for traceability. Concrete examples, accepted reviews and computed walkthroughs do not evidence an executed product implementation or domain acceptance. A later selection must be made explicitly and must name its scope of validity.

Provenance and older language variants are in the [language history](language-history.md); current executable evidence is with the [Exchange example](exchange-example/README.md).

---

## 11. Verification and realization

| Task | Limit of the statement |
| --- | --- |
| Structural check | Identities, closed fields, value types, cardinalities, meta-contracts and reference targets. |
| Semantic model review | Comprehensibility, domain suitability, contradictions, scope of validity and missing stipulations. |
| Realization verification | Fixed implementation candidate against the applicable model state; suitable tests, analyses and independent assessment. |

A valid graph does not prove a correct MessageId mapping, monotonic time measurement or tenant separation. A new obligation in the Kind body also applies to unchanged concrete Definition files. Obligations that were not examined remain open. Derived checklists are work results and not a second canonical truth.

Context building, domain change assessment, delegation, integration and Apply remain necessary Markitect tasks. This language contract does not already fully define their new generic connection and does not introduce their execution data as prescribed project vocabulary. Existing product obligations are not incidentally lifted by this language consolidation.

**Notation of the following procedure derivation:** The index walkthroughs from the previous session keep their package IDs, slash IDs and alias:id examples of that time, in particular where they quote concrete historical probe files. They illustrate the index procedures, not the author notation now chosen. For new Definition files and references, the Package/import rules with / and Alias.id from sections 7 and 8 apply; transferring these examples requires explicitly adapted names and references. The historical probe reports are not rewritten.

### 11.1 Proposal: context as a justified compilation

**Status of section 11 from here on: commissioned elaboration, not yet a chosen procedure decision or an implemented generic context/impact computation.** The proposal uses the chosen language without building new mandatory terms such as Requirement, Rule or Manager into it. For the new model, it makes concrete the tasks that the existing product already handles with its fixed project vocabulary. It does not replace that product's accepted routing and verification obligations through a document change.

Context answers: which stipulations and relationships must one understand for the task under consideration? Impact answers: which items and realizations must be assessed again because of this change? Context relevance is neither a change instruction nor proof that something can remain unchanged.

For a Definition under consideration, Markitect first provides:

1. The Definition with complete body and spec as well as its fixed identity and binding.
2. Its Kind and that Kind's complete Property contracts in use, including nested object contracts and the bodies. These stipulations apply even without repetition in the concrete Definition.
3. The referenced items from spec and the explicit body link targets with the contracts they require. It remains recognizable whether, for example, api is a concrete participant or Container is its Kind.
4. Incoming references and further statements as context candidates to be assessed. An obligation described elsewhere can concern the item under consideration.

The compilation names the reason and the source for each entry: for example “Exchange is the Kind of list-documents”, “Exchange/sender is its requesting-party Property” or “list-documents references api as requester”. Shared contracts are provided once and referenced from all affected places. The fixed meta-contract can be present in shared form instead of being written out again recursively for every item.

The model remains fully addressable. Further contracts can be loaded on demand. A bounded first context delivery is not a completed completeness check: candidates still to be assessed and missing context must remain visible. Summaries serve orientation; they do not replace the binding original statements.

### 11.2 Structurally derivable candidates of a change

Impact analysis compares two fixed model states. It captures added, removed and changed Definitions, bodies, Property contracts, values, reference occurrences and package bindings. Unchanged concrete source files do not mean unchanged applicable contracts. A package update is considered by its bound contents, not only by its version string.

The structural part can compute the following reasons:

| Starting point | Candidates or verification tasks that can be found reliably |
| --- | --- |
| Changed Definition | The Definition itself and its incoming and outgoing references. The references alone do not yet prove a domain impact on every neighboring target. |
| Changed Kind contract, even only in the body | Its Definitions and their now applicable contracts. |
| Changed Property, even if reused | All Kinds or object contracts that use it, and the Definitions affected by them. Even a missing optional value can need checking again when the contract has changed. |
| Changed reference value | Source field, field contract and the previous and new target as candidates. |
| Changed target of a body link or deleted target | The previous and new link occurrences or sources; missing targets are diagnosed as structural errors. |

For deleted or re-attached references, the old graph is also used. Otherwise, for example, the previous API could disappear from the area under consideration precisely because requester was switched to another application. A new structurally invalid state first yields diagnostics, not a valid realization assignment.

The edge kinds remain distinguishable: typing, contract use, typed value reference and body reference. Complete undirected reachability would capture almost the entire model through shared Kinds or the Core. It is therefore no proof of domain impact. The procedure extends and assesses candidates by their reasons; it does not treat every graph path as a domain dependency.

### 11.3 Assessing semantic applicability from prose

A body can state: “The application determined by requester must discard late replies.” The Core knows requester as a declared Property and can find the associated applications. But it does not know the meaning of this obligation from the name or link alone. Applying the sentence to api requires a semantic assessment of the entire contract and of the concrete assignment.

Such an assessment yields derived results with provenance: which statement concerns which item under which conditions, which further Definition is needed for it and which question remains open. These are work and verification results against a fixed model state, not a second manually maintained specification and not additionally required Core Definitions. They must not reinterpret the accepted prose or silently introduce new architecture decisions. Contradictions or decisions outside the granted freedom go to the user.

There is an unavoidable limit: free prose can state, even without a link, that all external accesses require a particular authentication. A search only along explicit edges cannot fully determine this scope of validity. For a reliable bounded context, Markitect must therefore examine the statements of the entire relevant model scope for such scopes of validity or use already verified, suitably bound derivations. Word search can help with this; a missing search hit does not prove non-involvement.

An index generated from this can provide context repeatedly without sending every agent the entire model. It must preserve statements and source locations, the reach considered, justifications and uncertainties. New or changed bodies, contracts, assignments and Definitions can change the mapping. In particular, a new matching Definition must not drop out of a previously assessed statement about “all”. Reuse therefore needs proven matching inputs and reach; the complete invalidation rule is still to be worked out.

This semantic analysis is fallible and needs independent review. If it cannot bound an area with justification, the larger area remains open for verification; if necessary, that is the entire modeled scope. A limitation by a context budget must not appear as a domain exclusion. The minimal language makes the description possible; it does not make its complete understanding deterministic.

### 11.4 Exchange counter-probes

For list-documents from section 4, its seven Property contracts, the Exchange body and the concretely referenced participants, Command contracts and Channel contracts belong to the initial context. Incoming statements and the analysis from section 11.3 complement it. The example yields these different cases:

| Change | Structural reason | Consequence to be clarified in content |
| --- | --- | --- |
| timeoutSeconds from 30 to 60 | Changed value together with Property contract and Exchange. | As requester, the API uses the timeout period; its realization and suitable checks are to be examined. A code change in the broker does not follow from this automatically. |
| Exchange.body now requires monotonic time measurement | Changed Kind contract affects all Exchange Definitions. | Determine the requesting party's obligation for each Exchange assignment; for list-documents it concerns api, although its Definition file is unchanged. |
| sender becomes requester with the same Property ID | Changed field contract affects its users; concrete old keys are structural errors. | Migrate model data. Whether software field names must be changed as well depends on the intended interfaces and the previous realization. |
| Topic formation in the body of requests changes | Changed Channel and its consumer references, among them list-documents. | Assess request publication and matching subscriptions on the API and Connector side together; broker configuration only insofar as it depends on this. |
| Link display or title changes | The change is preserved; identities and referenced values stay the same. | Check for a presentation change or a change of meaning; no blanket software change follows from it. |

The traceable base path of the time measurement is: changed obligation in the Exchange body → list-documents has this Kind → requester is declared by Exchange/sender → spec.requester points to api. The first and last mappings can be checked structurally; that the concrete time measurement is an obligation of this application comes from the semantic interpretation of the sentence.

### 11.5 From model involvement to work and evidence

The results distinguish at least an established structural error, a possible consequence still to be assessed, a needed context and a justified realization/verification task. A result “no implementation needed” likewise needs a reason that fits the case; it must not be derived from a missing direct edge. This distinction is not a new catalog of obligations in the canonical model and not a permission to omit previously required context checks.

The model alone does not yet yield concrete file paths or completely executed tests. For that, Markitect needs the bound mapping of the existing realization to the model items, the actual implementation candidate and suitable evidence. These data belong to the realization/Apply level. Mappings determined manually or by machine need a stated basis and coverage. For a new obligation, realization and evidence can so far be missing.

Unknown mappings remain investigation tasks; a path must not be guessed only from a similar name. Observed code dependencies can widen the area under consideration and reveal model gaps, but they do not silently change the intended world. The [accepted problem statement on code dependencies](../../../concepts/register.md#enh-001-observed-code-dependencies-widen-impact) remains relevant.

### 11.6 Binding, traceability and counter-probes still needed

A derived context or impact report names its model states, package bindings, considered scope, reasons with source locations, semantic assessments, open areas and, where applicable, the realization used. When inputs change, its validity is checked again. Cycles must not produce endless processing; a limit reports open work. A later compilation must be able to take up additional insights for verification without silently losing existing unresolved reasons.

These are procedure requirements for a next draft. Still to be stipulated and tried out are in particular the scope and independent verification of the semantic analysis, safe reuse, extension of the context and the binding to the realization. The tables are conceptual counter-probes, not a test report of a new generic impact procedure. An implementation attempt would have to cover, among other things, new global obligations without links, new instances under old obligations, removed references, reused Properties, contradictory statements and missing realization mappings.

### 11.7 Semantic index: source, interpretation and concrete applicability

The next user request makes concrete what is in the index, how its applicability derivations are verified and when they are to be assessed again. This elaboration, too, is a procedure proposal. The index is a re-creatable work result for bound model states, not an additional canonical model part and not an ontology of obligations built into the Core.

For the example of monotonic time measurement, three separate results are needed:

| Result | Concrete content |
| --- | --- |
| Source finding | The sentence in the body of `(example/connector-vocabulary, Exchange)`, its exact source location and the unchanged surrounding contract. |
| Verified interpretation | The time measurement obligation concerns the application determined by the Property `(example/connector-vocabulary, Exchange/sender)` in every Exchange and per attempt. |
| Concrete applicability | For `(example/connector-project, list-documents)` this application is `(example/connector-project, api)`. The obligation concerns api in this exchange; it is thus not across the board an obligation of all operations of this application. |

In the [bound vocabulary example](identities/vocabulary-2.0.0.yaml), the source sentence actually reads: “Der Sender muss die Wartefrist jedes Versuchs anhand monotoner Zeitmessung bestimmen.” (English translation: “The sender must determine the timeout period of each attempt by monotonic time measurement.”) That Sender here corresponds to the Property now called requester is an interpretation to be checked against the contract. The stable Property ID Exchange/sender or word similarity does not prove this meaning. The historical example file is not rewritten for this investigation.

An interpretation entry records at least the following:

- **Source:** canonical Definition identity, bound content, body source location and original statement. The source location is a provenance statement against this content, not a new permanent language ID for every sentence. If the passage cannot be found again unambiguously, the mapping is checked anew.
- **Interpretation and limits:** the meaning in one's own words, the intended terms and roles, conditions, exceptions and the scope of validity considered. For this statement, “per attempt” and “requesting party of this exchange” explicitly belong to it.
- **Basis:** Kind/Property contracts used, individual stipulations, reference values and other context read; structural facts remain distinguishable from semantic conclusions.
- **Verification:** proposed, independently confirmed, disputed or open; with justification and the bound subject of verification. The determination that a sentence is a binding obligation and not a mere explanation also belongs to the interpretation.

For each concrete application, the exchange under consideration, the role target, the condition assignment, the structural base path and the result are recorded in addition. Applicability and review state are different statements: “applies” is at first a claim; it can still be unverified. A negative result reads, for example, “from this statement no time measurement obligation solely because of the receiver role”, not “onprem has no obligations”.

For the example state, a readable application entry reads:

| Field | Value |
| --- | --- |
| Statement | Monotonic time measurement of the timeout period of each attempt. |
| Provenance | Exchange.body in the bound vocabulary package. |
| Exchange considered | list-documents in the project package. |
| Role | Property Exchange/sender, field name requester. |
| Affected item | api in the project package. |
| Base path | list-documents.kind → Exchange; Exchange uses Exchange/sender; list-documents.spec.requester → api. |
| Applicability claim | For this exchange, api must determine the timeout period of each attempt monotonically. |
| Review state of this example | Proposal in the design conversation; no independent semantic verification executed yet. |
| Realization evidence | Not part of this applicability finding; not yet provided. |

A general interpretation may describe a reusable selection such as “all Definitions of this Kind, in each case their application determined by this Property”. Only when this selection has been explicitly verified against the source and is executable with stipulated operations may Markitect afterwards determine its concrete matches deterministically. An expression freely generated by the agent does not become trustworthy compiler semantics merely by existing. The Core interprets neither the word requester nor this sentence as a built-in operation. For more complex prose, the selection itself remains semantic work.

### 11.8 Verifying the derived applicability independently

The verification has three tasks with different limits of their statements:

1. **Structural verification:** The source version is bound and the source location fits; Exchange is a Kind; list-documents has this Kind; Exchange/sender is its valid Property contract; the current field name is requester; the reference value denotes api. This can be checked against the compiled state.
2. **Semantic verification:** An independent reviewer receives the original statement, the complete required contracts, the concrete assignment and relevant further statements. The reviewer checks for themselves whether Sender here means requester, whether the obligation applies per attempt, whether conditions or exceptions are missing and whether the application to api is correct. The reviewer does not receive only the first agent's summary. Ambiguity that cannot be resolved remains open; a real contradiction is not overruled by an index decision.
3. **Coverage verification:** Possible omitted statements and cases were also considered. Confirming only proposed positive matches is not enough. Counter-probes must distinguish a swap of requester and receiver, a further Exchange, the same application in another role and additional restrictions in the individual body. This verifies the interpretation; it does not yet execute the software's time measurement.

Coverage includes its own evidence of the scope examined: which source bodies were analyzed, which statements were not yet clarified, which Definitions or candidate groups were considered, and which exclusions are verified with which reason? Cases with the same justification may be grouped; the group must be determined unambiguously at the fixed state. In the example, the selection of all Exchange Definitions can be bound instead of claiming that today's list of matches is complete forever. In the initial analysis, prose outside this selected group must also be examined for wider-reaching applicability.

An open candidate is not a negative determination. A review of the wrong or incomplete candidate space does not prove completeness either. The coverage statement therefore denotes the space actually examined and the remaining uncertainty. Independent verification reduces the risk of errors; it does not prove complete capture of arbitrary prose.

### 11.9 Changes: renewing interpretation, application and coverage separately

For a new model state, previous results do not at first count as current without verification. Markitect compares the inputs and distinguishes which interpretation, concrete application or coverage is to be assessed anew. Old results are kept as results of the old state. A re-evaluation can yield the same result; its current validity nevertheless needs a traceable reason.

| Change | Reconsideration required in the example |
| --- | --- |
| The source sentence or its context of meaning changes | Verify the interpretation semantically again; then renew its applications and the affected coverage. An unchanged sentence can acquire a different meaning through a new exception in the same contract. |
| Property.body, type or role meaning change | Assess again the mapping of the statement to Exchange/sender; where applicable, all applications that depend on it. Stable identity does not prove stable meaning. |
| requester switches from api to another application | Renew the application for this Exchange; take into account the previous and the new target. The verified general interpretation can remain if its complete bases still fit. |
| A new Exchange is added or a Definition receives this Kind | Evaluate selection and coverage again, verify the new role assignment and form a new application. An unchanged list of previous matches is not sufficient. |
| An Exchange or its role reference is dropped | Remove the previous application at the new state or mark it as no longer applicable. Other applications of the same obligation to api remain separately in place. |
| timeoutSeconds changes from 30 to 60 | The statement “monotonic time measurement per attempt” can continue to apply unchanged, provided its interpretation is value-independent. The new concrete stipulation and the realization must nevertheless be verified; an unchanged applicability finding is not unchanged verification evidence. |
| sender becomes requester with the same Property ID | Verify the structural mapping under the new field name again; renew the terminological interpretation and source locations semantically if needed. Do not use mere character replacement as proof of meaning. |
| Another statement introduces an exception or global obligation | Assess affected interpretations, applications and coverage anew, even without direct links. If the reach is unclear, include the wider modeled scope. |
| An import is updated | Compare bound contents and the sets of Definitions considered. Identical package/Definition identities are not sufficient for reuse. |
| Only a file name in the same Package or an import alias changes | Canonical identities persist; map provenance and, where applicable, body source text anew. An evidenced pure presentation change needs no new domain conclusion. |
| Only the software changes | Model applicability can remain; the realization evidence must fit the new candidate. |

Safe reuse therefore involves not only Definitions actually read. Also to be recorded are the selection spaces and negative preconditions on which a result rests: for example “all Exchange Definitions of this model scope” or the verified absence of a relevant exception. Otherwise an old index seemingly remains valid when a new matching Definition or a contrary statement is added. An agent report with a list of IDs read is on its own insufficient for this.

As a first conservative procedure rule, the following is proposed: every model change triggers a freshness check of the index; reuse happens only with a demonstrably unchanged basis and coverage. If this dependency is not yet reliably delimited, the larger semantic verification area is assessed again. This does not mean reinterpreting every sentence for every changed value. But it prevents an unproven local limitation from being treated as a safe cache hit. The exact data encoding, selection execution and proof of complete reuse bases remain implementation questions.

### 11.10 Walkthrough: a second Exchange and a change of its requester

At the user's request, the proposed index is now updated over three fixed model states. This is a complete conceptual walkthrough for the time measurement obligation considered, not an executed semantic verification run and not a claim about the entire software. For the flow, a general interpretation previously confirmed independently is assumed; the previous reports contain no such confirmation.

**Bounded example.** In the following, E1 denotes list-documents with requester api and receiver onprem. E2 is a new Exchange check-status. Required supporting Definitions are presupposed for the initial state: status-service as a permitted Container, check-status-command as a read-only status contract, and status-requests and status-replies as permitted Channel Definitions. Their contracts allow authorized requesting parties at both sites, instance-specific replies to the respective requester and the matching MessageId mapping. They prescribe neither api as the only requesting party nor an exception to the time measurement obligation. If these Definitions had to be newly created first, they too, with their own statements, would belong to the change to be verified.

These dedicated channels avoid a hidden assumption: the previous channels requests and replies contain, among other things, statements about the API as sender. The later role change is therefore not simply to be treated as already compatible with these older channel contracts.

Under these preconditions, the complete E2 Definition in state M1 reads:

```yaml
id: check-status
kind: connector:Exchange
body: >-
  The requester queries the operational status from the receiver.
  The operation is read-only. Replies keep the MessageId of the request.
  Late replies without a waiting state are discarded.
spec:
  requester: api
  receiver: status-service
  command: check-status-command
  requestChannel: status-requests
  replyChannel: status-replies
  timeoutSeconds: 10
  maxAttempts: 2
```

M2 changes exclusively E2.spec.requester from api to onprem. The role-related body stays as it is. The Property identity remains `(example/connector-vocabulary, Exchange/sender)`; api, onprem and both Exchanges are in the project package. E1, the package bindings and the general Kind/Property contracts are unchanged in all three states. M0, M1 and M2 are labels of this walkthrough, not new Core items.

**M0: bound starting point.** The index contains the assumed verified interpretation “the requester of every Exchange determines the timeout period per attempt monotonically”. For the selected Exchange area, the set `{E1}` was considered. The concrete application reads “api for E1”. Sources, interpretation, selection scope, individual statements and verified restrictions are bound to M0. There is no negative determination that onprem could never be a requesting party.

**M0 → M1: adding E2.** The update goes through these steps:

1. **Check the change and the structure.** E2 has been added. Its identity is free, its Kind is Exchange; all seven fields, cardinalities and target types are to be checked for validity. The requester entry is resolved to api via the current Property contract. A merely valid Container reference does not yet imply domain admissibility as a requesting party.
2. **Analyze new statements.** E2.body is assessed completely for the first time. Together with the required contracts, the reviewer examines whether the new text contains an exception, a contradiction or a statement with an effect beyond E2. Only after this examination can the change be semantically bounded as local. In this chosen example it is assumed not to introduce any further global stipulation; this is a precondition to be verified, not a conclusion from the small text diff.
3. **Reuse the interpretation, renew the selection.** The general time measurement interpretation can be kept if its sources and meaning bases are unchanged and the new statement does not change it. The selection, on the other hand, is now `{E1, E2}`. Its old coverage finding does not fit M1 unchanged. A verified executable selection may determine the set anew; otherwise this selection is to be assessed semantically again.
4. **Verify the new application.** For E2, the proposal “api for E2” arises. The structural base path is to be checked anew; the semantic verification considers the individual description, participant, Command and Channel contracts as well as further pertinent statements. If applicability is confirmed, this case receives its own current review state. A mere copy of the old entry “api for E1” is not sufficient.
5. **Carry over existing results.** The application “api for E1” can be carried over to M1 after the freshness check: unchanged bases and no newly determined contrary statement. Its earlier verification remains provenance; a new carry-over finding explains its validity at the new state. Unchanged bytes alone would be insufficient in the case of a new global exception.
6. **Form context and work.** The API context now contains two applications bound to different Exchanges. For E2, realization/verification tasks arise; previous evidence for E1 does not automatically cover E2. Whether shared implementation or integration requires further checks for E1 is to be assessed on the basis of the actual realization.

**M1 → M2: changing E2.requester from api to onprem.** Again, several parts are to be handled separately:

1. **Compare the old and the new reference.** There is no new Exchange Definition and no change of the selection set `{E1, E2}`. What has changed is the role assignment of E2. Markitect keeps the old target api for the change assessment and checks the new target onprem against the Container contract. An invalid new reference blocks a valid new model state.
2. **Check the general interpretation for freshness.** The statement continues to concern the respective requester, regardless of its identity or site. Under the stated unchanged bases, the general interpretation can be reused. This is not a new reading of the word requester from its name.
3. **Determine and verify the concrete application anew.** “api for E2” no longer applies in M2. Instead, “onprem for E2” arises. Suitability as requesting party, authorization, reply address, waiting state management and any restrictions by other contracts are assessed for the new assignment. In case of contradictions, the case remains disputed or open; the index grants no exception from the architecture. Under the chosen compatible contracts, the time measurement obligation for onprem in E2 is assumed as confirmed.
4. **Update coverage.** The set finding for the Exchange members can be kept after an equality check. The finding on their role assignments must be renewed. Coverage here means both “both Exchanges considered” and “their current assignments assessed”. Only the first statement is unchanged.
5. **Keep the other application and renew contexts.** “api for E1” remains valid after the freshness check. The API context loses only the application for E2; the onprem context receives it. Historical M1 results are not overwritten. References to api as participant of other operations are kept.
6. **Examine realization consequences separately.** The previous requesting-party realization for E2 on the API side must, where applicable, be dropped or switched; on the onprem side, request creation, monotonic time measurement and reply mapping are to be realized and verified. receiver, channel permissions and integration can also be affected. The change of model applicability does not yet yield a complete file list or a passed test.

The expected index after successful assessment in each case reads, for this one obligation:

| State | Exchange selection | Application to api | Application to onprem |
| --- | --- | --- | --- |
| M0 | E1 | For E1. | No application from this obligation in the state considered. |
| M1 | E1, E2 | For E1 and separately for E2. | No application from this obligation in the state considered. |
| M2 | E1, E2 | Still for E1. | New for E2. |

Reuse thus does not follow only from the equality of target IDs:

| Part of the index | M0 → M1 | M1 → M2 |
| --- | --- | --- |
| Original statement and general interpretation | Reusable after freshness checks and counter-checks. | Reusable under the same preconditions. |
| Application to api for E1 | Can be carried over after checking possible new influences. | Can be carried over after checking possible effects of the role change. |
| Application for E2 | Analyze and verify anew. | Verify the new role assignment; keep the previous application only historically. |
| Completeness of the Exchange selection | Determine the new set and renew coverage. | Confirm the same set structurally; renew the current role coverage. |
| Concrete participant/Channel verification for E2 | New. | Not reusable across the board; the new requesting party changes relevant preconditions. |
| Evidence of software conformance | Still to be provided for E2. | To be provided again for the new implementation candidate; a model review does not replace it. |

This walkthrough also proposes an important report property: the carry-over of an old finding is also a result with a reason and a new state binding. A previous status “confirmed” is not copied to M1 or M2 without a freshness check. Neither an open new case nor its prioritization as context may appear as a completed verification. The concrete automation of these steps is still to be designed and tried out.

### 11.11 Reusability: content read, selection spaces and new influences

The further user request makes concrete the dependencies with which an existing index entry is checked for reusability. The following proposal complements the walkthrough without stipulating a new Core language for rules or applicability predicates. Capturing dependencies is part of index processing. A list of the IDs visited at first reading is not sufficient for this: a previously unknown sentence can claim a global stipulation, restrict another statement or add a new obligation.

**Two separate bases.** Each entry depends on the one hand on its concrete sources and interpretation steps. On the other hand, its usability in the current context depends on changes in the entire bound model scope having been examined for possible additional or contrary statements. The content of an old finding can still be correct while the context generated from it is still incomplete. “Unchanged basis” and “no unresolved new influences” are therefore separate checks.

At least these dependencies are to be captured:

| Dependency | Content to be bound | Which change thereby becomes visible |
| --- | --- | --- |
| Source statement and context of meaning | Canonical identity, complete authoritative body, source location, Kind/Property contracts used and further interpreted context. Initially, whole Definitions rather than individual sentence fragments are bound. | New exception next to an unchanged sentence; changed meaning of the requesting-party Property; dropped justification. |
| Concrete structure and values | Typing used, stable Property IDs with current field name, reference targets, conditions and relevant values or their absence. Excerpts that are not sufficiently delimited are conservatively bound as complete Definition content. | requester switch, changed condition values, newly set optional entries or deleted relationships. |
| Selection space and selection procedure | Fixed model scope considered, verified meaning of the selection, operations used, members of the initial space and all inputs needed for the selection. | New Exchange Definition or a previously excluded item that now matches because of a change. |
| Negative preconditions and exclusions | Which area was checked for which restriction, associated sources, verified exclusion reasons and open cases. An empty result also needs a basis. | A new possible exception or the deletion of a previous exclusion reason. |
| Other derived findings | The exact versions of interpretations used, of their verifications and coverage, as well as preconditions for their reuse. | A disputed role interpretation or a coverage finding that has become invalid affects its dependent applications. |
| Model binding and processing contract | Package/content bindings, complete input inventory, resolution/normalization rules and index/verification procedures used. | Newly included library content, changed resolution or a change of the procedure whose result one relies on. |

IDs and digests evidence identity and content equality; they evidence neither the sufficient selection of the bases nor their correct interpretation. A content binding must hit the scope actually verified. A changed overall model digest alone does not yet say which finding is to be reinterpreted in domain terms, but it ensures that the old state is not simply presented as the current one.

**Selection dependencies go beyond previous matches.** For “all Exchanges”, membership in the complete selected Definition space is monitored, not only the list of applications already assessed. For a conditional selection such as “all Exchanges with requester at site cloud”, the requester references and site values of candidates that previously did not match also belong to the selection basis. Otherwise a site change could produce a new match unnoticed. For a verified executable selection procedure, these dependencies can be specified technically; for free semantic selections, the complete reach itself remains in need of verification. A reverse index on matches read does not solve this problem on its own.

**New global statements need a complete change check.** For a transition between two fixed states, a complete change set is determined from the entire bound input scope, independently of existing entry dependencies. It includes additions, changes and deletions of Definitions and imported contents as well as changed types, values and resolved relationships. Unchanged prose can acquire a different reach through a different assignment or binding. Therefore not only edited body strings are considered.

For each change area, this change check records: examination still open, possible effect on named areas, or justifiably no additional effect on the finding considered. Its basis contains the previous and the new content and the context needed for understanding. For semantic determinations of reach, independent verification is required. Working through the change set completely is structurally controllable; reliably bounding its meaning remains a semantic task.

In particular:

- A new or changed statement is not treated as local solely because of missing links, a local file path, its Kind or missing keywords. In the chosen Core, its body may describe a wider scope of validity.
- For deletions, the earlier statement is taken into account: a dropped restriction can widen existing obligations. For a changed statement, the earlier and the new reach count.
- A statement such as “no relevant exception found” is a verified finding against a particular scope. It is not stored as a structurally proven absence of any semantic exception. Newly added text is at first unverified and not a further negative match.
- An already excluded source is considered again when it changes materially. New Definitions and new import contents must not be excluded because they are missing from the old selection space.
- An unclear possible influence keeps the affected findings open for the new state. If its area cannot be bounded with justification, the larger bound model scope is to be assessed. There is no automatic relief through low priority or a context budget.

The complete input inventory and its binding form a conservative outer dependency for this. Every model change requires a current finding on the possible reach of its changes. Initially, this area is not partitioned on the basis of automatically assigned topic labels. A word filter or a vector search can supply candidates, but cannot exempt any new statement from the change check. A claim by the same old index that “nothing relevant can be here” must not itself be the only basis for excluding new texts unread.

**Flow for a reuse attempt.** For the entry under consideration, Markitect proceeds in this order:

1. Compile the new model state and determine the complete changes relative to the bound starting state. With an invalid structure, no valid new index state arises.
2. Compare or re-evaluate direct bases, presupposed interpretations and selection spaces. If, for example, requester has changed, the concrete application is formed anew. An unchanged selection set does not yet prove unchanged role assignments.
3. Examine the changes in the bound model scope for new, changed or dropped influences. An old global verification state alone is not sufficient for this. If relevant questions remain open here, the entry must not yet be used as fully current context.
4. Assign possible new influences and assess affected interpretations, applications or coverage again. An additional independent obligation can require a new entry without changing the old time measurement interpretation. A claimed exception, on the other hand, can create a conflict; it does not silently override existing obligations.
5. Bind only sufficiently evidenced reuse to the new state. The carry-over finding names unchanged or newly confirmed bases, current selection/coverage findings and the completed verification of new influences. Historical results are kept.

The result is either “reusable for this state”, “to be derived anew because of named changes” or “freshness still open”. The original verification evidence remains distinguishable from this current carry-over finding. A reused entry, too, can receive additional new context neighbors. A composed context only counts as current when its required entries and the verification of further influences fit the same target state. Work still open is not concealed by a mixture of old and new positive states.

For the time measurement obligation from the previous M2, these counter-probes result:

| New change | Triggering dependency | Consequence |
| --- | --- | --- |
| A previously uninvolved Definition now states “All requesting parties must additionally log the clock source used.” | Complete change check, even without a reference to Exchange or api. | Check the scope of validity and add new applications where applicable. The old time measurement finding can remain; the earlier context is not complete for that reason alone. |
| Another Definition claims “For status queries, the wall clock is sufficient.” | Check of new influences as well as the previously verified negative precondition on exceptions. | Assess a possible conflict with the Exchange contract for check-status. No automatic exception according to the principle “more specific wins”. |
| A third Exchange is created | Selection space of all Exchanges and new statements in the model scope. | Verify the new application and coverage; the existing interpretation can be kept if the bases are unchanged. |
| check-status.requester switches back from onprem to api | Concrete role assignment and the selection/application findings that depend on it. | Take both targets into account and renew the application; list-documents remains a separate case. |
| An already captured restriction is deleted | Old source version, preconditions and complete deletion finding. | Assess previously excluded cases again. A view derived only from the new graph would be insufficient. |

The proposal thereby ensures that new inputs do not remain unverified solely because of a too narrow cache dependency. It does not promise that an agent recognizes every semantic connection without error. Initially, the larger content bindings and the complete change check are to be preferred; finer reuse may only rest on verified sufficient capture of dependencies and coverage. Neither an exact data contract nor an executable reuse verifier is already implemented with this design counter-probe.

### 11.12 Walkthrough: global additional obligation and explicit exception

The next user request plays through a new global obligation and then an exception for check-status. The starting point is M2 from section 11.10: list-documents has requester api, check-status has requester onprem. The Definitions considered, role assignments and package bindings stay the same afterwards. All following confirmations describe expected results after successful independent verification in the hypothetical example; such a semantic verification was not executed here.

To delimit the case, the new obligation concerns the additional logging of the clock source. The later exception limits exclusively this new obligation. It does not lift the existing obligation of monotonic time measurement. In the example, the source statement is added to the body of the existing project Definition platform. Its global reach arises from its sentence, not from the Kind System, its storage location or an automatic containment rule.

As readable abbreviations for index findings we use T for the general interpretation of the time measurement obligation, T1 for its application to api in list-documents and T2 for its application to onprem in check-status. These abbreviations are not additional canonical Definitions or promised permanent sentence IDs.

**M2 → M3: adding a global obligation.** The new statement reads:

> In addition, for every Exchange the application determined by requester must log the clock source used in every attempt.

The compiler can capture the changed body and its bound new content. The sentence needs no links to be binding. That it means both Exchanges and their requesting parties is to be derived semantically and verified independently.

1. **Open freshness:** The old finding on the complete analysis of possible new influences does not fit M3. Previous contexts must not count as fully current until the addition has been examined. T, T1 and T2 are therefore not automatically wrong in domain terms; their use at the new state needs a current verification. If platform itself belonged to their concretely bound meaning bases, this content dependency has changed as well.
2. **Form a new interpretation:** L3 arises: the requesting parties of all Exchanges log the clock source used per attempt. To be verified are in particular the role reference, obligation character, selection scope, relationship to monotonic time measurement and possible other restrictions. The obligation applies per exchange and attempt, not arbitrarily to every operation of the referenced application.
3. **Verify new applications:** L3-1 concerns api for list-documents; L3-2 concerns onprem for check-status. The type/Property mappings and reference values can be carried over from current structural findings. The application of this new obligation and its compatibility with the concrete contracts are to be assessed anew.
4. **Confirm the existing obligation:** After verification of the relationship of both statements, T, T1 and T2 can be carried over or confirmed again for M3. The new obligation complements the time measurement; it does not override it. A whole old Definition hash must not be presented as unchanged if the meaning content considered has actually changed.
5. **Renew coverage and contexts:** The Exchange base set remains {list-documents, check-status}; its unchanged membership can be confirmed structurally. The new logging obligation, both applications and their possible restrictions were not covered so far. Only their verification establishes a new complete finding for this area considered. The API and onprem contexts each receive one additional obligation.

After successful assessment, the considered excerpt of the M3 index contains T, T1, T2 and additionally L3, L3-1, L3-2 together with new coverage. Historical M2 results remain unchanged. Software conformance with regard to the additional logging is thus not yet evidenced.

**M3 → M4: explicitly limiting the global statement.** For the contradiction-free main case, the previously added statement in the same contract is replaced by:

> In addition, for every Exchange except [[check-status]] the application determined by requester must log the clock source used in every attempt.

The explicit target of the body link is resolved in the project package. The exception arises from the sentence; the link alone knows no exception operation. The new source version remains under the same Definition identity platform, but has a different body and thus a different bound content basis.

1. **Mark the changed source and dependent findings:** L3 is not a current interpretation of the new source version. Its applications L3-1 and L3-2 as well as the finding “this obligation applies to both Exchanges” therefore cannot be carried over to M4 unchanged. An earlier finding “no pertinent exception found” for the logging is likewise no longer current. They remain valid historical results for M3.
2. **Verify the new interpretation:** L4 now reads “logging for all Exchanges except the explicitly named check-status”. The reviewer verifies the source comparison, an unambiguous exception target, the limited scope of the exception and contradictions in the wider context. In particular, it must remain clear that the time measurement obligation is unaffected by it. A matching exception must neither be guessed from a similarly named status Command nor extended to all status queries.
3. **Classify both previous applications again:** For list-documents, the current positive finding L4-1 on api arises or is confirmed. The unchanged role assignment and structural mapping can be reused; the application under the changed contract needs a new applicability/carry-over finding. For check-status, L3-2 no longer applies in M4. Instead, the coverage explicitly records: from L4 no logging obligation for onprem in this Exchange, because of the verified exception.
4. **Distinguish sets and negative results:** The Exchange base set considered still has two members. The positive selection of the logging obligation shrinks from two to one. check-status does not disappear from the coverage but remains included as a verified excluded case. This prevents a missing positive entry from being confused with an overlooked verification.
5. **Check time measurement and further obligations for interactions:** T1 and T2 remain in place after a suitable freshness/compatibility check. An exception from L4 is no evidence of an exception from T or from another logging requirement. If the wording reaches beyond the obligation considered or is ambiguous, the wider area remains open.
6. **Form current contexts and realization consequences:** The API remains obligated for list-documents to time measurement and to logging. onprem remains obligated for check-status to monotonic time measurement; the additional logging is no longer required by this statement. Its removal is not a prohibition on continuing to log. Implementation and tests must be assessed against the new set of obligations instead of automatically removing existing logs or tests.

The expected current applicability findings after successful verification in each case are:

| Obligation and application | M2 | M3 | M4 |
| --- | --- | --- | --- |
| Time measurement: api for list-documents | Applies. | Still valid after freshness check. | Still valid after freshness check. |
| Time measurement: onprem for check-status | Applies. | Still valid after freshness check. | Still valid after verification of the limited exception. |
| Additional logging: api for list-documents | Not yet required from this source. | New positive finding L3-1. | New or reconfirmed positive finding L4-1. |
| Additional logging: onprem for check-status | Not yet required from this source. | New positive finding L3-2. | Old positive finding no longer current; explicitly verified exclusion from L4. |

For “invalid”, the report must therefore distinguish: source or derivation are outdated for the new state; a concrete applicability claim no longer holds; or merely the freshness/completeness check is still open. These different findings require different work. A global addition does not make every existing individual obligation wrong; but a changed source interpretation cannot be attached to its old applications without a new carry-over reason either.

**Alternative case: claiming a local exception without adapting the global obligation.** If check-status.body merely stated “This logging is not required here” while the unrestricted global obligation remains, their compatibility would be open or contradictory. There is still no Core rule “local or specific wins”. The index must mark the counterclaim and the affected applicability finding for check-status for clarification instead of silently removing L3-2 or claiming a confirmed exemption. list-documents can only be confirmed as unaffected after a justified delimitation of the conflict. Whether the intended exception is resolved by sharpening the global statement or by another explicitly agreed rule is a model decision; the index must not invent it itself.

### 11.13 Proposal of a data contract for the semantic index

The user now commissions a concrete data contract from the walkthroughs. The following contract describes derived index data, not further Definitions or meta-Kinds of the canonical language. It is a draft of the data model; its serialization, digest encoding and production implementation are not already stipulated. Domain applicability, independent verification and freshness remain separate entries.

#### Envelope, identities and model binding

An index package contains formatVersion, targetSnapshot, procedures, records, reviews, bindings and requiredCoverage. In the proposal, formatVersion is semantic-index/v1. The three collections records, reviews and bindings are lists with unique content IDs; they contain or reference immutable records. requiredCoverage is a list of exact BindingRefs. All fields named are required; collections can be empty, but then possibly yield only an incomplete index. Every foreign record and every content manifest must be available in its exact version; a digest without resolvable content is not sufficient for re-verification. procedures denotes the exact Procedures structure or a resolvable content reference to it; the same rule applies in entries and review records.

| Type | Required fields and meaning |
| --- | --- |
| ModelRef | package and id, both non-empty strings. Denotes a canonical Definition without alias or built-in version number. |
| SnapshotRef | modelDigest, packageBindingsDigest, inventoryDigest and compilerContractDigest. The referenced manifests bind all loaded Definitions, package versions/contents and the resolution/normalization used. |
| Procedures | indexContractDigest, derivationProcedureDigest and reviewPolicyDigest. They bind data/derivation rules and requirements for verification; a model import does not install these procedures. |
| RecordRef, ReviewRef, BindingRef | In each case the exact immutable record version via its content ID. A readable label can be displayed but does not replace this identity. |
| InventoryRef | scope, snapshot and digest. The resolvable content contains the complete uniquely identified elements of this scope, including Definitions, candidates or changes and their necessary values. |

targetSnapshot denotes the state for which the envelope is assembled. The entire bound model is the fixed outer scope bound-model; smaller selection spaces must trace back to it. For an import change, the loaded contents changed by it also belong to the inventory difference. Snapshot and procedure references are to be compared exactly; differing versions require a freshness check and, where applicable, explicitly evidenced compatibility, not an assumption based on similar version names.

Record IDs are technical content identities of the index. They are not stable domain IDs of the canonical model. When their content is computed, the record's own ID field is omitted so that no self-hash is required. A change to the claim, dependencies or provenance creates a new record version. Optional replaces refers exclusively to the historical predecessor version; it is not a continuing domain precondition and not an entitlement to carry over its reviews.

#### Common entry form and four entry types

Every entry has the required fields recordId, recordType, originSnapshot, procedures, producer, sources, dependencies and claim. producer contains runId and the exact generating procedure version. sources and dependencies are lists; for interpretations at least one source location is required, for applications at least the verified interpretation and the concrete case basis. In other cases, the required basis must be present through source, inventory or prior-finding references. Unknown fields or recordType values are errors; optional fields are omitted instead of being set to null.

sources each contain definition as ModelRef, definitionDigest, pointer, startByte, endByte and quote. A pointer uses JSON Pointer notation; for prose, pointer is initially /body. The range [startByte, endByte) counts UTF-8 bytes in the decoded field value, without Unicode normalization, not positions in the YAML file. The boundaries must fall on character boundaries and the excerpt must yield exactly quote. The source file/line mapping is additional provenance. The complete bound contract remains available; a correct source location does not prove a sufficient context selection.

| recordType | Required claim content |
| --- | --- |
| interpretation | meaning as an interpretation in prose that is faithful to normative force, force as obligation, permission, prohibition, information or unresolved, as well as scope with description and inventory. The selection must fit the concrete version of a scope considered. |
| application | interpretation as RecordRef, subject as ModelRef, context as a non-empty list of ModelRefs, result as applies, excluded, uncertain or conflict, statement and reason as non-empty texts. Optionally, property as ModelRef denotes the role considered. |
| coverage | purpose as sources, cases or changes, inventory as InventoryRef and results as a list of the individual results defined below. An optional baseSnapshot is required for changes. |
| influence | baseSnapshot, targetSnapshot, changes as InventoryRef, considered as a list of exact RecordRefs or as bound-model, result as affects, no-additional-influence or uncertain, affected as a list of ModelRefs/RecordRefs or bound-model, as well as reason. The result refers to the stated comparison and the area considered. |

scope.inventory of an interpretation binds the initial space considered at its creation, not a list of matches closed forever. The general meaning “all Exchanges” can remain unchanged while its current case set is re-evaluated. A successful re-evaluation may replace only this set-related observation; statements and other direct meaning bases must still be verified as unchanged.

An application entry with subject onprem, context [check-status], property Exchange/sender and result excluded states exclusively: the referenced interpretation produces no such obligation for this participant in this exchange. It contains neither a global exemption of onprem nor a prohibition of the described action. uncertain and conflict are actual results in need of clarification and not negative applicability evidence. force denotes a derived interpretation, not a new language feature for requirements in the model.

#### Dependencies as closed variants

Every dependency has a key unique within the entry and a type. The remaining required fields depend on the type. Initially, complete normalized Definitions count as the content granularity; arbitrary field fragments selected by agents are not a reliable automatic optimization.

| type | Required further fields | Validity condition on reuse |
| --- | --- | --- |
| content | definition as ModelRef, expectedDigest | The complete Definition has the same normalized content version. This also captures values and their absence. |
| record | record as RecordRef | The exact presupposed finding is itself usable for the target state. A new interpretation with the same wording is not automatically the same precondition. |
| selection | basis as RecordRef of the verified selection meaning, input as InventoryRef, result as InventoryRef, evaluation as verified procedure reference | The selection is re-evaluated against the complete current input space. The relevant observed result set stays the same. Changed candidate/assignment coverage must nevertheless be verified anew. |
| absence | statement, scope as InventoryRef, coverage as RecordRef | The exactly named negative precondition is confirmed again for the current scope; a formerly empty result is not simply carried over. |
| watch | scope with value bound-model | A current complete verification of the new influences is available for the entry. Unequal model inventories require this verification, not necessarily a new interpretation of the entry. |

evaluation of a selection may denote only a stipulated verified procedure. If the selection cannot be executed structurally, the re-evaluation must supply a correspondingly verified semantic result. A freely generated program or a description is not executed as a trustworthy query. input also contains non-matches and relevant selection values; result contains not only a match count but the complete observed mappings. For applications, requester mappings belong to the observation, not only Exchange IDs.

Every entry to be used in domain terms for the current context is covered by watch or the equivalent mandatory check in its Binding. This check is never optional merely because dependencies contain no new source. If a content basis or an exact presupposed record version changes, the old entry cannot be carried over as unchanged. Even with the same conclusion, a new finding with current bases then arises. A successful renewed selection or absence check, on the other hand, may confirm the observation explicitly provided for this; that is a logged verification step, not a comparison of the old digests alone.

#### Coverage and complete change capture

A coverage.results element contains itemId from its inventory, outcome, reason and records as a list of precise finding references. For purpose sources the outcomes are interpreted, no-semantic-claim or unresolved; for cases assessed, excluded or unresolved; for changes assessed or unresolved. reason is also required for exclusions. assessed or interpreted need matching associated findings. A negative no-semantic-claim finding must be semantically justified and verified. There is exactly one result per inventory element; missing results are treated as open, foreign or duplicate itemIds are errors.

For sources, the mandatory outer inventory must capture all loaded Definition bodies and their meaning bases. For cases, the input set including relevant assignments must be determined unambiguously. For changes, the complete inventory is generated from the comparison of two fixed model states. Removed entries also receive an itemId and prior provenance. Neither search hits nor an old selection determine which changes are included in this inventory.

requiredCoverage of the envelope names exact bindings for the source analysis of the outer model scope, the required case areas and, in the case of a carry-over, the complete change check. A new state may alternatively be analyzed completely anew; an unbound old coverage finding does not replace this work. For coverage and influence, current versions with the target inventories or the exact state comparison are initially created. Their contents and independent verifications form an upstream basis: they are validated directly against the fixed input state without in turn presupposing their own complete influence clearance. The downstream interpretation/application Bindings consume this basis. Mere records references from coverage and influence to claims considered serve traceability and are not a mutual usability precondition. This prevents an entry and the change clearance from confirming each other's validity without an independent basis.

#### Review reports, freshness bindings and derived status

A review is an immutable record with reviewId, subject as exact RecordRef or BindingRef, snapshot, procedures, check as structural, semantic or coverage, verdict as pass, fail or open, reviewer with runId and role, contextManifest as content reference, evidence as a list of bound evidence items and reason. The justification is always required. The requirements for independent semantic verification are in the bound reviewPolicy. Different runIds or an independent: true claimed by the producer are not sufficient as evidence; the verification environment and the original context supplied must make the required separation traceable.

The Binding is the actual usability statement for a particular state. Required are bindingId, record as RecordRef, targetSnapshot, procedures, mode as original or reuse, dependencyChecks, coverageBindings and reason. With original, targetSnapshot must match the entry's originSnapshot. With reuse, previousBinding for exactly the same recordId is required. dependencyChecks is a list of objects with key, result as equal, reassessed, changed or unresolved, evidence as a list of exact evidence references and reason. Every dependency.key receives exactly one finding; unknown or duplicate keys are errors. reassessed needs current concrete evidence. coverageBindings references the source, case and influence verification needed for the entry; for their upstream validation, the order just described applies. The Binding receives its own structural review reports and, for semantic carry-over decisions, the reviews required by the policy. A fully mechanical carry-over step can rely on existing matching semantic verification evidence instead of having the same interpretation assessed again.

Even with mode original, dependencies and coverage must be verified. previousBinding and replaces document only provenance; they automatically fulfill no current verification obligation whatsoever. Exact references and validation dependencies are to be resolved finitely; circular justification is invalid. Historical and diagnostic references may show cycles but produce no proof of validity.

The status is computed from findings and bindings and is not stored as a freely settable approved field:

| Axis | Values and derivation |
| --- | --- |
| reviewState | unreviewed: required evidence is missing; confirmed: all required verifications succeeded; disputed: a decisive fail finding is present; open: a required verification explicitly remains unresolved. disputed takes precedence over open, open over unreviewed. |
| freshness | current: dependencies and required coverage fit the target state; stale: a decisive basis has demonstrably changed; pending: the freshness check is incomplete. stale takes precedence over pending. |
| usability | usable only with reviewState confirmed and freshness current and without an unresolved claim or required context. uncertain, unresolved or an unresolved force permit no usable applicability. Otherwise blocked with fail, conflict or stale; else pending. |

A reviewer may correctly confirm a claim with result conflict: the finding is then verified, but the conflict remains unresolved. It is provided as a diagnosis, not as a usable positive or negative obligation. Likewise, individual entries can be usable while the entire envelope supplies only a partial context because of incomplete requiredCoverage. A later positive review line does not overwrite an old fail finding of the same version according to the principle “last wins”. In this case the draft requires a new traceable finding/Binding version with its own verification; old versions remain unchanged.

#### Exchange: readable record view and carry-over

The following reference names are explicitly symbolic short forms of exact content/snapshot references, not claimed real digests or already valid product files. M4 denotes the state with the logging exception from section 11.12; T the general time measurement interpretation. D-CHECK, D-ROLE and D-ONPREM denote the respective complete Definition digests. This view shows an application record completely in its proposed form; referenced records and manifests are moved out of the view.

```yaml
recordId: A-TIME-CHECK
recordType: application
originSnapshot: M4
procedures: INDEX-PROCEDURES-1
producer:
  runId: derive-time-check
  procedure: DERIVATION-1
sources: []
dependencies:
  - key: meaning
    type: record
    record: T
  - key: exchange
    type: content
    definition: {package: example/connector-project, id: check-status}
    expectedDigest: D-CHECK
  - key: role
    type: content
    definition: {package: example/connector-vocabulary, id: Exchange/sender}
    expectedDigest: D-ROLE
  - key: participant
    type: content
    definition: {package: example/connector-project, id: onprem}
    expectedDigest: D-ONPREM
  - key: new-influences
    type: watch
    scope: bound-model
claim:
  interpretation: T
  subject: {package: example/connector-project, id: onprem}
  context:
    - {package: example/connector-project, id: check-status}
  property: {package: example/connector-vocabulary, id: Exchange/sender}
  result: applies
  statement: onprem determines the timeout period of each attempt of check-status monotonically.
  reason: check-status is an Exchange and its requester is onprem.
```

The sources of the obligation are bound here via T instead of being maintained as a manual second copy. The bases listed concretely are the essential direct sources of this example, not a blanket claim of a complete context list. Further bases needed by the reviewer must also be captured as dependencies; for complete validation, the resolved sources of T, review reports and coverage bindings are required.

For a hypothetical subsequent state M5, the same recordId can be carried over if its direct bases remain unchanged and the complete change check leaves no unresolved influences on this applicability. The Binding then contains targetSnapshot M5, mode reuse, previousBinding as the exact M4 Binding, equal findings for unchanged bases, a reassessed finding with a current influence check for new-influences and the M5 coverage bindings. The original semantic verification remains a verification of its original subject; the new Binding explains its permissible reuse.

If, on the other hand, check-status.requester changes, at least D-CHECK changes. A carry-over attempt for A-TIME-CHECK yields freshness stale for this state; the historical M4 Binding is not changed retroactively. A new application with a changed subject and current content dependencies needs a new recordId; it can continue to use T as a meaning precondition if T's own Binding remains current. If the global logging interpretation changes from L3 to L4, the same principle is to be applied to the applications that depend on it. A carried-over time measurement application does not replace a newly required logging application or its verified exclusion.

#### Minimum validation and remaining work

A later validator must check closed forms, enums, unique keys, referenced contents, source ranges, inventory completeness, exact subjects of verification, dependency/Binding references and status derivation. There are no silent name lookups or fallbacks to older confirmed versions. An old or incomplete Binding can be read as a diagnosis, but not as a confirmation for the current state.

The contract does not yet present a JSON Schema file, an executable query language or an independent review implementation. In particular, digest encoding, the fully machine-readable manifest forms, criteria for verified selection execution and the concrete enforcement of review separation remain to be worked out. The structured check of a later transport format will not create a semantic guarantee of the chosen dependencies or applicability judgments. The YAML example was checked for readability, not against a production index validator or an executed semantic review.

### 11.14 Complete record protocol for M3 → M4

The user commissions the complete instantiation of the data contract for the exception of check-status and a classification of the language state. The following catalog fills in all findings, reviews and Bindings needed in the transition considered, in a symbolic reading representation. The reference names stand for exact content references; they are not invented real SHA values. The protocol assumes an already verified M3 starting index and the successful new reviews described below. These reviews were not executed. This is not a self-contained loadable transport package of the whole project index; unchanged background data of the assumed starting index and the complete inventory manifests are named as available preconditions instead of being presented as new insights.

#### Firmly bound inputs and starting state

| Reference | Filled-in content of this example |
| --- | --- |
| M3 | Snapshot with both Exchanges, requester api for list-documents and onprem for check-status; unrestricted additional logging obligation. |
| M4 | The same set of Definitions, the same package bindings and role assignments; only the global logging statement in platform.body now exempts check-status. |
| P1 | Unchanged indexContract, derivationProcedure and reviewPolicy versions. The example policy requires structural verification of every new record/Binding version and independent semantic or coverage verification of the corresponding new findings. |
| D-EXCHANGE, D-ROLE, D-LIST, D-CHECK, D-API, D-ONPREM | Identical complete content versions of the respectively named Definitions in both states; D-ROLE denotes Exchange/sender in the vocabulary package. Further participant/Command/Channel contracts read are bound in the unchanged context manifest. |
| D-PLATFORM-3, D-PLATFORM-4 | Different complete content versions of the same project Definition platform. Claim no content equality. |
| SOURCES-3, SOURCES-4 | Complete inventories of the source bodies and meaning bases at the respective state; exactly one changed source area. The input/package manifests and original texts are available. |
| CASES-3, CASES-4 | The same two cases considered with the current role assignments: list-documents → api, check-status → onprem. The positive selection of the logging obligation is to be distinguished from this. |
| DELTA-34 | Complete comparison inventory M3 → M4 with the one body change area and its two source versions. |

The six domain starting entries are T, T1, T2, L3, L3-1 and L3-2 from section 11.12. T interprets the monotonic time measurement, T1 and T2 apply it to the two cases. L3 interprets the logging without exception, L3-1 and L3-2 apply it. All have exact M3 Bindings B3-T, B3-T1, B3-T2, B3-L3, B3-L3-1 and B3-L3-2 as well as existing passed record and Binding verifications. The complete background analysis and its M3 coverage are further assumed preconditions.

For the time measurement findings, the direct meaning bases in the example are Exchange, its requesting-party Property and the required concrete case contracts; the global logging source is not a direct content dependency of T to be compared for being unchanged. Its possible influences are assessed via watch and the current influence check. Had an actual finding bound D-PLATFORM-3 directly as a basis, exactly this old record version could not be carried over unchanged: then a newly justified finding would be required, even with the same conclusion. The following reuse applies explicitly to the dependencies stipulated here.

#### Eight new records and three reused domain records

For every new record, originSnapshot M4, procedures P1 and producer {runId: scenario-derive-M4, procedure: P1.derivationProcedure} are set. sources name the exact new source location in D-PLATFORM-4 insofar as the interpretation is affected; the applications bind this provenance via L4. dependencies contain the complete bases named below and, where applicable, watch. The abbreviations denote real ModelRefs or RecordRefs according to section 11.13; the entry type determines the claim structure fully defined there.

| recordId / recordType | claim and concrete basis |
| --- | --- |
| L4 / interpretation | meaning: requester logs the clock source per attempt for all Exchanges except check-status; force: obligation; scope.description: all Exchange cases except the explicitly named exchange; scope.inventory: CASES-4. Source: new global sentence in D-PLATFORM-4. dependencies: content platform=D-PLATFORM-4, content Exchange=D-EXCHANGE, content Exchange/sender=D-ROLE, content check-status=D-CHECK for the resolved exception target, watch bound-model. replaces: L3. |
| L4-1 / application | interpretation L4; subject api; context [list-documents]; property Exchange/sender; result applies; statement: api logs the clock source per attempt of list-documents; reason: this Exchange does not fall under the explicitly named exception. dependencies: record L4, content list-documents=D-LIST, api=D-API, requesting-party Property and further required unchanged case contracts, watch. replaces: L3-1. |
| L4-2 / application | interpretation L4; subject onprem; context [check-status]; property Exchange/sender; result excluded; statement: From L4 no logging obligation follows for onprem in check-status; reason: the exactly resolved exception target is this Exchange. dependencies: record L4, content check-status=D-CHECK, onprem=D-ONPREM, requesting-party Property and required case contracts, watch. replaces: L3-2. |
| I34 / influence | baseSnapshot M3; targetSnapshot M4; changes DELTA-34; considered [T,T1,T2,L3,L3-1,L3-2]; result affects; affected [L3,L3-1,L3-2]; reason: the new restriction changes the log interpretation and its applications, but does not concern the time measurement. Source: both global source versions. dependencies: both fixed comparison manifests and the verified meaning bases. No precondition of an already current T/L4 Binding. |
| CS4 / coverage | purpose sources; inventory SOURCES-4. Every source has a finding: the changed global sentence refers to L4; unchanged source areas refer, after a freshness check, to existing or, if necessary, newly bound background findings. No source is skipped because of missing links. This presupposed complete source coverage is to be supplied along with it, not proven by the eight foreground records alone. |
| CT4 / coverage | purpose cases; inventory CASES-4; results: list-documents → assessed with [T1], check-status → assessed with [T2]. reason in each case: the logging exception does not change the monotonic time measurement. |
| CL4 / coverage | purpose cases; inventory CASES-4; results: list-documents → assessed with [L4-1], check-status → excluded with [L4-2]. reason: the explicit exception concerns exactly the second case and exclusively L4. Both cases remain in the coverage. |
| CD34 / coverage | purpose changes; baseSnapshot M3; inventory DELTA-34; results: the one change area → assessed with [I34]. reason: earlier and new reach including interactions assessed; no further change areas in the complete inventory. |

For all four coverage records, the complete respective inventories form the direct input basis. Their related RecordRefs describe claims considered and are not mutual freshness preconditions. The exactly bound source/inventory evidence is in sources or dependencies respectively. For these upstream certificates, watch is not replaced by a demand for their own already completed clearance: the direct independent verification at the fixed state from section 11.13 applies.

T, T1 and T2 receive no new recordIds and no changed originSnapshots. Their content objects and original reviews are kept unchanged. L3, L3-1 and L3-2 remain historical M3 findings. They receive no successful M4 carry-over binding. In particular, L3-2 is not changed in place from applies to excluded.

#### Assigning the new reviews completely

All following intended reports have snapshot M4 and procedures P1. reviewer.runId distinguishes the structural validator run scenario-validate-M4 from the independent semantic review run scenario-review-M4. The names are explicitly scenario labels and no evidence of actual independence. contextManifest contains the original sources named in each case, both fixed model states insofar as needed and the findings concretely considered. evidence binds the source comparisons, structural evidence or independently assessed applicability reasons. verdict pass is here in each case the assumed successful assessment, not an executed result.

| Review IDs and exact subject | check | Filled-in verification reason and context |
| --- | --- | --- |
| RS-L4, RS-L4-1, RS-L4-2, RS-I34, RS-CS4, RS-CT4, RS-CL4, RS-CD34; each the record of the same name | structural | Closed record form, exact source/target references, inventories, dependency references and provenance are valid for the exact new version. |
| RM-L4 → L4 | semantic | The complete source comparison limits only the additional logging. check-status is identified unambiguously; the time measurement obligation is not changed. |
| RM-L4-1 → L4-1 | semantic | list-documents lies outside the exception; api is still requester there. The positive application fits the concrete contracts. |
| RM-L4-2 → L4-2 | semantic | The exclusion applies to onprem only in check-status and only from L4; it claims neither a logging prohibition nor a time measurement exception. |
| RM-I34 → I34 | semantic | The change claims no further exception. Old and new reach were verified on the basis of the sources and relevant counter-rules. The reusability of T/T1/T2 remains limited to their unchanged direct bases. |
| RC-CS4 → CS4 | coverage | All elements of SOURCES-4 have matching findings; unresolved background areas would prevent this pass report. The existing M3 coverage alone is not sufficient. |
| RC-CT4 → CT4 | coverage | Both current Exchange/requester cases are considered for T; neither drops out of the selection through a role or set change. |
| RC-CL4 → CL4 | coverage | The base set contains both cases; positive application and explicit exclusion are completely and correctly distinguished. |
| RC-CD34 → CD34 | coverage | DELTA-34 comes from the complete state comparison and is completely assessed. |

The original T/T1/T2 reviews remain evidence of their original record versions. No fictitious new semantic T reviews arise merely through a change of the snapshot name. For the carry-over, the new influence/coverage reports and the mechanically verified unchanged bases are drawn upon. Every fail or open at a required place would prevent the described successful end state.

#### Eleven new Bindings and their dependency checks

Every new Binding has targetSnapshot M4, procedures P1 and a non-empty reason from the following table. For every dependency.key of the bound record, exactly one dependencyChecks result is entered, with evidence pointing to the complete content comparison, the current predecessor finding or the named new assessment. The current equality of the direct bases is not inferred from an unchanged file label.

| bindingId → record | mode / previousBinding | dependencyChecks and coverageBindings |
| --- | --- | --- |
| B4-I34 → I34 | original | Comparison manifests and exact source versions verified; direct evidence from M3/M4. coverageBindings empty: the upstream influence check does not need its own clearance. |
| B4-CS4 → CS4 | original | Source inventory M4 completely verified. coverageBindings empty; direct self-contained coverage verification. |
| B4-CT4 → CT4 | original | Both case mappings verified. coverageBindings empty; the associated semantic evidence is in the source/influence verifications and record verifications. |
| B4-CL4 → CL4 | original | Both cases including the exclusion verified. coverageBindings empty; original source L4 and concrete assignment are in the review context. |
| B4-CD34 → CD34 | original | Complete delta and I34 verified. coverageBindings [B4-I34]; no back edge to B4-CD34. |
| B4-T → T | reuse / B3-T | Direct contents and exact prior findings equal; new-influences reassessed with I34 and CD34. coverageBindings [B4-CS4,B4-CD34]. Reason: the same time measurement interpretation, justifiably unaffected by the change. |
| B4-T1 → T1 | reuse / B3-T1 | Exact interpretation T usable via B4-T; direct case/Property/participant bases equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CT4]. Reason: api remains obligated for list-documents. |
| B4-T2 → T2 | reuse / B3-T2 | Exact interpretation T usable via B4-T; direct case/Property/participant bases equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CT4]. Reason: onprem remains obligated to time measurement for check-status. |
| B4-L4 → L4 | original | New source version and all direct content bases equal against their originSnapshot M4 values; watch reassessed with I34. coverageBindings [B4-CS4,B4-CD34]. Reason: verified new interpretation with a limited scope of validity. |
| B4-L4-1 → L4-1 | original | Interpretation L4 usable via B4-L4; concrete new record bases equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CL4]. Reason: positive logging obligation for api in list-documents. |
| B4-L4-2 → L4-2 | original | Interpretation L4 usable via B4-L4; concrete new record bases equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CL4]. Reason: verified exclusion of onprem in check-status from this obligation. |

For each of these eleven Bindings, a structural intended report RB-<bindingId> arises with the exact BindingRef as subject, snapshot M4, procedures P1, check structural and assumed verdict pass. Its contextManifest contains the record, the exact old Binding in the case of reuse, dependencyChecks, current coverage/influence bindings and the required independent semantic evidence. reason reads in each case: complete dependencies and permissible evidence carry-over verified at the target state. An equal finding on a presupposed record presupposes its current permissible Binding; it is not merely its character string that is compared.

The foreground considered thus contains **eight new records, eleven new Bindings and 27 new intended reviews**: eight record structure checks, four semantic reviews, four coverage reviews and eleven Binding structure checks. Added to this are the background findings presupposed as existing, which are either unchanged or to be updated on their own. This is a concrete simple breakdown of the example policy, not an obligation to 27 separate agent runs or a general cost statement. Reports can come from the same matching independent runs; their exact subjects of verification remain separate.

The M4 envelope references as requiredCoverage [B4-CS4,B4-CT4,B4-CL4,B4-CD34]. The three current time measurement Bindings and three current log Bindings form the domain excerpt considered. Only with actually confirmed verifications and complete background coverage would confirmed/current/usable be derived from this. Here the successful reviews are assumed; therefore this protocol is no current product evidence. The M3 Bindings remain valid historical Bindings for M3. A carry-over attempt of L3/L3-1/L3-2 to M4 would be stale because of changed source/interpretation preconditions; this diagnosis does not change the old Bindings.
