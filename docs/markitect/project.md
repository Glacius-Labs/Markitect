# Canonical Markitect model

Generated from acquired canonical v1 source bytes. This view is not execution or acceptance evidence.

## markitect

Canonical model digest: `sha256:e675437b02d093eda69e7d1fcdd3cedacca8451961a67c278bec416f4aaf232d`.

Manifest: `.markitect/model.yaml` (original bytes `sha256:d40feb9e02db275aed89d53b0a88849e463acaca076c8fa70fad38b367a68559`).

### documentation/current-guidance

Kind: `vocabulary/Rule`. Source: `.markitect/model/documentation/current-guidance.yaml` (original bytes `sha256:524b734a619e408c36d646548616cc9718bdec6be5a89c920c81bb415d3c73a0`).

Current guides describe the implemented command surface, requested stages, exact source inputs and incomplete execution honestly. Preserve accepted concepts, user source documents and historical evidence in their original scope. Check links and generated views separately from semantic assessment.

### examples/independent-oracles

Kind: `vocabulary/Rule`. Source: `.markitect/model/examples/independent-oracles.yaml` (original bytes `sha256:38384fe9a5d2944692cb2ebac4f9734e6c02579b8b53b3470de62bcbbc367848`).

Keep canonical Library M1/M2 rules and the independent oracle unchanged. Treat controlled fixture and source-smoke results as bounded mechanics evidence. Retire obsolete fake product engines and command consumers together; synthetic bookkeeping never proves delegated realization or provider performance.

### markitect/core/Kind

Kind: `markitect/core/Kind`. Source: `.markitect/model/markitect/core/kind.yaml` (original bytes `sha256:b8694323468d4d8c90e64740b39a10045fd21ae759066c4d99c9bc1d82003ebe`).

Deklariert einen Typ. properties benennt die erlaubten Felder der spec seiner Definitionen. Der body einer Typdefinition beschreibt deren gemeinsame Bedeutung und geltende Festlegungen.

Fields:

- `properties`: [`markitect/core/properties`]

### markitect/core/Property

Kind: `markitect/core/Kind`. Source: `.markitect/model/markitect/core/property.yaml` (original bytes `sha256:be6149acf2d5ebcca9e18bad13932ca3be9bb6d838ac5a142dd83d7bc6143590`).

Deklariert einen Feldvertrag mit Werttyp und Kardinalität. Der body beschreibt die Bedeutung seiner Werte oder Beziehungen. Bei type object benennt properties die erlaubten Felder des Objektwerts.

Fields:

- `properties`: [`markitect/core/maxCount`, `markitect/core/minCount`, `markitect/core/name`, `markitect/core/properties`, `markitect/core/target`, `markitect/core/type`, `markitect/core/values`]

### markitect/core/maxCount

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/max-count.yaml` (original bytes `sha256:206aa732af928dafc01b6949eca98db1fb5f9e3a001046d7adf889a1a1303789`).

Höchstanzahl vorhandener Werte. Der Wert -1 erlaubt beliebig viele Werte innerhalb der Werkzeuggrenzen. Andere Werte müssen mindestens minCount entsprechen. Genau 1 bezeichnet einen Einzelwert; andere Höchstwerte eine Liste.

Fields:

- `maxCount`: 1
- `minCount`: 1
- `name`: "maxCount"
- `type`: "integer"

### markitect/core/minCount

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/min-count.yaml` (original bytes `sha256:d2f2c064071c75f3f4b41cef16bd94cc7a3a7ef036f9eb7e8226559425a38744`).

Nichtnegative Mindestanzahl vorhandener Werte.

Fields:

- `maxCount`: 1
- `minCount`: 1
- `name`: "minCount"
- `type`: "integer"

### markitect/core/name

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/name.yaml` (original bytes `sha256:ec8abd72a1f308f00f905dacf2d8959054961f55e84b6c894aa3214a5da30d10`).

Optionaler ausdrücklich abweichender nichtleerer Feldname. Ohne Angabe gilt die vollständige lokale ID der Property als Feldname. Der effektive Feldname muss innerhalb jedes verwendenden Vertrags eindeutig sein.

Fields:

- `maxCount`: 1
- `minCount`: 0
- `name`: "name"
- `type`: "string"

### markitect/core/properties

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/properties.yaml` (original bytes `sha256:e3f430b729a355704ba208bcfb95bd206b66e91307f43a6af1af263f3c9879b8`).

