# Makerspace model snapshots

> **Note, 26 October 2026:** since [DEC-075](../../../../docs/concepts/register.md#dec-075-kind-extension-and-one-core-120) (Core 1.2.0), `testdata/core` and `software-basics` on Main hold the new editions. These snapshots bind Core 1.1.0 and software-basics 0.2.0 as of the Phase C materials freeze `4840fb20f52c1e0e0367e909d833919ca8a215c6`, and are not checkable with the current compiler. Nothing else here changed.

The canonical Markitect model of the Phase C Makerspace program, one complete snapshot per step. It is derived only from the neutral requirements: [w0.md](../requirements/w0.md) for W0, then each change document applied in order. The single-agent arm's documentation in [spec/](../spec) is rendered from these snapshots, and [mapping/](../mapping) ties every neutral statement to its Definitions.

## Layout

Each step folder (`w0`, `c1` … `c9`) holds `model.yaml`, `model-lock.yaml`, `model/<package>/<id>.yaml` and `files.sha256`, which lists `<sha256>  <path>` for every other file of the folder, sorted. A step is cumulative: `c3` is W0 with C1, C2 and C3 applied. `model.yaml` and `model-lock.yaml` are the same in every step. They bind the Library's dependencies, `markitect-core` 1.1.0 ([testdata/core](../../../../testdata/core)) and `software-basics` 0.2.0 ([testdata/library/vocabulary](../../../../testdata/library/vocabulary)), whose Kinds Concept, Rule and UseCase type every Definition. Arm M's fixture for a step replaces `.markitect/model/` wholesale with that step's `model/`.

## Packages and Managers

Every local Package has a Manager, including the empty intermediate ones, and Root is the 21st ([concept §7.1](../../../../docs/concepts/v1-concept.md#71-the-model-is-the-architecture)).

| Area | Packages (Definitions at W0) |
| --- | --- |
| Shared | `workshop` (14): overview, program, requests, responses, invalid requests, argument values, initial state, results, computed values, records, order of checks, refusal codes, order of listed results, delivery |
| Policies | `policies` (empty); `policies/authorization` (2), `policies/audit` (3), `policies/money` (2), `policies/time` (3) |
| Members | `members` (empty); `members/registry` (4), `members/memberships` (4) |
| Training | `training` (7) |
| Equipment | `equipment` (empty); `equipment/catalog` (5), `equipment/maintenance` (3) |
| Bookings | `bookings` (9) |
| Materials | `materials` (5) |
| Billing | `billing` (empty); `billing/invoices` (5), `billing/payments` (4) |
| Safety | `safety` (4) |
| Reporting | `reporting` (3) |

Revision 1, as merged in [PR #707](https://github.com/Glacius-Labs/Markitect/pull/707) (Main `fed1fe22`); the folders now hold [revision 2](#revision-2-computed-values-and-limits):

| Step | Definitions | Operations | Packages | Managers | Canonical model digest |
| --- | --- | --- | --- | --- | --- |
| w0 | 75 | 37 | 20 | 21 | `sha256:08187407dd8a775355fc6986d7f3403037b6fbeb4ed2cf52c506ce0056b3a1e1` |
| c1 | 75 | 37 | 20 | 21 | `sha256:e78a32ad243098ec80eb679b0b343fd6be08ae3b73a9cad54b3216a6d71a3f88` |
| c2 | 77 | 38 | 20 | 21 | `sha256:e614543d665b27b08de262288f15c52316a5740afeb4dd512ecd4ae0f961d7e5` |
| c3 | 78 | 38 | 20 | 21 | `sha256:5b18ab4736e8dde0fde31d88361d152de367bc8e8067675f575eaff55d117f09` |
| c4 | 78 | 38 | 20 | 21 | `sha256:e051ade25c07a7e88adb0ae60949ff11326c2e255c52a538f4cb9e7a8c719590` |
| c5 | 79 | 38 | 20 | 21 | `sha256:2ef85cdac6b6cf2df6e16745f1b4c76ef36e7883e9769308836389b1b68fbe1a` |
| c6 | 79 | 38 | 20 | 21 | `sha256:92679189d5ab9146636e75905f38315677202b80ce13f71c03b550c7654b2275` |
| c7 | 79 | 38 | 19 | 20 | `sha256:21ea3d12b4fa3ae6988a292e1c42ebd80135c92492500c71ff3878601361a0cd` |
| c8 | 79 | 38 | 19 | 20 | `sha256:33cd40acaf29ae2f4dec099d9d4bfc2642549b4d0e41befc792ce440fab9de34` |
| c9 | 79 | 38 | 19 | 20 | `sha256:33cd40acaf29ae2f4dec099d9d4bfc2642549b4d0e41befc792ce440fab9de34` |

C7 removes `billing/payments`: its four Definitions move to `billing/invoices` with unchanged local IDs, so their identities and every link to them change. The empty `billing` Package stays as the parent of `billing/invoices`; flattening billing into one `billing` Package would also change the identities of the five invoice Definitions, so the fold keeps the restructure to the four moved ones. C9 is byte-identical to C8.

## Revision 2: computed values and limits

Requirements [PR #713](https://github.com/Glacius-Labs/Markitect/pull/713) (Main `8a2231f1`) closed a W0 gap found by the uncounted smoke run. It inserted a paragraph at the end of w0 §2.5, the new §2.8 "Computed values and limits", and one statement each in c1, c2, c5 and c6. Requirements [PR #717](https://github.com/Glacius-Labs/Markitect/pull/717) (Main `a94c46f1`) then inserted two more W0 statements: the refusal for an `actor` that is not a valid identifier (after §2.5) and how nesting depth is counted (a §2.8 bullet). The model follows from that text only:

- **W0:** the new Rule `workshop/computed-values` carries §2.8. Its links reach the Definitions whose computed values it bounds: invoices, certifications, reports, maintenance windows, stock, credit, metering, money, the calendar of the time rule, generated identifiers, audit targets and the program limits. `workshop/argument-values` gains the two §2.5 paragraphs, and `workshop/computed-values` includes the "Counting depth" bullet.
- **C1, C2, C5 and C6:** their new statements amend `workshop/computed-values`.
- **C3, C4, C7, C8 and C9:** no new text; they inherit the W0 change and the earlier delta edits. At C7 the new Rule's link to `record-payment` moves from `billing/payments` to `billing/invoices` with the fold.

Against revision 1, every step adds one Definition and changes `workshop/argument-values`; [mapping/r2.md](../mapping/r2.md) lists the comparison per step. Packages, Managers and the execution configuration are unchanged. The single-agent documents were re-rendered; the renderer adds the section "Computed values and limits" and two listed substitutions ("rules of the model", "formula in the model" → "this document").

| Step | Definitions | Operations | Managers | Canonical model digest | SHA-256 of `files.sha256` |
| --- | --- | --- | --- | --- | --- |
| w0 | 76 | 37 | 21 | `sha256:cfb7a9fea665827567aa26d63050b52d943d091689ce077d38c3b7138aa1dd13` | `1f6522589460cbe77a64413912f073c0e79aceaa03517d82046d073e61d9ee54` |
| c1 | 76 | 37 | 21 | `sha256:f7a8bd292010908c64b972bd92965fb669fe57a02b88f5fa85a2863d87dab23e` | `82a06c012c7ea1382aa6575dc54a9faaa254835e8c4f9d96d3d762e02f4dfcef` |
| c2 | 78 | 38 | 21 | `sha256:daf68db95376cbcab4625fee945604691275c8d87ee9759deeb4d70d706bbd6e` | `76b8c0ad745236dba4ad60a1417619d9ce0c0e8dec5195aba2735d32ddd0b5d8` |
| c3 | 79 | 38 | 21 | `sha256:d9d631c784233e488fe7fb77952e87d0bfbf1496e49b44acf875f7170ca2fb1b` | `ac3d125d6990b14f5c1a1ffa4c1508ad91760090577f6d915efb0cf8cf402232` |
| c4 | 79 | 38 | 21 | `sha256:e1f3acb6aef5ec24124c3c9aeb04fc2d28de7d8552b07faa31bb58c51ce07365` | `7efb664e91b16866c6421b7492e94ea7954238ea6ac491fce748c00445bafe8c` |
| c5 | 80 | 38 | 21 | `sha256:5e61dc5212efa813f9263aec352c5ec1034d410aac9fc9f65276598fae3dcaab` | `4c4d5544fca64b709d750b905f65cc0d6b349c9b4a5bb937d363ff6dcf0221e1` |
| c6 | 80 | 38 | 21 | `sha256:3a081c995f9ef5ab84522796b40d1016f72cd8653e801a270b6b78a024a926b1` | `027611c9d29ac05b502e44b71e835937197e9f3c7b145ee2aaa8acfcdaf07995` |
| c7 | 80 | 38 | 20 | `sha256:4fb0c37a7ed795286130c78d417cd39a264cc94585c332c694b18f09d7da08de` | `f2a09010d45f3be05a1c94770774bd7a518dfb9346a77ff5f54b94b575658aec` |
| c8 | 80 | 38 | 20 | `sha256:2505dc4bc3be83a25072569cabf235b64a6dced6497514368a31ef032ff0d6eb` | `cd1df82af94fd2de2ba4ede79311464f52c228a0bae03fa18e1e3b467ae58792` |
| c9 | 80 | 38 | 20 | `sha256:2505dc4bc3be83a25072569cabf235b64a6dced6497514368a31ef032ff0d6eb` | `cd1df82af94fd2de2ba4ede79311464f52c228a0bae03fa18e1e3b467ae58792` |

**Validation of revision 2:** each snapshot passed the structural check described under [Validation](#validation) on 25 October 2026, this time with its proposed [execution configuration](#execution-configuration-for-arm-m) in place of the ownership-only file, and with `markitect` built from Main `3f065d4323062ecceda922ace988f7ad6a8aed47`. All ten returned exit code 0 and status `passed`, with the digests above.

## How the text became Definitions

- **Bodies keep the neutral wording.** One Definition per neutral subsection, record table, rule or operation. Section references became body links. The seven phrases that name the document ("of this document", "in this document", "agree with this document", "rules of this document", "formula in this document") read "the model" here; the renderer substitutes them back. Numbers stay in prose: there are no typed policy Kinds or Properties.
- **Every operation is a UseCase** that opens with "Operation `x`. Request processing, results and delivery apply." The three links name the rules that W0 states for every operation: the order of checks (§3.1), the shape of results (§2.7) and the delivery and test obligations (§15). The Library model did the same with its shared input rule.
- **Policy reach.** A Manager's context is its own Definitions, their direct link targets and the local Definitions that link to them ([select_context.go](../../../../internal/organization/select_context.go)). The operation index (§14) is split into the permission matrix in `policies/authorization` and the audit target list in `policies/audit`. Both list and link every operation, so each operation's Manager sees its permissions and audit target through incoming links. The authorization and audit rule bodies (the meaning of "own", the entry fields, the recording of refusals) reach the order of checks in `workshop` and their own Managers, but not every area Manager directly: the model does not say whether checks and auditing are realized centrally or per operation, which stays a realization choice. The money and time rules, the record table, the order of listed results and the cross-area rules (good standing, certification, maintenance, lockout, overdue block, charges, credit balance, opening hours, granularity and interval overlap) are linked where the text uses them.
- **Not modeled:** the W0 paragraph that frames the document, change-relative wording such as "Until now …" or "still", and "What stays as it is" statements, which are change control. Clarifications in those sections that state a rule are carried, and so is C8's "including dates before the change", which carries a rule; [mapping/](../mapping) lists each case.

## Validation

Each snapshot was checked structurally in a throwaway fixture repository on 25 October 2026, with `markitect` built from Main `528ef346d3149002f53bf8eca1c54ea5613453ed`. The fixture's `.markitect/` held the snapshot's `model.yaml`, `model-lock.yaml` and `model/`, the two dependencies as `dependencies/software` and `dependencies/core`, an `execution.yaml` with only `ownership: [{path: ./, owner: .}]` and an empty registry shard, all in one commit. Then:

```text
markitect check --repo FIXTURE --model .markitect/model.yaml --dependency .markitect/dependencies/software/model.yaml --dependency .markitect/dependencies/core/model.yaml --revision HEAD
```

All ten steps returned exit code 0, status `passed`, with the Package and Manager counts and digests above. This checks structure only: envelope, names, imports, links and the lock binding. It does not show that the model says what the neutral text says; the [mapping](../mapping) and the independent review cover that.

## Execution configuration for arm M

[execution/](../execution) proposes arm M's `.markitect/execution.yaml`, in the form of the Library fixture that [run-library-apply.ps1](https://github.com/Glacius-Labs/Markitect/blob/d4bc68ed9f6d808f807f616f5a122c90112c3ee6/scripts/run-library-apply.ps1) writes and that Phases A and B used. It holds nothing but Package ownership and the project checks:

| File | Steps | Content |
| --- | --- | --- |
| [w0.yaml](../execution/w0.yaml) | W0–C6 | Root owns `./`; every Package `P`, including the empty intermediate ones, owns `realization/P/`; `workshop` also owns `realization/cmd/makerspace/` |
| [c7.yaml](../execution/c7.yaml) | C7–C9 | The same without `billing/payments` |

Both declare two Root checks on the Go profile `go` of the [runtime-profile example](../../../../examples/runtime-profile.example.yaml): `project-build` runs `go build ./cmd/makerspace` (required by `workshop/program`, 2 minutes) and `project-tests` runs `go test ./...` (required by `workshop/delivery`, 5 minutes, more than the Library's 2 because the program is about five times larger), both from `realization/`. The directory names follow the Package paths. The Go package names and the source layout inside each directory remain the implementer's choice.

Each step's snapshot passes the same structural check with its configuration. C7 with the W0 configuration fails (`execution.package`: `billing/payments` no longer exists), so arm M's C7 fixture commit replaces `execution.yaml` together with the model. The registry stays as in the Library fixture: one empty Root shard, `apiVersion: markitect.registry/v1` with `artifacts: []`.

## Rendering and checks

```text
python -I -B tools/comparison/phase-c/spec/render.py write-all
python -I -B tools/comparison/phase-c/spec/render.py check-all
python -B -m unittest discover -s tools/comparison
```

`check-all` proves that each committed `spec/requirements-<step>.md` is the current rendering and contains every body verbatim. The unit tests also check every snapshot's `files.sha256`, that C9 equals C8, and the main content of each step.
