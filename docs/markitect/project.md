# Canonical Markitect model

Generated from acquired canonical v1 source bytes. This view is not execution or acceptance evidence.

## markitect

Canonical model digest: `sha256:e90871ba28bb91188686e85ea7bb3806bc17648e305a6e59f5246615d784fc10`.

Manifest: `.markitect/model.yaml` (original bytes `sha256:d40feb9e02db275aed89d53b0a88849e463acaca076c8fa70fad38b367a68559`).

### documentation/current-guidance

Kind: `vocabulary/Rule`. Source: `.markitect/model/documentation/current-guidance.yaml` (original bytes `sha256:524b734a619e408c36d646548616cc9718bdec6be5a89c920c81bb415d3c73a0`).

Current guides describe the implemented command surface, requested stages, exact source inputs and incomplete execution honestly. Preserve accepted concepts, user source documents and historical evidence in their original scope. Check links and generated views separately from semantic assessment.

### examples/independent-oracles

Kind: `vocabulary/Rule`. Source: `.markitect/model/examples/independent-oracles.yaml` (original bytes `sha256:38384fe9a5d2944692cb2ebac4f9734e6c02579b8b53b3470de62bcbbc367848`).

Keep canonical Library M1/M2 rules and the independent oracle unchanged. Treat controlled fixture and source-smoke results as bounded mechanics evidence. Retire obsolete fake product engines and command consumers together; synthetic bookkeeping never proves delegated realization or provider performance.

### markitect/core/Kind

Kind: `markitect/core/Kind`. Source: `.markitect/model/markitect/core/kind.yaml` (original bytes `sha256:94c9b4aed3c0acdedac13375d56039eb18554b2b28686a7becc3642fd81f2411`).

Declares a type. properties names the permitted fields of the spec of its definitions. The body of a type definition describes their shared meaning and applicable stipulations.

Fields:

- `properties`: [`markitect/core/properties`]

### markitect/core/Property

Kind: `markitect/core/Kind`. Source: `.markitect/model/markitect/core/property.yaml` (original bytes `sha256:46a615230b96ff6ceb6dfc4582fff98d9b8678ee9f81b725def9f851ec8a1e8b`).

Declares a field contract with value type and cardinality. The body describes the meaning of its values or relationships. For type object, properties names the permitted fields of the object value.

Fields:

- `properties`: [`markitect/core/maxCount`, `markitect/core/minCount`, `markitect/core/name`, `markitect/core/properties`, `markitect/core/target`, `markitect/core/type`, `markitect/core/values`]

### markitect/core/maxCount

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/max-count.yaml` (original bytes `sha256:7965ac2a6c4d014f9279fccb7899a4111649d78c3e6fdc1b2c2c081b190b8049`).

Maximum number of present values. The value -1 permits any number of values within the tool limits. Other values must be at least minCount. Exactly 1 denotes a single value; other maximum values denote a list.

Fields:

- `maxCount`: 1
- `minCount`: 1
- `name`: "maxCount"
- `type`: "integer"

### markitect/core/minCount

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/min-count.yaml` (original bytes `sha256:21b55a98cea8e61741cd569a61b35d464048751eaeb1da607d6e8da7aa005fee`).

Non-negative minimum number of present values.

Fields:

- `maxCount`: 1
- `minCount`: 1
- `name`: "minCount"
- `type`: "integer"

### markitect/core/name

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/name.yaml` (original bytes `sha256:0026b9de7a20f82c0b69a90fdbf1679369f69ff7a21c2227175b5b956c68da26`).

Optional explicitly deviating non-empty field name. If omitted, the complete local ID of the Property applies as the field name. The effective field name must be unique within every contract that uses it.

Fields:

- `maxCount`: 1
- `minCount`: 0
- `name`: "name"
- `type`: "string"

### markitect/core/properties

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/properties.yaml` (original bytes `sha256:fb82b590b233aea97e905bfa582cc9d8b793c839361b194be2fe33f2d0f566bb`).

References the permitted field contracts of the object described here. The effective field names must be unique within this contract.

Fields:

