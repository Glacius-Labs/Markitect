# Documentation authoring pilot

Recorded 2026-10-01. **Incomplete exercise:** one check-only run and one full-query run completed. The third actor stopped at a provider usage limit before making changes; three remaining runs were not started. The planned three runs per arm and blind semantic review did not complete. No comparative productivity, token, or cost conclusion follows.

[Measurement](measurement.md) owns the procedure. The earlier [core authoring exercise](authoring-assessment.md) is separate historical evidence.

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

The raw protocol, scorer, actor reports and local candidate repositories are retained in excluded development artifacts. They are local exercise evidence, not hosted release attestations. Complete repeated runs under a consistent model allowance before comparing the arms.

## Deterministic core benchmark

At the same product commit on Windows amd64, Go 1.27.1 and an AMD Ryzen 7 7700X, three repetitions of `go test ./internal/app -run '^$' -bench Authoring -benchmem -count=3` measured:

| Operation | Observed time per operation | Fixture/result |
|---|---|---|
| Context compilation | 164,733–173,283 ns | 482-resource graph; 10 selected entries; 8,021 serialized context bytes |
| Shared-Rule impact | 4,425,656–4,822,289 ns | 481 affected entries |

The benchmark excludes fixture parsing, Git reads and any model interaction. Its fixture initially failed because nine areas lacked an explicit import for the shared Rule. That fixture was corrected and a one-iteration CI smoke check now catches such drift. These measurements describe kernel operations; they do not estimate tokens, full-task latency or sustained savings.
