# Canonical Markitect model

Generated from acquired canonical v1 source bytes. This view is not execution or acceptance evidence.

## markitect

Canonical model digest: `sha256:69bcffcfdd8da4ee9293a2d1c28572c4872c4c57f0afb670a63054c1504b8b75`.

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

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/ports-and-adapters.yaml` (original bytes `sha256:f8d5bc7c83b1060662299de1169fb0edc969d645148c55da86cdab7f21ccfb0a`).

Keep model, snapshot, organization and realization values independent of execution and infrastructure. Apply owns consumer ports; adapters implement physical acquisition and protocols. Bootstrap composes actual services. All maintained Go code belongs to one root module and an explicit architecture responsibility; no old compiler, Manager DTO or runtime remains active.

### product/adapters/agent/closed-agent-metadata

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/agent/closed-agent-metadata.yaml` (original bytes `sha256:41a6dfd7742793234d55c6b5ba370246d603b6d68946808117b413b6764e8017`).

Prepare only explicitly selected source-bound Agent profile metadata in closed formats. Bind the immutable runtime image identity and the actual whole metadata file bytes. The configured container engine may inspect the image and copy build metadata from a created, never started container; preparation never reads a host provider executable, probes a provider version, exports a schema, starts an Agent Session or invokes a model. Preserve recorded CLI and agentd versions, diagnostics and observation limits. A build-recorded schema digest is an assertion, not an observed complete schema inventory. Instructions come only from captured inputs. A static pass establishes neither runtime nor provider capability.

### product/adapters/checks/closed-check-metadata

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/checks/closed-check-metadata.yaml` (original bytes `sha256:17e3b76f6998fdfdf11cffbc87fa39944cf6481515b3d3ca94e566e113ee3df8`).

Parse only explicitly selected source-bound executable and Check profile metadata in closed formats. Preserve literal arguments, environment policy, finite limits and original identities. Preparation does not execute a check and cannot establish execution success.

### product/adapters/repository/repository-boundaries

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/adapters/repository/repository-boundaries.yaml` (original bytes `sha256:9bc0d1bf7eb6fe3456baa22981fed765c0263d64d2142e59fb79013735362651`).

Capture the complete actual repository census and selected original inputs with exact paths, modes and bytes. Working and fixed previews grant no publication guard; committed capture requires clean HEAD and retains its comparison guard. Resolve a symbolic revision once, retain the full identity, reject redirects and changed directory identity, and never replace historical sources with current external bytes.

### product/apply/read-only-preparation

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/apply/read-only-preparation.yaml` (original bytes `sha256:b5b8e1cfc6261dbba1837dd446cb2bf6cbe43a76231c6f93f2d30856c8f9757f`).

Acquire explicitly selected model and control inputs through consumer-owned ports. Prepare the Model check, registry, complete strict file accounting, execution declarations and static context from one complete capture; recapture rather than extend an old guard. Preserve actual product defaults and profile metadata sources. Known failure differs from incomplete acquisition. Preparation never starts a session, executes a check, creates a workspace or publishes artifacts.

### product/cli/honest-read-only-interface

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/cli/honest-read-only-interface.yaml` (original bytes `sha256:c89e18fbcd585cc136a12645feebc774c917b82904fe33001a68a1461d8293dc`).

Expose check and context over actual PreparationService reports, with structural preparation as the default requested stage and explicitly selected Orchestrator metadata as an additional stage. Preserve actual statuses, sources and inclusion reasons. Apply, answer and extract act only with an explicit Runtime profile and run configuration and report the actual run status, Passes, findings, questions, publication, Run outcome and retained work; an answer binds to the exact run and question that asked. Usage, known failure and incomplete work have distinct exit statuses.

### product/cockpit/observed-run-inspection

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/cockpit/observed-run-inspection.yaml` (original bytes `sha256:d31c0fcf7d2de03bf231b129af226a50c5691b33be410a96f00fef2647cf26ed`).

Present runs only through their recorded evidence under the state directory: the events, transcripts and report of each run, read as consistent prefixes up to an explicit record boundary; nothing recorded after the boundary reaches its view, and the report appears only at the final boundary. Show the organization, current work, recorded waits with the records and participants they name, exchanges with their direction and their author (an Agent, the Orchestrator or the owner channel), Passes, candidates, checks, findings, questions and the publication's read-back as recorded. Never infer a delivery, a reading, a correlation, a success or an authority the record does not establish; show outdated waits, missing outcomes or reports, unreadable or unknown records and absent bindings as limitations. Reading, replay, pause, selection and reconnection start, answer, stop, repair and publish nothing, and the view serves only the local machine.

### product/documents/guarded-model-view

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/documents/guarded-model-view.yaml` (original bytes `sha256:76491d1fd66ab27815d4f311d56737bbbe035b60ca69870eaa8ee12fd2566057`).