- `maxCount`: -1
- `minCount`: 0
- `name`: "properties"
- `target`: `markitect/core/Property`
- `type`: "reference"

### markitect/core/target

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/target.yaml` (original bytes `sha256:91fa530485a5dc91d8a52f2de3b58eb05f426bb5dc3d823e44352aaf01076ad4`).

The Kind required of the referenced definitions. For type reference, this entry is required. For other value types, it is not permitted.

Fields:

- `maxCount`: 1
- `minCount`: 0
- `name`: "target"
- `target`: `markitect/core/Kind`
- `type`: "reference"

### markitect/core/type

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/type.yaml` (original bytes `sha256:9cbe3ee9abfa60915edf65f626934e046c4ae0a377a044ebfb934702c9b194cf`).

The value type of this Property, as understood by the Core.

Fields:

- `maxCount`: 1
- `minCount`: 1
- `name`: "type"
- `type`: "enum"
- `values`: ["boolean", "enum", "integer", "number", "object", "reference", "string"]

### markitect/core/values

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/values.yaml` (original bytes `sha256:f6de4bd1609a1ecf09777a27e1dffa9eebc5ff9db74058bb070b00b3926f7791`).

Permitted string values of an enum Property. For type enum, a non-empty list of distinct values is required. For other value types, this entry is not permitted.

Fields:

- `maxCount`: -1
- `minCount`: 0
- `name`: "values"
- `type`: "string"

### product/ports-and-adapters

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/ports-and-adapters.yaml` (original bytes `sha256:c3b60d25b03ebac270487b5697de1d39c334edbbd453800ca1c816a031358b5d`).

Keep model, snapshot, organization and realization values independent of execution and infrastructure. Project owns structural acquisition, checked model-derived contexts and immutable candidate validation. Run consumes Project and organization directly and owns the complete closed hierarchy over consumer-owned Session, Git, check and record ports. Workflow selects fully closed Runs through its Run-only Executor boundary. Apply owns command preflight, the original shared allowance and independently reopened stored-candidate publication. Outer runtime configuration owns acquired static native/check profile metadata; it does not resolve model intent. Concrete adapters implement physical acquisition and protocols. Bootstrap composes actual services. All maintained Go code belongs to one root module and an explicit architecture responsibility; obsolete execution engines and their consumers retire together.

### product/adapters/agent/closed-agent-metadata

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/agent/closed-agent-metadata.yaml` (original bytes `sha256:41a6dfd7742793234d55c6b5ba370246d603b6d68946808117b413b6764e8017`).

Prepare only explicitly selected source-bound Agent profile metadata in closed formats. Bind the immutable runtime image identity and the actual whole metadata file bytes. The configured container engine may inspect the image and copy build metadata from a created, never started container; preparation never reads a host provider executable, probes a provider version, exports a schema, starts an Agent Session or invokes a model. Preserve recorded CLI and agentd versions, diagnostics and observation limits. A build-recorded schema digest is an assertion, not an observed complete schema inventory. Instructions come only from captured inputs. A static pass establishes neither runtime nor provider capability.

### product/adapters/checks/closed-check-metadata

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/checks/closed-check-metadata.yaml` (original bytes `sha256:e042543f85fbefabcd03a89e0bc1d6cf1698465d4552ddc2e10a72336bdc8307`).

Parse only explicitly acquired pinned container Check profile metadata in closed formats. Preserve literal arguments, fixed environment policy, finite resource and output limits, exact image/executable identities and source bindings. Static preparation starts no check and establishes no execution success. Execute each declared check against the exact fixed candidate on an independently verified disposable copy and empty private home; retain actual image, tree, start, exit, exit code, output, timeout and closure observations. Unknown or uncertain execution and cleanup cannot count as a passing check.

### product/adapters/repository/repository-boundaries

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/repository/repository-boundaries.yaml` (original bytes `sha256:9bc0d1bf7eb6fe3456baa22981fed765c0263d64d2142e59fb79013735362651`).

Capture the complete actual repository census and selected original inputs with exact paths, modes and bytes. Working and fixed previews grant no publication guard; committed capture requires clean HEAD and retains its comparison guard. Resolve a symbolic revision once, retain the full identity, reject redirects and changed directory identity, and never replace historical sources with current external bytes.

### product/apply/read-only-preparation

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/apply/read-only-preparation.yaml` (original bytes `sha256:d549d34005d1b52a8e9169d6d46cc5d712942824b22b8018e7ee5707ae4f5547`).

