# Makerspace model snapshots

The canonical Markitect model of the Phase C Makerspace program, one complete snapshot per step. It is derived only from the neutral requirements: [w0.md](../requirements/w0.md) for W0, then each change document applied in order. The single-agent arm's documentation in [spec/](../spec) is rendered from these snapshots, and [mapping/](../mapping) ties every neutral statement to its Definitions.

## Layout

Each step folder (`w0`, `c1` … `c9`) holds `model.yaml`, `model-lock.yaml`, `model/<package>/<id>.yaml` and `files.sha256`, which lists `<sha256>  <path>` for every other file of the folder, sorted. A step is cumulative: `c3` is W0 with C1, C2 and C3 applied. `model.yaml` and `model-lock.yaml` are the same in every step. They bind the Library's dependencies, `markitect-core` 1.1.0 ([testdata/core](../../../../testdata/core)) and `software-basics` 0.2.0 ([testdata/library/vocabulary](../../../../testdata/library/vocabulary)), whose Kinds Concept, Rule and UseCase type every Definition. Arm M's fixture for a step replaces `.markitect/model/` wholesale with that step's `model/`.

## Packages and Managers

Every local Package has a Manager, including the empty intermediate ones, and Root is the 21st ([concept §7.1](../../../../docs/concepts/v1-concept.md#71-the-model-is-the-architecture)).

| Area | Packages (Definitions at W0) |
| --- | --- |
| Shared | `workshop` (13): overview, program, requests, responses, invalid requests, argument values, initial state, results, records, order of checks, refusal codes, order of listed results, delivery |
| Policies | `policies` (empty); `policies/authorization` (2), `policies/audit` (3), `policies/money` (2), `policies/time` (3) |
| Members | `members` (empty); `members/registry` (4), `members/memberships` (4) |
| Training | `training` (7) |
| Equipment | `equipment` (empty); `equipment/catalog` (5), `equipment/maintenance` (3) |
| Bookings | `bookings` (9) |
| Materials | `materials` (5) |
| Billing | `billing` (empty); `billing/invoices` (5), `billing/payments` (4) |
| Safety | `safety` (4) |
| Reporting | `reporting` (3) |

| Step | Definitions | Operations | Packages | Managers | Canonical model digest |
| --- | --- | --- | --- | --- | --- |
| w0 | 75 | 37 | 20 | 21 | `sha256:ae65df902f01047c7fd2edb5fa23e2e750034de9a67746527a65346a8a1e0173` |
| c1 | 75 | 37 | 20 | 21 | `sha256:502965d0df7af55c8fe3fa228427d2d8186607aa03ae6bf061641ccee5fa29d6` |
| c2 | 77 | 38 | 20 | 21 | `sha256:05e964fd962e2a3d4446f6f7541a2be8a0250a4ef62b54d6758b5ce11d2e3b4e` |
| c3 | 78 | 38 | 20 | 21 | `sha256:0a387d122c123a849e3deb6368a50e65117c4ba8fc1839116edae9c87d6479ee` |
| c4 | 78 | 38 | 20 | 21 | `sha256:cda3bfabcbdad71e7578c541bdc66b646e8ae0cf9ea08158bf765d4cb6c9e725` |
| c5 | 79 | 38 | 20 | 21 | `sha256:4ee96737ad4e3388186c0dc129488ea6aaf70201e887fa83ed1eae7f11ca95c6` |
| c6 | 79 | 38 | 20 | 21 | `sha256:e85d05c9f65e1040176090ae7a8a90316a89f9b8614af934067c7391a12abe90` |
| c7 | 79 | 38 | 19 | 20 | `sha256:738584fbe33a3a158ae05558263b3e14dd822cfe1680bcfc6688a31f39c89828` |
| c8 | 79 | 38 | 19 | 20 | `sha256:833beece8949ced69a9fcaf45464e4f7a82c690b2cf6960ad2604871b4c6a99a` |
| c9 | 79 | 38 | 19 | 20 | `sha256:833beece8949ced69a9fcaf45464e4f7a82c690b2cf6960ad2604871b4c6a99a` |

C7 removes `billing/payments`: its four Definitions move to `billing/invoices` with unchanged local IDs, so their identities and every link to them change. C9 is byte-identical to C8.

## How the text became Definitions

- **Bodies keep the neutral wording.** One Definition per neutral subsection, record table, rule or operation. Section references became body links. The five phrases that name the document ("of this document", "in this document", "agree with this document") read "the model" here; the renderer substitutes them back. Numbers stay in prose: there are no typed policy Kinds or Properties.
- **Every operation is a UseCase** that opens with "Operation `x`. Request processing and delivery apply." The two links name the rules that W0 states for every operation: the order of checks (§3.1) and the delivery and test obligations (§15). The Library model did the same with its shared input rule.
- **Cross-cutting policies reach every operation's Manager.** A Manager's context is its own Definitions, their direct link targets and the local Definitions that link to them ([select_context.go](../../../../internal/organization/select_context.go)). The operation index (§14) is therefore split into the permission matrix in `policies/authorization` and the audit target list in `policies/audit`. Both list and link every operation, so each operation's Manager sees both through incoming links. The money and time rules, the record table, the order of listed results and the cross-area rules (good standing, certification, maintenance, lockout, overdue block, charges, credit balance, opening hours and granularity) are linked where the text uses them.
- **Not modeled:** the W0 paragraph that frames the document, change-relative wording such as "Until now …", and "What stays as it is" statements, which are change control. The few clarifications in those sections that state a rule are carried; [mapping/](../mapping) lists each case.

## Validation

Each snapshot was checked structurally in a throwaway fixture repository on 25 October 2026, with `markitect` built from Main `528ef346d3149002f53bf8eca1c54ea5613453ed`. The fixture's `.markitect/` held the snapshot's `model.yaml`, `model-lock.yaml` and `model/`, the two dependencies as `dependencies/software` and `dependencies/core`, an `execution.yaml` with only `ownership: [{path: ./, owner: .}]` and an empty registry shard, all in one commit. Then:

```text
markitect check --repo FIXTURE --model .markitect/model.yaml --dependency .markitect/dependencies/software/model.yaml --dependency .markitect/dependencies/core/model.yaml --revision HEAD
```

All ten steps returned exit code 0, status `passed`, with the Package and Manager counts and digests above. This checks structure only: envelope, names, imports, links and the lock binding. It does not show that the model says what the neutral text says; the [mapping](../mapping) and the independent review cover that. Arm M's real fixture also needs its own `execution.yaml`, with ownership per Package and the project check; that belongs to the comparison fixture, not to this folder.

## Rendering and checks

```text
python -I -B tools/comparison/phase-c/spec/render.py write-all
python -I -B tools/comparison/phase-c/spec/render.py check-all
python -B -m unittest discover -s tools/comparison
```

`check-all` proves that each committed `spec/requirements-<step>.md` is the current rendering and contains every body verbatim. The unit tests also check every snapshot's `files.sha256`, that C9 equals C8, and the main content of each step.