Referenziert die erlaubten Feldverträge des hier beschriebenen Objekts. Die effektiven Feldnamen müssen innerhalb dieses Vertrags eindeutig sein.

Fields:

- `maxCount`: -1
- `minCount`: 0
- `name`: "properties"
- `target`: `markitect/core/Property`
- `type`: "reference"

### markitect/core/target

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/target.yaml` (original bytes `sha256:06dc15e8673124553bd2b71a73e1b7bb75a237d2b0a01e690da3230a467a0a63`).

Der verlangte Kind der referenzierten Definitionen. Bei type reference ist diese Angabe erforderlich. Bei anderen Werttypen ist sie nicht zulässig.

Fields:

- `maxCount`: 1
- `minCount`: 0
- `name`: "target"
- `target`: `markitect/core/Kind`
- `type`: "reference"

### markitect/core/type

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/type.yaml` (original bytes `sha256:acbcb268f34a4aedfcde704af97648dbda499c6a8d22522c3e4b93f2802d9579`).

Der vom Core verstandene Werttyp dieser Property.

Fields:

- `maxCount`: 1
- `minCount`: 1
- `name`: "type"
- `type`: "enum"
- `values`: ["boolean", "enum", "integer", "number", "object", "reference", "string"]

### markitect/core/values

Kind: `markitect/core/Property`. Source: `.markitect/model/markitect/core/values.yaml` (original bytes `sha256:e69f9a0d9720055274bf0854072e2cce495e94630359599bd73d4f5a2589fffd`).

Erlaubte Stringwerte einer enum-Property. Bei type enum ist eine nichtleere Liste eindeutiger Werte erforderlich. Bei anderen Werttypen ist diese Angabe nicht zulässig.

Fields:

- `maxCount`: -1
- `minCount`: 0
- `name`: "values"
- `type`: "string"

### product/ports-and-adapters

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/ports-and-adapters.yaml` (original bytes `sha256:9ec243e1c1a81a95f7a87c34be6484e94b72eb0f2708508d43ca5b7e0bd8bc03`).

Keep model, snapshot, organization and realization values independent of execution and infrastructure. Project owns structural acquisition, checked model-derived contexts and immutable candidate validation. Run consumes Project and organization directly and owns the complete closed hierarchy over consumer-owned Session, Git, check and record ports. Outer runtime configuration owns acquired static native/check profile metadata; it does not resolve model intent. Concrete adapters implement physical acquisition and protocols. Bootstrap composes actual services. All maintained Go code belongs to one root module and an explicit architecture responsibility; obsolete execution engines and their consumers retire together.

### product/adapters/agent/closed-agent-metadata

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/agent/closed-agent-metadata.yaml` (original bytes `sha256:41a6dfd7742793234d55c6b5ba370246d603b6d68946808117b413b6764e8017`).

Prepare only explicitly selected source-bound Agent profile metadata in closed formats. Bind the immutable runtime image identity and the actual whole metadata file bytes. The configured container engine may inspect the image and copy build metadata from a created, never started container; preparation never reads a host provider executable, probes a provider version, exports a schema, starts an Agent Session or invokes a model. Preserve recorded CLI and agentd versions, diagnostics and observation limits. A build-recorded schema digest is an assertion, not an observed complete schema inventory. Instructions come only from captured inputs. A static pass establishes neither runtime nor provider capability.

### product/adapters/checks/closed-check-metadata

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/checks/closed-check-metadata.yaml` (original bytes `sha256:4426df8b86a963298cd81b08b00a1eb18c1c5c86fbc202a923b2a2a6d6336fc4`).

Parse only explicitly acquired pinned container Check profile metadata in closed formats. Preserve literal arguments, fixed environment policy, finite resource and output limits, exact image/executable identities and source bindings. Static preparation starts no check and establishes no execution success. Execute each declared check against the exact fixed candidate on an independently verified disposable copy and empty private home; retain actual image, tree, start, exit, exit code, output, timeout and closure observations. Unknown or uncertain execution and cleanup cannot count as a passing check.

### product/adapters/repository/repository-boundaries

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/repository/repository-boundaries.yaml` (original bytes `sha256:9bc0d1bf7eb6fe3456baa22981fed765c0263d64d2142e59fb79013735362651`).

