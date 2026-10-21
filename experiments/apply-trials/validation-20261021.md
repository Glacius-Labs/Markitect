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

After the initial composition, Main advanced through #566, #570 and C's #562 to `f09f006ac575b7681a3baebe0649ce38d3124fe0`. B composed that integrated stand as `1003130150b10ff90cecbd10915ec9f7fe557eca`; the navigation conflict retained both B's detailed kit entry and C's method entry. The second product executable is built from that exact composition with `vcs.modified=false`, SHA256 `a4f169579357a753e14edd1d3f93ef77faba7cca94acfae58b57f626baff41b9`. Later kit-only changes do not alter that executable's identity.

The successor's first broad `evalcheck.py` and combined Stockroom/Depot/Hall `selfcheck.py` processes were explicitly interrupted after confirmed evaluator defects and before full output. Their empty/partial outputs and `interrupted-successor-checks.txt` are retained. They are neither successful full checks nor provider attempts. The renewed checks focus on the corrected contracts and affected fixtures; no complete successor fleet/full-kit result is claimed. The predecessor's actual full results remain separately bound above.

For coordination: all newly confirmed findings in this record are B evaluator/control defects, not demonstrated production failures for A. Production already provides the required `Pass.tree`, `Finding.candidate`, `closedPass`, `closedAs`, `closedReason`, `closedEvidence`, Event `pass` and Pass purpose bindings. B had dropped or failed to correlate them. An accepted model-only assessment is a sealed `Satisfied` with Run `Incomplete` and `AssessmentAccepted`; it is not artifact verification. Root-confirmed model problems are Failed and technical/unfinished assessment is incomplete, as C's integrated scorer records.

## Closing source, controls and evidence