Acquire explicitly selected model and control inputs through Project-owned ports. Prepare the Model check, registry, complete strict file accounting, authored ownership, scope, Manager instructions, declared checks and static contexts from one complete capture; recapture rather than extend an old guard. Preserve original bytes, source provenance and authored absence. Known invalid input differs from incomplete acquisition. Project does not decode runtime profiles, resolve strategies, roles or budgets, start Sessions, execute checks, allocate working copies or publish artifacts.

### product/cli/honest-read-only-interface

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/cli/honest-read-only-interface.yaml` (original bytes `sha256:cf9e7da39dd973491f091694a6409b0f4afac89686caaea03f52a5265a1a4f1f`).

Expose check and context over actual Project preparation, with structural preparation as the default requested stage and explicitly selected runtime metadata as an additional stage. Preserve actual statuses, sources and inclusion reasons. The generated model view has its own source/target-bound guarded write. Apply binds committed inputs and an isolated non-protected target branch before Manager work, uses complete fresh Assessment, Implement and Verify Runs with one shared finite allowance and publishes only independently reopened stored Verify approval after every Run resource closes. A Run is closed: the command line offers no answer command or preloaded owner decision. Invocation configuration is closed and requires explicit finite limits and normal account credential sources. Report actual technical and semantic results, cleanup, completed publication paths and readback; retained reports and printed JSON use the same final projection. Usage, known failure and incomplete work have distinct exit statuses. Source composition, native provider qualification and owner acceptance remain separate claims.

### product/documents/guarded-model-view

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/documents/guarded-model-view.yaml` (original bytes `sha256:76491d1fd66ab27815d4f311d56737bbbe035b60ca69870eaa8ee12fd2566057`).

Render a deterministic readable view from actual compiled canonical v1 Definitions. Keep canonical model digest, rendered byte digest and source/target-bound preview digest distinct. Write only the reviewed generated target under an independent documentation guard after source and target rechecks; never overwrite model, control or instruction inputs, and retain observed partial effects on failure.

### product/model/language-v1

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/model/language-v1.yaml` (original bytes `sha256:45b618528da0a7bce07e1760ca4e9233a201df718b598b3b4ed34e302f755259`).

Check the complete canonical language v1 from explicitly supplied immutable sources. Preserve exact identities, values, links, provenance and deterministic model digests. Reject invalid or incomplete language constructs without filesystem, Git, process, provider or legacy product dependencies. The fixed Core definitions retain the normative contracts.

### product/organization/delegated-context

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/organization/delegated-context.yaml` (original bytes `sha256:76be42ec9a6b960c89b7f780a8eed525fdc4309d5d451b7dcec1409c06dc6084`).

Derive one ordinary Root and one Manager per local canonical Package from authoritative checked facts. Every Manager, including empty and intermediate Packages, participates in every Assessment, Implement and Verify Run; no Agent prunes the required hierarchy by relevance. Deliver each Manager's selected Definitions and contracts in full, with original bodies, typed values, resolved links, provenance and ownership. Label foreign context and Root's complete project inventory clearly. Include declared instructions, owned areas, scoped checks, mandate and mandatory Findings. Refuse mandatory-context overflow explicitly rather than silently truncating it. A Manager realizes its own responsibilities and coordinates, reviews and integrates its direct children. Parents supply foundations and actual task bases, inspect complete fixed child diffs, return gaps to their owner and integrate accepted commits. They do not subtract accepted child paths from later parent review or excuse a child's obligations. Managers communicate directly through correlated Run-local delegation, questions and replies, including crossed questions and later questions to completed Managers in the same conversation. No question crosses Root to the invoking AI or owner, and no preloaded owner decision supplies missing intent. Answers clarify the fixed model and never override it. Material contradictions or missing canonical policy are model blockers; ordinary implementation choices stay with their responsible Manager. Root reports confirmed model defects without publication so intent can be corrected outside the Run. Apply changes artifacts only for modeled obligations, required checks or necessary consequences of that work. Preserve conforming artifacts even when another design is preferred. A Definition is realized by its owner's artifacts; other areas use that realization rather than rebuilding it. Parents settle cross-area interactions within accepted intent. Independent fresh Verify judges the fixed integrated candidate and the actual assertions and limits of its tests. Concrete Findings survive review and closure and reach the responsible correction Run. Green totals, provider completion and unsupported self-approval do not establish semantic completeness. Required late findings invalidate affected acceptance. Shared Notes are attributed noncanonical hints, never authority or hidden requirements. Generated prompts express the bound model and mandate without becoming a second owner of binding project rules. Complete context delivery does not prove that an Agent followed it. The confirmed-run-history Rule owns lifecycle, mandate decisions, finite limits and durable results.

