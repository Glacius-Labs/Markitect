# Implementation plan

Updated 2026-09-30. This is the Markitect product roadmap. The Konfyra work-item dossier remains the owner of that consumer's running delivery state and acceptance. [Architecture](architecture.md) and [refinement decisions](refinement.md) own design; [pilot assessment](konfyra-assessment.md) records observed results.

## 1. Establish the independent project — completed locally

- Preserve existing Go source history in an independent checkout.
- Move product architecture and planning ownership here; leave relocation pointers in the Cockpit.
- Provide an executable synthetic example and a concise source/release guide.
- Replace Markitect's Python bootstrap/tests with Go and exercise the integration against Konfyra's existing pinned RC2 engine.
- Verify standalone builds/tests, consumer bootstrap and generated-view parity without a hidden source dependency on either consumer.

Completion means a project that can be built, understood and changed on its own. The user selected the private GitHub repository `Glacius-Labs/Markitect` and a local checkout under Glacius Labs. The module now follows that repository identity. Reconcile the reviewed branch there, run hosted Windows/Linux gates and provision explicit commit artifacts. API-domain migration, public license and public releases remain separate decisions. Development identifies itself as `0.1.0-rc.3-dev`; existing consumer release pins do not float.

## 2. Improve agent authoring — implemented and exercised locally

Build core authoring using current types and ship its portable resources with the tool. Authoring is part of Markitect, not an optional package. Add deterministic resource lookup, owner, incoming references and context-reason queries to the CLI. Keep task intent interpretation in the authoring agent. Existing structured YAML output is the first interface; no MCP dependency.

Acceptance: a user asks for a rollback-plan requirement; an agent finds the owner, modifies the proper Rule, reports affected consumers, generates and validates a fixed candidate, and explains what semantic work remains. No duplicate rule, guessed global scope or manually maintained provider file.

The [actual exercise](authoring-assessment.md) records the initial failed boundary, the normalized-example fix and the successful fresh-agent rerun. The suite now includes structural queries, embedded authoring and deterministic impact/context scenarios. Consumer upgrade and acceptance remain separate below.

### Delivery plan from the vision

Work on `feature/core-authoring` from the independent source repository. Keep file ownership disjoint while implementing; integrate and review one candidate before changing a consumer pin.

| Work | Artifact and owner | Acceptance |
|---|---|---|
| Resolve structure once | Core graph relationship provenance and query tests | Existing context/impact edges remain compatible; reasons distinguish direct references, area rules and selected bindings. |
| Inspect without broad searching | Application queries and CLI `find`/`explain` | Sorted YAML results expose canonical paths, area ownership, direct references and Contract implementations; fixed revisions ignore later edits. |
| Ship authoring as core | Embedded canonical Rule, Text, Workflow and Skill with a small Project | `authoring` compiles these resources through the normal parser/graph/context pipeline; source packages contain the assets. |
| Test real editing behavior | Isolated synthetic consumer exercise | A separate agent uses the bundled workflow to update one owner, regenerate and check, then report fixed impact and semantic limits. |
| Establish measurement | Go correctness scenarios and benchmarks; [measurement protocol](measurement.md) | Exact expected effects for scoped/shared changes and deleted dependencies; unknown inputs remain conservative. No invented model-token measurements. |
| Review and integrate | Focused independent review and standalone checks | Windows tests/vet/schema/example, source-package build, Linux tests, read-only Konfyra queries and reviewed documentation. |

The supplied vision is a target, not an instruction to add every interface now. This slice does not require a package resolver, metrics database, MCP server, IDE extension or operator. Their order below follows concrete consumer value. The new vision's optional-authoring wording is superseded by the user's core decision.

## 3. Prove versioned content reuse

Implement the minimal package slice in [the refined design](refinement.md): package identity, exact version, exports/private closure, qualified references, a versioned lock, offline immutable archive and evidence integration. These are one feature boundary. Include a migration for the existing tool lock before changing its meaning.

Pilot one small generic engineering-policy package and two synthetic consumers. Keep core authoring owned and released by Markitect itself. Then extract only confirmed generic material from Konfyra. Local customer policy and delivery authority stay at their owner.

Acceptance: checks work offline; exports and version conflicts fail correctly; package-only changes affect context/impact/review; selected Contract implementations remain explicit; a consumer cannot reference private resources directly. No template sync, registry, ranges, solver or transitive imports in this first slice.

## 4. Finish Konfyra acceptance

Keep its dedicated pilot branch as the testbed. Re-evaluate evidence for any new integrated candidate. Test normal authoring and bounded changes, simultaneous isolated work and meaningful repeated-review savings. Complete the applicable independent semantic review, provider runtime exercise, hosted CI and human acceptance through the existing consumer workflow.

Use the previous RC2 baseline as evidence for that fixed candidate, not as approval of later changes. A renderer replacement is its own parity exercise; the existing consumer renderer currently owns provider mappings and retirements.

## 5. Apply the accepted system to the Cockpit

Migrate after agreed Konfyra acceptance. Use the same released tool and generic package; explicitly compose General, Consiliari, customer and project areas. Verify tasks that stop above project level and tests for cross-customer isolation. Do not create another tool implementation or inherit all customer rules through folder placement.

## 6. Add interfaces when they remove measured friction

- One local `init` template once package composition is proven.
- MCP over the existing query API when agent calls benefit.
- LSP/IDE navigation and diagnostics when editing/inspection needs justify it.
- Graph visualization as derived output.
- Runtime/operator integration only after there is real desired state to reconcile.

CI enforcement should arrive with hosted delivery, before optional user interfaces. No obligation to follow the concept note's illustrative LSP-before-MCP sequence.

## Glacius Labs reconciliation

The GitHub repository is the product source owner. Move the complete independent local repository into `Glacius Labs/Markitect`, repair its registered worktree paths, and preserve the existing history rather than copying a second implementation. Keep the active implementation worktree until reconciliation and verification finish. Update Cockpit relocation pointers to the stable repository URLs. CI publishes only commit-specific development artifacts after both platform gates pass; consumers adopt a selected archive, lock and Go bootstrap through their own candidate workflow. No branch-following tool dependency or automatic promotion is introduced.

## Measures

Record structural check duration, context size, affected-entry count, actual model calls/tokens, repeated evidence reuse and the number of manual steps for the same task. Compare bounded local changes, shared Rule changes and unmodelled changes. Passing tests and one reused report alone do not quantify sustained token savings.