Main subsequently integrated observation kinds (#571) and reusable Manager containers (#572), reaching `fd94e365f9c1c060c9419be67b70ca072b6c1128`. B composed those changes as `b0635a60b28320f692988a8da3b6103a7ca43585`; its closing product executable is built at `8198b318ee106501c1c8d16760c2c14b6f7100e0`, `vcs.modified=false`, SHA256 `5fb24d10df3a9fe408c178d5c1a7e5d415ed5d1c4a5c3d41bf7264e08c467570`. This adds no B production implementation or runtime evaluation.

The current source supplies `Finding.kind` (`deviation`, `suspected`, `model`, `suggestion`) and `closedCandidate`. The evaluator preserves them, does not label a properly investigated and disproved `suspected` observation a false factual claim, and rejects a disproof whose closing candidate differs. No missing production evidence field is established.

The focused successor script completed 53 declared verdict controls, A4 history, DEP/disproof controls and the conforming D-U stand (22/22 external checks plus own tests and three preserved tempting spots). Its aggregate result is **failed**: the new child-suite command used the wrong unittest discovery root and both child imports failed. The command is corrected to `discover -s CHILD -t .`. The separate corrective supplement invokes the corrected helper on the same frozen H-I bytes: four Rooms tests and five Bookings tests pass; the independent reserve-agreement check rejects H-I and accepts H-N. The supplement also renews DEP and the current observation/disproof schema controls. All its assertions pass. The failed original output remains unchanged; this is not a full new green selfcheck/fleet result.

The first working-tree structural check failed only on four ignored bytecode files created during review. They were preserved, without discarding bytes, under `preserved-bytecode/` with their own manifest. The review session confirmed Python-based repros as their likely source, without certifying the timestamp-specific invocation. Fixed-HEAD preparation passed; subsequent working-tree preparation and product/examples contexts passed after preservation. No source ignore exception was added.

Final input lock SHA256: `c18bfd3aa12db8990a1bb00aa600fa445c2f89de4652ac091d200f05df0e38fe`; kit digest: `6877e9a7c57e1063ded27bd2e800263ed2bc7d89844ba2064bfb1bd81e0ab63e`. All 31 Markitect packet heads and all 11 ordinary-Agent packet heads are unchanged from the prior checked lock. The four C packet heads are also unchanged. The prior output manifest checked every Markitect packet structurally, and all eleven plan packets passed complete-definition/metadata parity against the product. Changed evaluator code is bound by the new kit digest; the new observation/helper controls ran against that code.

The existing multi-area controls are frozen separately in `existing-controls.json`: D-M head `a2046fa5c5d5f598eb1c4857db64eba9ee0faadc`, tree `a34e2f545df0d467004da26a4a273de0640476a1`, structurally passed despite semantic faults; D-R head `bb88f0e88cdeb3a61005c903b8942b564e7ffd1f`, tree `975ec38caa0063592370a08f572f900ea1ebf842`, correctly refused by strict accounting. D-M has a matched ordinary-Agent plan packet in that manifest. This retains ownership/accounting controls alongside the selected S-A artifact slot, without allocating another provider case. The preserved predecessor Depot selfcheck contains its actual D-M fault/oracle results.

The 33 neutral blind-v2 model-input files were also preserved outside writer packets in `preserved-blind-v2.zip`, SHA256 `3350b4ab091f48c5547767e93791b8403639a97003d0eba61faae0ca3be14f0a`, with a per-file manifest. No blind review transcript is invented from those inputs.

All filenames below are relative to `C:/Users/Consiliari/.codex/evidence/markitect-session-b-20261021/`. They bind the actual retained outputs, including failures.

| Evidence | Actual scope/result | SHA256 |
| --- | --- | --- |
| `preserved-manifest.json` | Original scratchpad output/binary hash inventory | `b24cbfc4654aa8af0d9865e8555158ecb8c1fbb55005169f1fb845b1a0f85e18` |
| `quick-final.json` | Passed: lock, eleven complete plan packets, context and process controls at ecda26b7 | `5fd7d7f52ced78e29d5ddbe7ae258d004903075e83090f92925ba5e07fa77aff` |
| `packets-final.json` | Passed: all 31 Markitect packet structural checks | `6c371d6323119d0a4391c1b9b210609e94fe3f771f94519c63d41d3cec8c2b58` |
| `lock-linux-final.json` | Passed: Windows/Linux identity for prior c0d8501f lock | `5ab408816734165a70742e837b914f746f2d0895d2645510bb03db19e820644b` |
| `focused-final.json` | Failed aggregate: 53 matching verdicts; child discovery invocation failed | `b2c3d57974b9ce73e73f4aec5f0133844cb5358c8a47186bafb55ba5a1b6031c` |
| `supplement-final.json` | Passed corrected child discovery, parent positive/negative, observation/disproof and DEP controls | `2c13a6d09d19ef0623d84461a5bfd96a2ae719537aaee7faef81d7be35fe531f` |
| `existing-controls.json` | D-M/D-R exact packets and matched D-M plan | `39903890b1763ddde146973df29ec9912f2d3a3a3ee8d8d55d69c5ae5e6f5084` |
| `repo-final-check.json` | Failed working preparation: four unregistered bytecode files | `17059e5190be689c4afecfb9f1718f5fae4f52f0e4df6d0961aab9d4ab143f52` |
| `repo-final-fixed-check.json` | Passed fixed-HEAD structural preparation | `a0a7a6e92259bdd788d3085db3ab7055916ca0d7bbb59e4072ff966bf2a65632` |
| `repo-final-working-check.json` | Passed working preparation after preserving bytecode | `62930897e2ff5fda34e9aec66903efcc62932ba945defdd18bce45d3967a8b8d` |

The closing PR/chat records renewed final-head checks, review, merge and Main read-back. Code-source validation is reused only for unchanged packet bytes and unaffected assertions, with the corrected discovery and observation contracts tested separately. Hosted CI was not requested. B used zero provider evaluation attempts.

## Remaining D dependencies

No realistic project/driver was promoted from the unregistered P3 drafts. D still lacks a validated bounded realistic scenario, a real ordinary-Agent execution driver with final-record/stop/resource behavior, and the comparative measurement driver with actual usage/time/cessation evidence. A final integrated A build including C's method, matched environment/image/profile, selected D cases/order and explicit remaining run authority must be frozen by coordination. Do not substitute stubs or start a new framework/series to conceal those gaps. The separate ordinary-Agent public-gap comparison is unrun after #569 changed D's slots; A/C gap evidence does not replace it.