### product/realization/artifact-accounting

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/realization/artifact-accounting.yaml` (original bytes `sha256:3267a354d09c55f6e7b36f015387c452a5379e61738e55337acd1e9c630344ef`).

Account for every observed regular file exactly once as a validated Markitect input, a registered artifact or an explicitly ignored file; Markitect inputs are the immutable authority inputs and the owner-writable local registries, and neither may be registered or ignored as an artifact. Ownership is the declared set of repository-relative directory and exact-file areas, each owned by Root or one local Package; the most specific area owns a path, equal specificity and files without an owner fail, and a broad area never absorbs a more specific one. Each Manager maintains only its own local registry of exact-file or directory entries with Definition links, forms and reasons; a directory entry covers only its registrant's eligible files below it, the most specific entry registers a file, and no file has two effective registrations. Missing paths, unexplained artifacts, dangling links, duplicate accounting, wrong-owner registrations and changed authority inputs block, with machine-readable diagnostics addressed to the Manager that must act. Only the Orchestrator assembles the read-only derived registry view, from checked local versions, and records which versions it includes. Enforce writing ownership in every Agent's container: its own areas and its own registry writable, protected inputs and other owners' areas, deeper directories and exact files read-only; refuse an ownership shape that mounts cannot enforce instead of trusting prompts. Accept a basis or result only after the File check of the submitter's stable area composed with the accepted stand, so disjoint Managers work and finish concurrently without a mixed live view. Candidates preserve source identity and observed findings; pure accounting values do not schedule or publish work.

### product/run/confirmed-run-history

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/run/confirmed-run-history.yaml` (original bytes `sha256:69b33fd761a27de6ea6902a06b75889bfbffe45ceccb2e0fb7a93af04ab9df1d`).

