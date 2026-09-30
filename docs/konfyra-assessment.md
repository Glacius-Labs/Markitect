# Konfyra migration assessment

Assessment date: 2026-09-30. This evaluates the completed RC2 pilot at `705f90b072938324dfe7940f9c815ff19d3b79b4`, compared with local master baseline `74d18c56e20dbb272dc085a389bfdaf291ee4562`. The subsequent Go-bootstrap integration is a new candidate and needs its own checks. This document is a dated assessment; Konfyra dossier `01a0f27a-a479-76bf-886e-6583bbe49200` owns running delivery and acceptance.

## Judgment

The structural migration worked. It gives Konfyra a useful local compiler for its AI mechanisms and makes fixed-input reviews possible while other work continues. The pilot is a credible foundation for further use. It has not yet demonstrated complete semantic coverage, general provider runtime equivalence or production acceptance.

## What migrated

| Resource | Migrated definitions |
|---|---:|
| Rule | 12 |
| Workflow | 8 |
| Skill | 25 |
| Agent | 19 |
| Total original mechanisms | 64 |

These sources now live in adjacent YAML; existing Markdown paths are generated reading views. One documentation-review Contract and one review-snapshots Text were added, along with the Project configuration: 67 configured resources in total. Product code, ordinary ADRs, domain contracts, roles and work-item records were not converted.

The importer normalizes line endings and checks source text and supported metadata. Unknown metadata fails explicitly. Workflow links were reviewed before becoming typed `uses` relationships; the importer only proposes candidates. This preserves the distinction between navigation and an actual dependency.

Original provider models, effort, permissions, tools and limits were retained. The initial provider output regeneration was unchanged; RC2 deliberately clarified the authoring Skill description. The existing renderer still owns provider mappings and retirement lists. Empty Markitect `targets` prevents two renderers from owning the same provider files.

## How normal use works

1. The agent uses Konfyra's router and authoring Skill to select the owner.
2. It changes the canonical YAML body and any machine-relevant dependencies or declared files.
3. Markitect formats and renders adjacent views. Konfyra's renderer regenerates native provider entrypoints.
4. Structural and repository checks run locally, typically in seconds after caches are warm.
5. The complete candidate is committed once. Context, impact and verification use that fixed SHA.
6. A semantic reviewer receives the declared context and a bounded question. Eligible previous reports can be reused if their relevant inputs, tool/configuration and change analysis allow it.

Markitect currently resolves a named Skill/Agent/Workflow; the agent selects that entry from the task. It does not autonomously interpret the user's task, run agents or enforce their behavior. The portable authoring kit and reference/owner queries remain the next UX improvement.

### Parallel work

An in-flight review of commit A keeps reading A even when another checkout creates B. After B is integrated, evaluate its impact and evidence separately. The reviewer cannot accidentally read a mixture of working-tree versions through Markitect's fixed context. Separate worktrees and normal Git integration still matter for concurrent writes and merge conflicts.

This resolves the snapshot part of the user's hours-long consistency-review problem. It does not prove that every relevant prose dependency was declared. Reviews based on live browser state, undeclared code or external systems need their own recorded inputs and cannot automatically reuse a local context-only result.

## Evidence actually present

| Evidence | Result and boundary |
|---|---|
| Source distribution | RC2 archive SHA-256 matches lock: `0504656a794654d1301b472c3d2a8d41851e585739844291d690bbd0c5ae999f` |
| Immutable local verification | `verify-rc2.yaml` passes at the exact RC2 commit; snapshot `139aa97fff3fe8477a3dfbbb6a1e9da0603d31c353f0e858dcdcc533b57c3985` |
| Go core | Full tests and vet previously passed on Windows and isolated Linux |
| Repository regressions | RC2 Windows: 34 passed, one symlink privilege skip; Linux: 35 passed |
| Bounded semantic review | Actual Luna High report found no material findings for the authoring-context question; it did not review the entire migration diff or runtime behavior |
| Advisory reuse | Reused that report at the same commit in 802 ms without a model call; changed-input invalidation is covered separately by regression tests |
| Hosted CI | Configured, no successful hosted run evidenced |
| Claude runtime | Last pilot authentication check reported logged out; no runtime acceptance |

These are local records, not authenticated CI attestations. Historical test counts describe RC2, including its former Python bootstrap; they do not certify the new Go bootstrap. Exact reports remain in the consumer's `.artifacts/markitect/` evidence directory and its dossier.

Observed local samples: original governance run 7.473 s; migrated warm run 5.826 s; Markitect structural checks 1.041–1.975 s. These are small, uncontrolled samples. No sustained token-cost reduction has yet been measured. The 88,942-byte authoring context also shows that shared rules still have a real context cost.

## Gaps worth refining

- **Declared dependency completeness:** typing 64 files does not prove that every normative prose relationship is modelled. Audit affected entrypoints through real tasks, not repeated full-repository rereads.
- **Conservative invalidation:** all rules attached to the parent docs area reach descendant resources; unknown/configuration/inventory changes can affect every entry. Explain why a rule/input applies before considering narrower policy.
- **Authoring friction:** multiple explicit commands and two renderers remain. First improve the portable skill and query operations. Consolidate provider rendering only with parity for existing mappings and retirements.
- **Go consistency:** replace the Markitect-owned Python bootstrap/tests. Keep existing consumer governance ownership explicit; porting all Konfyra tooling is a separate scope.
- **Runtime evidence:** generated-file parity and a Codex-based semantic reviewer do not establish a real Codex/Claude workflow runtime acceptance.
- **Reuse measurement:** demonstrate unchanged, local, shared-rule and unknown-input cases over several actual tasks, including a different commit with genuinely unaffected declared context.

## Next acceptance exercise

Use one small authoring request, one shared Rule change and one simultaneous independent branch change. Record selected entry, declared inputs, changed/affected resources, regenerated files, checks, semantic review scope and actual model work. Run the installed provider workflows, hosted gates and human acceptance on the final candidate. Only then advance the agreed Cockpit migration.
