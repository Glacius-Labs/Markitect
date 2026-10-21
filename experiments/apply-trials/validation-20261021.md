# Session B successor validation, 21 October 2026

Operator evidence for [#564](https://github.com/Glacius-Labs/Markitect/pull/564), under the [current handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#primary-goal-supplement-and-current-handoff). No provider evaluation was run by B. Local deterministic controls, source review, actual product execution, hosted CI and owner acceptance remain separate.

## Source and ownership

The supplied worktree `C:/Users/Consiliari/.claude/worktrees/session-b-fixtures`, branch `dev/session-b-eval-fixtures`, was clean at `831a919855ff14696dc8286ebac1c0122fa0bd73`. Remote Main was `3d100e2498034f5e7fe802c85e5833d2eb9c7178`; composing it produced `21e005457602fdd95a5475bdbe2e8a047d5cc283`. The first successor product executable was built from that composition with Go 1.27.1, `vcs.modified=true` because the trial-kit fixes were present. Its SHA256 is `ec2c5c175604a40a370267fbd6cdd8f3322be063b64b76ef477d30fcab2e22a9`. That build is not the subsequent closed-Run product of #566.

B writes only the existing trial kit, its exact examples registry and navigation. The original checkout's foreign untracked `internal/apply/assessment_test.go` was preserved. A's production/runtime/bootstrap/self-model paths, C's method files and the P3 WIP were not edited. `wip/session-b-p3-drafts` at `e514b9543c99c049c550bf177a433047bdb00c6d` remains excluded.

## Review findings and corrections

The successor independently reviewed the full predecessor candidate, especially `f1f517742f0b9f4bd11cd39d9196935f7040e5c9` and `831a919855ff14696dc8286ebac1c0122fa0bd73`. A separate read-only reviewer covered the full diff and independently reviewed the successor's evaluator and controls. The authoring reviewer's two narrow control fixes were reviewed by the successor.

- A boolean blocking flag or a correction verdict on an oracle-conforming stand is insufficient to prove a false factual blocker. Retain the actual finding, immutable candidate, disputes and independent closure evidence. An unresolved concrete concern remains incomplete; a factual claim independently disproved with evidence is a quality error. The permitted recovery through independent disproof and a fresh complete Verify of the unchanged candidate remains valid separately from that quality score.
- Repeated references to one note, or equal wording under distinct ids, do not prove reopening as a new duty. Retain possible duplicates for semantic reading; do not automatically fail the process for their text alone. Actual optional writes and unjustified extra work remain failures.
- The plan verifier previously accepted missing kind, owner, source/digest or changed title metadata. It now checks those fields against product context, with omission/substitution controls.
- The context selector computed missing connected contracts but still returned passed. Missing connected contracts now fail, with a regression control.
- #566 changes the exact source-bound Model assessment stop reason. The evaluator must recognize that reason without treating a technical failure or arbitrary mention of model errors as a semantic stop.
- Independent disproof now also requires the actual later review Pass, its `tree`, and the raising Pass to match the finding's `candidate`; missing/wrong closure-Pass and changed-candidate controls reject fabricated bindings.
- DEP cannot borrow assessment or Verify questions/acceptances as realization evidence. Plan declarations may legitimately target later Prepare/Implement work; the evaluator keeps that relation, matches prerequisite acceptance and premature-work analysis to the declared purpose, and rejects an acceptance from the wrong phase.

These fixes change the new evaluator/lock. Prior attempts and their original rubric, failures and lock remain historical; none is silently rescored.

## Semantic controls for #569

| Dimension | Conforming and faulty control | Independent discriminator and limit |
| --- | --- | --- |
| Artifact conformance | S-N versus existing S-A (`ARTIFACTS`) | Code/configuration, green wrong exact-issue test, README and sorting follow the unchanged model. Configuration repair alone or green own tests cannot pass all external checks. |
| Material clause within complete IDs | S-N with isolated a4 | The entire `stockroom/list` Definition remains present, but ascending normalized-SKU order is omitted in behavior. Own tests stay green; the independent `list-order` check rejects it. Isolated a2 similarly has green wrong tests but fails `issue-exact`. |
| Semantic parent integration | H-N versus H-I (`DEP-SEMANTIC`) | Both child suites pass, but rooms displays one more bookable seat than bookings admits. `reserve-agreement` rejects it. Local acceptance, routing and version events alone do not establish this invariant. |
| Preservation and necessary repair | A2 v2 D-U versus A1 v2 D-UH | The same unattractive stand with and without the real hyphen-name defect. Repair must preserve the three tempting areas. |
| Closed-run policy | A4-gap versus A4-fixed; A5 separately | The gap stops before realization; corrected model is a new Run. A5 changes an already explicit 60 rule to 40 before a new Run. |

The selected new D slot candidates have plan packets with the same starting artifact bytes and complete visible requirements. Fault edits, answers, expected findings and oracle code stay outside writer packets. They are preparation for existing slots, not extra provider attempts. Coordinator selection, exact A product/profile/image/order freeze and execution authority remain necessary.

## Independent model reading and C handoff

Reading the source Definitions and case edits independently supports these expectations: MA-COMPLETE v2 S-GO is accepted because omitted error wording, internal organization and storage layout leave ordinary implementation freedom; MA-FREE v2 S-FREEU is accepted because a bounded name maximum is expressly delegated and internal control characters are an unusual but consistent rule. MA-GAP S-QV refers to a public maximum but supplies neither a value nor delegation, so `model_gap` is material. MA-CONTRA S-XG requires distinct case-sensitive SKUs while inputs/add-item normalize and reject their duplicate, so `model_error` is sourced across those Definitions. This is source reasoning, not observed team/provider behavior. The key retains minor ambiguities and earlier blind-reader evidence.

All controls use `direct`, profiles `execute: project-agent` / `verify: project-agent`, and packet ceilings 30 minutes / 20-minute turns / 400 Agent turns / 12 Passes. C's separate series ceiling remains 12 attempts / 90 minutes / 10 minutes per attempt, with consumed attempts preserved. MA-NEAR is only a prepared counter-control, outside those four authorized cases. No C or D evaluation is started by this kit.

The assessment-key SHA256 remains `cb757240c947ef254c800baec3b0fcdb4e4ccdbf8029aafccd0024e600795bd4`; context expectation SHA256 remains `8b52172b41f2bf25fa4926c64f41a5953c50b1316aa4dfdc1e10b7a31933ccc7`. Exact packet commits, trees, budgets and individual file hashes live in `scenarios.lock.json`, rather than in a second editable input manifest.

## Preserved predecessor evidence

The temporary handoff scratchpad was copied without alteration to `C:/Users/Consiliari/.codex/evidence/markitect-session-b-20261021/`. `preserved-manifest.json` binds every selected original filename, size and SHA256. The copied original executable hashes to `02571dd97a40d294950e3ddeb3d2cdec1b8ad958ef3740f5dddb2bf4d40f5bfe`. Its handoff described Main including #559, but did not supply an exact full build source SHA; do not invent one.

Preserved files include every `full-r3-*` JSON/error/completion file, `full-r4-hall.*`, `full-r4-stockroom.*`, `evalcheck-quick-r4.json`, `repo-check-r4.json` and the original executable. The first Hall check failed because the invalid reserve-3 variant's own-test failure was undeclared. The first Stockroom check failed on undeclared own-test failures for free-40/free-120/free-30/free-121/free-undocumented/s1-chose-80. The later Hall/Stockroom outputs retain the actual reruns after `831a9198` corrected the variant declarations/boundary tests. The initial successful evalcheck/depot checks are retained alongside those failures. No blind input/key was copied into a writer packet.

## Local validation and final freeze

The delivering PR and final chat bind the reviewed head and merge SHA. The evidence table below records actual outputs and product identities; no hosted CI was requested under the current integration rule. Local controls do not establish provider success, Apply acceptance, release or owner acceptance.

After the initial composition, Main advanced through #566, #570 and C's #562 to `f09f006ac575b7681a3baebe0649ce38d3124fe0`. B composed that integrated stand as `1003130150b10ff90cecbd10915ec9f7fe557eca`; the navigation conflict retained both B's detailed kit entry and C's method entry. The final product executable is built from that exact composition with `vcs.modified=false`, SHA256 `a4f169579357a753e14edd1d3f93ef77faba7cca94acfae58b57f626baff41b9`. Later kit-only changes do not alter that executable's identity.

The successor's first broad `evalcheck.py` and combined Stockroom/Depot/Hall `selfcheck.py` processes were explicitly interrupted after confirmed evaluator defects and before full output. Their empty/partial outputs and `interrupted-successor-checks.txt` are retained. They are neither successful full checks nor provider attempts. The renewed checks focus on the corrected contracts and affected fixtures; no complete successor fleet/full-kit result is claimed. The predecessor's actual full results remain separately bound above.

For coordination: all newly confirmed findings in this record are B evaluator/control defects, not demonstrated production failures for A. Production already provides the required `Pass.tree`, `Finding.candidate`, `closedPass`, `closedAs`, `closedReason`, `closedEvidence`, Event `pass` and Pass purpose bindings. B had dropped or failed to correlate them. An accepted model-only assessment is a sealed `Satisfied` with Run `Incomplete` and `AssessmentAccepted`; it is not artifact verification. Root-confirmed model problems are Failed and technical/unfinished assessment is incomplete, as C's integrated scorer records.

## Remaining D dependencies

No realistic project/driver was promoted from the unregistered P3 drafts. D still lacks a validated bounded realistic scenario, a real ordinary-Agent execution driver with final-record/stop/resource behavior, and the comparative measurement driver with actual usage/time/cessation evidence. A final integrated A build including C's method, matched environment/image/profile, selected D cases/order and explicit remaining run authority must be frozen by coordination. Do not substitute stubs or start a new framework/series to conceal those gaps. The separate ordinary-Agent public-gap comparison is unrun after #569 changed D's slots; A/C gap evidence does not replace it.