Every Assessment, Implement and Verify executes the complete model-derived Manager hierarchy, including Root, intermediate and empty Packages, in fresh Run environments. Project intent and its complete input inventory remain fixed; an invalid or incompletely acquired input starts no Manager work. Before ordinary work, open one fresh environment and working copy per Manager and one Git Workspace Service for the Run. Run has no domain Passes or Turns. Every Manager receives its own selected context, mandate, bound candidate, concrete findings and direct tools. Managers delegate to their direct children, remain available for questions, inspect complete fixed child diffs, review exact commits and integrate accepted work upward through actual Git history. In Implement every submitted child candidate is fixed and reaches its parent with its deterministic check result, also when the check fails; an invalid candidate cannot be approved or integrated; invalidity refers to the checks that apply to the fixed child state, and integration still outstanding elsewhere is not invalidity. A child can run the same check on its current copy before submitting. Questions and replies remain inside the Run and follow the tree: a Manager asks only its direct parent or a direct child, sibling coordination goes through the common parent and cross-level coordination step by step through the Managers in between. Correlate actual incoming questions and answers, support crossed waits and keep completed Managers reachable in their same ordinary conversation. No question crosses Root to the invoking AI or owner. Required late findings invalidate affected acceptance; an answer alone does not change an already fixed Git handoff. A judgment belongs to the assignment and the assessed state, not to a message. Parent rejection or correction, new mandatory findings, a late adverse child finding or a change of the assessed state invalidate it; a message that only handles questions or answers does not, also before the first judgment. The end of a provider response is not a completion attempt, and Run does not demand a judgment merely because a response ended; completion validity is checked when the Manager judges and before Run relies on an earlier judgment (Root completion). The host keeps an explicit assignment state per Manager and states the concrete reason when a completion is not valid. Harness text and messages carry role, rules, context, task and answer format, not host process states. Correction stays with the responsible Manager and uses concrete findings and the actual integrated parent baseline. Parent integration never subtracts previously accepted child paths from the complete diff. Actual delegation bases, fixed results, parent before/after commits, accepted child commits and judgments survive closure. Implement submits a structurally checked candidate and required scoped check results; it cannot approve publication. Assessment and Verify judge the immutable input independently in fresh Runs, and scratch edits never become product. Root checks the complete declared suite on the final fixed candidate. Required check failures cannot be waived by a semantic approval; actual startup, acquisition or observation failures remain technical failures. Managers assess material conditions, exceptions, interfaces and composed workflows from the bound model, challenging the actual work and evidence of their children. Green totals and provider completion do not establish semantic completeness. Reject and model blocker require concrete findings; no finding quota or custom evidence-mapping protocol is required. One finite outer deadline and message allowance bound the invocation, including correction Runs; Manager count is explicitly bounded before environments start. Exhausted capacity, cancellation, incomplete observations or uncertain closure never become success. Cleanup and persistence have separate finite bounds and grant no further productive work. Candidates retain their exact commit, tree and complete ancestry in self-contained Git storage. Verify independent reopening before exposing the stored reference. Mandatory terminal results include the exact input reference, binding, technical and semantic outcomes, findings, handoffs, checks and actual cleanup observations. Return no live resource handle and confirm terminal storage by read-back after resource closure. Shared project Notes persist across Runs, retain their originating Run, Manager and context and preserve concurrent appends. Managers read them in bounded, stably ordered pages within the tool transport limit, without silent truncation. They are operational memory, not authority, assignments or model decisions. Assessment approves model readiness, including an unrealized or outdated product; it validates fixed structural inputs without requiring already-passing realization commands. Assessment Approve always enables a complete Implement, even when its complete captured Census remains unchanged. Assessment Reject or a model blocker stops without publication. Verify judges the exact fixed realization; its Reject carries mandatory Findings and the exact candidate into fresh Implement and Verify Runs under the original allowance. Apply alone composes Assessment, Implement, fresh Verify and bounded corrections and publishes only the exact stored acceptance after every Run resource has closed and target bytes have been read back. Standalone Run completion is not Apply, publication, provider qualification, release or owner acceptance.

### product/runtime/container-runtime

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/runtime/container-runtime.yaml` (original bytes `sha256:7346c0b835b5b08460b786ad1632385b2e176fedae0f8e9243702f82b6ef78cc`).

Every Run opens one fresh private Manager environment, working copy, provider home and conversation per required Manager, and one owned Git Workspace Service. Assessment, Implement and Verify use separate fresh Run environments; no Manager or Git service survives into a correction Run. Keep each Manager available for ordinary questions and renewed tasks until that Run closes. Native Managers and project checks execute in hardened Linux containers from explicitly prepared immutable engine image IDs. Give a Manager only its own private copy through an owned named-volume subpath; never synchronize a second writable host mirror or overlay other owners' paths. Responsibilities and reviewed handoffs govern work; per-write ACLs and custom evidence protocols are not prerequisites. Run as non-root with a read-only root filesystem, no capabilities or privilege escalation, and explicit finite CPU, memory, process and temporary-space limits. A narrow fixed initialization of a newly created empty owned volume may establish its non-root ownership; it grants no product work. Refuse mismatched images, unsafe mounts or incompatible frame protocols. markitect-agentd starts the native provider, relays framed streams and Run tools, and reports actual exits without owning model policy. Provision dedicated normal-account input privately into each Manager home; never put secrets into arguments, environment, transcripts, Notes or candidate bundles, and never share a mutable host login home. Codex has no API-key route. Provider support requires its own actual native evidence. Declared project checks use independent exact input and empty-home copies, preserve bytes, executable modes and required empty directories, and retain actual image, start, exit, timeout and closure observations. Publication reconstructs the stored candidate with host Git after Run closure and materializes its check copies independently; it never borrows a closed Run volume or Agent. Retain candidates before complete bounded resource closure. Persist and read back mandatory terminal results after closure, including actual cleanup observations and any uncertainty. Observe the end of every owned process, container, home, temporary copy and volume, including partial-start failures. Uncertain cleanup, acquisition, storage or transport blocks success and publication. The confirmed-run-history Rule owns the shared outer deadline, message allowance and closed terminal result.

### product/snapshot/fixed-values

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/snapshot/fixed-values.yaml` (original bytes `sha256:26588b48ed9b6e0f28a946f4ea63b86eecff58b2ef0deece8899d5289e0a88b5`).