Capture the complete actual repository census and selected original inputs with exact paths, modes and bytes. Working and fixed previews grant no publication guard; committed capture requires clean HEAD and retains its comparison guard. Resolve a symbolic revision once, retain the full identity, reject redirects and changed directory identity, and never replace historical sources with current external bytes.

### product/apply/read-only-preparation

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/apply/read-only-preparation.yaml` (original bytes `sha256:7f5c012ef50c8c96bb2421791d7e60b1c02b6a9871abfe21f2635dd37dc40637`).

Acquire explicitly selected model and control inputs through Project-owned ports. Prepare the Model check, registry, complete strict file accounting, authored ownership, scope, Manager instructions, declared checks and static contexts from one complete capture; recapture rather than extend an old guard. Preserve original bytes, source provenance and authored absence. Known invalid input differs from incomplete acquisition. Project does not decode runtime profiles, resolve strategies, roles or budgets, start Sessions, execute checks, allocate working copies or publish artifacts.

### product/cli/honest-read-only-interface

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/cli/honest-read-only-interface.yaml` (original bytes `sha256:3f1292e64c3d6559eb2fd167d0b4ae0521e3f6b23fe96e2eb9aa40542c1d13e6`).

Expose check and context over actual Project preparation, with structural preparation as the default requested stage and explicitly selected runtime metadata as an additional stage. Preserve actual statuses, sources and inclusion reasons. The generated model view has its own source/target-bound guarded write. At the S2 boundary Apply explicitly reports unavailable until the S3 workflow and publication are integrated; it starts no Run or resource. A Run is closed: the command line offers no answer command or preloaded owner decision. Invocation configuration is closed and requires explicit finite limits and normal account credential sources. Usage, known failure and incomplete work have distinct exit statuses.

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

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/run/confirmed-run-history.yaml` (original bytes `sha256:a0869a3fcb983995b3ea51954a5aba5c5535c27b3cb1b8e2e43cc770e6158c99`).

Every Assessment, Implement and Verify executes the complete model-derived Manager hierarchy, including Root, intermediate and empty Packages, in fresh Run environments. Project intent and its complete input inventory remain fixed; an invalid or incompletely acquired input starts no Manager work. Before ordinary work, open one fresh environment and working copy per Manager and one Git Workspace Service for the Run. Run has no domain Passes or Turns. Every Manager receives its own selected context, mandate, bound candidate, concrete findings and direct tools. Managers delegate to their direct children, remain available for questions, inspect complete fixed child diffs, review exact commits and integrate accepted work upward through actual Git history. Questions and replies remain inside the Run. Correlate actual incoming questions and answers, support crossed waits and keep completed Managers reachable in their same ordinary conversation. No question crosses Root to the invoking AI or owner. Required late findings invalidate affected acceptance; an answer alone does not change an already fixed Git handoff. Correction stays with the responsible Manager and uses concrete findings and the actual integrated parent baseline. Parent integration never subtracts previously accepted child paths from the complete diff. Actual delegation bases, fixed results, parent before/after commits, accepted child commits and judgments survive closure. Implement submits a structurally checked candidate and required scoped check results; it cannot approve publication. Assessment and Verify judge the immutable input independently in fresh Runs, and scratch edits never become product. Root checks the complete declared suite on the final fixed candidate. Required check failures cannot be waived by a semantic approval; actual startup, acquisition or observation failures remain technical failures. Managers assess material conditions, exceptions, interfaces and composed workflows from the bound model, challenging the actual work and evidence of their children. Green totals and provider completion do not establish semantic completeness. Reject and model blocker require concrete findings; no finding quota or custom evidence-mapping protocol is required. One finite outer deadline and message allowance bound the invocation, including correction Runs; Manager count is explicitly bounded before environments start. Exhausted capacity, cancellation, incomplete observations or uncertain closure never become success. Cleanup and persistence have separate finite bounds and grant no further productive work. Candidates retain their exact commit, tree and complete ancestry in self-contained Git storage. Verify independent reopening before exposing the stored reference. Mandatory terminal results include the exact input reference, binding, technical and semantic outcomes, findings, handoffs, checks and actual cleanup observations. Return no live resource handle and confirm terminal storage by read-back after resource closure. Shared project Notes persist across Runs, retain their originating Run, Manager and context and preserve concurrent appends. They are operational memory, not authority, assignments or model decisions. Assessment approves model readiness, including an unrealized or outdated product; it validates fixed structural inputs without requiring already-passing realization commands. Assessment Approve always enables a complete Implement, even when its complete captured Census remains unchanged. Assessment Reject or a model blocker stops without publication. Verify judges the exact fixed realization; its Reject carries mandatory Findings and the exact candidate into fresh Implement and Verify Runs under the original allowance. Apply alone composes Assessment, Implement, fresh Verify and bounded corrections and publishes only the exact stored acceptance after every Run resource has closed and target bytes have been read back. Standalone Run completion is not Apply, publication, provider qualification, release or owner acceptance.

### product/runtime/container-runtime

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/runtime/container-runtime.yaml` (original bytes `sha256:b9d020c5f735f6697f9ea30254894a722f4ba15dba873ed8f9d6883fbe377aaa`).

