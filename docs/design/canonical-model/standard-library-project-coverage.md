# Web Library: project-coverage walkthrough

Date: 17 October 2026. After selecting the fourteen Kinds, the owner selected this six-area coverage: "Okay, dann halten wir diese Abdeckung so fest und lassen die konkreten Projektwerte offen." This document owns the selected qualitative coverage, its bounded assessment and illustrative reference-project Definitions. The examples make that coverage concrete; they do not select a web identity mechanism, stack, provider, database technology, numeric operating objectives or every detail of the prospective Web Library.

## Result and scope

The fourteen selected Kinds can describe all six requested areas in this example; the walkthrough exposes no necessary new Kind or Property. Rules state binding conditions and required procedures/outputs; Interfaces state interaction contracts; Actors describe roles; Scenarios state concrete expectations; Checks describe examination and success criteria. Environment, Deployment and OperatingResource place the operating conditions. These are modeling judgments from a bounded example, not a universal completeness proof.

The reference Web Library is not yet a complete implementation assignment: its access contract, configuration requirements, build/release bindings and operating responsibilities need explicit project choices and complete clauses. A vocabulary's ability to describe these subjects does not supply those choices or prove their realization. This example is distinct from the existing local, sequential, process-lifetime-only M1/M2 fixture, whose [realization](library-example/m1/model/library/realization.yaml) and [callable interface](library-example/m1/model/library/callable-interface.yaml) already specify their bounded delivery. The Web design does not add authentication, persistence, networking or deployment requirements to that fixture.

## Selected coverage and open project values

| Area | Proposed canonical owners | Concrete Web Library obligations and distinguishing examination |
| --- | --- | --- |
| Security | operator Actor; access-policy Rule; applicable HTTP/data Interfaces; production trust/resource boundaries | Explicit API-side permission checks for every protected operation, protected transport at declared boundaries, limited database/release identities and safe handling of untrusted input/output. An anonymous or unentitled caller cannot read protected data or change domain state, including through a direct API call. Complete the trust, authentication/session and browser-protection design against the chosen architecture; naming an Actor grants no permission. |
| Configuration | configuration-contract Rule plus the actual Deployment/OperatingResource binding | Declare required settings, source/precedence, defaults, validation and lifetime. Distinguish public browser settings from server-only secret references. Invalid required configuration must not lead to readiness or an empty/local database fallback. Test invalid/missing values and environment isolation; do not expose actual credentials in model, browser assets, diagnostics or evidence. |
| Build | build-contract Rule; required output obligations; realizing files in the registry | A clean checkout can build documented frontend/API deliverables using fixed source, model/locks, toolchain, dependency and target inputs. Record actual artifact identities and build inputs. Assess the actual commands and resulting deliverables; do not claim byte reproducibility from a successful single build. |
| Tests | verification-contract Rule; feature Scenarios and Checks | Cover behavior/refusals and the new web boundaries, not merely successful responses. Reuse domain expectations while adapting to the actual web interface. Include permission denial, invalid configuration, multi-instance consistency and the existing ambiguous-commit/migration/restore distinctions. Bind results to the actual candidate and relevant environment; fakes or source tests do not establish live database/runtime behavior. |
| Release | release-contract Rule; deployment compatibility obligations | Publish identified immutable deliverables through the project's declared authorized path, with evidence applicable to those bytes and the intended combination of frontend/API/schema/configuration. Promote those bytes; document admission and recovery/rollback behavior. Verify what was actually selected and served after publication; a source tag alone does not identify deployed bytes. |
| Operation | operations-contract Rule; Environment/Deployment/OperatingResource; existing recovery contracts and Checks | State traffic admission/readiness, observation, response responsibility, finite resource/retry behavior, maintenance and recovery. Examine startup with missing dependencies, drain/replacement and an isolated restore within their actual scope. Select meaningful load/failure cases, service objectives and recovery/retention requirements before promising production readiness. Two replicas do not themselves prove availability. |

Use one owner per mutable fact. For example, configuration-contract owns resolution/validation rules; a Deployment owns its actual selected binding; access-policy owns entitlement; an Interface owns its exact error/protocol representation. Checks examine them without becoming another requirements catalogue. The existing [data-and-view contract](standard-library-data-and-views.md) remains the owner of detailed transaction, retry, migration, recovery and C4 recommendations; link it rather than repeat its full clauses here.

The security recommendations are informed by OWASP's [authorization guidance](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html), including default denial and request-by-request enforcement, and [secret-management guidance](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html). They inform this reference-project proposal; neither document becomes a blanket imported normative standard. Artifact/input traceability is informed by [SLSA provenance](https://slsa.dev/spec/v1.2/provenance), without selecting a SLSA level, attestation system or signing service.

## Concrete proposed Definitions

