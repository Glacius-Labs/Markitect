# Documentation authoring pilot

Recorded 2026-10-01. The first exercise was **incomplete**: one check-only run and one full-query run completed. The third actor stopped at a provider usage limit before making changes; three remaining runs were not started. The planned three runs per arm and blind semantic review did not complete. A separate repeat with a corrected task card is recorded below. Neither exercise establishes a comparative productivity, token, or cost conclusion.

[Measurement](measurement.md) owns the procedure. The earlier [core authoring exercise](authoring-assessment.md) is separate historical evidence.

The recorded actor outcomes below belong to the dated source, tool and fixture identified here. The current example contract uses canonical `.markitect/areas/` resources and explicitly selected Markdown views under `docs/markitect/`; this change does not reinterpret those pilot outcomes as a replay.

## Fixed inputs

- Product source: `070f51c1828a123530167d985890039579b28068`.
- Windows native CLI: Markitect 0.4.0; SHA-256 `d50bd4ec1b37a0ead90a0022441753463c10cb1d21b264574005de7cf1ef4d4c`.
- Seed: a copy of the complete `examples/documentation` fixture, committed as `edf003cbc2ec656913654195a523712b399865b8`. Clones used `core.autocrlf=false` from checkout onward.
- Fresh actors: `gpt-6-luna`, high effort, no inherited task history; sequential execution and identical reference documentation.
- Both arms could use `check`, `format`, `render`, help, Git and file tools. The full-query arm additionally had `authoring`, `find`, `explain`, `context` and `impact`.

The task named the owner, identity and path of a new `implementation/Rule/retry-safety`, required a binding from `startup-review`, and preserved the existing `worker-behavior` dependency and Go input. The Rule had to require prospective review of idempotence or duplicate-effect protection, bounded elapsed time and attempts, transient/permanent error handling, and tests for both classes. It could not assert those behaviors already existed. Go code, Project topology and the independent operations area were outside the requested change.

The executable, fixture, task and Go structural scorer were frozen before actors started. No earlier result was supplied to a later actor. The source and tool hashes above identify the measured candidate; later release additions, including bundled notices, are not retroactively measured by this exercise.

## Observed results

| Run | Arm | Candidate | Actor-recorded UTC interval | Independent fixed checks | Strict mutation oracle |
|---|---|---|---|---|---|
| 01 | Check-only | `ad4c387b516c953d3b945f85c42f7c2f02a8c3dd` | 03:40:02–03:47:27; 445 s | check, format and render passed | Rejected extra Workflow prose change |
| 02 | Full queries | `31fad57abe16775960065669eeae30f19528230f` | 03:48:29–03:52:19; 230 s | check, format and render passed | Rejected extra Workflow prose change |
| 03 | Check-only | No candidate; unchanged seed | Unavailable | Not run | Provider usage-limit interruption |
| 04–06 | Planned alternating arms | Not started | Unavailable | Not run | Not run |

Both completed candidates preserved the Go input, Project and operations files and the existing resolved dependency. Their new Rule texts contain all four requested review criteria and frame them as requirements rather than implemented behavior. This was checked by the coordinating reviewer; the planned separate blind semantic review did not run.

The strict predeclared oracle allowed only the new Rule and its view, the Workflow relationship and its view, plus an optional area-router link. Both actors also changed Workflow prose. Run 01 narrowed the original broad startup wording to the requested attempt-limit case; run 02 retained the original sentence and appended an explanation of the new Rule. The mechanical oracle rejected both. The task card did not explicitly forbid every Workflow text edit, so that strict result cannot be treated as an unambiguous failure to follow the request. The oracle and task were kept unchanged for both runs; no failed result was replaced.

The actors reported path-quoting, executable-path and YAML/line-ending corrections. These are self-reported categories, not authenticated command counts. Both final working trees were clean. Independent fixed checks confirmed the final artifacts; they do not authenticate the entire reported command history. Duration uses actor-recorded UTC values and is descriptive. Tokens and monetary cost are unavailable.

## Measurement and product findings

- The task supplied exact ownership and paths, leaving little ambiguity for discovery queries. A later comparison should test a realistic ownership decision and specify the allowed mutation boundary as clearly as its oracle.
- New resource identities conservatively broaden impact across the inventory. Run 02's operations entry was affected for this reason even though its source stayed unchanged. This is a review-invalidation boundary, not evidence of an operations edit.
- The preparation self-check found Windows checkout newline conversion when `core.autocrlf` was set after cloning. Recreating the fixture with that setting applied during clone removed the confound before actors ran. The corrected scorer accepted a valid hand-authored candidate and rejected an unauthorized Go edit.
- Shared-machine, instruction-based isolation is not a hard filesystem sandbox. The fixture contains only neutral synthetic content.

The raw protocol, scorer, actor reports and local candidate repositories are retained in excluded development artifacts. They are local exercise evidence, not hosted release attestations. The incomplete and ambiguous result motivated the repeat below.

## Controlled repeat with an explicit mutation boundary