Render a deterministic readable view from actual compiled canonical v1 Definitions. Keep canonical model digest, rendered byte digest and source/target-bound preview digest distinct. Write only the reviewed generated target under an independent documentation guard after source and target rechecks; never overwrite model, control or instruction inputs, and retain observed partial effects on failure.

### product/model/language-v1

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/model/language-v1.yaml` (original bytes `sha256:45b618528da0a7bce07e1760ca4e9233a201df718b598b3b4ed34e302f755259`).

Check the complete canonical language v1 from explicitly supplied immutable sources. Preserve exact identities, values, links, provenance and deterministic model digests. Reject invalid or incomplete language constructs without filesystem, Git, process, provider or legacy product dependencies. The fixed Core definitions retain the normative contracts.

### product/organization/delegated-context

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/organization/delegated-context.yaml` (original bytes `sha256:0eb79ee243f8210dc2372509384d0c58cccc1f603312b232191925c315727849`).

Derive one ordinary Root and one Manager per local canonical Package from authoritative checked facts; every required Manager, including empty and intermediate Packages, takes part in every Pass, and no Agent prunes the tree or selects participants by relevance. Give each Manager its own obligations with provenance, labeled foreign context with its owners, its owned areas and local registry, the accepted plans and bases it builds on, the required checks and its team; Root receives the project and boundary obligations explicitly, attributed to their actual owners. Every Manager plans and realizes its own obligations; a parent supplies its own foundation, reviews its children's plans, bases and results for complete responsibilities and fit, returns gaps to their owner and integrates their accepted results; it never does a child's work or releases a child from its obligations. A Definition is realized by its owner's artifacts; other areas use that realization instead of rebuilding it, and the parent settles cross-area matters between its children. Managers communicate only along direct parent and child edges, in both directions, with correlated questions; a cross-branch exchange travels through the common ancestor and back; only Root asks the AI that invoked Markitect, which may consult the user. Answers clarify within the model and never override it; Root ends the Run when it judges the model erroneous or contradictory. A Manager may delegate bounded tasks to configured helper roles, each in its own container with read access to the whole stand and write access only to an explicit subset of its own area that no other writer holds; helpers report to their Manager only, gain no ownership and complete nothing by finishing. Reviewers are fresh, read-only and independent of the authors; the Orchestrator records their findings, and no author can hide them. Derive technical risks and suitable evidence from the model and the realization without a mandatory advance expectations phase. Preserve intent; changed policy or a necessary missing canonical link needs a separate model decision. Prompts state each role's task, inputs and answer format; they never become a second owner of binding project rules or of the process.

### product/realization/artifact-accounting

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/realization/artifact-accounting.yaml` (original bytes `sha256:3267a354d09c55f6e7b36f015387c452a5379e61738e55337acd1e9c630344ef`).

Account for every observed regular file exactly once as a validated Markitect input, a registered artifact or an explicitly ignored file; Markitect inputs are the immutable authority inputs and the owner-writable local registries, and neither may be registered or ignored as an artifact. Ownership is the declared set of repository-relative directory and exact-file areas, each owned by Root or one local Package; the most specific area owns a path, equal specificity and files without an owner fail, and a broad area never absorbs a more specific one. Each Manager maintains only its own local registry of exact-file or directory entries with Definition links, forms and reasons; a directory entry covers only its registrant's eligible files below it, the most specific entry registers a file, and no file has two effective registrations. Missing paths, unexplained artifacts, dangling links, duplicate accounting, wrong-owner registrations and changed authority inputs block, with machine-readable diagnostics addressed to the Manager that must act. Only the Orchestrator assembles the read-only derived registry view, from checked local versions, and records which versions it includes. Enforce writing ownership in every Agent's container: its own areas and its own registry writable, protected inputs and other owners' areas, deeper directories and exact files read-only; refuse an ownership shape that mounts cannot enforce instead of trusting prompts. Accept a basis or result only after the File check of the submitter's stable area composed with the accepted stand, so disjoint Managers work and finish concurrently without a mixed live view. Candidates preserve source identity and observed findings; pure accounting values do not schedule or publish work.

### product/run/confirmed-run-history

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/run/confirmed-run-history.yaml` (original bytes `sha256:d26c320e24b41ea8957fa4aa0c5fdde7da254497f0db935c024f9bb3c33aaa23`).

