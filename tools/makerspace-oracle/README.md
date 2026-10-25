# Makerspace oracle

Hidden Phase C oracle: behavior cases and probes for the Makerspace program, written only from the [neutral requirements](../comparison/phase-c/requirements/README.md). Assessor material; no writer receives it.

The author saw only `tools/comparison/phase-c/requirements/` at Main `795b8a2a` and the Library oracle; not the Markitect model, the rendered spec, the mapping, the reference implementations or any candidate. Every case and every probe family cites the requirement sentences it checks, as they stand at the step under test: a change's own sentences from that change on, w0 sentences until a change voids them (`facts.VOIDED`), and w0 section 11 as c7 restates it from C7. A self-test proves each cited fragment occurs verbatim in the named section.

## Files

| File | Use |
| --- | --- |
| [facts.py](facts.py) | The requirements as plain data per step: operation index (permissions, audit targets, argument types), fees and discounts, bulk table, opening hours, granularity and the record shapes of every step. |
| [builder.py](builder.py) | Case builder: explicit requests with explicit expected responses, deterministic generated identifiers, the expected audit log, user roles, the request-time order and unchanged-state snapshots around refusals. Authoring guards refuse a case that goes back in time without expecting `time_regression`. |
| [world.py](world.py) | The shared starting world (Monday 2026-03-02) with fully stated results. |
| [behavior.py](behavior.py) | Strict behavior cases: B-* per business area of W0, D-C1 to D-C8 per change. Every result is compared in full and the audit log is compared at the end. |
| [probes.py](probes.py) | The 31 probe families, one rule cluster each (`FAMILIES` holds names and neutral references), including the end-of-calendar cases of w0 2.8. |
| [invariants.py](invariants.py) | Ten cross-area invariant checks over a mixed scenario; the scenario's statuses are judged once (INV-SCENARIO), each check fails only its own family. |
| [census.py](census.py) | The expected change census for C3, C7 and C9 and a differential runner. |
| [catalog.py](catalog.py) | The cumulative case set of a step. |
| [harness.py](harness.py) | Matcher, case runner, report and the container entry point. |
| `test_*.py` | Offline self-tests. |

## Reuse of the Library oracle

Only [driver.py](../library-oracle/driver.py) is reused, loaded unchanged from its file (`Process`, `Limits`, `TransportFailure`): the Makerspace program speaks the Library program's envelope. Nothing in `tools/library-oracle` changes, so its hashes, its CI step and the frozen Phase A and B revisions stay as they are. The Library's `run_case` and matcher are bound to library records (unordered lists keyed by `loanId`, its snapshot model), so `harness.run_case` mirrors their report shape instead: one fresh process per case, no retry, stop at the first failing step, exact request and response bytes in the trace.

## Case form

A case is the Library form `{id, families, steps}` plus `cite`; a step is `{request | raw, expect, capture?, cite?, snapshot?}`. Strict cases state generated identifiers literally; probe cases capture them (`capture` names a path in the response) and refer to them, so a probe family depends only on its own rule: numbering is P-IDENTIFIERS' subject, and money families compare their own lines (`$filter`) and fields, reading invoice balances first where a payment depends on them. Responses must have exactly the stated members (requirements README, "Transport"), JSON types are strict and lists are ordered (w0 4.5). Expectation tokens:

| Token | Meaning |
| --- | --- |
| `{"$nonempty": true}` | a string with at least one non-whitespace character (`reason`) |
| `{"$any": true}` | any value; probe setups assert only `status: ok` |
| `{"$has": {...}}` | an object with at least these members, compared exactly |
| `{"$record": k}` / `{"$same": k}` | remember a value / require it unchanged (the no-change snapshots) |
| `{"$extends": k, "tail": [...]}` | the recorded list plus these items; `"seq": {"$next": true}` numbers them after it |
| `{"$capture": k}` | remember this value for later steps |
| `{"$ref": k}` / `{"$calc": [op, ...]}` | a captured value, or `+`, `-` or `min` over captured values and integers; also usable in request arguments |
| `{"$filter": {field: value}, "items": [...]}` | only the list items with these field values, in their order |

## Families and counts

Probe families (calibration: one mutant each), behavior families, the scenario statuses (INV-SCENARIO) and each invariant check count as one hidden case family in the report's `families` and `failed_families`. The probe families and their references are in [probes.py](probes.py) `FAMILIES`; the list sent to session I is the same.

| Step | Behavior cases | Probe families | Probe cases | Invariant checks | Distinct requests |
| --- | --- | --- | --- | --- | --- |
| W0 | 7 | 30 | 36 | 10 | 3746 |
| C1 | 8 | 30 | 36 | 10 | 3759 |
| C2 | 9 | 31 | 37 | 10 | 3979 |
| C3 | 10 | 31 | 37 | 10 | 4005 |
| C4 | 11 | 31 | 37 | 10 | 4093 |
| C5 | 12 | 31 | 38 | 10 | 4163 |
| C6 | 13 | 31 | 38 | 10 | 4188 |
| C7 | 13 | 31 | 38 | 10 | 4188 |
| C8 | 14 | 31 | 38 | 10 | 4220 |
| C9 | 14 | 31 | 38 | 10 | 4220 |