Every Run opens one fresh private Manager environment, working copy, provider home and conversation per required Manager, and one owned Git Workspace Service. Assessment, Implement and Verify use separate fresh Run environments; no Manager or Git service survives into a correction Run. Keep each Manager available for ordinary questions and renewed tasks until that Run closes. Native Managers and project checks execute in hardened Linux containers from explicitly prepared immutable engine image IDs. Give a Manager only its own private copy through an owned named-volume subpath; never synchronize a second writable host mirror or overlay other owners' paths. Responsibilities and reviewed handoffs govern work; per-write ACLs and custom evidence protocols are not prerequisites. Run as non-root with a read-only root filesystem, no capabilities or privilege escalation, and explicit finite CPU, memory, process and temporary-space limits. A narrow fixed initialization of a newly created empty owned volume may establish its non-root ownership; it grants no product work. Refuse mismatched images, unsafe mounts or incompatible frame protocols. markitect-agentd starts the native provider, relays framed streams and Run tools, and reports actual exits without owning model policy. Provision dedicated normal-account input privately into each Manager home; never put secrets into arguments, environment, transcripts, Notes or candidate bundles, and never share a mutable host login home. Codex has no API-key route. Provider support requires its own actual native evidence. Declared project checks use independent exact input and empty-home copies, preserve bytes, executable modes and required empty directories, and retain actual image, start, exit, timeout and closure observations. Publication reconstructs the stored candidate with host Git after Run closure and materializes its check copies independently; it never borrows a closed Run volume or Agent. Retain candidates and mandatory results before complete bounded resource closure. Observe the end of every owned process, container, home, temporary copy and volume, including partial-start failures. Uncertain cleanup, acquisition, storage or transport blocks success and publication. The confirmed-run-history Rule owns the shared outer deadline, message allowance and closed terminal result.

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

Kind: `vocabulary/Rule`. Source: `.markitect/model/tooling/maintenance-boundaries.yaml` (original bytes `sha256:633fbc919470ea2717d400147e6f27c8349365cca9032c81b17e4414ae9c0615`).

Maintain strict root-layout source packaging, deterministic release archives, licenses, distribution and guarded installation as independent maintenance services. Reject unsupported files, redirection, nested modules and archive collisions. Preserve Git tag identities and required historical release evidence. Retire public release pages and downloads only on explicit owner instruction; retain their source/asset identities and attestations without implying current availability. A source smoke is not release, installation, provider or Apply acceptance.

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

Kind: `vocabulary/Check`. Source: `.markitect/model/tooling/gates/library-oracle.yaml` (original bytes `sha256:9476604c25ff1e2ba800d9023949ede73a79f5c5b4cc70bbef53501e6c34a06c`).

Run the declared library-oracle source maintenance check and retain failures separately from runtime and acceptance evidence.

Fields:

- `command`: "python -B -m unittest discover -s experiments/library-playground"

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