Bind each Run to the exact starting commit, model and dependency locks, ownership, execution settings, ignore policy, request, authority and finite limits; these authority inputs stay immutable, while owners change their own artifacts and local registry metadata only within the current Pass's write rights. Before any Agent work, run the strict Model check and File check of the captured stand; a failure ends the Run with its diagnostics, starts nothing and repairs nothing. Apply composes Plan, Prepare and Implement and then a separate Verify as Passes of one recursive base process, each with a new identity; only the mandate, effects, purpose payload and acceptance differ, and no purpose or correction has an engine of its own. For every Manager, Root and leaves included: receive the Pass mandate, work locally, and submit an explicit basis that the Orchestrator checks technically on the stable submitted area and that the direct parent accepts semantically, Root's against the declared Pass requirements; only then release its children and stay available for their questions; review each child's basis and complete result, return gaps to their owner and integrate accepted results; submit the complete result; the Pass closes after whole-stand gates with one sealed result. A submitted proposal wakes its parent before the child is Done; a returned proposal stays its owner's work in the same Pass; a changed accepted basis explicitly invalidates its consumers. Managers are Ready, Working, Waiting or Done in each Pass, problems are structured reports, and a Turn ending, a provider exit, a timeout or an uncertain stop never establishes completion. A Manager its parent's basis released may answer routed questions in read-only turns while its productive dependencies are blocked; that neither releases its children nor completes its work. At every Pass boundary quiesce the writers, capture the whole stand, require it to equal the accepted stand and pass the Model check and the File check; a failure ends the Run failed without a repair Pass. Verify assesses one immutable candidate with fresh independent read-only reviewers and the required checks; a genuine finding stays open until a fresh independent assessment of its responsibility closes it, resolved on a corrected candidate or disproved with evidence on the unchanged one in a new complete Verify, and a failed required check cannot be disputed away. Route corrections through the earliest necessary normal work Pass followed by a complete fresh Verify; clean means no open genuine finding and passing required checks. Success requires clean Verify and an applied or no-change publication with exact read-back of the verified bytes; definite rejections fail, and unfinished work, exhausted limits, unresolved waits or uncertainty stay visibly incomplete with their evidence. Keep execution semantics independent of SQL, provider protocols and repository acquisition.

### product/runtime/container-runtime

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/runtime/container-runtime.yaml` (original bytes `sha256:1b5bbe332d5dde94e39eb52d99742a8d38071a45c2deaf01a16bcd63cbae3c3d`).

Run every Agent session and project check in its own hardened Linux container from a runtime image named by its engine image ID. markitect-agentd is the only program the Orchestrator talks to: it starts the provider CLI, relays its standard streams, serves the Orchestrator tools over MCP on a Unix socket inside the container and reports exits, adding no policy. The Orchestrator passes only validated specs to the engine, gives containers named volumes and never host paths, mounts the shared tree read-only with only the session's permitted directories and exact files overlaid writable and other owners' deeper ones read-only, runs them as a non-root user with a read-only root filesystem, no capabilities and bounded resources, and refuses a different frame protocol.

### product/snapshot/fixed-values

Kind: `vocabulary/Rule`. Source: `.markitect/model/product/snapshot/fixed-values.yaml` (original bytes `sha256:26588b48ed9b6e0f28a946f4ea63b86eecff58b2ef0deece8899d5289e0a88b5`).

Represent exact repository-relative paths, modes, immutable bytes, trees and deterministic content identities. Tree and content digests remain distinct from canonical model digests. Invalid spellings, collisions and unsupported entries fail explicitly; this package performs no repository I/O.

### repository/intent-and-evidence

Kind: `vocabulary/Rule`. Source: `.markitect/model/repository/intent-and-evidence.yaml` (original bytes `sha256:911ecd62e7e0659aebc1fcac712474584734ea7d3c2d5f1b17e8a09e36024830`).

People maintain the canonical intended world. Artifacts realize its obligations. One ordinary root and one Manager per local Package organize delegated realization with independent review, integration, verification and guarded Apply. A committed model binds the specification for its revision; source checks and reports do not authenticate human acceptance or establish runtime readiness.

### repository/learning-driven-workflow

Kind: `vocabulary/Rule`. Source: `.markitect/model/repository/learning-driven-workflow.yaml` (original bytes `sha256:f050bb80233c979e76ecbe8f4c4e181bad5b390e263f26df5a62921b59031658`).

Develop Markitect through explicit learning and complete user workflows. Separate observed facts, owner reports, assistant assessments, recommendations and accepted decisions. Similar independent-looking designs are evidence of conceptual stability, not proof that the method works.

Before a substantial redesign, record briefly in the existing owning design or decision document: the new observation, the prior assumption it challenges, the resulting design change, and the observable result that would demonstrate improvement. An accepted foundation gets a coherent period of actual use. Keep the current complete-workflow target stable; fix concrete failures, missing accepted obligations and necessary blocked steps before unrelated generalization. New evidence can justify a bounded correction or an owner decision; this is not a ban on necessary redesign.

Investigate a consequential uncertain technical boundary early with the smallest useful real probe before dependent design or implementation hardens. Record its observations and limits, and preserve separate finite authority for provider runs, studies, releases and deployments. Independent design review examines whether each added mechanism addresses an observed failure or accepted obligation and whether a simpler approach would serve it. Existing quality, review, evidence and write safeguards remain in force.

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
