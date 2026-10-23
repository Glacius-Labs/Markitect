# Canonical Markitect model

Generated from acquired canonical v1 source bytes. This view is not execution or acceptance evidence.

## markitect

Canonical model digest: `sha256:fec67c1e2946953bc340b93bd4671edb13e50e7d53ea685374ad059a5a8a9526`.

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

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/organization/delegated-context.yaml` (original bytes `sha256:053c899a26050f7624789a14cda7540a606ea1199c79d926eb00d3b512ff2b74`).

Derive one ordinary Root and one Manager per local canonical Package from authoritative checked facts; every required Manager, including empty and intermediate Packages, takes part in every Pass, and no Agent prunes the tree or selects participants by relevance. Give each Manager its own obligations with provenance, labeled foreign context with its owners, its owned areas and local registry, the accepted plans, where its strategy plans, and bases it builds on, the required checks and its team; Root receives the project and boundary obligations explicitly, attributed to their actual owners. Every Manager plans and realizes its own obligations; a parent supplies its own foundation, reviews its children's plans, bases and results for complete responsibilities and fit, returns gaps to their owner and integrates their accepted results; it never does a child's work or releases a child from its obligations. A Definition is realized by its owner's artifacts; other areas use that realization instead of rebuilding it, and the parent settles cross-area matters between its children. Before any work, every Manager assesses the bound model read-only as a fresh reviewer of the same recursive Pass, with no gate, required check or probe; only an accepted assessment releases the work, and a material problem Root confirms ends the Run Failed without publication. Managers communicate only along direct parent and child edges, in both directions, with correlated questions; a cross-branch exchange travels through the common ancestor and back; nobody outside the tree answers during a Run, and no preloaded owner decision exists. Answers clarify within the model and never override it; Root ends the Run when it confirms that the model is erroneous, contradictory or materially open without delegating the decision, and the model is corrected outside the Run for a new one. A Manager may delegate bounded tasks to configured helper roles, each in its own container with read access to the whole stand and write access only to an explicit subset of its own area that no other writer holds; helpers report to their Manager only, gain no ownership and complete nothing by finishing. Reviewers are fresh, read-only and independent of the authors; the Orchestrator records their findings, and no author can hide them. Every finding states its kind: a confirmed deviation states the obligation, the outcome it requires and the observed mismatch and names its correction; a suspicion blocks until a fresh review investigates and classifies it and grants no work; a model problem belongs to the Model assessment; a suggestion never blocks, causes a correction or grants a write. A finding a review of the unchanged candidate disproved is not raised again without new evidence; a new reviewer or new wording is none. Apply fulfills the model: an artifact changes only for a modeled obligation, a required check or evidence, or a necessary consequence of such work, and a conforming stand is preserved even where another design is preferred. Derive risks, distinct claims and model-permitted counterexamples within the same Pass; record source-derived expectations before their probes, inspect the actual evidence and return unsupported success claims without a separate expectations Pass. Choose implementation details within declared freedom; report an unresolved material public policy to the parent as a model problem rather than inferring authority from existing artifacts or a plausible default; an internal question still unanswered at the limit leaves the work unfinished, while Root ends the Run on a confirmed model problem. Preserve intent; changed policy or a necessary missing canonical link needs a separate model decision. Deliver each Manager's and reviewer's own obligations and selected contracts in full, with every condition, value, resolved link target and owner, and keep Root's project inventory a labeled reference list; never cut required content silently: a packet the provider channel cannot carry refuses the turn. Answer a lookup of any Definition or of the standing context from the bound inputs, record how each turn received its standing context and each lookup with digests of what was sent, and deliver the standing context again after an observable compaction or a conversation the provider did not continue. Retention a provider does not expose stays an evidence gap, and complete delivery does not establish that an Agent followed it. Prompts state each role's task, inputs and answer format; they never become a second owner of binding project rules or of the process.

### product/realization/artifact-accounting

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/realization/artifact-accounting.yaml` (original bytes `sha256:3267a354d09c55f6e7b36f015387c452a5379e61738e55337acd1e9c630344ef`).