Represent exact repository-relative paths, modes, immutable bytes, trees and deterministic content identities. Tree and content digests remain distinct from canonical model digests. Invalid spellings, collisions and unsupported entries fail explicitly; this package performs no repository I/O.

### repository/intent-and-evidence

Kind: `vocabulary/Rule`. Source: `.markitect/model/repository/intent-and-evidence.yaml` (original bytes `sha256:911ecd62e7e0659aebc1fcac712474584734ea7d3c2d5f1b17e8a09e36024830`).

People maintain the canonical intended world. Artifacts realize its obligations. One ordinary root and one Manager per local Package organize delegated realization with independent review, integration, verification and guarded Apply. A committed model binds the specification for its revision; source checks and reports do not authenticate human acceptance or establish runtime readiness.

### repository/learning-driven-workflow

Kind: `vocabulary/Rule`. Source: `.markitect/model/repository/learning-driven-workflow.yaml` (original bytes `sha256:92548c9c56e72b8d54d69883ca366df1a8363a15cff1ed8ebcc0c7e4c8ef8f3b`).

Develop Markitect through explicit learning and complete user workflows. Separate observed facts, owner reports, assistant assessments, recommendations and accepted decisions. Similar independent-looking designs are evidence of conceptual stability, not proof that the method works.

Before a substantial redesign, record briefly in the existing owning design or decision document: the new observation, the prior assumption it challenges, the resulting design change, and the observable result that would demonstrate improvement. An accepted foundation gets a coherent period of actual use. Keep the current complete-workflow target stable; fix concrete failures, missing accepted obligations and necessary blocked steps before unrelated generalization. New evidence can justify a bounded correction or an owner decision; this is not a ban on necessary redesign.

Investigate a consequential uncertain technical boundary early with the smallest useful real probe before dependent design or implementation hardens. Record its observations and limits, and preserve separate finite authority for provider runs, studies, releases and deployments. Independent design review examines whether each added mechanism addresses an observed failure or accepted obligation and whether a simpler approach would serve it. Existing quality, review, evidence and write safeguards remain in force.

Evaluate the Agent process against ordinary delegated coding practice. Prefer the smallest process that improves trust through model intent, clear responsibilities, independent perspectives and parent integration. Do not substitute expanding prompts, mandatory evidence fields or completeness declarations for semantic judgment. Review removals as well as additions when protocol work obstructs the complete workflow. Separate an accepted guideline from proposed contract changes and their actual implementation; technical supervision failures are not evidence that semantic review failed.

At design and delivery milestones, the responsible chat gives the owner a short workflow-oriented account: what the owner can now do, what Markitect takes over, where human intervention remains, and which next observation could change the assessment. Derive this account from existing sources and exact evidence; do not create a parallel status system, mandatory periodic automation or another standing approval gate. Keep one current owner per knowledge subject, date historical status and link to current owners.


### tooling/maintenance-boundaries

Kind: `vocabulary/Rule`. Source: `.markitect/model/tooling/maintenance-boundaries.yaml` (original bytes `sha256:562ee375e5d23a79f3d25473ae1b4f7a2c48c66c15fb332e6d85ce9d0c630539`).