Each code block is one Definition file with body links and an empty spec. These are reference-project proposals, not new standard Kinds or a compiled fixture. They reference the subjects from the [connected architecture example](standard-library-architecture-platform.md#connected-web-library-example). Domain Scenarios and detailed data/recovery obligations remain at their existing feature owners.

Keep the access role/policy in Package library, environment-specific operating policy in library/production, and coherent project delivery rules in library/delivery. The last Package creates an intended delivery responsibility under the Manager design; it does not relocate all feature tests there. These project Package choices are illustrative, not compulsory standard-library organization.

### Security

File: library/operator.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
id: operator
kind: Software.Actor
body: >-
  The role that operates [[library]] through [[library-web]] or
  [[library-http-api]]. Entitlement follows [[access-policy]];
  declaring this role does not grant access. A member record does
  not establish an operator identity or an entitlement.
spec: {}
```

File: library/access-policy.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
id: access-policy
kind: Software.Rule
body: >-
  Every protected Library operation through [[library-http-api]]
  requires a verified caller identity entitled to act as [[operator]].
  [[library-api]] enforces this for each request independently of
  [[library-web]]. Missing or invalid identity or entitlement denies
  access, discloses no protected data and makes no domain changes.
  Identity proof must come from the selected trusted mechanism,
  not a caller-supplied role label. The production binding must
  specify that trust mechanism and its validity/revocation rules.
spec: {}
```

This proposal introduces a protected Web Library, not member self-service or an account-management domain. The selected mechanism must address its actual transport, session/token and browser threats, including request forgery where applicable. Identify where transport protection terminates, which downstream links remain trusted/protected, database/release authority boundaries, and data/log/backup access and lifecycle requirements. Operational logs need suitable access and minimization; retention/deletion and recovery must agree about retained data. These missing clauses cannot be supplied merely by writing "secure" in a Rule. No arbitrary retention duration, role hierarchy or identity vendor is selected here.

### Configuration

File: library/production/configuration-contract.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
id: configuration-contract
kind: Software.Rule
body: >-
  [[api-production]], [[data-production]] and [[web-production]]
  declare required configuration, its authoritative sources,
  precedence, defaults, validation and restart/reload behavior.
  Public browser settings are distinct from server-only secret references.
  Required invalid or unresolved settings prevent admission to service;
  there is no fallback to unrelated or initially empty domain state.
  Model sources, browser assets and evidence contain no live credentials.
  Environment configuration does not silently override domain policy.
spec: {}
```

For this example the concrete configuration must bind the browser's public API endpoint, API-to-store connection, trusted identity/entitlement source, relevant origins/transport boundaries and selected operational limits. Every setting must state whether it is public, confidential or a secret reference and which subject owns it. Rotation/revocation and restart/reload must preserve the access/data contracts; an unresolved required credential cannot silently disable them. Actual names, supported values and permitted defaults need one maintained owner. A required exact value may live in an owning Definition's body; structured generation/validation can later justify a Property proposal. Do not introduce another informal field catalogue maintained independently of that owner.

### Build

File: library/delivery/build-contract.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
  - package: library
    as: Library
id: build-contract
kind: Software.Rule
body: >-
  Deliver [[Library.library-web]] and [[Library.library-api]]
  from a clean checkout through documented build commands.
  Bind source/model revisions, dependency locks, toolchain and target
  inputs; do not select changing dependencies implicitly.
  Identify the produced deliverable bytes and their actual build inputs.
  Include required migration/configuration assets and dependency/license
  information under the project's declared delivery requirements.
  Build outputs contain no production credentials.
spec: {}
```

The actual web stack, targets and commands must be selected or explicitly left within a documented realization freedom. Existing M1 Go commands remain M1 commands, not an invented build contract for a browser stack. Required outputs are intent expressed here; exact source files, pipelines, manifests and build artifacts are realizations registered by their owning Managers. Their mere presence does not establish correct or reproducible builds.

### Tests

File: library/delivery/verification-contract.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
  - package: library
    as: Library
id: verification-contract
kind: Software.Rule
body: >-
  Examine the applicable behavior and constraints of [[Library.library]]
  through linked feature Scenarios and Checks with observable criteria.
  [[Library.access-denial-check]] examines the proposed access boundary.
  Include relevant refusals and web access/configuration boundaries,
  not only successful operation responses.
  Bind results to the actual candidate and relevant inputs/environment;
  missing or skipped required observations are not success.
  Before [[release-contract]], assess which evidence applies to the
  deliverables produced under [[build-contract]] and expose any gaps.
spec: {}
```

The complete model must supply the actual linked feature Checks; an instruction to test everything is insufficient. The small distinguishing set below demonstrates the proposed method for one new boundary. Existing domain/concurrency/recovery cases stay at their feature owners. Source checks, actual runtime/database checks and independent review retain different scopes. Source-test results can support a deliverable claim only through an assessed binding to its construction; run the relevant packaged checks against the identified deliverable rather than an unexamined rebuild. No arbitrary coverage percentage or new test framework is selected.

### Release

File: library/delivery/release-contract.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
  - package: library/production
    as: Production
id: release-contract
kind: Software.Rule
body: >-
  Publish identified immutable deliverables from [[build-contract]]
  only through the project's declared authorized release path,
  with required applicable evidence under [[verification-contract]].
  Bind frontend/API versions, schema compatibility and the intended
  configuration contract for [[Production.production]].
  Promote the identified bytes rather than an unexamined rebuild.
  Record what was actually selected and served, and state the permitted
  admission, correction and recovery path before changing service.
spec: {}
```

A release is not the same object as a Deployment: this Rule governs the intended publication procedure, while Deployment specifies the desired environment arrangement. Actual release manifests, digests, check results and rollout history are records/artifacts. The project selects its authority and automation; this proposal does not impose human approval on every release. A source tag alone is insufficient deployment identity. Frontend asset caching and any mixed-version window must match the selected compatibility policy. The existing maintenance migration/recovery proposal does not promise arbitrary database down-migration or rollback to an incompatible old application. A production recovery overwriting current data needs its own explicit operational authority and loss assessment.

### Operation

File: library/production/operations-contract.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
id: operations-contract
kind: Software.Rule
body: >-
  Operate [[api-production]], [[data-production]] and [[web-production]]
  under explicit service, failure and recovery requirements.
  [[public-entry]] admits traffic only to suitable ready API occurrences;
  a live process alone is not evidence of readiness.
  Define startup, drain/replacement, dependency-failure handling,
  finite resource/retry behavior and relevant observations.
  Assign response responsibility and an actionable incident/recovery path.
  Logs and diagnostics protect secrets and sensitive data;
  observation must preserve the access and transaction contracts.
spec: {}
```

A useful first observation set distinguishes refusal, authorization failure, technical failure and uncertain commit; it supports correlation without recording secret values or unnecessary member data. State health/admission semantics and permitted public diagnostic content, choose meaningful metrics and response conditions against actual service requirements, and account for database capacity and backup/recovery. Assign responsibility for dependency/security updates and vulnerability response through the same declared build/release path. Select actual thresholds, failure-domain requirements, retention, loss/time objectives and response responsibilities from the project requirements. A generic vocabulary does not require invented defaults. The [recovery contract](standard-library-data-and-views.md#6-backup-and-recovery) remains the owner of the proposed isolated restore method and its limits.

## One expectation from Rule to Scenario to Check

Files: library/unauthorized-borrow.yaml and library/access-denial-check.yaml, respectively. No new kind or structured relationship list is needed.

```yaml
imports:
  - package: markitect/verification
    as: Verification
  - package: lending
    as: Lending
id: unauthorized-borrow
kind: Verification.Scenario
body: >-
  A member and an available copy satisfy the domain conditions for
  [[Lending.borrow]]. A caller without the entitlement required by
  [[access-policy]] sends an otherwise valid request directly to
  [[library-http-api]]. Access is denied without protected data in
  the response; member, copy and loan state remain unchanged.
spec: {}
```

```yaml
imports:
  - package: markitect/verification
    as: Verification
id: access-denial-check
kind: Verification.Check
body: >-
  Examine [[unauthorized-borrow]] through the actual [[library-http-api]]
  under [[access-policy]] at the identified candidate/configuration.
  Use controlled domain fixtures and legitimate operator inspection
  to observe state before and after an otherwise valid request,
  first without identity and then with identity but no entitlement.
  Success requires denial without protected response data, unchanged
  member/copy/loan state, and a permitted operator request succeeding.
  Retain evidence without live credentials. This Check establishes
  this access boundary, not complete web security.
spec: {}
```

The positive control prevents an always-denying or unusable service from passing this Check. These are planned examination methods; no requests, tests, release, credential rotation, provider execution or restore were performed by this documentation work.

## What remains before closure

For library design, the owner selected the fourteen Kinds and this coverage, with no additional mandatory security/configuration/build/test/release/operations Package or Kind. Project rules above can already express the requested subjects. Specialized reusable libraries remain possible when actual repeated contracts justify them; a generic label such as SecurityPolicy or Pipeline alone supplies no missing capability.

For this particular Web Library, complete these requirement groups before claiming it is fully specified:

1. Access/trust and data lifecycle: actual operator entitlement, identity/session trust, transport/browser protection and relevant retained-data/log/backup handling.
2. Executable boundary and configuration: complete HTTP/operation/error semantics, chosen or permitted technology, concrete settings/defaults/source precedence and validation, and usable local build/test commands.
3. Delivery compatibility: actual deliverables and evidence subjects, frontend/API/schema compatibility, release authority/path and migration/correction/recovery admission.
4. Operation: meaningful load/failure assumptions, response ownership, service/recovery objectives, retention and observation/incident criteria.

These are project decisions or explicitly bounded realization freedoms, not evidence that another universal Kind is required. A generic vocabulary can be selected and implemented before every production value of this reference project is fixed; production readiness cannot be claimed from that vocabulary selection. Keep the requested observation of spec: exact configuration consumers or artifact-processing contracts may justify future structure, but the six-topic walkthrough alone does not.

Return to the [discussion record](standard-library.md) or [design map](README.md).
