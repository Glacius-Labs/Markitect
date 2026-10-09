# Independent Knowledge Graph variant handoff

Implementation source: `d360ec0199d4333271fd6ec6c66227a2da4aca83`, pushed on `codex/knowledge-graph` in the independent product-knowledge-graph worktree. Starting Management reference: `1495e1be7b0046711531fa42c8407fba67b8e814`. Evidence additions after the implementation commit are documentation and ownership declarations; their commit is not a replacement source label for the full suite.

Status: fixed-source full Verify at d360ec01 FAILED in two projectcli test fixtures. Both fixture repairs passed targeted reruns and parent review; a fresh fixed-source full replay is next. Production behavior is unchanged from d360ec01. Accepted-Main reconciliation remains pending because Overseer has not supplied the exact accepted Main SHA. This variant has not been merged into Main, published, installed, or evaluated in a real actor study.

## Delivered behavior

YAML remains canonical. ProjectModel supports explicit Decisions and Statement IdentityChanges, with source provenance, scoped visibility and conservative Impact. A derived pure graph index supports deterministic bounded navigation. Host composes actual definition/file ownership, declared dependencies and explicitly selected existing operational records into a scoped read-only view. CLI and MCP expose graph, relations, explain, trace, history and coverage through one operation. The user guide and standalone executable example document those queries and their boundaries.

The graph preserves distinct relationship effects; it does not replace Context, Impact, execution dependencies or guarded Apply. Forbidden and absent lookups match. Direct public contract owners carry identity only. Missing runs/tasks are expected links, not observed work; stale, historical, partial and unknown record bindings remain explicit. Decisions record actors without authenticating them. IdentityChanges declare previous identities without proving historical existence or semantic equivalence. No graph database, RDF engine, provider integration or new dependency was added.

## Validation and evidence

Independent Luna High model and Host reviews closed their material findings and reread the targeted fixes. Focused component tests, architecture, whole-repository vet, artifact ownership and module checks passed on the coordinated candidate before freezing it.

The final actual CLI/MCP process journey was built from exactly `d360ec0199d4333271fd6ec6c66227a2da4aca83`. Its standalone fixture revision is `963a5f1e00ada83eb59072f60f426c793117d034`; binary SHA256 is `ac709206ef05e14780e423cca899d154a676ac5ae2667f25bc755032206ae24a`. Project check passed with accounted/conforming coverage, all six queries passed, and actual stdio initialization, six-tool listing and graph invocation passed. CLI and MCP returned the same graph digest. Provider calls: zero. The [machine receipt](evidence/user-journey-d360ec01.json) records individual query digests and output hashes.

Full fixed-source Verify at d360ec01 exited 1 after 1,748,072 ms. Architecture, artifact ownership and module checks passed; all Go packages except projectcli passed. The two failing tests concern an omitted child Manager delegation list and a negative privacy fixture that targeted an intentionally visible public-contract owner identity. Claude/Codex protocol checks were NOT RUN. See the [validation ledger](continuation-validation.md), [raw failed log](evidence/verify-d360ec01.log) and [receipt](evidence/verify-d360ec01-receipt.json); the failed log SHA256 is `08d096b27385fc97c2ba61053e573758196c54ff3e10c1c086b3863c297b129c`. This is retained separately from subsequent repairs/checks. The [older failed Verify](evidence/verify-08b273f6.log) remains byte exact with SHA256 `4b8eaaf56d6c286de054eb8e4676873780fc63e56c5804ece1f580e716eaf68b`. It has not been relabeled as passing or replayed as a measured failure of the starting reference.

The validated ProjectRun record port tests cover a complete stored candidate/check/review/verification/Apply chain. Evidence-to-graph real-loader fixtures cover Exploration, Brownfield and a planned run with missing execution; the complete operational chain was not exercised as a fresh end-to-end evidence fixture or real actor run. Runtime fingerprints are not compared against a current provider fingerprint and remain explicitly unknown. Hosted Windows/Linux CI, native provider proof, autonomous engineering benefit and human acceptance have not been established by this local validation.

## Remaining integration and evaluation

1. Overseer supplies the exact accepted Main SHA after the Designer/Integration Main package is finished.
2. Merge that SHA into this branch, resolve source/config/generated-output conflicts, and preserve both Main behavior and scoped KG contracts. Do not merge KG into Main automatically.
3. Run the combined candidate's relevant Main/KG checks and independent review. Bind the report to the combined source SHA and actual configuration.
4. Evaluate the same twelve questions in the existing sandbox with equal staged requirements and resources. Separate deterministic mechanics, actual actor results, semantic quality and human acceptance; do not turn fixture checks into study results.

[RDF fit decision](RDF-fit-decision.md) recommends an optional derived RDF export/SPARQL adapter only after concrete interchange or user-composed query needs justify it and parity/privacy/determinism gates pass. Canonical RDF migration and engine selection remain separate decisions. Existing native queries may be sufficient; agent familiarity and compiler simplification are hypotheses to measure rather than guarantees.

See the [backlog](backlog.md), [validation ledger](continuation-validation.md), [twelve-question outcomes](KG07-question-outcomes.md), and [user guide](../../project-knowledge.md) for concrete contracts and commands.
