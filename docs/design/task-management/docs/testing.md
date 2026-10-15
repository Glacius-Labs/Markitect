# Quality assurance

## Current evidence

This is a test strategy and acceptance plan. No application code, test suite, Windows/Linux execution, performance result or user observation is supplied yet. Documentation review can identify contradictions; it does not prove the software exists or works.

## Test levels

| Level | Purpose | Isolation |
|---|---|---|
| Domain | Text, reference and state meaning. | Pure functions and synthetic values. |
| Task actions | Action sequencing, outcomes and no-write branches. | Controlled load/save functions; no real user data. |
| File adapter | Strict JSON validation, safe replacement and failure stages. | Temporary directories and deterministic failure injection. |
| CLI | Real invocation, parsing, output streams and status. | Child processes using the built source package and explicit temporary `--file` paths. |
| Restart integration | Actual persistence across separate processes. | Real file adapter and known temporary files. |
| Review/walkthrough | Architecture, documentation usability and intended benefit. | An exact candidate source version and synthetic tasks. |

The planned entrypoint is `python -m unittest discover -s tests -p "test_*.py" -v` from the future source root. The [development guide](../CONTRIBUTING.md) owns setup. Python's [unittest command-line interface](https://docs.python.org/3.13/library/unittest.html#command-line-interface) is the runner reference. Expected results below are test oracles, not observed passes.

## Required scenarios

| Area | Cases and expected observation |
|---|---|
| Creation | First reference, next reference, open state, Unicode text and duplicate wording. All previously stored tasks remain unchanged. |
| Text validation | Empty/whitespace-only text; control, surrogate and Unicode line/paragraph separators; trimming before persistence; preserve internal spacing/case. Invalid data is refused before file access. |
| Listing | Missing/empty file, all tasks, completed tasks, numeric ordering, exact tab-separated fields and no write. |
| Completion | Open task, already completed task, unknown valid reference, missing/empty file. No write for repeated completion or refusal. |
| Reference boundaries | Noncanonical syntax, zero/negative/overflow; allocate after the highest ID. With the maximum ID stored, adding is refused while list/complete still work. |
| CLI | Missing/extra arguments, unknown command/option, repeated/misplaced `--file`, empty path and quoted spaces; `add --` with dash-prefixed text and a literal `--`, refusal without the delimiter, dash-prefixed filenames through `./`; exact output channels, messages and exit statuses. |
| File selection | Stable default under a controlled test home, relative/absolute overrides, independent lists, absent home resolution and parent-directory behavior. Never use the developer's real default file. |
| Strict data | Invalid UTF-8/BOM, invalid JSON, duplicate/unknown/missing keys, booleans/floats as integers, unsupported version, duplicate IDs, invalid text/state and trailing data. Whole-file refusal without mutation. |
| Safe save | Failure during preparation, serialization, writing, flush/synchronization and replacement. Before commit, existing bytes stay identical; an absent selected file stays absent. Default directory may remain empty. |
| Commit/output | After replacement succeeds, simulated output failure does not roll back data. An output error does not justify automatically repeating Add. |
| Privacy/scope | Source review finds no network call, telemetry, task-content logs or unintended domain features. Task text is data, not evaluated code. |

For strict-format tests, include both valid-format fixtures and invalid bytes; a fake repository cannot prove the real decoder. For storage safety, test at least one real adapter failure before replacement and one successful replacement on each target. Permission-only tests are insufficient if they pass merely because the process has elevated rights; deterministic injection identifies the actual stage. Cleanup failures must not turn a committed state into a false pre-commit failure.

## End-to-end persistence case

1. In a temporary home or with an explicit new file, list and confirm no data file was created.
2. Add `Buy milk`, exit, start another process and verify reference 1, text and open state.
3. Add identical text and verify a distinct reference.
4. Complete reference 1, exit, list from another process and verify both tasks and their states.
5. Repeat completion and verify no file write and the documented no-change response.
6. Request a valid unknown reference and check refusal without mutation.
7. Repeat with an explicitly selected relative file and an absolute file path containing spaces.

Also prepare a valid file with a reference larger than a 32-bit integer, then list, complete, add the next reference and restart. This catches truncation in parsing, formatting and storage. Exercise the declared maximum separately without trying to create that many tasks.

## Architecture and documentation review

Review module responsibilities and the actual import/call direction. Confirm CLI code does not decode JSON, task logic does not print and storage code does not decide action policy. Reconcile command behavior, examples, file format and error handling across affected documents. Keep a requirement in its natural owning document rather than adding an independent checklist of competing rules.

Ask a person to set up the application from the README, record two synthetic tasks, complete one and identify outstanding work. Record misunderstandings and whether the error/recovery instructions were actionable. This is limited usability evidence, not proof of productivity improvement.

## Release readiness

Before a release:

- The implementation matches the documented scope, CLI, data format and safety boundaries; no unresolved data-loss or contract issue remains.
- Syntax/import checking and the complete automated suite pass on Python 3.13.x on Windows 11 and Ubuntu 24.04. Record each platform result separately.
- The source archive is unpacked and the documented setup/run/test/example procedures succeed from that clean package.
- Commands are measured with the reproducible 1,000-task fixture from Product; record the environment/results and resolve any unacceptable interaction delay before release. No numerical latency guarantee is implied by the measurement.
- The data/recovery and user walkthroughs have recorded outcomes; architecture and changed-document reviews are complete.
- Version, changelog, archive filename and checksum agree. No user data or caches are included.
- The project owner explicitly decides release readiness from the evidence. A passing suite alone is not that decision.

Record exact source revision, Python/platform versions, invocation, result, limitations and relevant output. An unavailable target is an evidence gap, not a pass. There is no release approval recorded in this design delivery.