Account for every observed regular file exactly once as a validated Markitect input, a registered artifact or an explicitly ignored file; Markitect inputs are the immutable authority inputs and the owner-writable local registries, and neither may be registered or ignored as an artifact. Ownership is the declared set of repository-relative directory and exact-file areas, each owned by Root or one local Package; the most specific area owns a path, equal specificity and files without an owner fail, and a broad area never absorbs a more specific one. Each Manager maintains only its own local registry of exact-file or directory entries with Definition links, forms and reasons; a directory entry covers only its registrant's eligible files below it, the most specific entry registers a file, and no file has two effective registrations. Missing paths, unexplained artifacts, dangling links, duplicate accounting, wrong-owner registrations and changed authority inputs block, with machine-readable diagnostics addressed to the Manager that must act. Only the Orchestrator assembles the read-only derived registry view, from checked local versions, and records which versions it includes. Enforce writing ownership in every Agent's container: its own areas and its own registry writable, protected inputs and other owners' areas, deeper directories and exact files read-only; refuse an ownership shape that mounts cannot enforce instead of trusting prompts. Accept a basis or result only after the File check of the submitter's stable area composed with the accepted stand, so disjoint Managers work and finish concurrently without a mixed live view. Candidates preserve source identity and observed findings; pure accounting values do not schedule or publish work.

### product/run/confirmed-run-history

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/run/confirmed-run-history.yaml` (original bytes `sha256:b7be09c3d7e5f111bde6c3a8d7931bdadf7b2e8d6d3910a7b5390db845f96e5f`).

Every Assessment, Implement and Verify executes the complete model-derived Manager hierarchy, including Root, intermediate and empty Packages, in fresh Run environments. Project intent and its complete input inventory remain fixed; an invalid or incompletely acquired input starts no Manager work. Before ordinary work, open one fresh environment and working copy per Manager and one Git Workspace Service for the Run. Run has no domain Passes or Turns. Every Manager receives its own selected context, mandate, bound candidate, concrete findings and direct tools. Managers delegate to their direct children, remain available for questions, inspect complete fixed child diffs, review exact commits and integrate accepted work upward through actual Git history. Questions and replies remain inside the Run. Correlate actual incoming questions and answers, support crossed waits and keep completed Managers reachable in their same ordinary conversation. No question crosses Root to the invoking AI or owner. Required late findings invalidate affected acceptance; an answer alone does not change an already fixed Git handoff. Correction stays with the responsible Manager and uses concrete findings and the actual integrated parent baseline. Parent integration never subtracts previously accepted child paths from the complete diff. Actual delegation bases, fixed results, parent before/after commits, accepted child commits and judgments survive closure. Implement submits a structurally checked candidate and required scoped check results; it cannot approve publication. Assessment and Verify judge the immutable input independently in fresh Runs, and scratch edits never become product. Root checks the complete declared suite on the final fixed candidate. Required check failures cannot be waived by a semantic approval; actual startup, acquisition or observation failures remain technical failures. Managers assess material conditions, exceptions, interfaces and composed workflows from the bound model, challenging the actual work and evidence of their children. Green totals and provider completion do not establish semantic completeness. Reject and model blocker require concrete findings; no finding quota or custom evidence-mapping protocol is required. One finite outer deadline and message allowance bound the invocation, including correction Runs; Manager count is explicitly bounded before environments start. Exhausted capacity, cancellation, incomplete observations or uncertain closure never become success. Cleanup and persistence have separate finite bounds and grant no further productive work. Candidates retain their exact commit, tree and complete ancestry in self-contained Git storage. Verify independent reopening before exposing the stored reference. Mandatory terminal results include the exact input reference, binding, technical and semantic outcomes, findings, handoffs, checks and actual cleanup observations. Return no live resource handle and confirm terminal storage by read-back after resource closure. Shared project Notes persist across Runs, retain their originating Run, Manager and context and preserve concurrent appends. They are operational memory, not authority, assignments or model decisions. Apply alone composes Assessment, Implement, fresh Verify and bounded corrections and publishes only the exact stored acceptance after every Run resource has closed and target bytes have been read back. Standalone Run completion is not Apply, publication, provider qualification, release or owner acceptance.

### product/runtime/container-runtime

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/runtime/container-runtime.yaml` (original bytes `sha256:0bb08e25733d9ca7ff1c43038b8899c7a34254c0e22977891546924824564f3e`).

Run every Agent Turn and project check in its own fresh hardened Linux container from a runtime image named by its engine image ID; no container outlives its Turn. Container lifetime is separate from Manager identity and conversation: a Manager's work Turns keep one provider home and continue its permitted conversation, while every reviewer starts fresh. A Turn ends only when the end of its container, and with it of every process it started, was observed. markitect-agentd is the only program the Orchestrator talks to: it starts the provider CLI, relays its standard streams, serves the Orchestrator tools over MCP on a Unix socket inside the container and reports exits, adding no policy. The Orchestrator passes only validated specs to the engine, gives containers named volumes and never host paths, mounts the shared tree read-only with only the session's permitted directories and exact files overlaid writable and other owners' deeper ones read-only, runs them as a non-root user with a read-only root filesystem, no capabilities and bounded resources, and refuses a different frame protocol.

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