Maintain strict root-layout source packaging, deterministic release archives, licenses, distribution and guarded installation as independent maintenance services. Reject unsupported files, redirection, nested modules and archive collisions. Keep required historical release evidence and the named archive refs. Retire public release pages and downloads only on explicit owner instruction; retain their source/asset identities and attestations without implying current availability. A source smoke is not release, installation, provider or Apply acceptance.

### tooling/gates/architecture

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/architecture.yaml` (original bytes `sha256:e9eecbe38438492123cf86d681312762d9768fa6e1797840f391181a610abfa4`).

Run the declared architecture source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "go run ./cmd/markitect-check-architecture --repo ."

### tooling/gates/coverage

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/coverage.yaml` (original bytes `sha256:6e357ab6d707aa7b6b5a94b085185bf4e39b1eed050dd5f670b52e653e31f299`).

Run the declared coverage source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "go run ./cmd/markitect check --repo ."

### tooling/gates/go-build

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/go-build.yaml` (original bytes `sha256:e9b5e5c24fa4949d353cb81608737ce1162a08e65edd7d0c18d34d8849a60aed`).

Run the declared go-build source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "go build ./..."

### tooling/gates/go-tests

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/go-tests.yaml` (original bytes `sha256:559d1963174e07d8eddcf529453e7fbefaeb02042d26c2ee953a953ef3771a14`).

Run the declared go-tests source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "go test ./... -count=1"

### tooling/gates/go-vet

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/go-vet.yaml` (original bytes `sha256:82a72081f52138d8a7b854ea6ede2b1cae9a2048320e9c63d4c45dd71518e325`).

Run the declared go-vet source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "go vet ./..."

### tooling/gates/library-oracle

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/library-oracle.yaml` (original bytes `sha256:3007de0954ce6dfab6cca0a5e44aeeed02e8afadb900ad9e75d600f1177651ff`).

Run the declared library-oracle source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "python -B -m unittest discover -s tools/library-oracle"

### tooling/gates/modules

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/modules.yaml` (original bytes `sha256:e47488caa558b1be454decdec52f244402007baca152d59c6d39e0a122ad8bf1`).

Run the declared modules source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "go run ./cmd/markitect-check-modules --repo . --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config"

### tooling/gates/source-checks

Kind: `vocabulary/Rule`. Source: `.markitect/model/tooling/gates/source-checks.yaml` (original bytes `sha256:df903ccfaefbaab4e3fd55e27aeec6cd58c2e7fd979f71555a61d70e76d49320`).

Check exact package ownership and permitted imports, repository coverage, generated view freshness, module pins and literal pipeline commands against actual canonical preparation. Tests exercise failures and finite boundaries. No disabled gate, fake success or compatibility branch may make an unimplemented replacement appear complete.

### vocabulary/Check

Kind: `markitect/core/Kind`. Source: `.markitect/model/vocabulary/check.yaml` (original bytes `sha256:22926aeabe2064c6baaba5629a851138b02f5d44c921f7fc82af00c038e2054e`).

A required source maintenance check. Its command is a literal command string for contributor and pipeline wiring; declaring it never executes it.

Fields:

- `properties`: [`vocabulary/command`]

### vocabulary/Rule

Kind: `markitect/core/Kind`. Source: `.markitect/model/vocabulary/rule.yaml` (original bytes `sha256:f2e42005ea9542249085f36526ea144039c178370d88bd0f8bd6f9f049627fe5`).

States intended behavior or an invariant in its named scope. Structural checks establish references and value contracts; independent assessment establishes whether artifacts satisfy the natural-language obligation.

Fields:

- `properties`: []

### vocabulary/command

Kind: `markitect/core/Property`. Source: `.markitect/model/vocabulary/command.yaml` (original bytes `sha256:ecac1893689a487599ff7157422d6e9babd62c4c5109ad8ac93f146a81c15a9c`).

The exact literal maintenance command required by this Check.

Fields:

- `maxCount`: 1
- `minCount`: 1
- `name`: "command"
- `type`: "string"