P-MONEY-LOCKOUT starts at C2. P-LIMITS (request size and depth, 129-character audit targets, the largest line amounts and century-long windows, w0 2.8) was added with the computed-value rules; the end-of-calendar cases of w0 2.8 (due dates and expiry dates in 2100, the last invoiceable month, the last utilization day, the block point) belong to P-TIME-BILLING, P-TIME-CERTIFICATION, P-REPORT-UTILIZATION and P-TIME-FORMAT. Every case of a step also runs at every later step, rendered for that step (for example, certification records lose `expiresOn` from C6 and opening hours change from C8).

Change census (`census.py`): at C3 every hidden case without an accepted material purchase of 10 or more units must answer as the C2 program does, apart from the free `reason` wording; at C7 every case must answer as the C6 program does, apart from `reason`; at C9 every case must answer exactly as the C8 program does, `reason` included. The path census per arm stays with the comparison assessor.

## Run

Self-check, from the repository root (on Windows set `TMP` and `TEMP` to a user temp directory first):

```text
python -B -m unittest discover -s tools/makerspace-oracle -p "test_*.py"
```

The tests materialize every case's expected transcript at every step and replay it through the matcher, replay chosen cases through the real Library transport with tiny replay processes, plant faults that must fail, check every citation against the requirements, check that the strict cases satisfy all ten invariants and that each invariant catches a planted inconsistency, and check that the census selects only cases the change leaves alone. They start no program, container or provider.

An independent review (25 October 2026) ran every case at every step against a simulator written from the requirements alone, plus about 120 plausible implementation faults. It found one false failure (the billing-once check ignored the member) and coverage gaps in metering, C2 lockout scope and minutes, invoice months, schedule order, `-0` and utilization above 100 %; all are fixed here. The simulator stays outside the repository.

Against an already built program (local calibration):

```text
python -B tools/makerspace-oracle/harness.py --step W0 --program /absolute/makerspace --cwd /absolute/realization [--census-start /absolute/start-program]
```

Inside the assessment container, with the mounts of [assess_candidate.py](../comparison/assess_candidate.py) (`/evaluator` the Library oracle, `/harness` this folder, `/out/makerspace` the built `realization/cmd/makerspace`):

```text
python -I /harness/harness.py --step C3 --source S --commit C --limits JSON --protected JSON [--census-start /out/start]
```

`--protected` binds the reused `driver.py` and every non-test file here, checked before and after the run. Building the program and wiring this entry point into the comparison assessment belong to the comparison assessor.

## Readings and ambiguities

Tested only where the requirements are unambiguous. Points for session V and the coordinator:

1. **`actor` with a string that is not a valid identifier** (for example `"bad id!"`): `invalid_argument` if `actor` counts as an identifier argument (w0 2.2, 2.5, 3.1 check 1), or `unknown_actor` (3.1 check 2). No argument list types `actor`. Not tested; missing, `null` and wrong JSON types (2.5, closing paragraph) and unknown actors are.
2. **`at` read as a timestamp argument**: a malformed or out-of-range `at` is tested as `invalid_argument` (w0 2.5 timestamp row, 4.4 first bullet). Session V confirmed this reading.
3. **Reads advance the request-time order**: w0 4.4 says "the most recent request answered `ok`", and c1 repeats it, so an accepted read moves the point too. Tested; easy to miss, not ambiguous; session V confirmed it.
4. **Census at C3 and C7 ignores `reason`**: its wording is free (w0 2.3), and neither change fixes it. At C9 it is compared, because the requirements want no code change.
5. Not testable or not tested: a last line without a line feed (the reused transport always sends one), duplicate member names and invalid UTF-8 (w0 2.4 excludes them), and characters whose Unicode `White_Space` property changed between versions (only stable whitespace is tested). The 65,536-byte line of w0 2.8 is tested exactly.
6. **Nesting depth at exactly 100**: w0 2.8 leaves values nested "more than 100 levels deep" unspecified but does not say whether the request object itself counts as a level. Only 90 levels are tested, which is inside the limit under either reading.
7. Reachable only with impractically many requests, so not tested: generated identifiers of 20 or 21 characters and integers near 2^53 − 1 (w0 2.8). The largest values tested are a 52,592,519-minute usage line of 525,925,190,000 cents and a material line of 10^10 cents.

## Limits

Behavior through the public interface only. The process transport filters the environment but is not a sandbox; Agent-produced code runs only in the hardened assessment container. The census needs both the start and the candidate program. The self-tests are not part of CI, like the other comparison tools; they need no engine.