The repeat used the same pinned product source `070f51c1828a123530167d985890039579b28068`, with a newly built fixed Windows CLI (SHA-256 `3760f9415c1b9faae08df95f229c811175f7e88b94a30ce5861d96e11a1b5181`). Its 11-file seed was commit `c83c17ee0408b35a4d33986d20b6945f9948e416`, tree `650375629def122c67b82b39dbfbcdc5fa90331f`. The frozen task card's SHA-256 was `a9befe28d16faff25da99ab9ff7a324a8afcfaad04ac7cbb1dd8a78d9eea5f5`. It explicitly allowed only the new Rule YAML and view and the existing Workflow YAML and view. The semantic task and the check-only/full-query command permissions stayed the same as in the first exercise.

Six fresh `gpt-6-luna` high-effort actors ran sequentially in alternating arms, three per arm. Each isolated checkout began at the same clean seed with `core.autocrlf=false` and `core.eol=lf`; the coordinator verified its commit, tree and status before dispatch. Earlier candidates and reports were not provided to later actors. The semantic review rubric was frozen before candidates were unblinded (SHA-256 `032a24d9fa190534fcfd353a8c3b5371110bab490ea818b907f595c461e2ac2d`).

| Run | Arm | Candidate commit | Observed elapsed time | Fixed structural and output checks | Four-path boundary |
|---|---|---|---|---|---|
| 01 | Check-only | `5353f02a2a8e0f23f04742e75ddecc7d3550a659` | 18m24s | Passed | Passed |
| 02 | Full queries | `f8bfb105d847a747b900d7746007c494eabdd1a7` | 11m03s | Passed | Passed |
| 03 | Check-only | `7b62cbb6b6a41730e504a76863c0b417d3440146` | 18m31s | Passed | Passed |
| 04 | Full queries | `b93748388c2e9984c15fc32fe47bf6297ef8b7b4` | 19m51s, including approval wait | Passed | Passed |
| 05 | Check-only | `e83793d9c01160a6715aa74e22d7af89ea141753` | 7m48s | Passed | Passed |
| 06 | Full queries | `7a234b4ef2aead88eb283a57d4387c857bb373db` | 8m32s | Passed | Passed |

The coordinator independently reran the same pinned CLI's fixed `check`, `format --check` and `render --check` for each commit. All six committed working trees were clean and each commit changed exactly the four permitted paths. Fixed `impact` in full-query runs conservatively included resources outside the changed paths because adding a resource changes the inventory; this did not represent an edit outside the boundary.

An independent reviewer saw only six neutrally labeled snapshots of the four changed files, without arm, run order, reports, timing, or commit identity. All six received the overall rating *partially satisfied* under the frozen rubric. Every candidate met the Rule identity and binding, prospective framing, duplicate-effect safeguard, both bounds, tests for transient and permanent errors, and source/view consistency. Three candidates met the rubric's requirement for *different* transient and permanent handling; three were rated partial. The task card required handling the two error classes but did not explicitly demand different treatment, so this split cannot be attributed unambiguously to a product or actor failure. The reviewer could not score preservation of `worker-behavior` and its Go input from those four-file snapshots alone, which contributed to every partial overall rating. A separate commit-file-list and hash audit confirmed both unchanged in every candidate: `worker-behavior.yaml` SHA-256 `984b571de3068ca2b3bc2ac0445a43d3cff3282add0c08e9719af733290134f5`; `worker.go` SHA-256 `5252d4852401734a59a2e8df95764fa1454a0d9b2400b64e335d2a5c231006ca`.

The earlier setup attempts were excluded before scoring: checkout newline conversion made the first parallel attempt dirty, and a later candidate lacked the required clean-baseline proof. Both are retained as unscored preparation evidence. In the scored repeat, actors also recovered from path assumptions, missing directories, protected-branch write refusal, Git ownership checks, and in run 05 an initial `uses` relationship where the Rule required `rules`. Run 04 paused after automatic review rejected an authorized temporary-fixture write; the actor resumed only after explicit human approval. Its 19m51s includes an unmeasured approval wait and cannot be used as productive-work duration.

Token counts, monetary cost and per-command timings were unavailable. The six elapsed times include discovery and recovery but were observed on a shared host, and the sample is small. Only two of the three full-query elapsed times are usable for ordinary timing description because run 04 includes approval waiting. The task supplied exact ownership and paths, limiting the discovery value of full queries. These results establish whether each actor completed this synthetic authoring task under its assigned command access; they do not establish a speed or cost advantage, runtime retry behavior, or human acceptance. The task card, rubric, anonymized snapshots and six actor reports remain in the local excluded `verified-sequential` pilot artifacts.

## Deterministic core benchmark

At the same product commit on Windows amd64, Go 1.27.1 and an AMD Ryzen 7 7700X, three repetitions of `go test ./internal/app -run '^$' -bench Authoring -benchmem -count=3` measured:

| Operation | Observed time per operation | Fixture/result |
|---|---|---|
| Context compilation | 164,733–173,283 ns | 482-resource graph; 10 selected entries; 8,021 serialized context bytes |
| Shared-Rule impact | 4,425,656–4,822,289 ns | 481 affected entries |

The benchmark excludes fixture parsing, Git reads and any model interaction. Its fixture initially failed because nine areas lacked an explicit import for the shared Rule. That fixture was corrected and a one-iteration CI smoke check now catches such drift. These measurements describe kernel operations; they do not estimate tokens, full-task latency or sustained savings.
